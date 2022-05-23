package task

// Config holds the per-task configuration.
type Config struct {
	KillConsensus KillConsensusConfig `yaml:"kill_consensus"`
	KillExecution KillExecutionConfig `yaml:"kill_execution"`
}

var (
	ConsensusClientNames = []string{"prysm", "lighthouse", "lodestar", "nimbus", "teku"}
	ExecutionClientNames = []string{"geth", "besu", "nethermind", "erigon"}
)

type KillConsensusConfig struct {
	Command []string `yaml:"command"`
}

type KillExecutionConfig struct {
	Command []string `yaml:"command"`
}

func DefaultConfig() Config {
	return Config{
		KillConsensus: KillConsensusConfig{
			Command: []string{},
		},
		KillExecution: KillExecutionConfig{
			Command: []string{},
		},
	}
}
