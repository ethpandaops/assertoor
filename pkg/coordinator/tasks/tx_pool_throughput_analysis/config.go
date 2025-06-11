package txpoolcheck

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	QPS                  int `yaml:"qps" json:"qps"`
	Duration_s           int `yaml:"duration_s" json:"duration_s"`
	MeasureInterval      int `yaml:"measureInterval" json:"measureInterval"`
	SecondsBeforeRunning int `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		QPS:                  100,
		Duration_s:           60,
		MeasureInterval:      100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
