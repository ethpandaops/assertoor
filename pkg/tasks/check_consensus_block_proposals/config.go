package checkconsensusblockproposals

import "math/big"

type Config struct {
	CheckLookback                int    `yaml:"checkLookback" json:"checkLookback" desc:"Number of slots to look back when checking for block proposals."`
	BlockCount                   int    `yaml:"blockCount" json:"blockCount" desc:"Number of matching blocks required to pass the check."`
	PayloadTimeout               int    `yaml:"payloadTimeout" json:"payloadTimeout" desc:"Timeout in seconds to wait for execution payload (gloas+). Default: 12"`
	GraffitiPattern              string `yaml:"graffitiPattern" json:"graffitiPattern" desc:"Regex pattern to match block graffiti."`
	ValidatorNamePattern         string `yaml:"validatorNamePattern" json:"validatorNamePattern" desc:"Regex pattern to match validator names."`
	ExtraDataPattern             string `yaml:"extraDataPattern" json:"extraDataPattern" desc:"Regex pattern to match execution payload extra data."`
	MinAttestationCount          int    `yaml:"minAttestationCount" json:"minAttestationCount" desc:"Minimum number of attestations required in the block."`
	MinDepositCount              int    `yaml:"minDepositCount" json:"minDepositCount" desc:"Minimum number of deposits required in the block."`
	MinExitCount                 int    `yaml:"minExitCount" json:"minExitCount" desc:"Minimum number of voluntary exits required in the block."`
	MinSlashingCount             int    `yaml:"minSlashingCount" json:"minSlashingCount" desc:"Minimum number of slashings (attester + proposer) required in the block."`
	MinAttesterSlashingCount     int    `yaml:"minAttesterSlashingCount" json:"minAttesterSlashingCount" desc:"Minimum number of attester slashings required in the block."`
	MinProposerSlashingCount     int    `yaml:"minProposerSlashingCount" json:"minProposerSlashingCount" desc:"Minimum number of proposer slashings required in the block."`
	MinBlsChangeCount            int    `yaml:"minBlsChangeCount" json:"minBlsChangeCount" desc:"Minimum number of BLS to execution changes required in the block."`
	MinWithdrawalCount           int    `yaml:"minWithdrawalCount" json:"minWithdrawalCount" desc:"Minimum number of withdrawals required in the block."`
	MinTransactionCount          int    `yaml:"minTransactionCount" json:"minTransactionCount" desc:"Minimum number of transactions required in the block."`
	MinBlobCount                 int    `yaml:"minBlobCount" json:"minBlobCount" desc:"Minimum number of blob sidecars required in the block."`
	MinDepositRequestCount       int    `yaml:"minDepositRequestCount" json:"minDepositRequestCount" desc:"Minimum number of deposit requests required in the block."`
	MinWithdrawalRequestCount    int    `yaml:"minWithdrawalRequestCount" json:"minWithdrawalRequestCount" desc:"Minimum number of withdrawal requests required in the block."`
	MinConsolidationRequestCount int    `yaml:"minConsolidationRequestCount" json:"minConsolidationRequestCount" desc:"Minimum number of consolidation requests required in the block."`

	ExpectDeposits  []string `yaml:"expectDeposits" json:"expectDeposits" desc:"List of validator public keys expected to have deposits in the block."`
	ExpectExits     []string `yaml:"expectExits" json:"expectExits" desc:"List of validator public keys expected to have exits in the block."`
	ExpectSlashings []struct {
		PublicKey    string `yaml:"publicKey" json:"publicKey" desc:"Public key of the slashed validator."`
		SlashingType string `yaml:"slashingType" json:"slashingType" desc:"Type of slashing: 'attester' or 'proposer'."`
	} `yaml:"expectSlashings" json:"expectSlashings" desc:"List of expected slashings in the block."`
	ExpectBlsChanges []struct {
		PublicKey string `yaml:"publicKey" json:"publicKey" desc:"Public key of the validator."`
		Address   string `yaml:"address" json:"address" desc:"Target execution layer address."`
	} `yaml:"expectBlsChanges" json:"expectBlsChanges" desc:"List of expected BLS to execution changes in the block."`
	ExpectWithdrawals []struct {
		PublicKey string   `yaml:"publicKey" json:"publicKey" desc:"Public key of the validator."`
		Address   string   `yaml:"address" json:"address" desc:"Withdrawal address."`
		MinAmount *big.Int `yaml:"minAmount" json:"minAmount" desc:"Minimum withdrawal amount."`
		MaxAmount *big.Int `yaml:"maxAmount" json:"maxAmount" desc:"Maximum withdrawal amount."`
	} `yaml:"expectWithdrawals" json:"expectWithdrawals" desc:"List of expected withdrawals in the block."`
	ExpectDepositRequests []struct {
		PublicKey             string   `yaml:"publicKey" json:"publicKey" desc:"Public key of the validator."`
		WithdrawalCredentials string   `yaml:"withdrawalCredentials" json:"withdrawalCredentials" desc:"Withdrawal credentials."`
		Amount                *big.Int `yaml:"amount" json:"amount" desc:"Deposit amount."`
	} `yaml:"expectDepositRequests" json:"expectDepositRequests" desc:"List of expected deposit requests in the block."`
	ExpectWithdrawalRequests []struct {
		SourceAddress   string   `yaml:"sourceAddress" json:"sourceAddress" desc:"Source address initiating the withdrawal."`
		ValidatorPubkey string   `yaml:"validatorPubkey" json:"validatorPubkey" desc:"Public key of the validator."`
		Amount          *big.Int `yaml:"amount" json:"amount" desc:"Withdrawal amount."`
	} `yaml:"expectWithdrawalRequests" json:"expectWithdrawalRequests" desc:"List of expected withdrawal requests in the block."`
	ExpectConsolidationRequests []struct {
		SourceAddress string `yaml:"sourceAddress" json:"sourceAddress" desc:"Source address initiating the consolidation."`
		SourcePubkey  string `yaml:"sourcePubkey" json:"sourcePubkey" desc:"Public key of the source validator."`
		TargetPubkey  string `yaml:"targetPubkey" json:"targetPubkey" desc:"Public key of the target validator."`
	} `yaml:"expectConsolidationRequests" json:"expectConsolidationRequests" desc:"List of expected consolidation requests in the block."`
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
