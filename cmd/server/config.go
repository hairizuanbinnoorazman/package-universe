package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server  ServerConfig
	Storage StorageConfig
	Log     LogConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// StorageConfig holds blob storage configuration.
type StorageConfig struct {
	Type            string        // "local" or "s3"
	BaseDir         string        // For local: "./uploads"
	S3Bucket        string        // For S3: bucket name
	S3Region        string        // For S3: AWS region
	S3PresignExpiry time.Duration // Presigned URL expiration
}

// LogConfig holds logging configuration.
type LogConfig struct {
	Level string
}

// LoadConfig loads configuration from file and environment variables.
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
	}

	// Enable environment variable overrides
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "15s")
	v.SetDefault("server.write_timeout", "15s")

	v.SetDefault("storage.type", "local")
	v.SetDefault("storage.base_dir", "./uploads")
	v.SetDefault("storage.s3_bucket", "")
	v.SetDefault("storage.s3_region", "us-east-1")
	v.SetDefault("storage.s3_presign_expiry", "15m")

	v.SetDefault("log.level", "info")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; using defaults
	}

	// Parse configuration
	var config Config

	config.Server.Host = v.GetString("server.host")
	config.Server.Port = v.GetInt("server.port")
	config.Server.ReadTimeout = v.GetDuration("server.read_timeout")
	config.Server.WriteTimeout = v.GetDuration("server.write_timeout")

	config.Storage.Type = v.GetString("storage.type")
	config.Storage.BaseDir = v.GetString("storage.base_dir")
	config.Storage.S3Bucket = v.GetString("storage.s3_bucket")
	config.Storage.S3Region = v.GetString("storage.s3_region")
	config.Storage.S3PresignExpiry = v.GetDuration("storage.s3_presign_expiry")

	config.Log.Level = v.GetString("log.level")

	return &config, nil
}
