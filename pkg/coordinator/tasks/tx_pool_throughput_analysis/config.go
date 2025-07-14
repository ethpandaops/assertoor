package txpoolthroughputanalysis

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	StartingTPS          int `yaml:"tps" json:"tps"`
	EndingTPS            int `yaml:"endingTps" json:"endingTps"`
	IncrementTPS         int `yaml:"incrementTps" json:"incrementTps"`
	DurationS            int `yaml:"durationS" json:"durationS"`
	LogInterval          int `yaml:"logInterval" json:"logInterval"`
	SecondsBeforeRunning int `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		StartingTPS:          100,
		EndingTPS:            1000,
		IncrementTPS:         100,
		DurationS:            60,
		LogInterval:          100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
