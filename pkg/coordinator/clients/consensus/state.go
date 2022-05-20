package consensus

type State struct {
	Healthy    bool
	SyncStatus SyncStatus
	ChainState map[CheckpointName]Checkpoint
	Spec       Spec
}

func NewState() State {
	return State{
		Healthy:    false,
		SyncStatus: SyncStatus{},
		ChainState: NewChainState(),
		Spec:       Spec{},
	}
}
