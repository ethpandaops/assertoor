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
	HezeForkEpoch        uint64         `yaml:"HEZE_FORK_EPOCH"`
	SlotDurationMs       uint64         `yaml:"SLOT_DURATION_MS"`
	SlotsPerEpoch        uint64         `yaml:"SLOTS_PER_EPOCH"`
	MaxCommitteesPerSlot uint64         `yaml:"MAX_COMMITTEES_PER_SLOT"`
}

// FarFutureEpoch marks a fork that is not scheduled on this chain. Beacon nodes
// report it for known-but-unscheduled forks and omit the key entirely for forks
// they predate; NewChainSpec seeds every fork epoch with it so both cases read
// the same, and a reported epoch of 0 keeps its literal meaning of "genesis".
const FarFutureEpoch = ^uint64(0)

// NewChainSpec returns a ChainSpec with every fork unscheduled, ready to be
// filled from a beacon node's /eth/v1/config/spec response.
func NewChainSpec() *ChainSpec {
	return &ChainSpec{
		AltairForkEpoch:    FarFutureEpoch,
		BellatrixForkEpoch: FarFutureEpoch,
		CappellaForkEpoch:  FarFutureEpoch,
		DenebForkEpoch:     FarFutureEpoch,
		ElectraForkEpoch:   FarFutureEpoch,
		FuluForkEpoch:      FarFutureEpoch,
		GloasForkEpoch:     FarFutureEpoch,
		HezeForkEpoch:      FarFutureEpoch,
	}
}

// isForkActive reports whether the fork scheduled at forkEpoch is active at the
// given slot. Unscheduled forks carry FarFutureEpoch (see NewChainSpec), so a
// fork epoch of 0 correctly reads as "active since genesis". The comparison
// divides rather than multiplying, so FarFutureEpoch cannot overflow it.
func (chain *ChainSpec) isForkActive(forkEpoch uint64, slot phase0.Slot) bool {
	return chain.SlotsPerEpoch > 0 && uint64(slot)/chain.SlotsPerEpoch >= forkEpoch
}

// IsGloasActive returns true if the gloas fork is active at the given slot.
func (chain *ChainSpec) IsGloasActive(slot phase0.Slot) bool {
	return chain.isForkActive(chain.GloasForkEpoch, slot)
}

// DataVersionAtSlot returns the consensus fork active at the given slot, as the
// spec.DataVersion that selects the arm of a versioned type and drives the
// Eth-Consensus-Version header.
func (chain *ChainSpec) DataVersionAtSlot(slot phase0.Slot) spec.DataVersion {
	switch {
	case chain.isForkActive(chain.HezeForkEpoch, slot):
		return spec.DataVersionHeze
	case chain.isForkActive(chain.GloasForkEpoch, slot):
		return spec.DataVersionGloas
	case chain.isForkActive(chain.FuluForkEpoch, slot):
		return spec.DataVersionFulu
	case chain.isForkActive(chain.ElectraForkEpoch, slot):
		return spec.DataVersionElectra
	case chain.isForkActive(chain.DenebForkEpoch, slot):
		return spec.DataVersionDeneb
	case chain.isForkActive(chain.CappellaForkEpoch, slot):
		return spec.DataVersionCapella
	case chain.isForkActive(chain.BellatrixForkEpoch, slot):
		return spec.DataVersionBellatrix
	case chain.isForkActive(chain.AltairForkEpoch, slot):
		return spec.DataVersionAltair
	default:
		return spec.DataVersionPhase0
	}
}

func (chain *ChainSpec) CheckMismatch(chain2 *ChainSpec) []string {
	mismatches := []string{}

	chainT := reflect.ValueOf(chain).Elem()
	chain2T := reflect.ValueOf(chain2).Elem()

	for i := range chainT.NumField() {
		if chainT.Field(i).Interface() != chain2T.Field(i).Interface() {
			mismatches = append(mismatches, chainT.Type().Field(i).Name)
		}
	}

	return mismatches
}
