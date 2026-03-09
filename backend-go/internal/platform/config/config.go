package config

import (
	"github.com/spf13/viper"
	"log"
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
	Driver string
	DSN    string
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
	viper.SetDefault("database.driver", "sqlite")
	viper.SetDefault("database.dsn", "rss_reader.db")
	viper.SetDefault("cors.origins", []string{"http://localhost:3001", "http://localhost:3000"})
	viper.SetDefault("cors.methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	viper.SetDefault("cors.allow_headers", []string{"Content-Type", "Authorization"})

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
		log.Println("Config file not found, using defaults")
	}

	AppConfig = &Config{}

	if err := viper.Unmarshal(AppConfig); err != nil {
		return err
	}

	return nil
}
