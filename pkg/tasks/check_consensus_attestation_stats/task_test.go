package checkconsensusattestationstats

import (
	"testing"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/prysmaticlabs/go-bitfield"
	"github.com/sirupsen/logrus"
)

func newTestTask(config *Config) *Task {
	return &Task{
		ctx:    &types.TaskContext{Outputs: vars.NewVariables(nil)},
		config: *config,
		logger: logrus.New(),
	}
}

func TestAggregateAttestationVotesSkipsUnmatchedValidators(t *testing.T) {
	config := DefaultConfig()
	task := newTestTask(&config)
	votes := task.newEpochVotes(nil)
	votes.attesterDuties = &attesterDuties{
		duties: map[string][]*attesterDuty{
			"8-0": {
				{validator: 1, balance: 32, matches: true},
				{validator: 2, balance: 32, matches: false},
				{validator: 3, balance: 32, matches: true},
			},
		},
	}

	bits := bitfield.NewBitlist(3)
	bits.SetBitAt(0, true)
	bits.SetBitAt(1, true)

	amount, count, size := task.aggregateAttestationVotes(votes, 8, 0, bits, 0)
	if amount != 32 || count != 1 || size != 3 {
		t.Fatalf("got amount=%d count=%d size=%d, want 32/1/3", amount, count, size)
	}
}

func TestCheckEpochVotesReason(t *testing.T) {
	config := DefaultConfig()
	config.MinTargetPercent = 98
	task := newTestTask(&config)

	votes := task.newEpochVotes(nil)
	votes.attesterDuties = &attesterDuties{validatorCount: 4, validatorBalance: 100}
	votes.currentEpoch.targetVoteAmount = 94

	passed, reason := task.checkEpochVotes(10, votes)
	if passed || reason != "target votes 94.00% < 98%" {
		t.Fatalf("got passed=%v reason=%q", passed, reason)
	}

	votes.attesterDuties = &attesterDuties{}

	passed, reason = task.checkEpochVotes(10, votes)
	if passed || reason != "no active validators match" {
		t.Fatalf("got passed=%v reason=%q", passed, reason)
	}
}
