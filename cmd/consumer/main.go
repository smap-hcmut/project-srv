package main

// import (
// 	"context"
// 	"fmt"
// 	"os"
// 	"os/signal"
// 	"syscall"

// 	"project-srv/config"
// 	configPostgre "project-srv/config/postgre"
// 	"project-srv/internal/consumer"
// 	pkgLog "project-srv/pkg/log"

// 	_ "github.com/lib/pq"
// )

// // @Name SMAP Consumer Service
// // @description Consumer service for processing async tasks (Audit Logging, etc.)
// // @version 1.0
// func main() {
// 	// 1. Load configuration
// 	cfg, err := config.Load()
// 	if err != nil {
// 		fmt.Printf("Failed to load config: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// 2. Initialize logger
// 	logger := pkgLog.Init(pkgLog.ZapConfig{
// 		Level:        cfg.Logger.Level,
// 		Mode:         cfg.Logger.Mode,
// 		Encoding:     cfg.Logger.Encoding,
// 		ColorEnabled: cfg.Logger.ColorEnabled,
// 	})

// 	// 3. Initialize PostgreSQL
// 	ctx := context.Background()
// 	postgresDB, err := configPostgre.Connect(ctx, cfg.Postgres)
// 	if err != nil {
// 		logger.Errorf(ctx, "Failed to connect to PostgreSQL: %v", err)
// 		os.Exit(1)
// 	}
// 	defer configPostgre.Disconnect(ctx, postgresDB)
// 	logger.Infof(ctx, "PostgreSQL connected successfully to %s:%d/%s",
// 		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DBName)

// 	// 4. Initialize consumer service
// 	consumerService, err := consumer.New(logger, consumer.Config{
// 		PostgresDB:   postgresDB,
// 		KafkaBrokers: cfg.Kafka.Brokers,
// 	})
// 	if err != nil {
// 		logger.Errorf(ctx, "Failed to initialize consumer service: %v", err)
// 		os.Exit(1)
// 	}
// 	defer consumerService.Close()

// 	// 5. Start all consumers
// 	// Consumers are registered internally in consumer/handler.go
// 	go func() {
// 		if err := consumerService.Start(ctx); err != nil {
// 			logger.Errorf(ctx, "Consumer service error: %v", err)
// 			os.Exit(1)
// 		}
// 	}()

// 	logger.Info(ctx, "Consumer service ready - processing events from all registered consumers")

// 	// 6. Wait for shutdown signal
// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
// 	<-sigChan

// 	logger.Info(ctx, "Shutting down consumer service gracefully...")
// 	consumerService.Close()
// 	logger.Info(ctx, "Consumer service stopped gracefully")
// }
