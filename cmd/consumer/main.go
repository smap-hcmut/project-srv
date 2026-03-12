package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"project-srv/config"
	"project-srv/internal/consumer"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/postgres"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		return
	}

	// Initialize logger
	logger := log.NewZapLogger(log.ZapConfig{
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
	kafkaProducer, err := kafka.NewProducer(kafka.Config{
		Brokers: cfg.Kafka.Brokers,
		Topic:   cfg.Kafka.Topic,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Kafka producer: %v", err)
		return
	}
	defer kafkaProducer.Close()
	logger.Info(ctx, "Kafka producer initialized")

	// Redis
	redisClient, err := redis.New(redis.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to Redis: %v", err)
		return
	}
	defer redisClient.Close()
	logger.Info(ctx, "Redis client initialized")

	// PostgreSQL
	postgresDB, err := postgres.New(postgres.Config{
		Host:     cfg.Postgres.Host,
		Port:     cfg.Postgres.Port,
		User:     cfg.Postgres.User,
		Password: cfg.Postgres.Password,
		DBName:   cfg.Postgres.DBName,
		SSLMode:  cfg.Postgres.SSLMode,
	})
	if err != nil {
		logger.Errorf(ctx, "Failed to connect to PostgreSQL: %v", err)
		return
	}
	defer postgresDB.Close()
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
		PostgresDB:    postgresDB.GetDB(),
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
