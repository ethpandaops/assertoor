package checkconsensusblockproposals

type Config struct {
	BlockCount               int    `yaml:"blockCount" json:"blockCount"`
	GraffitiPattern          string `yaml:"graffitiPattern" json:"graffitiPattern"`
	ValidatorNamePattern     string `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	MinAttestationCount      int    `yaml:"minAttestationCount" json:"minAttestationCount"`
	MinDepositCount          int    `yaml:"minDepositCount" json:"minDepositCount"`
	MinExitCount             int    `yaml:"minExitCount" json:"minExitCount"`
	MinSlashingCount         int    `yaml:"minSlashingCount" json:"minSlashingCount"`
	MinAttesterSlashingCount int    `yaml:"minAttesterSlashingCount" json:"minAttesterSlashingCount"`
	MinProposerSlashingCount int    `yaml:"minProposerSlashingCount" json:"minProposerSlashingCount"`
	MinBlsChangeCount        int    `yaml:"minBlsChangeCount" json:"minBlsChangeCount"`
	MinWithdrawalCount       int    `yaml:"minWithdrawalCount" json:"minWithdrawalCount"`
	MinTransactionCount      int    `yaml:"minTransactionCount" json:"minTransactionCount"`
	MinBlobCount             int    `yaml:"minBlobCount" json:"minBlobCount"`
}

func DefaultConfig() Config {
	return Config{
		BlockCount: 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
