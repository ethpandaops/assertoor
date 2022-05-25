package runcommand

import "errors"

type Config struct {
	Command []string `yaml:"command" json:"command"`
}

func DefaultConfig() Config {
	return Config{
		Command: []string{},
	}
}

func (c *Config) Validate() error {
	if len(c.Command) == 0 {
		return errors.New("command must be specified")
	}

	return nil
}
