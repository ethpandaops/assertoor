package runshell

import "errors"

type Config struct {
	Shell     string            `yaml:"shell" json:"shell"`
	ShellArgs []string          `yaml:"shellArgs" json:"shellArgs"`
	EnvVars   map[string]string `yaml:"envVars" json:"envVars"`
	Command   string            `yaml:"command" json:"command"`
}

func DefaultConfig() Config {
	return Config{
		Shell:     "bash",
		ShellArgs: []string{},
	}
}

func (c *Config) Validate() error {
	if c.Command == "" {
		return errors.New("command must be specified")
	}

	return nil
}
