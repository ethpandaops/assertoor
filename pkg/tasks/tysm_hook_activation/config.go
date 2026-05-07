package tysmhookactivation

import (
	"errors"
	"net/url"
	"strings"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

// Config drives a single POST to the TYSM hook-control API.
//
// AuthToken is intentionally yaml-tagged with snake_case: it is expected
// to be supplied via the task's configVars (e.g. configVars: { auth_token:
// "tysmApiToken" }) so the secret never lives in the playbook source.
// Inline assignment under config: still works, it is just discouraged.
type Config struct {
	Endpoint    string                 `yaml:"endpoint" json:"endpoint" require:"A" desc:"Base URL of the TYSM API, e.g. http://beacon:8080"`
	AuthToken   string                 `yaml:"auth_token" json:"auth_token" desc:"Bearer token sent in the Authorization header. Prefer supplying via configVars."`
	Hook        string                 `yaml:"hook" json:"hook" require:"A" desc:"Name of the TYSM hook to activate (e.g. blob-mutator)."`
	Enabled     *bool                  `yaml:"enabled,omitempty" json:"enabled,omitempty" desc:"Override the hook's enabled flag while the activation is in force."`
	ConfigPatch map[string]interface{} `yaml:"configPatch,omitempty" json:"configPatch,omitempty" desc:"Top-level config keys to overlay on top of the hook's baseline configuration."`
	Duration    helper.Duration        `yaml:"duration" json:"duration" require:"A" desc:"Activation TTL (Go duration: 10m, 1h, ...). The server enforces a hard cap; values exceeding it are rejected."`
	Replace     bool                   `yaml:"replace" json:"replace" desc:"If true, replace any existing activation for the same hook instead of returning 409 Conflict."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Endpoint) == "" {
		return errors.New("endpoint is required")
	}

	if _, err := url.Parse(c.Endpoint); err != nil {
		return errors.New("endpoint must be a valid URL")
	}

	if strings.TrimSpace(c.Hook) == "" {
		return errors.New("hook is required")
	}

	if c.Duration.Duration <= 0 {
		return errors.New("duration must be greater than 0")
	}

	if c.Enabled == nil && len(c.ConfigPatch) == 0 {
		return errors.New("at least one of enabled or configPatch must be set")
	}

	return nil
}
