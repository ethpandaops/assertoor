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
