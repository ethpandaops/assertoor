package consensus

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

func TestConsensusForkVersionAtSlot(t *testing.T) {
	specs := &ChainSpec{
		ElectraForkEpoch: 0,
		FuluForkEpoch:    10,
		GloasForkEpoch:   20,
		SlotsPerEpoch:    32,
	}

	tests := []struct {
		name     string
		spec     *ChainSpec
		slot     phase0.Slot
		expected string
	}{
		{name: "ElectraBeforeFulu", spec: specs, slot: 0, expected: ForkVersionElectra},
		{name: "FuluAtForkEpoch", spec: specs, slot: phase0.Slot(10 * 32), expected: ForkVersionFulu},
		{name: "FuluBeforeGloas", spec: specs, slot: phase0.Slot(20*32 - 1), expected: ForkVersionFulu},
		{name: "GloasAtForkEpoch", spec: specs, slot: phase0.Slot(20 * 32), expected: ForkVersionGloas},
		{name: "GloasAfterForkEpoch", spec: specs, slot: phase0.Slot(25 * 32), expected: ForkVersionGloas},
		{
			name:     "GloasNotScheduled",
			spec:     &ChainSpec{FuluForkEpoch: 10, SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: ForkVersionFulu,
		},
		{
			name:     "NoForksScheduled",
			spec:     &ChainSpec{SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: ForkVersionElectra,
		},
		{
			name:     "ZeroSlotsPerEpoch",
			spec:     &ChainSpec{FuluForkEpoch: 10, GloasForkEpoch: 20},
			slot:     phase0.Slot(100 * 32),
			expected: ForkVersionElectra,
		},
		{
			name:     "FarFutureForkEpochsNoOverflow",
			spec:     &ChainSpec{FuluForkEpoch: ^uint64(0), GloasForkEpoch: ^uint64(0), SlotsPerEpoch: 32},
			slot:     phase0.Slot(100 * 32),
			expected: ForkVersionElectra,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.spec.ConsensusForkVersionAtSlot(test.slot); got != test.expected {
				t.Errorf("ConsensusForkVersionAtSlot(%d) = %q, want %q", test.slot, got, test.expected)
			}
		})
	}
}
