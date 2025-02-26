package txpoolcheck

type Config struct {
	TxCount            int      `yaml:"txCount" json:"txCount"`
	MeasureInterval    int      `yaml:"measureInterval" json:"measureInterval"`
	ExpectedLatency    int64    `yaml:"expectedLatency" json:"expectedLatency"`
	FailOnHighLatency  bool     `yaml:"failOnHighLatency" json:"failOnHighLatency"`
	ClientPattern      string   `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
}

func DefaultConfig() Config {
	return Config{
		TxCount:         1000,
		MeasureInterval: 100,
		ExpectedLatency: 500, // in milliseconds
	}
}

func (c *Config) Validate() error {
	return nil
}
