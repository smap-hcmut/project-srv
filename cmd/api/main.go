package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"project-srv/config"
	configPostgre "project-srv/config/postgre"

	_ "project-srv/docs" // Import swagger docs
	"project-srv/internal/httpserver"
	"project-srv/pkg/discord"
	"project-srv/pkg/encrypter"
	"project-srv/pkg/log"
	pkgRedis "project-srv/pkg/redis"
	"syscall"
)

// @title       SMAP Project Service API
// @description SMAP Project Service API documentation.
// @version     1
// @host        project-srv.tantai.dev
// @schemes     https
// @BasePath    /project
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name smap_auth_token
// @description Authentication token stored in HttpOnly cookie. Set automatically by /login endpoint.
//
// @securityDefinitions.apikey Bearer
// @in header
// @name Authorization
// @description Legacy Bearer token authentication (deprecated - use cookie authentication instead). Format: "Bearer {token}"
func main() {
	// 1. Load configuration
	// Reads config from YAML file and environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// 2. Initialize logger
	logger := log.Init(log.ZapConfig{
		Level:        cfg.Logger.Level,
		Mode:         cfg.Logger.Mode,
		Encoding:     cfg.Logger.Encoding,
		ColorEnabled: cfg.Logger.ColorEnabled,
	})

	// 3. Register graceful shutdown
	registerGracefulShutdown(logger)

	// 4. Initialize encrypter
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)

	// 5. Initialize PostgreSQL
	ctx := context.Background()
	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Error(ctx, "Failed to connect to PostgreSQL: ", err)
		return
	}
	defer configPostgre.Disconnect(ctx, postgresDB)
	logger.Infof(ctx, "PostgreSQL connected successfully to %s:%d/%s", cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

	// 6. Initialize Discord (optional)
	discordClient, err := discord.New(logger, &discord.DiscordWebhook{
		ID:    cfg.Discord.WebhookID,
		Token: cfg.Discord.WebhookToken,
	})
	if err != nil {
		logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		discordClient = nil // Continue without Discord
	} else {
		logger.Infof(ctx, "Discord webhook initialized successfully")
	}

	// 7. Initialize Redis
	// 7. Initialize Redis
	redisClient, err := pkgRedis.NewRedis(pkgRedis.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis: ", err)
		return
	}
	logger.Infof(ctx, "Redis connected successfully to %s:%d (DB %d)", cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)

	logger.Infof(ctx, "JWT Manager initialized with algorithm: %s", cfg.JWT.Algorithm)

	// 11. Initialize HTTP server
	// 11. Initialize HTTP server
	// Main application server that handles all HTTP requests and routes
	httpServer, err := httpserver.New(logger, httpserver.Config{
		// Server Configuration
		Logger:      logger,
		Host:        cfg.HTTPServer.Host,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		// Database Configuration
		PostgresDB: postgresDB,

		// Storage Configuration

		// Redis Configuration
		RedisClient: redisClient,

		// Authentication & Security Configuration
		JwtSecretKey: cfg.JWT.SecretKey,
		CookieConfig: cfg.Cookie,
		Encrypter:    encrypterInstance,
		InternalKey:  cfg.InternalConfig.InternalKey,

		// Monitoring & Notification Configuration
		Discord: discordClient,
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
