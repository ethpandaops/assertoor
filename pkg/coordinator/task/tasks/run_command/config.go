package runcommand

import "errors"

type Config struct {
	AllowedToFail bool     `yaml:"allowed_to_fail" json:"allowed_to_fail"`
	Command       []string `yaml:"command" json:"command"`
}

func DefaultConfig() Config {
	return Config{
		Command:       []string{},
		AllowedToFail: false,
	}
}

func (c *Config) Validate() error {
	if len(c.Command) == 0 {
		return errors.New("command must be specified")
	}

	return nil
}
