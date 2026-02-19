package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"project-srv/config"
	"project-srv/config/kafka"
	"project-srv/config/postgre"
	"project-srv/config/redis"
	"project-srv/internal/consumer"
	"project-srv/pkg/discord"
	"project-srv/pkg/log"
)

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

	logger.Info(ctx, "Starting Project Consumer Service...")

	// Kafka Producer (for publishing events)
	kafkaProducer, err := kafka.ConnectProducer(cfg.Kafka)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Kafka producer: %v", err)
		return
	}
	defer kafka.DisconnectProducer()
	logger.Info(ctx, "Kafka producer initialized")

	// Redis
	redisClient, err := redis.Connect(ctx, cfg.Redis)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer redis.Disconnect()
	logger.Info(ctx, "Redis client initialized")

	// PostgreSQL
	postgresDB, err := postgre.Connect(ctx, cfg.Postgres)
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to PostgreSQL: %v", err)
		return
	}
	defer postgre.Disconnect(ctx, postgresDB)
	logger.Info(ctx, "PostgreSQL client initialized")

	// Discord (optional)
	var discordClient discord.IDiscord
	if cfg.Discord.WebhookURL != "" {
		discordClient, err = discord.New(logger, cfg.Discord.WebhookURL)
		if err != nil {
			logger.Warnf(ctx, "Discord webhook not configured (optional): %v", err)
		} else {
			logger.Info(ctx, "Discord client initialized")
		}
	}

	// Consumer server
	srv, err := consumer.New(consumer.Config{
		Logger:        logger,
		KafkaConfig:   cfg.Kafka,
		RedisClient:   redisClient,
		PostgresDB:    postgresDB,
		Discord:       discordClient,
		KafkaProducer: kafkaProducer,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to create consumer server: %v", err)
		return
	}

	// Run consumer server
	logger.Info(ctx, "Consumer server starting...")
	if err := srv.Run(ctx); err != nil {
		logger.Errorf(ctx, "Consumer server error: %v", err)
		return
	}

	logger.Info(ctx, "Consumer server stopped gracefully")
}
