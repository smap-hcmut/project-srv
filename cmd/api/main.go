package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"project-srv/config"
	"project-srv/config/postgre"
	"project-srv/config/redis"

	_ "project-srv/docs" // Import swagger docs
	"project-srv/internal/httpserver"
	"project-srv/pkg/discord"
	"project-srv/pkg/encrypter"
	"project-srv/pkg/log"
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

	// Create context with signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger.Info(ctx, "Starting Project API Service...")

	// Initialize encrypter
	encrypterInstance := encrypter.New(cfg.Encrypter.Key)
	logger.Info(ctx, "Encrypter initialized")

	// Initialize PostgreSQL
	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to PostgreSQL: %v", err)
		return
	}
	defer postgre.Disconnect(ctx, postgresDB)
	logger.Infof(ctx, "PostgreSQL connected successfully to %s:%d/%s",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

	// Initialize Redis
	redisClient, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer redis.Disconnect()
	logger.Infof(ctx, "Redis connected successfully to %s:%d (DB %d)",
		cfg.Redis.Host, cfg.Redis.Port, cfg.Redis.DB)

	// Initialize Discord (optional)
	var discordClient discord.IDiscord
	if cfg.Discord.WebhookURL != "" {
		discordClient, err = discord.New(logger, cfg.Discord.WebhookURL)
		if err != nil {
			logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		} else {
			logger.Info(ctx, "Discord webhook initialized")
		}
	}

	// Initialize HTTP server
	httpServer, err := httpserver.New(logger, httpserver.Config{
		// Server Configuration
		Logger:      logger,
		Port:        cfg.HTTPServer.Port,
		Mode:        cfg.HTTPServer.Mode,
		Environment: cfg.Environment.Name,

		// Database Configuration
		PostgresDB: postgresDB,

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
		logger.Errorf(ctx, "Failed to initialize HTTP server: %v", err)
		return
	}

	// Start HTTP server
	logger.Infof(ctx, "Starting HTTP server on port %d (mode: %s)", cfg.HTTPServer.Port, cfg.HTTPServer.Mode)
	if err := httpServer.Run(); err != nil {
		logger.Errorf(ctx, "Failed to run server: %v", err)
		return
	}

	// Wait for shutdown signal
	<-ctx.Done()
	logger.Info(ctx, "Shutting down gracefully...")
}
