package txpoolcheck

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`

	TxCount            int      `yaml:"txCount" json:"txCount"`
	MeasureInterval    int      `yaml:"measureInterval" json:"measureInterval"`
}

func DefaultConfig() Config {
	return Config{
		TxCount:         1000,
		MeasureInterval: 100,
	}
}

func (c *Config) Validate() error {
	return nil
}
