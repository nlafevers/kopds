package config

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds the application configuration.
type Config struct {
	LibraryPath         string        `mapstructure:"library_path"`
	DatabasePath        string        `mapstructure:"database_path"`
	BaseURL             string        `mapstructure:"base_url"`
	Port                int           `mapstructure:"port"`
	LogLevel            string        `mapstructure:"log_level"`
	JSONLog             bool          `mapstructure:"json_log"`
	SyncInterval        time.Duration `mapstructure:"sync_interval"`
	ImageCachePath      string        `mapstructure:"image_cache_path"`
	ImageCacheMaxCount  int           `mapstructure:"image_cache_max_count"`
}

// Load loads the configuration from file and environment variables.
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	viper.SetDefault("port", 8080)
	viper.SetDefault("database_path", "kopds.db")
	viper.SetDefault("base_url", "http://localhost:8080")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("json_log", false)
	viper.SetDefault("sync_interval", "30m")
	viper.SetDefault("image_cache_path", "cache/images")
	viper.SetDefault("image_cache_max_count", 1000)

	viper.SetEnvPrefix("KOPDS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate ensures the configuration is valid.
func (c *Config) Validate() error {
	if c.LibraryPath == "" {
		return errors.New("library_path is required")
	}

	// Basic check for Calibre library
	info, err := os.Stat(c.LibraryPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("library_path must be a directory")
	}

	if c.ImageCacheMaxCount <= 0 {
		return errors.New("image_cache_max_count must be greater than 0")
	}

	return nil
}
