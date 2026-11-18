package checkconsensusreorgs

import (
	"io"
	"testing"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/sirupsen/logrus"
)

func newEvalTask(cfg Config, totalReorgs uint64) *Task {
	log := logrus.New()
	log.SetOutput(io.Discard)

	return &Task{
		config:      cfg,
		logger:      log,
		totalReorgs: totalReorgs,
	}
}

// TestEvaluateReorgsMaxTotal covers the maxTotalReorgs bound, which was declared and
// documented but never enforced, so a run exceeding it still passed.
func TestEvaluateReorgsMaxTotal(t *testing.T) {
	tests := []struct {
		name        string
		maxTotal    uint64
		totalReorgs uint64
		epochCount  uint64
		want        types.TaskResult
	}{
		{"under the bound passes", 5, 3, 10, types.TaskResultSuccess},
		{"at the bound passes", 5, 5, 10, types.TaskResultSuccess},
		{"over the bound fails", 5, 6, 10, types.TaskResultFailure},
		{"unset bound is ignored", 0, 1000, 10, types.TaskResultSuccess},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := newEvalTask(Config{MaxTotalReorgs: tc.maxTotal}, tc.totalReorgs)

			if got := task.evaluateReorgs(tc.epochCount); got != tc.want {
				t.Fatalf("maxTotalReorgs=%d totalReorgs=%d: got %v, want %v", tc.maxTotal, tc.totalReorgs, got, tc.want)
			}
		})
	}
}

// TestEvaluateReorgsOtherBounds guards the neighbouring thresholds so the added
// check does not shadow them.
func TestEvaluateReorgsOtherBounds(t *testing.T) {
	// minCheckEpochCount not yet reached -> inconclusive
	task := newEvalTask(Config{MinCheckEpochCount: 5}, 0)
	if got := task.evaluateReorgs(2); got != types.TaskResultNone {
		t.Fatalf("min epoch not reached: got %v, want None", got)
	}

	// maxReorgsPerEpoch exceeded -> failure
	task = newEvalTask(Config{MaxReorgsPerEpoch: 1.0}, 30)
	if got := task.evaluateReorgs(10); got != types.TaskResultFailure {
		t.Fatalf("max reorgs per epoch exceeded: got %v, want Failure", got)
	}
}
