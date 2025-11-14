// Package config handles application configuration loading and validation using Viper.
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	GitLab       GitLabConfig       `mapstructure:"gitlab"`
	Mattermost   MattermostConfig   `mapstructure:"mattermost"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Teams        []TeamConfig       `mapstructure:"teams"`
	Roulette     RouletteConfig     `mapstructure:"roulette"`
	Scheduler    SchedulerConfig    `mapstructure:"scheduler"`
	Metrics      MetricsConfig      `mapstructure:"metrics"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Badges       []BadgeConfig      `mapstructure:"badges"`
	Availability AvailabilityConfig `mapstructure:"availability"`
}

// ServerConfig contains HTTP server configuration.
type ServerConfig struct {
	Port        int    `mapstructure:"port"`
	Environment string `mapstructure:"environment"`
	Language    string `mapstructure:"language"` // Language for bot responses (en, fr)
}

// GitLabConfig contains GitLab API connection and authentication settings.
type GitLabConfig struct {
	URL           string `mapstructure:"url"`
	Token         string `mapstructure:"token"`
	BotUsername   string `mapstructure:"bot_username"`
	WebhookSecret string `mapstructure:"webhook_secret"`
}

// MattermostConfig contains Mattermost webhook notification settings.
type MattermostConfig struct {
	WebhookURL string `mapstructure:"webhook_url"`
	Channel    string `mapstructure:"channel"`
	Enabled    bool   `mapstructure:"enabled"`
}

// DatabaseConfig contains database connection settings for PostgreSQL and Redis.
type DatabaseConfig struct {
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

// PostgresConfig contains PostgreSQL database connection and pool settings.
type PostgresConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Database        string `mapstructure:"database"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

// RedisConfig contains Redis cache connection and pool settings.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
	PoolSize int    `mapstructure:"pool_size"`
}

// TeamConfig represents a team with its members.
type TeamConfig struct {
	Name    string         `mapstructure:"name"`
	Members []MemberConfig `mapstructure:"members"`
}

// MemberConfig represents a team member with their role.
type MemberConfig struct {
	Username string `mapstructure:"username"`
	Role     string `mapstructure:"role"`
}

// RouletteConfig contains reviewer selection algorithm configuration.
type RouletteConfig struct {
	Weights   WeightsConfig   `mapstructure:"weights"`
	Expertise ExpertiseConfig `mapstructure:"expertise"`
}

// WeightsConfig contains scoring weights for reviewer selection algorithm.
type WeightsConfig struct {
	CurrentLoad    int `mapstructure:"current_load"`
	RecentReview   int `mapstructure:"recent_review"`
	ExpertiseBonus int `mapstructure:"expertise_bonus"`
}

// ExpertiseConfig defines file patterns for developer and operations expertise.
type ExpertiseConfig struct {
	Dev []string `mapstructure:"dev"`
	Ops []string `mapstructure:"ops"`
}

// SchedulerConfig contains daily notification scheduler settings.
type SchedulerConfig struct {
	Enabled             bool   `mapstructure:"enabled"`
	Time                string `mapstructure:"time"`
	BadgeEvaluationTime string `mapstructure:"badge_evaluation_time"` // Cron expression for badge evaluation
	Timezone            string `mapstructure:"timezone"`
	SkipWeekends        bool   `mapstructure:"skip_weekends"`
	SkipHolidays        bool   `mapstructure:"skip_holidays"`
}

// MetricsConfig contains metrics collection and retention settings.
type MetricsConfig struct {
	RetentionDays int              `mapstructure:"retention_days"`
	Prometheus    PrometheusConfig `mapstructure:"prometheus"`
}

// PrometheusConfig contains Prometheus metrics exporter settings.
type PrometheusConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"`
	Path    string `mapstructure:"path"`
}

// LoggingConfig contains application logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// BadgeConfig represents a gamification badge with earning criteria.
type BadgeConfig struct {
	Name        string                 `mapstructure:"name"`
	Description string                 `mapstructure:"description"`
	Icon        string                 `mapstructure:"icon"`
	Criteria    map[string]interface{} `mapstructure:"criteria"`
}

// AvailabilityConfig contains reviewer availability checking settings.
type AvailabilityConfig struct {
	CacheTTL    int      `mapstructure:"cache_ttl"`
	OOOKeywords []string `mapstructure:"ooo_keywords"`
}

