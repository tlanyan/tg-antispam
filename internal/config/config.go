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
	Database DatabaseConfig `mapstructure:"database"`
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

// DatabaseConfig contains database connection settings
type DatabaseConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

// AntispamConfig contains anti-spam feature settings
type AntispamConfig struct {
	BanRandomUsername bool `mapstructure:"ban_random_username"`
	BanEmojiName      bool `mapstructure:"ban_emoji_name"`
	BanBioLink        bool `mapstructure:"ban_bio_link"`
	UseCAS            bool `mapstructure:"use_cas"`
	BanPremium        bool `mapstructure:"ban_premium"`
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

	// Database defaults
	v.SetDefault("database.enabled", false)
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.username", "root")
	v.SetDefault("database.password", "")
	v.SetDefault("database.dbname", "tgantispam")
	v.SetDefault("database.charset", "utf8mb4")

	// Antispam defaults
	v.SetDefault("antispam.ban_random_username", true)
	v.SetDefault("antispam.ban_emoji_name", true)
	v.SetDefault("antispam.ban_bio_link", true)
	v.SetDefault("antispam.use_cas", true)
	v.SetDefault("antispam.ban_premium", true)
	v.SetDefault("antispam.cache_size", 30)
}
