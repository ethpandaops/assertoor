package tysmhookdeactivation

import (
	"errors"
	"net/url"
	"strings"
)

// Config drives a single DELETE against the TYSM hook-control API.
//
// ActivationID is yaml-tagged with snake_case so that playbooks can wire it
// up via configVars from a previous tysm_hook_activation task's outputs:
//
//	configVars:
//	  activation_id: "tasks.kzg_chaos.outputs.activation_id"
//
// AuthToken follows the same convention so the bearer token can be
// supplied via a runtime variable rather than hard-coded in the playbook.
type Config struct {
	Endpoint       string `yaml:"endpoint" json:"endpoint" require:"A" desc:"Base URL of the TYSM API, e.g. http://beacon:8080"`
	AuthToken      string `yaml:"auth_token" json:"auth_token" desc:"Bearer token sent in the Authorization header. Prefer supplying via configVars."`
	ActivationID   string `yaml:"activation_id" json:"activation_id" require:"A" desc:"Activation ID returned by tysm_hook_activation. Typically supplied via configVars from the upstream task's outputs."`
	IgnoreNotFound bool   `yaml:"ignoreNotFound" json:"ignoreNotFound" desc:"If true (default), treat HTTP 404 as success — the activation may have already expired via TTL."`
}

func DefaultConfig() Config {
	return Config{
		IgnoreNotFound: true,
	}
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Endpoint) == "" {
		return errors.New("endpoint is required")
	}

	if _, err := url.Parse(c.Endpoint); err != nil {
		return errors.New("endpoint must be a valid URL")
	}

	if strings.TrimSpace(c.ActivationID) == "" {
		return errors.New("activation_id is required")
	}

	return nil
}
