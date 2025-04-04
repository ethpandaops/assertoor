package txpoolcheck

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	TxCount              int `yaml:"txCount" json:"txCount"`
	MeasureInterval      int `yaml:"measureInterval" json:"measureInterval"`
	SecondsBeforeRunning int `yaml:"secondsBeforeRunning" json:"secondsBeforeRunning"`
}

func DefaultConfig() Config {
	return Config{
		TxCount:            1000,
		MeasureInterval:    100,
		SecondsBeforeRunning: 0,
	}
}

func (c *Config) Validate() error {
	return nil
}
