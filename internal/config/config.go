package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// Config is the global configuration structure
type Config struct {
	Bot      BotConfig      `mapstructure:"bot"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Antispam AntispamConfig `mapstructure:"antispam"`
}

// BotConfig contains Telegram bot configuration
type BotConfig struct {
	Token   string        `mapstructure:"token"`
	Webhook WebhookConfig `mapstructure:"webhook"`
}

// WebhookConfig contains webhook server configuration
type WebhookConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	ListenPort string `mapstructure:"listen_port"`
	DebugPath  string `mapstructure:"debug_path"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
}

// LoggerConfig contains logging configuration
type LoggerConfig struct {
	Directory string            `mapstructure:"directory"`
	Rotation  LogRotationConfig `mapstructure:"rotation"`
}

// LogRotationConfig contains log rotation settings
type LogRotationConfig struct {
	MaxSize    int  `mapstructure:"max_size"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAge     int  `mapstructure:"max_age"`
	Compress   bool `mapstructure:"compress"`
}

// AntispamConfig contains anti-spam feature settings
type AntispamConfig struct {
	CheckRandomUsername bool `mapstructure:"check_random_username"`
	CheckEmojiUsername  bool `mapstructure:"check_emoji_username"`
	CheckBioLinks       bool `mapstructure:"check_bio_links"`
	UseCAS              bool `mapstructure:"use_cas"`
	RestrictPremiumUser bool `mapstructure:"restrict_premium_user"`
}

// Global configuration instance
var cfg *Config

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	v := viper.New()

	// Set default values
	setDefaults(v)

	// Set config file
	v.SetConfigFile(configPath)

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	log.Printf("Using config file: %s", v.ConfigFileUsed())

	// Unmarshal configuration
	cfg = &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	return cfg, nil
}

// Get returns the global configuration instance
func Get() *Config {
	if cfg == nil {
		log.Fatal("Configuration not initialized, call Load() first")
	}
	return cfg
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Bot defaults
	v.SetDefault("bot.webhook.listen_port", "8443")
	v.SetDefault("bot.webhook.debug_path", "/debug")
	v.SetDefault("bot.webhook.cert_file", "")
	v.SetDefault("bot.webhook.key_file", "")

	// Logger defaults
	v.SetDefault("logger.directory", "logs")
	v.SetDefault("logger.rotation.max_size", 10)
	v.SetDefault("logger.rotation.max_backups", 30)
	v.SetDefault("logger.rotation.max_age", 90)
	v.SetDefault("logger.rotation.compress", true)

	// Antispam defaults
	v.SetDefault("antispam.check_random_username", true)
	v.SetDefault("antispam.check_emoji_username", true)
	v.SetDefault("antispam.check_bio_links", true)
	v.SetDefault("antispam.use_cas", true)
	v.SetDefault("antispam.restrict_premium_user", true)
	v.SetDefault("antispam.cache_size", 30)
}
