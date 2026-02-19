package runshell

import "errors"

type Config struct {
	Shell     string            `yaml:"shell" json:"shell" desc:"Shell interpreter to use (e.g., bash, sh, zsh)."`
	ShellArgs []string          `yaml:"shellArgs" json:"shellArgs" desc:"Additional arguments to pass to the shell."`
	EnvVars   map[string]string `yaml:"envVars" json:"envVars" format:"expressionMap" desc:"Environment variables to set for the shell command."`
	Command   string            `yaml:"command" json:"command" require:"A" format:"shell" desc:"The shell command to execute."`
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
