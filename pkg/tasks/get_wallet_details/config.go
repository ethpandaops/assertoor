package getwalletdetails

import "errors"

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey" desc:"Private key of the wallet to get details for."`
	Address    string `yaml:"address" json:"address" desc:"Address of the wallet to get details for (alternative to privateKey)."`
}

func DefaultConfig() Config {
	return Config{}
}

func (c *Config) Validate() error {
	if c.PrivateKey == "" && c.Address == "" {
		return errors.New("either privateKey or address must be set")
	}

	if c.PrivateKey != "" && c.Address != "" {
		return errors.New("only one of privateKey or address must be set")
	}

	return nil
}
