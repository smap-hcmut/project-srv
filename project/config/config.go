package config

import (
	"fmt"

	"github.com/caarlos0/env/v9"
)

type Config struct {
	// Environment Configuration
	// Determines CORS behavior and other environment-specific settings
	// Values: "production" (strict), "staging" (permissive), "dev" (permissive)
	// Default: "production" (fail-safe for security)
	Environment EnvironmentConfig

	// Server Configuration
	HTTPServer HTTPServerConfig
	Logger     LoggerConfig

	// Database Configuration
	Postgres PostgresConfig

	// Message Queue Configuration
	RabbitMQ RabbitMQConfig

	// Cache Configuration
	Redis RedisConfig

	// Authentication & Security Configuration
	JWT            JWTConfig
	Cookie         CookieConfig
	Encrypter      EncrypterConfig
	InternalConfig InternalConfig

	// Monitoring & Notification Configuration
	Discord DiscordConfig

	// External Services
	Identity IdentityConfig
	LLM      LLMConfig

	// Dry-Run Sampling Configuration
	DryRunSampling DryRunSamplingConfig
}

// LLMConfig is the configuration for the LLM provider.
type LLMConfig struct {
	Provider   string `env:"LLM_PROVIDER" envDefault:"gemini"`
	APIKey     string `env:"LLM_API_KEY"`
	Model      string `env:"LLM_MODEL" envDefault:"gemini-2.0-flash"`
	Timeout    int    `env:"LLM_TIMEOUT" envDefault:"30"`
	MaxRetries int    `env:"LLM_MAX_RETRIES" envDefault:"3"`
}

// JWTConfig is the configuration for the JWT,
// which is used to generate and verify the JWT.
type JWTConfig struct {
	SecretKey string `env:"JWT_SECRET"`
}

// EnvironmentConfig is the configuration for the deployment environment.
// It controls environment-specific behavior such as CORS validation.
type EnvironmentConfig struct {
	// Name is the environment name: "production", "staging", or "dev"
	// Production uses strict CORS origins, non-production allows private subnets and localhost
	Name string `env:"ENV" envDefault:"production"`
}

// CookieConfig is the configuration for HttpOnly cookies,
// which is used for secure authentication token storage.
type CookieConfig struct {
	Domain         string `env:"COOKIE_DOMAIN" envDefault:".smap.com"`
	Secure         bool   `env:"COOKIE_SECURE" envDefault:"true"`
	SameSite       string `env:"COOKIE_SAMESITE" envDefault:"Lax"`
	MaxAge         int    `env:"COOKIE_MAX_AGE" envDefault:"7200"`
	MaxAgeRemember int    `env:"COOKIE_MAX_AGE_REMEMBER" envDefault:"2592000"`
	Name           string `env:"COOKIE_NAME" envDefault:"smap_auth_token"`
}

// HTTPServerConfig is the configuration for the HTTP server,
// which is used to start, call API, etc.
type HTTPServerConfig struct {
	Host string `env:"HOST" envDefault:""`
	Port int    `env:"APP_PORT" envDefault:"8080"`
	Mode string `env:"API_MODE" envDefault:"debug"`
}

// RabbitMQConfig is the configuration for the RabbitMQ,
// which is used to connect to the RabbitMQ.
type RabbitMQConfig struct {
	URL string `env:"RABBITMQ_URL"`
}

// RedisConfig is the configuration for Redis,
// which is used for pub/sub and caching.
// StateDB is the project progress tracking's Redis database
// Note: Only standalone mode is supported (cluster mode not needed for this service)
type RedisConfig struct {
	Host         string `env:"REDIS_HOST" envDefault:"localhost:6379"`
	Password     string `env:"REDIS_PASSWORD"`
	DB           int    `env:"REDIS_DB" envDefault:"0"`
	StateDB      int    `env:"REDIS_STATE_DB" envDefault:"1"`
	MinIdleConns int    `env:"REDIS_MIN_IDLE_CONNS" envDefault:"10"`
	PoolSize     int    `env:"REDIS_POOL_SIZE" envDefault:"100"`
	PoolTimeout  int    `env:"REDIS_POOL_TIMEOUT" envDefault:"30"`
}

// LoggerConfig is the configuration for the logger,
// which is used to log the application.
type LoggerConfig struct {
	Level        string `env:"LOGGER_LEVEL" envDefault:"debug"`
	Mode         string `env:"LOGGER_MODE" envDefault:"debug"`
	Encoding     string `env:"LOGGER_ENCODING" envDefault:"console"`
	ColorEnabled bool   `env:"LOGGER_COLOR_ENABLED" envDefault:"true"`
}

