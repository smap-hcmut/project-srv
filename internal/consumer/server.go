package consumer

import (
	"context"
	"database/sql"

	"project-srv/config"

	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/kafka"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

// ConsumerServer is the Kafka consumer orchestrator
type ConsumerServer struct {
	// Core Configuration
	l           log.Logger
	kafkaConfig config.KafkaConfig

	// Infrastructure clients
	redisClient   redis.IRedis
	postgresDB    *sql.DB
	kafkaProducer kafka.IProducer

	// Monitoring & Notification
	discord discord.IDiscord
}

// Config holds all dependencies for the consumer server
type Config struct {
	// Core Configuration
	Logger      log.Logger
	KafkaConfig config.KafkaConfig

	// Infrastructure clients
	RedisClient   redis.IRedis
	PostgresDB    *sql.DB
	KafkaProducer kafka.IProducer

	// Monitoring & Notification
	Discord discord.IDiscord
}

// Run starts the consumer server and blocks until context is cancelled.
// It initializes all domain layers, starts consumers, and handles graceful shutdown.
func (srv *ConsumerServer) Run(ctx context.Context) error {
	consumers, err := srv.setupDomains(ctx)
	if err != nil {
		srv.l.Errorf(ctx, "Failed to setup domains: %v", err)
		return err
	}

	if err := srv.startConsumers(ctx, consumers); err != nil {
		srv.l.Errorf(ctx, "Failed to start consumers: %v", err)
		return err
	}

	srv.l.Info(ctx, "Consumer Server is running")

	<-ctx.Done()
	srv.l.Info(ctx, "Shutdown signal received, stopping consumers...")

	srv.stopConsumers(ctx, consumers)

	srv.l.Info(ctx, "Consumer Server stopped gracefully")
	return nil
}
