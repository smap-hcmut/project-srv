package httpserver

import (
	"database/sql"
	"errors"
	"project-srv/config"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/discord"
	"github.com/smap-hcmut/shared-libs/go/encrypter"
	"github.com/smap-hcmut/shared-libs/go/log"
	"github.com/smap-hcmut/shared-libs/go/redis"
)

type HTTPServer struct {
	// Server Configuration
	gin         *gin.Engine
	l           log.Logger
	host        string
	port        int
	mode        string
	environment string

	// Database Configuration
	postgresDB *sql.DB

	// Storage Configuration

	// // Message Queue Configuration
	// Redis Configuration
	mainRedisClient  redis.IRedis // DB 0: job mapping, pub/sub
	stateRedisClient redis.IRedis // DB 1: project progress tracking

	// Authentication & Security Configuration
	jwtSecretKey string
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter
	internalKey  string

	// Monitoring & Notification Configuration
	discord discord.IDiscord
}

type Config struct {
	// Server Configuration
	Logger      log.Logger
	Port        int
	Mode        string
	Environment string

	// Database Configuration
	PostgresDB *sql.DB

	// Redis Configuration
	RedisClient redis.IRedis

	// Authentication & Security Configuration
	JwtSecretKey string
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter
	InternalKey  string

	// Monitoring & Notification Configuration
	Discord discord.IDiscord
}

// New creates a new HTTPServer instance with the provided configuration.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server Configuration
		l:           logger,
		gin:         gin.New(),
		host:        "",
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		// Database Configuration
		postgresDB: cfg.PostgresDB,

		// Redis Configuration
		mainRedisClient:  cfg.RedisClient,
		stateRedisClient: cfg.RedisClient,

		// Authentication & Security Configuration
		jwtSecretKey: cfg.JwtSecretKey,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,
		internalKey:  cfg.InternalKey,

		// Monitoring & Notification Configuration
		discord: cfg.Discord,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	// Add middlewares
	srv.gin.Use(srv.zapLoggerMiddleware())
	srv.gin.Use(gin.Recovery())

	return srv, nil
}

func (srv *HTTPServer) zapLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		if path == "/health" || path == "/ready" || path == "/live" {
			return
		}

		if srv.environment == "production" {
			srv.l.Infof(c.Request.Context(),
				"HTTP Request - Method: %s, Path: %s, Status: %d, IP: %s, Latency: %v, UserAgent: %s, Query: %s",
				c.Request.Method, path, status, c.ClientIP(), latency, c.Request.UserAgent(), query)
		} else {
			srv.l.Infof(c.Request.Context(), "%s %s %d %s %s", c.Request.Method, path, status, latency, c.ClientIP())
		}
	}
}

// validate validates that all required dependencies are provided.
func (srv HTTPServer) validate() error {
	// Server Configuration
	if srv.l == nil {
		return errors.New("logger is required")
	}
	if srv.mode == "" {
		return errors.New("mode is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}

	// Database Configuration
	if srv.postgresDB == nil {
		return errors.New("postgresDB is required")
	}

	// Authentication & Security Configuration
	if srv.jwtSecretKey == "" {
		return errors.New("jwtSecretKey is required")
	}
	if srv.encrypter == nil {
		return errors.New("encrypter is required")
	}
	if srv.internalKey == "" {
		return errors.New("internalKey is required")
	}

	return nil
}
