package coordinator

type Consensus struct {
	URL string `yaml:"url"`
}

// ExecutionNode represents a single ethereum execution client.
type Execution struct {
	URL string `yaml:"url"`
}

type Config struct {
	// Test is the name of the test to run.
	Test string
	// Execution is the execution node to use.
	Execution Execution `yaml:"execution"`
	// Consensus is the consensus node to use.
	Consensus Consensus `yaml:"consensus"`
}

// DefaultConfig represents a sane-default configuration.
func DefaultConfig() *Config {
	return &Config{
		Test: "both_synced",
		Execution: Execution{
			URL: "http://localhost:8545",
		},
		Consensus: Consensus{
			URL: "http://localhost:5052",
		},
	}
}
