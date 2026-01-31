package checkconsensusblockproposals

import "math/big"

type Config struct {
	CheckLookback                int    `yaml:"checkLookback" json:"checkLookback"`
	BlockCount                   int    `yaml:"blockCount" json:"blockCount"`
	GraffitiPattern              string `yaml:"graffitiPattern" json:"graffitiPattern"`
	ValidatorNamePattern         string `yaml:"validatorNamePattern" json:"validatorNamePattern"`
	ExtraDataPattern             string `yaml:"extraDataPattern" json:"extraDataPattern"`
	MinAttestationCount          int    `yaml:"minAttestationCount" json:"minAttestationCount"`
	MinDepositCount              int    `yaml:"minDepositCount" json:"minDepositCount"`
	MinExitCount                 int    `yaml:"minExitCount" json:"minExitCount"`
	MinSlashingCount             int    `yaml:"minSlashingCount" json:"minSlashingCount"`
	MinAttesterSlashingCount     int    `yaml:"minAttesterSlashingCount" json:"minAttesterSlashingCount"`
	MinProposerSlashingCount     int    `yaml:"minProposerSlashingCount" json:"minProposerSlashingCount"`
	MinBlsChangeCount            int    `yaml:"minBlsChangeCount" json:"minBlsChangeCount"`
	MinWithdrawalCount           int    `yaml:"minWithdrawalCount" json:"minWithdrawalCount"`
	MinTransactionCount          int    `yaml:"minTransactionCount" json:"minTransactionCount"`
	MinBlobCount                 int    `yaml:"minBlobCount" json:"minBlobCount"`
	MinDepositRequestCount       int    `yaml:"minDepositRequestCount" json:"minDepositRequestCount"`
	MinWithdrawalRequestCount    int    `yaml:"minWithdrawalRequestCount" json:"minWithdrawalRequestCount"`
	MinConsolidationRequestCount int    `yaml:"minConsolidationRequestCount" json:"minConsolidationRequestCount"`

	ExpectDeposits  []string `yaml:"expectDeposits" json:"expectDeposits"`
	ExpectExits     []string `yaml:"expectExits" json:"expectExits"`
	ExpectSlashings []struct {
		PublicKey    string `yaml:"publicKey" json:"publicKey"`
		SlashingType string `yaml:"slashingType" json:"slashingType"`
	} `yaml:"expectSlashings" json:"expectSlashings"`
	ExpectBlsChanges []struct {
		PublicKey string `yaml:"publicKey" json:"publicKey"`
		Address   string `yaml:"address" json:"address"`
	} `yaml:"expectBlsChanges" json:"expectBlsChanges"`
	ExpectWithdrawals []struct {
		PublicKey string   `yaml:"publicKey" json:"publicKey"`
		Address   string   `yaml:"address" json:"address"`
		MinAmount *big.Int `yaml:"minAmount" json:"minAmount"`
		MaxAmount *big.Int `yaml:"maxAmount" json:"maxAmount"`
	} `yaml:"expectWithdrawals" json:"expectWithdrawals"`
	ExpectDepositRequests []struct {
		PublicKey             string   `yaml:"publicKey" json:"publicKey"`
		WithdrawalCredentials string   `yaml:"withdrawalCredentials" json:"withdrawalCredentials"`
		Amount                *big.Int `yaml:"amount" json:"amount"`
	} `yaml:"expectDepositRequests" json:"expectDepositRequests"`
	ExpectWithdrawalRequests []struct {
		SourceAddress   string   `yaml:"sourceAddress" json:"sourceAddress"`
		ValidatorPubkey string   `yaml:"validatorPubkey" json:"validatorPubkey"`
		Amount          *big.Int `yaml:"amount" json:"amount"`
	} `yaml:"expectWithdrawalRequests" json:"expectWithdrawalRequests"`
	ExpectConsolidationRequests []struct {
		SourceAddress string `yaml:"sourceAddress" json:"sourceAddress"`
		SourcePubkey  string `yaml:"sourcePubkey" json:"sourcePubkey"`
		TargetPubkey  string `yaml:"targetPubkey" json:"targetPubkey"`
	} `yaml:"expectConsolidationRequests" json:"expectConsolidationRequests"`
}

func DefaultConfig() Config {
	return Config{
		CheckLookback: 1,
		BlockCount:    1,
	}
}

func (c *Config) Validate() error {
	return nil
}
