package config

import "time"

type Config struct {
	App       AppConfig       `yaml:"app"`
	MySQL     MySqlConfig     `yaml:"mysql"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JwtConfig       `yaml:"jwt"`
	Email     EmailConfig     `yaml:"email"`
	RateLimit RateLimitConfig `yaml:"rate_limit"`
}

type AppConfig struct {
	Host              string        `yaml:"host" env:"APP_HOST"`
	Port              int           `yaml:"port" env:"APP_PORT"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"APP_READ_HEADER_TIMEOUT"`
	ReadTimeout       time.Duration `yaml:"read_timeout" env:"APP_READ_TIMEOUT"`
	WriteTimeout      time.Duration `yaml:"write_timeout" env:"APP_WRITE_TIMEOUT"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" env:"APP_IDLE_TIMEOUT"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout" env:"APP_SHUTDOWN_TIMEOUT"`
}

type MySqlConfig struct {
	DSN                   string        `yaml:"dsn" env:"MYSQL_DSN"`
	MaxOpenConnections    int           `yaml:"max_open_connections" env:"MYSQL_MAX_OPEN_CONNECTIONS"`
	MaxIdleConnections    int           `yaml:"max_idle_connections" env:"MYSQL_MAX_IDLE_CONNECTIONS"`
	ConnectionMaxLifetime time.Duration `yaml:"connection_max_lifetime" env:"MYSQL_CONNECTION_MAX_LIFETIME"`
}

type RedisConfig struct {
	Address  string        `yaml:"address" env:"REDIS_ADDRESS"`
	Password string        `yaml:"password" env:"REDIS_PASSWORD"`
	Database int           `yaml:"database" env:"REDIS_DATABASE"`
	TaskTTL  time.Duration `yaml:"task_ttl" env:"REDIS_TASK_TTL"`
}

type JwtConfig struct {
	Secret   string        `yaml:"secret" env:"JWT_SECRET"`
	Lifetime time.Duration `yaml:"lifetime" env:"JWT_LIFETIME"`
}

type EmailConfig struct {
	BaseURL string        `yaml:"base_url" env:"EMAIL_BASE_URL"`
	Timeout time.Duration `yaml:"timeout" env:"EMAIL_TIMEOUT"`
}

type RateLimitConfig struct {
	Requests int           `yaml:"requests" env:"RATE_LIMIT_REQUESTS"`
	Window   time.Duration `yaml:"window" env:"RATE_LIMIT_WINDOW"`
}
