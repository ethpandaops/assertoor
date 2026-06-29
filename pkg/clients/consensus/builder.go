package consensus

import (
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
)

// BuilderInfo wraps a gloas.Builder with its index in the builder list.
type BuilderInfo struct {
	Index   gloas.BuilderIndex
	Builder *gloas.Builder
}
