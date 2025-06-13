package txpool_latency_analysis

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	TPS                  int   `yaml:"tps" json:"tps"`
	Duration_s           int   `yaml:"duration_s" json:"duration_s"`
	MeasureInterval      int   `yaml:"measureInterval" json:"measureInterval"`
	SecondsBeforeRunning int64 `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		TPS:                  100,
		Duration_s:           60,
		MeasureInterval:      100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
