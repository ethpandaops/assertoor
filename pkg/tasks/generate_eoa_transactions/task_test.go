package generateeoatransactions

import (
	"sync"
	"sync/atomic"
	"testing"
)

// TestAwaitReceiptsRequired covers the condition that decides whether the task waits
// for receipt callbacks before evaluating its result. failOnReject and failOnSuccess
// must force the wait even when awaitReceipt is off; otherwise the verdict reads the
// counters before the receipts arrive and reports success even if every transaction
// reverted.
func TestAwaitReceiptsRequired(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{"nothing set", Config{}, false},
		{"awaitReceipt only", Config{AwaitReceipt: true}, true},
		{"failOnReject forces wait", Config{FailOnReject: true}, true},
		{"failOnSuccess forces wait", Config{FailOnSuccess: true}, true},
		{"failOnReject without awaitReceipt still waits", Config{FailOnReject: true, AwaitReceipt: false}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := awaitReceiptsRequired(tc.cfg); got != tc.want {
				t.Fatalf("awaitReceiptsRequired(%+v) = %v, want %v", tc.cfg, got, tc.want)
			}
		})
	}
}

// TestReceiptCountersAreConcurrencySafe reproduces the receipt callbacks, which run in
// their own goroutines, incrementing the shared counters. The counters must be atomic
// so no increments are lost (and so this does not race under -race).
func TestReceiptCountersAreConcurrencySafe(t *testing.T) {
	const n = 500

	var revertCount atomic.Int64

	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			revertCount.Add(1)
		}()
	}

	wg.Wait()

	if got := revertCount.Load(); got != n {
		t.Fatalf("lost updates: revertCount = %d, want %d", got, n)
	}
}
