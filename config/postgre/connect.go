package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"project-srv/config"

	_ "github.com/lib/pq"
)

var (
	instance *sql.DB
	once     sync.Once
	mu       sync.RWMutex
	initErr  error
)

// Connect initializes and connects to PostgreSQL using singleton pattern.
func Connect(ctx context.Context, cfg config.PostgresConfig) (*sql.DB, error) {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		return instance, nil
	}

	if initErr != nil {
		once = sync.Once{}
		initErr = nil
	}

	var err error
	once.Do(func() {
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
		)

		// Add search_path if schema is specified
		if cfg.Schema != "" {
			dsn += fmt.Sprintf(" search_path=%s", cfg.Schema)
		}

		db, e := sql.Open("postgres", dsn)
		if e != nil {
			err = fmt.Errorf("failed to open PostgreSQL connection: %w", e)
			initErr = err
			return
		}

		// Test connection
		if e := db.PingContext(ctx); e != nil {
			err = fmt.Errorf("failed to ping PostgreSQL: %w", e)
			initErr = err
			return
		}

		// Set connection pool settings
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)

		instance = db
	})

	return instance, err
}

// GetDB returns the singleton PostgreSQL database instance.
func GetDB() *sql.DB {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		panic("PostgreSQL not initialized. Call Connect() first")
	}
	return instance
}

// HealthCheck checks if PostgreSQL connection is healthy
func HealthCheck(ctx context.Context) error {
	mu.RLock()
	defer mu.RUnlock()

	if instance == nil {
		return fmt.Errorf("PostgreSQL not initialized")
	}

	return instance.PingContext(ctx)
}

// Disconnect closes the PostgreSQL connection and resets the singleton.
func Disconnect(ctx context.Context, db *sql.DB) error {
	mu.Lock()
	defer mu.Unlock()

	if instance != nil {
		if err := instance.Close(); err != nil {
			return err
		}
		instance = nil
		once = sync.Once{}
		initErr = nil
	}
	return nil
}
