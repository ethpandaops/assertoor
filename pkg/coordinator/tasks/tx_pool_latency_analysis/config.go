package txpoollatencyanalysis

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	TxCount              int   `yaml:"txCount" json:"txCount"`
	MeasureInterval      int   `yaml:"measureInterval" json:"measureInterval"`
	ExpectedLatency      int64 `yaml:"expectedLatency" json:"expectedLatency"`
	FailOnHighLatency    bool  `yaml:"failOnHighLatency" json:"failOnHighLatency"`
	SecondsBeforeRunning int64 `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		TxCount:              1000,
		MeasureInterval:      100,
		ExpectedLatency:      5000, // in microseconds
		FailOnHighLatency:    true,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
