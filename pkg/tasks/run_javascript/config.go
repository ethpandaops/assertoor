package runjavascript

import "errors"

type Config struct {
	NodePath string            `yaml:"nodePath" json:"nodePath" desc:"Path to the node interpreter. Default: 'node' from PATH."`
	NodeArgs []string          `yaml:"nodeArgs" json:"nodeArgs" desc:"Additional arguments to pass to node."`
	EnvVars  map[string]string `yaml:"envVars" json:"envVars" format:"expressionMap" desc:"Variables to expose to the script. Each value is a runtime variable query (e.g. 'tasks') resolved at task start. JSON-encoded into env vars and also surfaced as the 'env' object inside the script."`
	Script   string            `yaml:"script" json:"script" require:"A" format:"javascript" desc:"The JavaScript source to execute. Wrapped in an async function so 'await' works at the top level."`
}

func DefaultConfig() Config {
	return Config{
		NodePath: "node",
		NodeArgs: []string{},
	}
}

func (c *Config) Validate() error {
	if c.Script == "" {
		return errors.New("script must be specified")
	}

	if c.NodePath == "" {
		c.NodePath = "node"
	}

	return nil
}
