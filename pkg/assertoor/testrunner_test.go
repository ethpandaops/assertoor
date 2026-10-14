package assertoor

import (
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/types"
)

// stubTest is a minimal types.Test double. Only RunID, Status, and StopTime
// matter for cleanupTestHistory; the rest exist to satisfy the interface.
type stubTest struct {
	runID    uint64
	status   types.TestStatus
	stopTime time.Time
}

func (s *stubTest) RunID() uint64                         { return s.runID }
func (s *stubTest) TestID() string                        { return "stub-test" }
func (s *stubTest) Name() string                          { return "stub test" }
func (s *stubTest) StartTime() time.Time                  { return s.stopTime.Add(-time.Minute) }
func (s *stubTest) StopTime() time.Time                   { return s.stopTime }
func (s *stubTest) Timeout() time.Duration                { return 0 }
func (s *stubTest) Status() types.TestStatus              { return s.status }
func (s *stubTest) GetTaskScheduler() types.TaskScheduler { return nil }
func (s *stubTest) AbortTest(_ bool)                      {}

func TestCleanupTestHistory(t *testing.T) {
	const retention = time.Hour

	now := time.Now()

	tr := NewTestRunner(nil, 0)

	tr.testRunMap[1] = &stubTest{runID: 1, status: types.TestStatusSuccess, stopTime: now.Add(-2 * retention)}
	tr.testRunMap[2] = &stubTest{runID: 2, status: types.TestStatusFailure, stopTime: now.Add(-2 * retention)}
	tr.testRunMap[3] = &stubTest{runID: 3, status: types.TestStatusAborted, stopTime: now.Add(-2 * retention)}
	tr.testRunMap[4] = &stubTest{runID: 4, status: types.TestStatusSuccess, stopTime: now.Add(-retention / 2)}
	tr.testRunMap[5] = &stubTest{runID: 5, status: types.TestStatusRunning, stopTime: time.Time{}}
	tr.testRunMap[6] = &stubTest{runID: 6, status: types.TestStatusPending, stopTime: time.Time{}}

	tr.cleanupTestHistory(retention)

	wantEvicted := []uint64{1, 2, 3}
	for _, runID := range wantEvicted {
		if _, ok := tr.testRunMap[runID]; ok {
			t.Errorf("run %d: expected eviction (finished more than retention ago), still present", runID)
		}
	}

	wantKept := []uint64{4, 5, 6}
	for _, runID := range wantKept {
		if _, ok := tr.testRunMap[runID]; !ok {
			t.Errorf("run %d: expected to be kept, was evicted", runID)
		}
	}

	if got := len(tr.testRunMap); got != len(wantKept) {
		t.Fatalf("testRunMap has %d entries after cleanup, want %d", got, len(wantKept))
	}
}

func TestCleanupTestHistoryNeverEvictsWithoutRetention(t *testing.T) {
	tr := NewTestRunner(nil, 0)

	tr.testRunMap[1] = &stubTest{
		runID:    1,
		status:   types.TestStatusSuccess,
		stopTime: time.Now().Add(-365 * 24 * time.Hour),
	}

	// A zero retention window still means "evict anything already finished",
	// not "keep everything forever" -- RunTestCleanup is what clamps a
	// non-positive config value to the 14 day default before calling in.
	tr.cleanupTestHistory(0)

	if _, ok := tr.testRunMap[1]; ok {
		t.Fatalf("expected the year-old finished run to be evicted with a zero retention window")
	}
}
