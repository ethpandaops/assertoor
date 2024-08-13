package getrandommnemonic

type Config struct {
	MnemonicResultVar string `yaml:"mnemonicResultVar" json:"mnemonicResultVar"`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
