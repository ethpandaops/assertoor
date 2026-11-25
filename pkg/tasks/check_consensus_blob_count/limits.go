package checkconsensusblobcount

import (
	"fmt"
	"math"
)

// Blob limits a proposed block can be checked against.
const (
	BlobLimitTarget = "target"
	BlobLimitMax    = "max"
)

// blobLimits are the per-block blob bounds that apply at a given fork, as
// reported by the execution client in the eth_config blobSchedule (EIP-7910).
// The consensus spec only carries the max, so the execution config is the only
// place both numbers are available.
type blobLimits struct {
	Target uint64
	Max    uint64
}

// blobCheckResult classifies a block's blob count against the blob limits.
type blobCheckResult int

const (
	blobCheckBelow blobCheckResult = iota
	blobCheckReached
	blobCheckOverMax
)

// parseBlobLimits reads the target and max blob counts from an eth_config
// blobSchedule object.
func parseBlobLimits(schedule map[string]interface{}) (*blobLimits, error) {
	if schedule == nil {
		return nil, fmt.Errorf("blob schedule is not present in eth_config")
	}

	target, err := blobScheduleValue(schedule, BlobLimitTarget)
	if err != nil {
		return nil, err
	}

	maxBlobs, err := blobScheduleValue(schedule, BlobLimitMax)
	if err != nil {
		return nil, err
	}

	if target > maxBlobs {
		return nil, fmt.Errorf("blob schedule target (%v) is above max (%v)", target, maxBlobs)
	}

	return &blobLimits{Target: target, Max: maxBlobs}, nil
}

// check reports whether blobCount reached the requested limit, and returns the
// blob count that limit required.
func (l *blobLimits) check(blobCount int, limit string) (blobCheckResult, uint64, error) {
	var threshold uint64

	switch limit {
	case BlobLimitTarget:
		threshold = l.Target
	case BlobLimitMax:
		threshold = l.Max
	default:
		return blobCheckBelow, 0, fmt.Errorf("unknown blob limit %q, want %q or %q", limit, BlobLimitTarget, BlobLimitMax)
	}

	if blobCount < 0 {
		return blobCheckBelow, threshold, fmt.Errorf("negative blob count %v", blobCount)
	}

	count := uint64(blobCount)

	switch {
	case count > l.Max:
		return blobCheckOverMax, threshold, nil
	case count >= threshold:
		return blobCheckReached, threshold, nil
	default:
		return blobCheckBelow, threshold, nil
	}
}

// blobScheduleValue reads a single unsigned integer from a blob schedule.
func blobScheduleValue(schedule map[string]interface{}, key string) (uint64, error) {
	value, ok := schedule[key]
	if !ok {
		return 0, fmt.Errorf("blob schedule has no %q entry", key)
	}

	switch typed := value.(type) {
	case float64:
		if typed < 0 || typed > math.MaxUint32 || typed != math.Trunc(typed) {
			return 0, fmt.Errorf("blob schedule %q is not a plain unsigned integer: %v", key, typed)
		}

		return uint64(typed), nil
	default:
		return 0, fmt.Errorf("blob schedule %q has unexpected type %T", key, value)
	}
}
