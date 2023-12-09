package runshell

import "errors"

type Config struct {
	Shell   string            `yaml:"shell" json:"shell"`
	EnvVars map[string]string `yaml:"envVars" json:"envVars"`
	Command string            `yaml:"command" json:"command"`
}

func DefaultConfig() Config {
	return Config{
		Shell: "sh",
	}
}

func (c *Config) Validate() error {
	if c.Command == "" {
		return errors.New("command must be specified")
	}

	return nil
}
