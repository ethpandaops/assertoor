package scheduler

import (
	"sync"
	"testing"

	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
)

// TestTaskStateStatusNoRace exercises the concurrent access pattern between the
// web/watcher readers (GetTaskStatus) and the writers (setTaskResult, SetProgress)
// on a single task state. All of them must go through resultMutex, otherwise the
// shared result/progress fields race (run with -race).
func TestTaskStateStatusNoRace(t *testing.T) {
	state := &taskState{
		index:          1,
		taskStatusVars: vars.NewVariables(nil),
	}

	var wg sync.WaitGroup

	stop := make(chan struct{})

	// writer: task result transitions
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
				state.setTaskResult(types.TaskResultSuccess, false)
				state.setTaskResult(types.TaskResultFailure, false)
			}
		}
	}()

	// writer: progress updates
	wg.Add(1)
	go func() {
		defer wg.Done()

		for {
			select {
			case <-stop:
				return
			default:
				state.SetProgress(50, "half way")
			}
		}
	}()

	// reader: the status snapshot served to watchers and the web api
	for i := 0; i < 200000; i++ {
		status := state.GetTaskStatus()
		_ = status.Result
		_ = status.Progress

		if status.Error != nil {
			_ = status.Error.Error()
		}
	}

	close(stop)
	wg.Wait()
}
