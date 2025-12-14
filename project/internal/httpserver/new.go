package httpserver

import (
	"database/sql"
	"errors"

	"smap-project/config"
	"smap-project/pkg/discord"
	"smap-project/pkg/encrypter"
	"smap-project/pkg/log"
	pkgRabbitMQ "smap-project/pkg/rabbitmq"
	pkgRedis "smap-project/pkg/redis"

	"github.com/gin-gonic/gin"
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

	// // Storage Configuration
	// minio miniopkg.MinIO

	// // Message Queue Configuration
	amqpConn *pkgRabbitMQ.Connection

	// Redis Configuration
	mainRedisClient  pkgRedis.Client // DB 0: job mapping, pub/sub
	stateRedisClient pkgRedis.Client // DB 1: project progress tracking

	// Authentication & Security Configuration
	jwtSecretKey string
	cookieConfig config.CookieConfig
	encrypter    encrypter.Encrypter
	internalKey  string

	// Monitoring & Notification Configuration
	discord *discord.Discord

	// External Services
	llmConfig config.LLMConfig

	// Dry-Run Configuration
	dryRunSamplingConfig config.DryRunSamplingConfig
}

type Config struct {
	// Server Configuration
	Logger      log.Logger
	Host        string
	Port        int
	Mode        string
	Environment string

	// Database Configuration
	PostgresDB *sql.DB

	// // Storage Configuration
	// MinIO miniopkg.MinIO

	// // Message Queue Configuration
	AmqpConn *pkgRabbitMQ.Connection

	// Redis Configuration
	MainRedisClient  pkgRedis.Client // DB 0: job mapping, pub/sub
	StateRedisClient pkgRedis.Client // DB 1: project progress tracking

	// Authentication & Security Configuration
	JwtSecretKey string
	CookieConfig config.CookieConfig
	Encrypter    encrypter.Encrypter
	InternalKey  string

	// Monitoring & Notification Configuration
	Discord *discord.Discord

	// External Services
	LLMConfig config.LLMConfig

	// Dry-Run Configuration
	DryRunSamplingConfig config.DryRunSamplingConfig
}

// New creates a new HTTPServer instance with the provided configuration.
func New(logger log.Logger, cfg Config) (*HTTPServer, error) {
	gin.SetMode(cfg.Mode)

	srv := &HTTPServer{
		// Server Configuration
		l:           logger,
		gin:         gin.Default(),
		host:        cfg.Host,
		port:        cfg.Port,
		mode:        cfg.Mode,
		environment: cfg.Environment,

		// Database Configuration
		postgresDB: cfg.PostgresDB,

		// // Storage Configuration
		// minio: cfg.MinIO,

		// // Message Queue Configuration
		amqpConn: cfg.AmqpConn,

		// Redis Configuration
		mainRedisClient:  cfg.MainRedisClient,
		stateRedisClient: cfg.StateRedisClient,

		// Authentication & Security Configuration
		jwtSecretKey: cfg.JwtSecretKey,
		cookieConfig: cfg.CookieConfig,
		encrypter:    cfg.Encrypter,
		internalKey:  cfg.InternalKey,

		// Monitoring & Notification Configuration
		discord: cfg.Discord,

		// External Services
		llmConfig: cfg.LLMConfig,

		// Dry-Run Configuration
		dryRunSamplingConfig: cfg.DryRunSamplingConfig,
	}

	if err := srv.validate(); err != nil {
		return nil, err
	}

	return srv, nil
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
	if srv.host == "" {
		return errors.New("host is required")
	}
	if srv.port == 0 {
		return errors.New("port is required")
	}

	// Database Configuration
	if srv.postgresDB == nil {
		return errors.New("postgresDB is required")
	}

	// Storage Configuration
	// if srv.minio == nil {
	// 	return errors.New("minio is required")
	// }

	// // Message Queue Configuration
	// if srv.amqpConn == nil {
	// 	return errors.New("amqp connection is required")
	// }

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

	// Monitoring & Notification Configuration
	if srv.discord == nil {
		return errors.New("discord is required")
	}

	// External Services
	if srv.llmConfig.APIKey == "" {
		return errors.New("LLM API key is required")
	}

	return nil
}
