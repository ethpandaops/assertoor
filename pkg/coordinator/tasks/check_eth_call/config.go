package checkethcall

type Config struct {
	EthCallData    string   `yaml:"ethCallData" json:"ethCallData"`
	ExpectResult   string   `yaml:"expectResult" json:"expectResult"`
	IgnoreResults  []string `yaml:"ignoreResults" json:"ignoreResults"`
	CallAddress    string   `yaml:"callAddress" json:"callAddress"`
	BlockNumber    uint64   `yaml:"blockNumber" json:"blockNumber"`
	FailOnMismatch bool     `yaml:"failOnMismatch" json:"failOnMismatch"`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern"`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern"`
}

func DefaultConfig() Config {
	return Config{
		EthCallData:  "0x",
		ExpectResult: "",
		CallAddress:  "0x0000000000000000000000000000000000000000",
	}
}

func (c *Config) Validate() error {
	return nil
}
