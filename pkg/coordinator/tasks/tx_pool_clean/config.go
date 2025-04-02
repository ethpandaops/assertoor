package txpoolclean

// import "time"

type Config struct {
	// WaitTime           time.Duration      `yaml:"waitTime" json:"waitTime"`
}

func DefaultConfig() Config {
	return Config{
		// WaitTime: time.Duration(5), // in seconds
	}
}

func (c *Config) Validate() error {
	return nil
}
