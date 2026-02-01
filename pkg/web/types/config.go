package types

import "time"

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
