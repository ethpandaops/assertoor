package checkethcall

type Config struct {
	EthCallData  string `yaml:"EthCallData" json:"EthCallData"`
	ExpectResult string `yaml:"ExpectResult" json:"ExpectResult"`
	CallAddress  string `yaml:"CallAddress" json:"CallAddress"`
}

func DefaultConfig() Config {
	return Config{
		EthCallData:  "0x0000000000000000000000000000000000000000000000000000000000000000",
		ExpectResult: "0x0000000000000000000000000000000000000000000000000000000000000000",
		CallAddress:  "0x0000000000000000000000000000000000000000",
	}
}

func (c *Config) Validate() error {
	return nil
}
