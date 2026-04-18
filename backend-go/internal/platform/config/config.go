package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
	"my-robot-backend/internal/platform/logging"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	CORS     CORSConfig
}

type ServerConfig struct {
	Port string
	Mode string // debug, release, test
}

type DatabaseConfig struct {
	Driver   string
	DSN      string
	Postgres PostgresConfig
}

type PostgresConfig struct {
	MaxIdleConns           int `mapstructure:"max_idle_conns"`
	MaxOpenConns           int `mapstructure:"max_open_conns"`
	ConnMaxLifetimeMinutes int `mapstructure:"conn_max_lifetime_minutes"`
	ConnMaxIdleTimeMinutes int `mapstructure:"conn_max_idle_time_minutes"`
}

type CORSConfig struct {
	Origins      []string
	Methods      []string
	AllowHeaders []string
}

var AppConfig *Config

func LoadConfig(configPath string) error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configPath)
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")

	// Set defaults
	viper.SetDefault("server.port", "5000")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.dsn", "host=127.0.0.1 user=postgres password=postgres dbname=rss_reader port=5432 sslmode=disable TimeZone=Asia/Shanghai")
	viper.SetDefault("database.postgres.max_idle_conns", 5)
	viper.SetDefault("database.postgres.max_open_conns", 25)
	viper.SetDefault("database.postgres.conn_max_lifetime_minutes", 60)
	viper.SetDefault("database.postgres.conn_max_idle_time_minutes", 10)
	viper.SetDefault("cors.origins", []string{"http://localhost:3000", "http://localhost:3000"})
	viper.SetDefault("cors.methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allow_headers", []string{"Content-Type", "Authorization"})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		logging.Infof("Config file not found, using defaults")
	}

	AppConfig = &Config{}

	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	applyEnvOverrides(AppConfig)

	return nil
}

func applyEnvOverrides(cfg *Config) {
	if cfg == nil {
		return
	}

	if value := strings.TrimSpace(os.Getenv("SERVER_PORT")); value != "" {
		cfg.Server.Port = value
	}

	if value := strings.TrimSpace(os.Getenv("SERVER_MODE")); value != "" {
		cfg.Server.Mode = value
	}

	if value := strings.TrimSpace(os.Getenv("DATABASE_DRIVER")); value != "" {
		cfg.Database.Driver = value
	}

	if value := strings.TrimSpace(os.Getenv("DATABASE_DSN")); value != "" {
		cfg.Database.DSN = value
	}

	if value := strings.TrimSpace(os.Getenv("CORS_ORIGINS")); value != "" {
		cfg.CORS.Origins = splitCommaSeparated(value)
	}
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}
