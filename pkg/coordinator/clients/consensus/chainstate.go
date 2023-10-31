package consensus

type CheckpointName string

const (
	Justified = CheckpointName("justified")
	Finalized = CheckpointName("finalized")
)

var CheckpointNames = []CheckpointName{
	Justified,
	Finalized,
}
