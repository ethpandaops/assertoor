package consensus

type CheckpointName string

const (
	Head      = CheckpointName("head")
	Justified = CheckpointName("justified")
	Finalized = CheckpointName("finalized")
)

var CheckpointNames = []CheckpointName{
	Head,
	Justified,
	Finalized,
}

type ChainState map[CheckpointName]Checkpoint

type Checkpoint struct {
	Slot  uint64
	Epoch uint64
}

func NewChainState() ChainState {
	return ChainState{
		Head: Checkpoint{
			Slot:  0,
			Epoch: 0,
		},
		Justified: Checkpoint{
			Slot:  0,
			Epoch: 0,
		},
		Finalized: Checkpoint{
			Slot:  0,
			Epoch: 0,
		},
	}
}
