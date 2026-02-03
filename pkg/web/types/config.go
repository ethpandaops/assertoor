package types

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type WebConfig struct {
	Server       *ServerConfig   `yaml:"server"`
	PublicServer *ServerConfig   `yaml:"publicServer"`
	Frontend     *FrontendConfig `yaml:"frontend"`
	API          *APIConfig      `yaml:"api"`
}

type ServerConfig struct {
	Port string `yaml:"port" envconfig:"WEB_SERVER_PORT"`
	Host string `yaml:"host" envconfig:"WEB_SERVER_HOST"`

	ReadTimeout  time.Duration `yaml:"readTimeout" envconfig:"WEB_SERVER_READ_TIMEOUT"`
	WriteTimeout time.Duration `yaml:"writeTimeout" envconfig:"WEB_SERVER_WRITE_TIMEOUT"`
	IdleTimeout  time.Duration `yaml:"idleTimeout" envconfig:"WEB_SERVER_IDLE_TIMEOUT"`

	AuthHeader string `yaml:"authHeader" envconfig:"WEB_SERVER_AUTH_HEADER"`
	TokenKey   string `yaml:"tokenKey" envconfig:"WEB_SERVER_TOKEN_KEY"`
}

type FrontendConfig struct {
	Enabled  bool   `yaml:"enabled" envconfig:"WEB_FRONTEND_ENABLED"`
	Debug    bool   `yaml:"debug" envconfig:"WEB_FRONTEND_DEBUG"`
	Pprof    bool   `yaml:"pprof" envconfig:"WEB_FRONTEND_PPROF"`
	Minify   bool   `yaml:"minify" envconfig:"WEB_FRONTEND_MINIFY"`
	SiteName string `yaml:"siteName" envconfig:"WEB_FRONTEND_SITE_NAME"`
}

type APIConfig struct {
	Enabled     bool `yaml:"enabled" envconfig:"WEB_API_ENABLED"`
	DisableAuth bool `yaml:"disableAuth" envconfig:"WEB_API_DISABLE_AUTH"`
}

// AIConfig holds configuration for the AI assistant feature.
type AIConfig struct {
	// Enable AI features
	Enabled bool `yaml:"enabled" json:"enabled"`

	// OpenRouter API key (can also be set via AI_OPENROUTER_KEY env var)
	OpenRouterKey string `yaml:"openRouterKey" json:"openRouterKey"`

	// Default model to use (default: "anthropic/claude-sonnet-4")
	DefaultModel string `yaml:"defaultModel" json:"defaultModel"`

	// Allowed models (empty means all models allowed)
	AllowedModels []string `yaml:"allowedModels" json:"allowedModels"`

	// Max tokens per response (default: 8192)
	MaxTokens int `yaml:"maxTokens" json:"maxTokens"`
}

// ApplyEnvironment applies environment variables to AIConfig.
// Environment variables override config file values.
func (c *AIConfig) ApplyEnvironment() {
	if v := os.Getenv("AI_ENABLED"); v != "" {
		c.Enabled = v == "true" || v == "1"
	}

	if v := os.Getenv("AI_OPENROUTER_KEY"); v != "" {
		c.OpenRouterKey = v
	}

	if v := os.Getenv("AI_DEFAULT_MODEL"); v != "" {
		c.DefaultModel = v
	}

	if v := os.Getenv("AI_MAX_TOKENS"); v != "" {
		if maxTokens, err := strconv.Atoi(v); err == nil {
			c.MaxTokens = maxTokens
		}
	}
}

// Validate validates the AI configuration.
func (c *AIConfig) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.OpenRouterKey == "" {
		return fmt.Errorf("openRouterKey is required when AI is enabled")
	}

	if c.DefaultModel == "" {
		return fmt.Errorf("defaultModel cannot be empty when AI is enabled")
	}

	if c.MaxTokens <= 0 {
		return fmt.Errorf("maxTokens must be positive")
	}

	return nil
}

// DefaultAIConfig returns sensible defaults for AI configuration.
func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		Enabled:      false,
		DefaultModel: "anthropic/claude-sonnet-4",
		MaxTokens:    8192,
		AllowedModels: []string{
			"anthropic/claude-sonnet-4",
			"anthropic/claude-opus-4",
			"openai/gpt-4o",
			"google/gemini-2.0-flash-001",
		},
	}
}
