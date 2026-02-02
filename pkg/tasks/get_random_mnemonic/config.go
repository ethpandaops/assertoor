package getrandommnemonic

type Config struct {
	MnemonicResultVar string `yaml:"mnemonicResultVar" json:"mnemonicResultVar" deprecated:"true" desc:"Deprecated: Use task outputs instead."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
