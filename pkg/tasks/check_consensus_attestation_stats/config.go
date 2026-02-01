package checkconsensusattestationstats

type Config struct {
	MinTargetPercent uint64 `yaml:"minTargetPercent" json:"minTargetPercent"`
	MaxTargetPercent uint64 `yaml:"maxTargetPercent" json:"maxTargetPercent"`
	MinHeadPercent   uint64 `yaml:"minHeadPercent" json:"minHeadPercent"`
	MaxHeadPercent   uint64 `yaml:"maxHeadPercent" json:"maxHeadPercent"`
	MinTotalPercent  uint64 `yaml:"minTotalPercent" json:"minTotalPercent"`
	MaxTotalPercent  uint64 `yaml:"maxTotalPercent" json:"maxTotalPercent"`
	FailOnCheckMiss  bool   `yaml:"failOnCheckMiss" json:"failOnCheckMiss"`
	MinCheckedEpochs uint64 `yaml:"minCheckedEpochs" json:"minCheckedEpochs"`
	// ContinueOnPass keeps the task running after the check passes.
	// When false (default), the task exits immediately on success.
	// When true, the task continues monitoring and may report failure if attestation stats change.
	ContinueOnPass bool `yaml:"continueOnPass" json:"continueOnPass"`
}

func DefaultConfig() Config {
	return Config{
		MaxTargetPercent: 100,
		MaxHeadPercent:   100,
		MaxTotalPercent:  100,
		MinCheckedEpochs: 1,
	}
}

func (c *Config) Validate() error {
	return nil
}
