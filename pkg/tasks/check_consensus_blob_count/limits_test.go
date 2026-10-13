package checkconsensusblobcount

import (
	"encoding/json"
	"testing"
)

func TestParseBlobLimits(t *testing.T) {
	tests := []struct {
		name       string
		schedule   string
		wantTarget uint64
		wantMax    uint64
		wantErr    bool
	}{
		{
			name:       "eip-7910 cancun schedule",
			schedule:   `{"baseFeeUpdateFraction":3338477,"max":6,"target":3}`,
			wantTarget: 3,
			wantMax:    6,
		},
		{
			name:       "eip-7910 prague schedule",
			schedule:   `{"baseFeeUpdateFraction":5007716,"max":9,"target":6}`,
			wantTarget: 6,
			wantMax:    9,
		},
		{
			name:       "bpo schedule",
			schedule:   `{"baseFeeUpdateFraction":8346193,"max":15,"target":10}`,
			wantTarget: 10,
			wantMax:    15,
		},
		{
			name:     "missing target",
			schedule: `{"max":6}`,
			wantErr:  true,
		},
		{
			name:     "missing max",
			schedule: `{"target":3}`,
			wantErr:  true,
		},
		{
			name:     "target above max",
			schedule: `{"max":3,"target":6}`,
			wantErr:  true,
		},
		{
			name:     "non numeric max",
			schedule: `{"max":"0x6","target":3}`,
			wantErr:  true,
		},
		{
			name:     "empty schedule",
			schedule: `{}`,
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var schedule map[string]interface{}
			if err := json.Unmarshal([]byte(test.schedule), &schedule); err != nil {
				t.Fatalf("could not unmarshal test schedule: %v", err)
			}

			limits, err := parseBlobLimits(schedule)

			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got limits %+v", limits)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if limits.Target != test.wantTarget {
				t.Errorf("target: want %v, have %v", test.wantTarget, limits.Target)
			}

			if limits.Max != test.wantMax {
				t.Errorf("max: want %v, have %v", test.wantMax, limits.Max)
			}
		})
	}
}

func TestParseBlobLimitsNilSchedule(t *testing.T) {
	if _, err := parseBlobLimits(nil); err == nil {
		t.Error("expected error for nil blob schedule")
	}
}

func TestBlobLimitsCheck(t *testing.T) {
	limits := &blobLimits{Target: 6, Max: 9}

	tests := []struct {
		name          string
		blobCount     int
		limit         string
		wantResult    blobCheckResult
		wantThreshold uint64
		wantErr       bool
	}{
		{
			name:          "below target",
			blobCount:     5,
			limit:         BlobLimitTarget,
			wantResult:    blobCheckBelow,
			wantThreshold: 6,
		},
		{
			name:          "at target",
			blobCount:     6,
			limit:         BlobLimitTarget,
			wantResult:    blobCheckReached,
			wantThreshold: 6,
		},
		{
			name:          "above target but below max",
			blobCount:     8,
			limit:         BlobLimitTarget,
			wantResult:    blobCheckReached,
			wantThreshold: 6,
		},
		{
			name:          "at target threshold but max not reached",
			blobCount:     6,
			limit:         BlobLimitMax,
			wantResult:    blobCheckBelow,
			wantThreshold: 9,
		},
		{
			name:          "at max",
			blobCount:     9,
			limit:         BlobLimitMax,
			wantResult:    blobCheckReached,
			wantThreshold: 9,
		},
		{
			name:          "over max is a protocol violation",
			blobCount:     10,
			limit:         BlobLimitTarget,
			wantResult:    blobCheckOverMax,
			wantThreshold: 6,
		},
		{
			name:       "unknown limit",
			blobCount:  9,
			limit:      "maximum",
			wantErr:    true,
			wantResult: blobCheckBelow,
		},
		{
			name:      "empty limit",
			blobCount: 9,
			limit:     "",
			wantErr:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, threshold, err := limits.check(test.blobCount, test.limit)

			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got result %v", result)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != test.wantResult {
				t.Errorf("result: want %v, have %v", test.wantResult, result)
			}

			if threshold != test.wantThreshold {
				t.Errorf("threshold: want %v, have %v", test.wantThreshold, threshold)
			}
		})
	}
}

func TestBlobLimitsCheckNegativeCount(t *testing.T) {
	limits := &blobLimits{Target: 6, Max: 9}

	if _, _, err := limits.check(-1, BlobLimitTarget); err == nil {
		t.Error("expected error for a negative blob count")
	}
}

func TestBlobLimitsCheckZeroMax(t *testing.T) {
	limits := &blobLimits{Target: 0, Max: 0}

	result, _, err := limits.check(0, BlobLimitTarget)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != blobCheckReached {
		t.Errorf("a zero blob limit is reached by an empty block, got %v", result)
	}
}
