package consensus

import (
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// BuilderIndexFlag is the bit flag that indicates a ValidatorIndex
// should be treated as a BuilderIndex (BUILDER_INDEX_FLAG = 2^40).
const BuilderIndexFlag = uint64(1 << 40)

// BuilderInfo wraps a gloas.Builder with its index in the builder list.
type BuilderInfo struct {
	Index   gloas.BuilderIndex
	Builder *gloas.Builder
}

// ConvertBuilderIndexToValidatorIndex returns the ValidatorIndex
// representation of a builder index (builder_index | BUILDER_INDEX_FLAG).
func ConvertBuilderIndexToValidatorIndex(builderIndex gloas.BuilderIndex) phase0.ValidatorIndex {
	return phase0.ValidatorIndex(uint64(builderIndex) | BuilderIndexFlag)
}
