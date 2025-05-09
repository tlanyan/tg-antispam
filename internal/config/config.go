package config

import (
	"fmt"
	"log"

	"github.com/spf13/viper"
)

// global configuration structure
type Config struct {
	Bot      BotConfig      `mapstructure:"bot"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Antispam AntispamConfig `mapstructure:"antispam"`
	Database DatabaseConfig `mapstructure:"database"`
}

// Telegram bot configuration
type BotConfig struct {
	Token   string        `mapstructure:"token"`
	Webhook WebhookConfig `mapstructure:"webhook"`
}

// webhook server configuration
type WebhookConfig struct {
	Endpoint   string `mapstructure:"endpoint"`
	ListenPort string `mapstructure:"listen_port"`
	DebugPath  string `mapstructure:"debug_path"`
	CertFile   string `mapstructure:"cert_file"`
	KeyFile    string `mapstructure:"key_file"`
}

// logging configuration
type LoggerConfig struct {
	Directory  string            `mapstructure:"directory"`
	Rotation   LogRotationConfig `mapstructure:"rotation"`
	Timezone   string            `mapstructure:"timezone"`
	Format     string            `mapstructure:"format"`
	TimeFormat string            `mapstructure:"time_format"`
	Level      string            `mapstructure:"level"`
}

// log rotation settings
type LogRotationConfig struct {
	MaxSize    int  `mapstructure:"max_size"`
	MaxBackups int  `mapstructure:"max_backups"`
	MaxAge     int  `mapstructure:"max_age"`
	Compress   bool `mapstructure:"compress"`
}

type DatabaseConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	Charset  string `mapstructure:"charset"`
}

// anti-spam feature settings
type AntispamConfig struct {
	BanRandomUsername bool `mapstructure:"ban_random_username"`
	BanEmojiName      bool `mapstructure:"ban_emoji_name"`
	BanBioLink        bool `mapstructure:"ban_bio_link"`
	UseCAS            bool `mapstructure:"use_cas"`
	BanPremium        bool `mapstructure:"ban_premium"`
}

var cfg *Config

func Load(configPath string) (*Config, error) {
	if configPath == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	v := viper.New()

	setDefaults(v)

	v.SetConfigFile(configPath)

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

func Get() *Config {
	if cfg == nil {
		log.Fatal("Configuration not initialized, call Load() first")
	}
	return cfg
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("bot.webhook.listen_port", "8443")
	v.SetDefault("bot.webhook.debug_path", "/debug")
	v.SetDefault("bot.webhook.cert_file", "")
	v.SetDefault("bot.webhook.key_file", "")

	v.SetDefault("logger.directory", "logs")
	v.SetDefault("logger.rotation.max_size", 10)
	v.SetDefault("logger.rotation.max_backups", 30)
	v.SetDefault("logger.rotation.max_age", 90)
	v.SetDefault("logger.rotation.compress", true)
	v.SetDefault("logger.timezone", "Local")
	v.SetDefault("logger.format", "[%{level}] %{time} %{file}:%{line}: %{message}")
	v.SetDefault("logger.time_format", "2006/01/02 15:04:05")
	v.SetDefault("logger.level", "INFO")

	v.SetDefault("database.enabled", false)

	v.SetDefault("antispam.ban_random_username", true)
	v.SetDefault("antispam.ban_emoji_name", true)
	v.SetDefault("antispam.ban_bio_link", true)
	v.SetDefault("antispam.use_cas", true)
	v.SetDefault("antispam.ban_premium", true)
}