// PostgresConfig is the configuration for the Postgres,
// which is used to connect to the Postgres.
type PostgresConfig struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER" envDefault:"postgres"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"postgres"`
	DBName   string `env:"POSTGRES_DB" envDefault:"postgres"`
	SSLMode  string `env:"POSTGRES_SSLMODE" envDefault:"prefer"`
}

type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:"minioadmin"`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:"minioadmin"`
	UseSSL    bool   `env:"MINIO_USE_SSL" envDefault:"false"`
	Region    string `env:"MINIO_REGION" envDefault:"us-east-1"`
	Bucket    string `env:"MINIO_BUCKET"`

	// Async upload settings
	AsyncUploadWorkers   int `env:"MINIO_ASYNC_UPLOAD_WORKERS" envDefault:"4"`
	AsyncUploadQueueSize int `env:"MINIO_ASYNC_UPLOAD_QUEUE_SIZE" envDefault:"100"`
}

type DiscordConfig struct {
	WebhookID    string `env:"DISCORD_WEBHOOK_ID"`
	WebhookToken string `env:"DISCORD_WEBHOOK_TOKEN"`
}

// EncrypterConfig is the configuration for the encrypter,
// which is used to encrypt and decrypt the data.
type EncrypterConfig struct {
	Key string `env:"ENCRYPT_KEY"`
}

// InternalConfig is the configuration for the internal,
// which is used to check the internal request.
type InternalConfig struct {
	InternalKey string `env:"INTERNAL_KEY"`
}

// IdentityConfig is the configuration for the Identity Service.
type IdentityConfig struct {
	BaseURL string `env:"IDENTITY_SERVICE_URL" envDefault:"http://localhost:8085"`
	Timeout int    `env:"IDENTITY_TIMEOUT" envDefault:"30"`
	// InternalAPIKey is automatically set from InternalConfig.InternalKey
	// No need to set IDENTITY_INTERNAL_KEY env variable
	InternalAPIKey string
}

// DryRunSamplingConfig is the configuration for dry-run keyword sampling.
type DryRunSamplingConfig struct {
	// Percentage-based sampling
	Percentage  float64 `env:"DRY_RUN_PERCENTAGE" envDefault:"10"`
	MinKeywords int     `env:"DRY_RUN_MIN_KEYWORDS" envDefault:"3"`
	MaxKeywords int     `env:"DRY_RUN_MAX_KEYWORDS" envDefault:"5"`

	// Timing configuration
	KeywordTimeEstimate string `env:"DRY_RUN_KEYWORD_TIME_ESTIMATE" envDefault:"16s"`

	// Strategy selection
	DefaultStrategy string `env:"DRY_RUN_SAMPLING_STRATEGY" envDefault:"percentage"`

	// Emergency fallback
	EmergencyThreshold string `env:"DRY_RUN_EMERGENCY_THRESHOLD" envDefault:"70s"`
	EmergencyKeywords  int    `env:"DRY_RUN_EMERGENCY_KEYWORDS" envDefault:"3"`
}

// Load is the function to load the configuration from the environment variables.
func Load() (*Config, error) {
	cfg := &Config{}
	err := env.Parse(cfg)
	if err != nil {
		fmt.Printf("Error loading configuration: %v", err)
		return nil, err
	}

	// Auto-sync: Use INTERNAL_KEY for Identity service internal calls
	cfg.Identity.InternalAPIKey = cfg.InternalConfig.InternalKey

	// Validate DryRun Sampling configuration
	if err := cfg.validateDryRunSampling(); err != nil {
		return nil, fmt.Errorf("invalid dry-run sampling configuration: %w", err)
	}

	return cfg, nil
}

// validateDryRunSampling validates the dry-run sampling configuration
func (c *Config) validateDryRunSampling() error {
	cfg := &c.DryRunSampling

	// Validate percentage range
	if cfg.Percentage <= 0 || cfg.Percentage > 100 {
		return fmt.Errorf("percentage must be between 0 and 100, got %f", cfg.Percentage)
	}

	// Validate minimum keywords
	if cfg.MinKeywords < 1 {
		return fmt.Errorf("minimum keywords must be at least 1, got %d", cfg.MinKeywords)
	}

	// Validate max >= min
	if cfg.MaxKeywords < cfg.MinKeywords {
		return fmt.Errorf("maximum keywords (%d) cannot be less than minimum (%d)",
			cfg.MaxKeywords, cfg.MinKeywords)
	}

	// Validate emergency settings
	if cfg.EmergencyKeywords < 1 {
		return fmt.Errorf("emergency keywords must be at least 1, got %d", cfg.EmergencyKeywords)
	}

	// Validate strategy
	switch cfg.DefaultStrategy {
	case "percentage", "fixed", "tiered":
		// Valid strategies
	default:
		return fmt.Errorf("invalid sampling strategy: %s", cfg.DefaultStrategy)
	}

	return nil
}
