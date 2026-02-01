package runcommand

import "errors"

type Config struct {
	AllowedToFail bool     `yaml:"allowed_to_fail" json:"allowed_to_fail" desc:"If true, the task succeeds even if the command fails."`
	Command       []string `yaml:"command" json:"command" desc:"The command and arguments to execute."`
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
