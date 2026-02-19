package checkethcall

type Config struct {
	EthCallData    string   `yaml:"ethCallData" json:"ethCallData" desc:"Hex-encoded call data to send with eth_call."`
	ExpectResult   string   `yaml:"expectResult" json:"expectResult" desc:"Expected hex-encoded result from eth_call."`
	IgnoreResults  []string `yaml:"ignoreResults" json:"ignoreResults" desc:"List of hex-encoded results to ignore (not treat as failures)."`
	CallAddress    string   `yaml:"callAddress" json:"callAddress" desc:"Target contract address for eth_call."`
	BlockNumber    uint64   `yaml:"blockNumber" json:"blockNumber" desc:"Block number to execute eth_call at (0 for latest)."`
	FailOnMismatch bool     `yaml:"failOnMismatch" json:"failOnMismatch" desc:"If true, fail the task when eth_call result does not match expected."`

	ClientPattern        string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for eth_call."`
	ExcludeClientPattern string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	ContinueOnPass       bool   `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
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