// Load reads configuration from file and environment variables.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set config file
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("/etc/reviewer-roulette/")
	}

	// Bind specific environment variables (explicit bindings for 12-factor app compliance)
	// Server configuration
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.environment", "SERVER_ENVIRONMENT")
	_ = v.BindEnv("server.language", "SERVER_LANGUAGE")

	// GitLab configuration
	_ = v.BindEnv("gitlab.url", "GITLAB_URL")
	_ = v.BindEnv("gitlab.token", "GITLAB_TOKEN", "GITLAB_BOT_TOKEN")
	_ = v.BindEnv("gitlab.bot_username", "GITLAB_BOT_USERNAME")
	_ = v.BindEnv("gitlab.webhook_secret", "GITLAB_WEBHOOK_SECRET")

	// Mattermost configuration
	_ = v.BindEnv("mattermost.webhook_url", "MATTERMOST_WEBHOOK_URL")
	_ = v.BindEnv("mattermost.channel", "MATTERMOST_CHANNEL")
	_ = v.BindEnv("mattermost.enabled", "MATTERMOST_ENABLED")

	// PostgreSQL configuration
	_ = v.BindEnv("database.postgres.host", "POSTGRES_HOST")
	_ = v.BindEnv("database.postgres.port", "POSTGRES_PORT")
	_ = v.BindEnv("database.postgres.database", "POSTGRES_DB")
	_ = v.BindEnv("database.postgres.user", "POSTGRES_USER")
	_ = v.BindEnv("database.postgres.password", "POSTGRES_PASSWORD")
	_ = v.BindEnv("database.postgres.ssl_mode", "POSTGRES_SSL_MODE")
	_ = v.BindEnv("database.postgres.max_open_conns", "POSTGRES_MAX_OPEN_CONNS")
	_ = v.BindEnv("database.postgres.max_idle_conns", "POSTGRES_MAX_IDLE_CONNS")
	_ = v.BindEnv("database.postgres.conn_max_lifetime", "POSTGRES_CONN_MAX_LIFETIME")

	// Redis configuration
	_ = v.BindEnv("database.redis.host", "REDIS_HOST")
	_ = v.BindEnv("database.redis.port", "REDIS_PORT")
	_ = v.BindEnv("database.redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("database.redis.db", "REDIS_DB")
	_ = v.BindEnv("database.redis.pool_size", "REDIS_POOL_SIZE")

	// Logging configuration
	_ = v.BindEnv("logging.level", "LOG_LEVEL")
	_ = v.BindEnv("logging.format", "LOG_FORMAT")
	_ = v.BindEnv("logging.output", "LOG_OUTPUT")

	// Scheduler configuration
	_ = v.BindEnv("scheduler.enabled", "SCHEDULER_ENABLED")
	_ = v.BindEnv("scheduler.time", "SCHEDULER_TIME")
	_ = v.BindEnv("scheduler.badge_evaluation_time", "SCHEDULER_BADGE_EVALUATION_TIME")
	_ = v.BindEnv("scheduler.timezone", "SCHEDULER_TIMEZONE")
	_ = v.BindEnv("scheduler.skip_weekends", "SCHEDULER_SKIP_WEEKENDS")
	_ = v.BindEnv("scheduler.skip_holidays", "SCHEDULER_SKIP_HOLIDAYS")

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.GitLab.URL == "" {
		return fmt.Errorf("gitlab.url is required")
	}
	if c.GitLab.Token == "" {
		return fmt.Errorf("gitlab.token is required")
	}
	if c.GitLab.WebhookSecret == "" {
		return fmt.Errorf("gitlab.webhook_secret is required")
	}
	if c.Database.Postgres.Host == "" {
		return fmt.Errorf("database.postgres.host is required")
	}
	if c.Database.Postgres.Database == "" {
		return fmt.Errorf("database.postgres.database is required")
	}
	if c.Database.Postgres.User == "" {
		return fmt.Errorf("database.postgres.user is required")
	}
	if c.Database.Redis.Host == "" {
		return fmt.Errorf("database.redis.host is required")
	}
	if len(c.Teams) == 0 {
		return fmt.Errorf("at least one team must be configured")
	}

	return nil
}

// GetLocation returns the timezone location.
func (c *SchedulerConfig) GetLocation() (*time.Location, error) {
	return time.LoadLocation(c.Timezone)
}

// GetTeamByName returns a team configuration by name.
func (c *Config) GetTeamByName(name string) *TeamConfig {
	for i := range c.Teams {
		if c.Teams[i].Name == name {
			return &c.Teams[i]
		}
	}
	return nil
}

// GetAllUsers returns all users from all teams.
func (c *Config) GetAllUsers() []MemberConfig {
	var users []MemberConfig
	for _, team := range c.Teams {
		users = append(users, team.Members...)
	}
	return users
}

// GetUsersByRole returns all users with a specific role.
func (c *Config) GetUsersByRole(role string) []MemberConfig {
	var users []MemberConfig
	for _, team := range c.Teams {
		for _, member := range team.Members {
			if member.Role == role {
				users = append(users, member)
			}
		}
	}
	return users
}
