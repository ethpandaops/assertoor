package consensus

import (
	"errors"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

func TestDataVersionAtSlot(t *testing.T) {
	specs := &ChainSpec{
		ElectraForkEpoch: 0,
		FuluForkEpoch:    10,
		GloasForkEpoch:   20,
		SlotsPerEpoch:    32,
	}

	tests := []struct {
		name         string
		spec         *ChainSpec
		slot         phase0.Slot
		expected     spec.DataVersion
		expectPreErr bool
	}{
		{name: "ElectraBeforeFulu", spec: specs, slot: 0, expected: spec.DataVersionElectra},
		{name: "FuluAtForkEpoch", spec: specs, slot: phase0.Slot(10 * 32), expected: spec.DataVersionFulu},
		{name: "FuluBeforeGloas", spec: specs, slot: phase0.Slot(20*32 - 1), expected: spec.DataVersionFulu},
		{name: "GloasAtForkEpoch", spec: specs, slot: phase0.Slot(20 * 32), expected: spec.DataVersionGloas},
		{name: "GloasAfterForkEpoch", spec: specs, slot: phase0.Slot(25 * 32), expected: spec.DataVersionGloas},
		{
			name:     "GloasNotScheduled",
			spec:     &ChainSpec{FuluForkEpoch: 10, SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: spec.DataVersionFulu,
		},
		{
			name:     "NoForksScheduled",
			spec:     &ChainSpec{SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: spec.DataVersionElectra,
		},
		{
			name:     "ZeroSlotsPerEpoch",
			spec:     &ChainSpec{FuluForkEpoch: 10, GloasForkEpoch: 20},
			slot:     phase0.Slot(100 * 32),
			expected: spec.DataVersionElectra,
		},
		{
			name:     "FarFutureForkEpochsNoOverflow",
			spec:     &ChainSpec{FuluForkEpoch: ^uint64(0), GloasForkEpoch: ^uint64(0), SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: spec.DataVersionElectra,
		},
		{
			// Electra scheduled at a later epoch is the one case where the slot
			// is positively known to predate electra, so the caller has to be
			// told rather than handed an electra-shaped submission.
			name:         "BeforeScheduledElectra",
			spec:         &ChainSpec{ElectraForkEpoch: 5, FuluForkEpoch: 10, SlotsPerEpoch: 32},
			slot:         phase0.Slot(4 * 32),
			expectPreErr: true,
		},
		{
			name:     "AtScheduledElectra",
			spec:     &ChainSpec{ElectraForkEpoch: 5, FuluForkEpoch: 10, SlotsPerEpoch: 32},
			slot:     phase0.Slot(5 * 32),
			expected: spec.DataVersionElectra,
		},
		{
			// Electra unreported (0) is indistinguishable from genesis-activated,
			// so it must not be reported as pre-electra.
			name:     "UnreportedElectraNotPreElectra",
			spec:     &ChainSpec{SlotsPerEpoch: 32},
			slot:     0,
			expected: spec.DataVersionElectra,
		},
		{
			name:     "ZeroSlotsPerEpochWithScheduledElectra",
			spec:     &ChainSpec{ElectraForkEpoch: 5},
			slot:     0,
			expected: spec.DataVersionElectra,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.spec.DataVersionAtSlot(test.slot)

			if test.expectPreErr {
				if !errors.Is(err, ErrPreElectraFork) {
					t.Fatalf("DataVersionAtSlot(%d) error = %v, want ErrPreElectraFork", test.slot, err)
				}

				if got != spec.DataVersionUnknown {
					t.Errorf("DataVersionAtSlot(%d) = %v, want unknown alongside the error", test.slot, got)
				}

				return
			}

			if err != nil {
				t.Fatalf("DataVersionAtSlot(%d) failed: %v", test.slot, err)
			}

			if got != test.expected {
				t.Errorf("DataVersionAtSlot(%d) = %v, want %v", test.slot, got, test.expected)
			}
		})
	}
}
