package consensus

import (
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
		name     string
		spec     *ChainSpec
		slot     phase0.Slot
		expected spec.DataVersion
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
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.spec.DataVersionAtSlot(test.slot); got != test.expected {
				t.Errorf("DataVersionAtSlot(%d) = %v, want %v", test.slot, got, test.expected)
			}
		})
	}
}
