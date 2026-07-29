package checkconsensusreorgs

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/helper"
	"github.com/ethpandaops/assertoor/pkg/logger"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
)

type fakeServices struct {
	pool *clients.ClientPool
}

func (f *fakeServices) Database() *db.Database                { return nil }
func (f *fakeServices) ClientPool() *clients.ClientPool       { return f.pool }
func (f *fakeServices) WalletManager() *txmgr.Spamoor         { return nil }
func (f *fakeServices) ValidatorNames() *names.ValidatorNames { return nil }
func (f *fakeServices) EventBus() *events.EventBus            { return nil }

type fakeSchedulerRunner struct {
	services types.TaskServices
}

func (f *fakeSchedulerRunner) GetServices() types.TaskServices { return f.services }
func (f *fakeSchedulerRunner) GetTestRunID() uint64            { return 1 }
func (f *fakeSchedulerRunner) GetTestRunCtx() context.Context  { return context.Background() }
func (f *fakeSchedulerRunner) ParseTaskOptions(_ helper.IRawMessage) (*types.TaskOptions, error) {
	return nil, nil
}
func (f *fakeSchedulerRunner) ExecuteTask(_ context.Context, _ types.TaskIndex, _ func(ctx context.Context, cancelFn context.CancelFunc, taskIndex types.TaskIndex)) error {
	return nil
}
func (f *fakeSchedulerRunner) TestResultPath() (string, error)                { return "", nil }
func (f *fakeSchedulerRunner) GetTaskState(_ types.TaskIndex) types.TaskState { return nil }
func (f *fakeSchedulerRunner) GetTaskCount() uint64                           { return 0 }
func (f *fakeSchedulerRunner) GetAllTasks() []types.TaskIndex                 { return nil }
func (f *fakeSchedulerRunner) GetRootTasks() []types.TaskIndex                { return nil }
func (f *fakeSchedulerRunner) GetAllCleanupTasks() []types.TaskIndex          { return nil }
func (f *fakeSchedulerRunner) GetRootCleanupTasks() []types.TaskIndex         { return nil }

func header(slot phase0.Slot, parent phase0.Root) *phase0.SignedBeaconBlockHeader {
	return &phase0.SignedBeaconBlockHeader{
		Message: &phase0.BeaconBlockHeader{
			Slot:       slot,
			ParentRoot: parent,
		},
	}
}

// ensureHeader sets a block's header the same way the real client
// ingestion path does, so AwaitHeader resolves immediately instead of
// waiting out its timeout.
func ensureHeader(t *testing.T, block *consensus.Block, h *phase0.SignedBeaconBlockHeader) {
	t.Helper()

	if err := block.EnsureHeader(func() (*phase0.SignedBeaconBlockHeader, error) {
		return h, nil
	}); err != nil {
		t.Fatalf("failed setting header: %v", err)
	}
}

func newTestTask(t *testing.T) (*Task, *consensus.BlockCache) {
	t.Helper()

	log := logrus.New()
	log.SetOutput(io.Discard)

	pool, err := clients.NewClientPool(log)
	if err != nil {
		t.Fatalf("failed constructing client pool: %v", err)
	}

	taskCtx := &types.TaskContext{
		Scheduler: &fakeSchedulerRunner{services: &fakeServices{pool: pool}},
		Logger: logger.NewLogger(&logger.ScopeOptions{
			Parent: log,
		}),
		SetResult:      func(types.TaskResult) {},
		ReportProgress: func(float64, string) {},
	}

	task := &Task{
		ctx:    taskCtx,
		logger: taskCtx.Logger.GetLogger(),
		config: Config{},
	}

	return task, pool.GetConsensusPool().GetBlockCache()
}

// handleBlock is Execute()'s per-block handling (the arm of its select
// statement), copied verbatim minus the channel receive itself, which
// can't be driven from this package: it is fired by an unexported method
// deep in the consensus client's ingestion pipeline, reachable in
// production only through a live beacon client goroutine.
func handleBlock(t *testing.T, task *Task, lastBlock **consensus.Block, block *consensus.Block, checkCount *int) {
	t.Helper()

	blockHeader := block.AwaitHeader(context.Background(), 500*time.Millisecond)

	switch {
	case blockHeader == nil:
	case *lastBlock != nil && !bytes.Equal(blockHeader.Message.ParentRoot[:], (*lastBlock).Root[:]):
		task.processChainReorg(*lastBlock, block)
	}

	*checkCount++
	*lastBlock = block
}

// TestNoPhantomReorgOnDroppedHeader is the regression test for the phantom
// reorg bug: a linear chain A -> B -> C, where B's header is never
// delivered in time, must not be counted as a reorg.
func TestNoPhantomReorgOnDroppedHeader(t *testing.T) {
	task, blockCache := newTestTask(t)

	var rootA, rootB, rootC phase0.Root
	rootA[0] = 0xAA
	rootB[0] = 0xBB
	rootC[0] = 0xCC

	blockA, _ := blockCache.AddBlock(rootA, 100)
	ensureHeader(t, blockA, header(100, phase0.Root{}))

	blockB, _ := blockCache.AddBlock(rootB, 101)
	// blockB's header is deliberately never set.

	blockC, _ := blockCache.AddBlock(rootC, 102)
	ensureHeader(t, blockC, header(102, rootB)) // C's real parent is B: a linear chain.

	var lastBlock *consensus.Block

	checkCount := 0
	for _, block := range []*consensus.Block{blockA, blockB, blockC} {
		handleBlock(t, task, &lastBlock, block, &checkCount)
	}

	if task.totalReorgs != 0 {
		t.Fatalf("expected no reorgs on a linear chain with one dropped header, got %d", task.totalReorgs)
	}
}

// TestRealReorgStillDetected guards against the fix over-correcting: a
// genuine reorg, where every header is available, must still be counted.
func TestRealReorgStillDetected(t *testing.T) {
	task, blockCache := newTestTask(t)

	var rootA, rootB, rootC phase0.Root
	rootA[0] = 0xAA
	rootB[0] = 0xBB
	rootC[0] = 0xCC

	blockA, _ := blockCache.AddBlock(rootA, 100)
	ensureHeader(t, blockA, header(100, phase0.Root{}))

	blockB, _ := blockCache.AddBlock(rootB, 101)
	ensureHeader(t, blockB, header(101, rootA))

	// blockC forks off blockA directly, not blockB: a genuine reorg.
	blockC, _ := blockCache.AddBlock(rootC, 102)
	ensureHeader(t, blockC, header(102, rootA))

	var lastBlock *consensus.Block

	checkCount := 0
	for _, block := range []*consensus.Block{blockA, blockB, blockC} {
		handleBlock(t, task, &lastBlock, block, &checkCount)
	}

	if task.totalReorgs != 1 {
		t.Fatalf("expected exactly 1 reorg for a genuine fork, got %d", task.totalReorgs)
	}
}
