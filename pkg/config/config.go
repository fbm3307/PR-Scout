package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for pr-scout.
type Config struct {
	GitHub   GitHubConfig   `yaml:"github"`
	LLM      LLMConfig      `yaml:"llm"`
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
}

type GitHubConfig struct {
	Org            string   `yaml:"org"`
	Username       string   `yaml:"username"`
	TokenEnv       string   `yaml:"token_env"`
	Repos          []string `yaml:"repos"`             // If set, only scan these repos (empty = scan all)
	MaxPRsPerRepo  int      `yaml:"max_prs_per_repo"`  // Limit PRs per repo (0 = no limit)
}

// Token resolves the GitHub token from the environment.
func (c GitHubConfig) Token() string {
	if c.TokenEnv == "" {
		return os.Getenv("GITHUB_TOKEN")
	}
	return os.Getenv(c.TokenEnv)
}

type LLMConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Provider      string `yaml:"provider"`
	Model         string `yaml:"model"`
	ProjectIDEnv  string `yaml:"project_id_env"`
	Region        string `yaml:"region"`
	MaxPRAgeDays  int    `yaml:"max_pr_age_days"` // Skip LLM for PRs older than this (0 = no limit)
}

// ProjectID resolves the Vertex AI project ID from the environment.
func (c LLMConfig) ProjectID() string {
	if c.ProjectIDEnv == "" {
		return os.Getenv("ANTHROPIC_VERTEX_PROJECT_ID")
	}
	return os.Getenv(c.ProjectIDEnv)
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // "sqlite" or "postgres"

	// PostgreSQL fields (resolved from env)
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`

	// SQLite fields
	Path string `yaml:"path"`
}

type ServerConfig struct {
	Port         int    `yaml:"port"`
	DashboardURL string `yaml:"dashboard_url"`
}

// Load reads config from a YAML file, then applies env var overrides.
// It also loads .env from the same directory if present.
func Load(configDir string) (*Config, error) {
	// Try loading .env file (non-fatal if missing)
	envPath := configDir + "/.env"
	if err := godotenv.Load(envPath); err != nil {
		// Silently continue — .env is optional
	}

	yamlPath := configDir + "/pr-scout.yaml"
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", yamlPath, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", yamlPath, err)
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.DashboardURL == "" {
		c.Server.DashboardURL = "http://localhost:5173"
	}
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.Database.Path == "" {
		c.Database.Path = "./pr-scout.db"
	}
	if c.Database.SSLMode == "" {
		c.Database.SSLMode = "disable"
	}
	if c.Database.Port == 0 {
		c.Database.Port = 5432
	}
	if c.LLM.Region == "" {
		c.LLM.Region = "us-east5"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "claude-sonnet-4@20250514"
	}
}

func (c *Config) applyEnvOverrides() {
	// PostgreSQL env overrides (match tarsy conventions)
	if v := os.Getenv("DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		c.Database.DBName = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		c.Database.SSLMode = v
	}
}
