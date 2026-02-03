package checkconsensusreorgs

type Config struct {
	MinCheckEpochCount uint64  `yaml:"minCheckEpochCount" json:"minCheckEpochCount" desc:"Minimum number of epochs to monitor before evaluating reorg conditions."`
	MaxReorgDistance   uint64  `yaml:"maxReorgDistance" json:"maxReorgDistance" desc:"Maximum allowed reorg distance (depth) in slots."`
	MaxReorgsPerEpoch  float64 `yaml:"maxReorgsPerEpoch" json:"maxReorgsPerEpoch" desc:"Maximum allowed average number of reorgs per epoch."`
	MaxTotalReorgs     uint64  `yaml:"maxTotalReorgs" json:"maxTotalReorgs" desc:"Maximum total number of reorgs allowed during monitoring."`
	ContinueOnPass     bool    `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
}

func DefaultConfig() Config {
	return Config{
		MinCheckEpochCount: 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
