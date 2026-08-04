package consensus

import (
	"errors"
	"fmt"
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

// isForkActive reports whether a fork scheduled at forkEpoch is active at the
// given slot. A fork epoch of 0 reads as "not scheduled" rather than "active
// since genesis": beacon nodes that predate a fork omit its epoch from
// /eth/v1/config/spec and it unmarshals to 0, and reporting a fork the node has
// never heard of is the worse failure mode. The comparison divides rather than
// multiplies so a FAR_FUTURE fork epoch (2^64-1) cannot overflow.
func (chain *ChainSpec) isForkActive(forkEpoch uint64, slot phase0.Slot) bool {
	return forkEpoch > 0 && chain.SlotsPerEpoch > 0 && uint64(slot)/chain.SlotsPerEpoch >= forkEpoch
}

// IsGloasActive returns true if the gloas fork is active at the given slot.
func (chain *ChainSpec) IsGloasActive(slot phase0.Slot) bool {
	return chain.isForkActive(chain.GloasForkEpoch, slot)
}

// ErrPreElectraFork reports that the fork active at a slot predates electra.
// The versioned beacon-API pool endpoints, and the versioned spec types they
// take, only exist from electra onwards.
var ErrPreElectraFork = errors.New("fork active at slot predates electra")

// DataVersionAtSlot returns the consensus fork active at the given slot, as the
// spec.DataVersion that selects the arm of a versioned type and drives the
// Eth-Consensus-Version header.
//
// It resolves gloas, fulu and electra. When electra is positively scheduled at
// a later epoch the slot is known to predate it, and ErrPreElectraFork is
// returned so callers fail on the fork rather than on an opaque response from
// an endpoint the node does not serve. Electra is the floor otherwise: a
// genesis-activated electra is indistinguishable from an unreported one under
// the fork-epoch convention isForkActive documents.
func (chain *ChainSpec) DataVersionAtSlot(slot phase0.Slot) (spec.DataVersion, error) {
	switch {
	case chain.IsGloasActive(slot):
		return spec.DataVersionGloas, nil
	case chain.isForkActive(chain.FuluForkEpoch, slot):
		return spec.DataVersionFulu, nil
	case chain.ElectraForkEpoch > 0 && chain.SlotsPerEpoch > 0 && uint64(slot)/chain.SlotsPerEpoch < chain.ElectraForkEpoch:
		return spec.DataVersionUnknown, fmt.Errorf("%w: slot %d is in epoch %d, electra activates at epoch %d",
			ErrPreElectraFork, slot, uint64(slot)/chain.SlotsPerEpoch, chain.ElectraForkEpoch)
	default:
		return spec.DataVersionElectra, nil
	}
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
