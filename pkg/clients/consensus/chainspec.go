package consensus

import (
	"reflect"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

type ForkVersion struct {
	Epoch           uint64
	CurrentVersion  []byte
	PreviousVersion []byte
}

// https://github.com/ethereum/consensus-specs/blob/dev/configs/mainnet.yaml
type ChainSpec struct {
	PresetBase           string         `yaml:"PRESET_BASE"`
	ConfigName           string         `yaml:"CONFIG_NAME"`
	MinGenesisTime       time.Time      `yaml:"MIN_GENESIS_TIME"`
	GenesisForkVersion   phase0.Version `yaml:"GENESIS_FORK_VERSION"`
	AltairForkVersion    phase0.Version `yaml:"ALTAIR_FORK_VERSION"`
	AltairForkEpoch      uint64         `yaml:"ALTAIR_FORK_EPOCH"`
	BellatrixForkVersion phase0.Version `yaml:"BELLATRIX_FORK_VERSION"`
	BellatrixForkEpoch   uint64         `yaml:"BELLATRIX_FORK_EPOCH"`
	CappellaForkVersion  phase0.Version `yaml:"CAPELLA_FORK_VERSION"`
	CappellaForkEpoch    uint64         `yaml:"CAPELLA_FORK_EPOCH"`
	DenebForkEpoch       uint64         `yaml:"DENEB_FORK_EPOCH"`
	ElectraForkEpoch     uint64         `yaml:"ELECTRA_FORK_EPOCH"`
	FuluForkEpoch        uint64         `yaml:"FULU_FORK_EPOCH"`
	GloasForkEpoch       uint64         `yaml:"GLOAS_FORK_EPOCH"`
	SlotDurationMs       uint64         `yaml:"SLOT_DURATION_MS"`
	SlotsPerEpoch        uint64         `yaml:"SLOTS_PER_EPOCH"`
	MaxCommitteesPerSlot uint64         `yaml:"MAX_COMMITTEES_PER_SLOT"`
}

// IsGloasActive returns true if the gloas fork is active at the given slot.
func (chain *ChainSpec) IsGloasActive(slot phase0.Slot) bool {
	if chain.GloasForkEpoch == 0 || chain.SlotsPerEpoch == 0 {
		return false
	}

	return uint64(slot) >= chain.GloasForkEpoch*chain.SlotsPerEpoch
}

// DataVersionAtSlot returns the consensus fork active at the given slot, as the
// spec.DataVersion that selects the arm of a versioned type and drives the
// Eth-Consensus-Version header. Electra is the floor: the versioned pool
// endpoints it is used for only exist from electra onwards.
//
// A fork epoch of 0 is treated as "not scheduled" rather than "active since
// genesis", matching IsGloasActive. Beacon nodes that predate a fork omit its
// epoch from /eth/v1/config/spec, which unmarshals to 0, and reporting a fork
// the node has never heard of is the worse failure mode.
func (chain *ChainSpec) DataVersionAtSlot(slot phase0.Slot) spec.DataVersion {
	if chain.IsGloasActive(slot) {
		return spec.DataVersionGloas
	}

	// Division-based comparison: FAR_FUTURE fork epochs (2^64-1) would overflow
	// an epoch*SlotsPerEpoch multiplication.
	if chain.FuluForkEpoch > 0 && chain.SlotsPerEpoch > 0 && uint64(slot)/chain.SlotsPerEpoch >= chain.FuluForkEpoch {
		return spec.DataVersionFulu
	}

	return spec.DataVersionElectra
}

func (chain *ChainSpec) CheckMismatch(chain2 *ChainSpec) []string {
	mismatches := []string{}

	chainT := reflect.ValueOf(chain).Elem()
	chain2T := reflect.ValueOf(chain2).Elem()

	for i := 0; i < chainT.NumField(); i++ {
		if chainT.Field(i).Interface() != chain2T.Field(i).Interface() {
			mismatches = append(mismatches, chainT.Type().Field(i).Name)
		}
	}

	return mismatches
}
