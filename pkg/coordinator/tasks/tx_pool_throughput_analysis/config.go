package txpoolcheck

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	QPS              		 int `yaml:"qps" json:"qps"`
	MeasureInterval      int `yaml:"measureInterval" json:"measureInterval"`
	SecondsBeforeRunning int `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		QPS:            			1000,
		MeasureInterval:    	100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
