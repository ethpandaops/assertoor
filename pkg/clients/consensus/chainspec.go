package consensus

import (
	"reflect"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
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
	SecondsPerSlot       time.Duration  `yaml:"SECONDS_PER_SLOT"`
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
