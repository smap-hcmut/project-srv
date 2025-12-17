package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"smap-project/config"
	"smap-project/config/postgre"
	"smap-project/internal/httpserver"
	"smap-project/pkg/discord"
	"smap-project/pkg/encrypter"
	"smap-project/pkg/log"
	"smap-project/pkg/rabbitmq"
	pkgRedis "smap-project/pkg/redis"
	"syscall"
)

// @title       SMAP Project Service API
// @description SMAP Project Service API documentation.
// @version     1
// @host        smap-api.tantai.dev
// @schemes     https
// @BasePath    /project
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by Identity service /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// Register graceful shutdown
	registerGracefulShutdown(logger)

	// Initialize encrypter
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)

	// Initialize PostgreSQL
	ctx := context.Background()
	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL: ", err)
		return
	}
	defer postgre.Disconnect(ctx, postgresDB)
	logger.Infof(ctx, "PostgreSQL connected successfully to %s:%d/%s", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

	// Initialize RabbitMQ
	amqpConn, err := rabbitmq.Dial(cfg.RabbitMQ.URL, true)
	if err != nil {
		logger.Error(ctx, "Failed to connect to RabbitMQ: ", err)
		return
	}
	defer amqpConn.Close()

	// Initialize Redis (Main - DB 0: job mapping, pub/sub)
	mainRedisOpts := pkgRedis.NewClientOptions().SetOptions(cfg.Redis)
	mainRedisClient, err := pkgRedis.Connect(mainRedisOpts)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis (main): ", err)
		return
	}
	defer mainRedisClient.Disconnect()
	logger.Infof(ctx, "Redis (main) connected successfully to DB %d", cfg.Redis.DB)

	// Initialize Redis (State - DB 1 for project progress tracking)
	stateRedisOpts := pkgRedis.NewClientOptions().SetOptions(cfg.Redis).SetDB(cfg.Redis.StateDB)
	stateRedisClient, err := pkgRedis.Connect(stateRedisOpts)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis (state): ", err)
		return
	}
	defer stateRedisClient.Disconnect()
	logger.Infof(ctx, "Redis (state) connected successfully to DB %d", cfg.Redis.StateDB)

	// Initialize Discord
	discordClient, err := discord.New(logger, &discord.DiscordWebhook{
		ID:    cfg.Discord.WebhookID,
		Token: cfg.Discord.WebhookToken,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize Discord: ", err)
		return
	}

	// Initialize HTTP server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		// Server Configuration
		Logger:      logger,
		Host:        cfg.HTTPServer.Host,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		// Database Configuration
		PostgresDB: postgresDB,

		// // Storage Configuration
		// MinIO: minioClient,

		// // Message Queue Configuration
		AmqpConn: amqpConn,

		// Redis Configuration
		MainRedisClient:  mainRedisClient,
		StateRedisClient: stateRedisClient,

		// Authentication & Security Configuration
		JwtSecretKey: cfg.JWT.SecretKey,
		CookieConfig: cfg.Cookie,
		Encrypter:    encrypterInstance,
		InternalKey:  cfg.InternalConfig.InternalKey,

		// External Services
		Discord:   discordClient,
		LLMConfig: cfg.LLM,

		// Dry-Run Configuration
		DryRunSamplingConfig: cfg.DryRunSampling,
	})
	if err != nil {
		logger.Error(ctx, "Failed to initialize HTTP server: ", err)
		return
	}

	if err := httpServer.Run(); err != nil {
		logger.Error(ctx, "Failed to run server: ", err)
		return
	}
}

// registerGracefulShutdown registers a signal handler for graceful shutdown.
func registerGracefulShutdown(logger log.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info(context.Background(), "Shutting down gracefully...")

		logger.Info(context.Background(), "Cleanup completed")
		os.Exit(0)
	}()
}
