package scheduler

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func TestTimeoutError(t *testing.T) {
	plainErr := errors.New("boom")

	tests := []struct {
		name      string
		state     *taskState
		err       error
		want      string
		wantIsDDL bool
	}{
		{"no timeout keeps error", &taskState{progressMessage: "waiting"}, plainErr, "boom", false},
		{"no timeout keeps nil", &taskState{progressMessage: "waiting"}, nil, "", false},
		{"parent deadline adds status", &taskState{progressMessage: "epoch 14 failed: target votes 94.00% < 98%"}, fmt.Errorf("wrapped: %w", context.DeadlineExceeded), "wrapped: context deadline exceeded, last status: epoch 14 failed: target votes 94.00% < 98%", true},
		{"own timeout with nil error", &taskState{isTimeout: true, progressMessage: "Waiting for sync... 2/3"}, nil, "task timed out: context deadline exceeded, last status: Waiting for sync... 2/3", true},
		{"own timeout with canceled error", &taskState{isTimeout: true}, context.Canceled, "task timed out: context deadline exceeded", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := timeoutError(test.state, test.err)

			got := ""
			if err != nil {
				got = err.Error()
			}

			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}

			if errors.Is(err, context.DeadlineExceeded) != test.wantIsDDL {
				t.Fatalf("errors.Is(DeadlineExceeded) = %v, want %v", !test.wantIsDDL, test.wantIsDDL)
			}
		})
	}
}
