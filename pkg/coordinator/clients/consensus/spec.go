package consensus

import "github.com/spf13/cast"

type Spec struct {
	SafeSlotsToUpdateJustified uint64
	SlotsPerEpoch              uint64
}

func (s *Spec) update(spec map[string]interface{}) {
	if safeSlotsToUpdateJustified, exists := spec["SAFE_SLOTS_TO_UPDATE_JUSTIFIED"]; exists {
		s.SafeSlotsToUpdateJustified = cast.ToUint64(safeSlotsToUpdateJustified)
	}

	if slotsPerEpoch, exists := spec["SLOTS_PER_EPOCH"]; exists {
		s.SlotsPerEpoch = cast.ToUint64(slotsPerEpoch)
	}
}
