package checkethconfig

type Config struct {
	ClientPattern         string `yaml:"clientPattern" json:"clientPattern" desc:"Regex pattern to select specific client endpoints for checking configuration."`
	ExcludeClientPattern  string `yaml:"excludeClientPattern" json:"excludeClientPattern" desc:"Regex pattern to exclude certain client endpoints."`
	FailOnMismatch        bool   `yaml:"failOnMismatch" json:"failOnMismatch" desc:"If true, fail the task when client configurations do not match."`
	ExcludeSyncingClients bool   `yaml:"excludeSyncingClients" json:"excludeSyncingClients" desc:"If true, exclude clients that are still syncing from the check."`
}

func DefaultConfig() Config {
	return Config{
		FailOnMismatch:        true,
		ExcludeSyncingClients: false,
	}
}

func (c *Config) Validate() error {
	return nil
}
