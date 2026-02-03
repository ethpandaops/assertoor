package checkconsensusattestationstats

type Config struct {
	MinTargetPercent uint64 `yaml:"minTargetPercent" json:"minTargetPercent" desc:"Minimum percentage of correct target votes required."`
	MaxTargetPercent uint64 `yaml:"maxTargetPercent" json:"maxTargetPercent" desc:"Maximum percentage of correct target votes allowed."`
	MinHeadPercent   uint64 `yaml:"minHeadPercent" json:"minHeadPercent" desc:"Minimum percentage of correct head votes required."`
	MaxHeadPercent   uint64 `yaml:"maxHeadPercent" json:"maxHeadPercent" desc:"Maximum percentage of correct head votes allowed."`
	MinTotalPercent  uint64 `yaml:"minTotalPercent" json:"minTotalPercent" desc:"Minimum total attestation participation percentage required."`
	MaxTotalPercent  uint64 `yaml:"maxTotalPercent" json:"maxTotalPercent" desc:"Maximum total attestation participation percentage allowed."`
	FailOnCheckMiss  bool   `yaml:"failOnCheckMiss" json:"failOnCheckMiss" desc:"If true, fail the task when attestation stats condition is not met."`
	MinCheckedEpochs uint64 `yaml:"minCheckedEpochs" json:"minCheckedEpochs" desc:"Minimum number of epochs to check before evaluating conditions."`
	ContinueOnPass   bool   `yaml:"continueOnPass" json:"continueOnPass" desc:"If true, continue monitoring after the check passes instead of completing immediately."`
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
