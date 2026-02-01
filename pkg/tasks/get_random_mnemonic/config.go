package getrandommnemonic

type Config struct {
	MnemonicResultVar string `yaml:"mnemonicResultVar" json:"mnemonicResultVar" desc:"Variable name to store the generated mnemonic phrase."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	return nil
}
