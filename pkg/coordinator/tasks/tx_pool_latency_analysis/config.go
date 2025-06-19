package txpoollatencyanalysis

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	TPS                  int   `yaml:"tps" json:"tps"`
	DurationS            int   `yaml:"durationS" json:"durationS"`
	LogInterval          int   `yaml:"logInterval" json:"logInterval"`
	SecondsBeforeRunning int64 `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		TPS:                  100,
		DurationS:            60,
		LogInterval:          100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
