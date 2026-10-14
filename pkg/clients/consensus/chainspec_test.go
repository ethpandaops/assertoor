package consensus

import (
	"context"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

func TestDataVersionAtSlot(t *testing.T) {
	// Every fork scheduled, one epoch apart, so each rung of the ladder is
	// reachable and the boundaries are exact.
	scheduled := &ChainSpec{
		AltairForkEpoch:    1,
		BellatrixForkEpoch: 2,
		CappellaForkEpoch:  3,
		DenebForkEpoch:     4,
		ElectraForkEpoch:   5,
		FuluForkEpoch:      6,
		GloasForkEpoch:     7,
		HezeForkEpoch:      8,
		SlotsPerEpoch:      32,
	}

	// A chain that never leaves electra: later forks unscheduled, as a beacon
	// node reports them.
	electraOnly := &ChainSpec{
		AltairForkEpoch:    0,
		BellatrixForkEpoch: 0,
		CappellaForkEpoch:  0,
		DenebForkEpoch:     0,
		ElectraForkEpoch:   0,
		FuluForkEpoch:      FarFutureEpoch,
		GloasForkEpoch:     FarFutureEpoch,
		HezeForkEpoch:      FarFutureEpoch,
		SlotsPerEpoch:      32,
	}

	tests := []struct {
		name     string
		spec     *ChainSpec
		slot     phase0.Slot
		expected spec.DataVersion
	}{
		{name: "Phase0BeforeAltair", spec: scheduled, slot: 0, expected: spec.DataVersionPhase0},
		{name: "Altair", spec: scheduled, slot: 1 * 32, expected: spec.DataVersionAltair},
		{name: "Bellatrix", spec: scheduled, slot: 2 * 32, expected: spec.DataVersionBellatrix},
		{name: "Capella", spec: scheduled, slot: 3 * 32, expected: spec.DataVersionCapella},
		{name: "Deneb", spec: scheduled, slot: 4 * 32, expected: spec.DataVersionDeneb},
		{name: "DenebUntilElectra", spec: scheduled, slot: 5*32 - 1, expected: spec.DataVersionDeneb},
		{name: "Electra", spec: scheduled, slot: 5 * 32, expected: spec.DataVersionElectra},
		{name: "Fulu", spec: scheduled, slot: 6 * 32, expected: spec.DataVersionFulu},
		{name: "Gloas", spec: scheduled, slot: 7 * 32, expected: spec.DataVersionGloas},
		{name: "Heze", spec: scheduled, slot: 8 * 32, expected: spec.DataVersionHeze},
		{name: "HezeStaysAtHead", spec: scheduled, slot: 500 * 32, expected: spec.DataVersionHeze},

		// A fork epoch of 0 means genesis, not "unscheduled".
		{name: "GenesisElectra", spec: electraOnly, slot: 0, expected: spec.DataVersionElectra},
		{name: "UnscheduledForksNeverActivate", spec: electraOnly, slot: 1e6, expected: spec.DataVersionElectra},

		{
			// SlotsPerEpoch is only zero before the spec has been fetched.
			name:     "ZeroSlotsPerEpoch",
			spec:     &ChainSpec{ElectraForkEpoch: 0},
			slot:     100 * 32,
			expected: spec.DataVersionPhase0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.spec.DataVersionAtSlot(test.slot); got != test.expected {
				t.Errorf("DataVersionAtSlot(%d) = %v, want %v", test.slot, got, test.expected)
			}
		})
	}
}

// TestSetClientSpecsForkDefaults pins the distinction the fork ladder rests on:
// a fork epoch a beacon node does not report at all must read as unscheduled,
// while a reported 0 must read as genesis. Both arrive as the same Go zero
// value unless the spec is seeded first.
func TestSetClientSpecsForkDefaults(t *testing.T) {
	cache, err := NewBlockCache(context.Background(), logrus.New(), 10)
	if err != nil {
		t.Fatalf("failed creating block cache: %v", err)
	}

	// A node that knows nothing past electra omits FULU/GLOAS/HEZE entirely.
	err = cache.SetClientSpecs(map[string]interface{}{
		"SLOTS_PER_EPOCH":    uint64(32),
		"ALTAIR_FORK_EPOCH":  uint64(0),
		"DENEB_FORK_EPOCH":   uint64(0),
		"ELECTRA_FORK_EPOCH": uint64(0),
	})
	if err != nil {
		t.Fatalf("failed setting client specs: %v", err)
	}

	specs := cache.GetSpecs()

	if specs.FuluForkEpoch != FarFutureEpoch {
		t.Errorf("unreported FuluForkEpoch = %d, want FarFutureEpoch", specs.FuluForkEpoch)
	}

	if specs.ElectraForkEpoch != 0 {
		t.Errorf("reported ElectraForkEpoch = %d, want 0", specs.ElectraForkEpoch)
	}

	if got := specs.DataVersionAtSlot(1e6); got != spec.DataVersionElectra {
		t.Errorf("DataVersionAtSlot = %v, want electra: unreported forks must not activate", got)
	}
}
