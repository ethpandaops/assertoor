package checkethconfig

type Config struct {
	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
	FailOnMismatch       bool   `yaml:"failOnMismatch" json:"failOnMismatch"`
}

func DefaultConfig() Config {
	return Config{
		FailOnMismatch: true,
	}
}

func (c *Config) Validate() error {
	return nil
}
