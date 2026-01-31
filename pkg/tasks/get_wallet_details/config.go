package getwalletdetails

import "errors"

type Config struct {
	PrivateKey string `yaml:"privateKey" json:"privateKey"`
	Address    string `yaml:"address" json:"address"`
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
