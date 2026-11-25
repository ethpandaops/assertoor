package checkconsensusblobcount

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "check_consensus_blob_count"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Check that proposed blocks reach the target or max number of blobs.",
		Category:    "consensus",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "matchingBlockRoots",
				Type:        "array",
				Description: "Array of block roots that reached the blob limit.",
			},
			{
				Name:        "blobTarget",
				Type:        "uint64",
				Description: "The target number of blobs per block reported by the execution client.",
			},
			{
				Name:        "blobMax",
				Type:        "uint64",
				Description: "The max number of blobs per block reported by the execution client.",
			},
			{
				Name:        "maxBlobCount",
				Type:        "int",
				Description: "The highest blob count seen in any checked block.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx          *types.TaskContext
	options      *types.TaskOptions
	config       Config
	logger       logrus.FieldLogger
	maxBlobCount int
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	// parse static config
	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	// load dynamic vars
	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	// validate config
	if err := config.Validate(); err != nil {
		return err
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	consensusPool := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool()

	blockSubscription := consensusPool.GetBlockCache().SubscribeBlockEvent(10)
	defer blockSubscription.Unsubscribe()

	totalMatches := 0
	matchingBlocks := []*consensus.Block{}
	checkCount := 0

	checkBlockMatch := func(block *consensus.Block) (bool, error) {
		checkCount++

		matches, err := t.checkBlock(ctx, block)
		if err != nil {
			t.ctx.SetResult(types.TaskResultFailure)
			return true, err
		}

		if matches {
			matchingBlocks = append(matchingBlocks, block)
			totalMatches++
		}

		if totalMatches >= t.config.BlockCount {
			t.setMatchingBlocksOutput(matchingBlocks)
			t.ctx.SetResult(types.TaskResultSuccess)
			t.ctx.ReportProgress(100, fmt.Sprintf("Found %d blocks reaching the %v blob count", totalMatches, t.config.BlobLimit))

			return true, nil
		}

		t.ctx.SetResult(types.TaskResultNone)
		t.ctx.ReportProgress(0, fmt.Sprintf("Waiting for blocks reaching the %v blob count... %d/%d (attempt %d)", t.config.BlobLimit, totalMatches, t.config.BlockCount, checkCount))

		return false, nil
	}

	// check cached blocks
	for _, block := range t.getLookbackBlocks(consensusPool) {
		if done, err := checkBlockMatch(block); done {
			return err
		}
	}

	for {
		select {
		case block := <-blockSubscription.Channel():
			if done, err := checkBlockMatch(block); done {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// getLookbackBlocks returns the cached blocks to check before waiting for new
// proposals, oldest first.
func (t *Task) getLookbackBlocks(consensusPool *consensus.Pool) []*consensus.Block {
	if t.config.CheckLookback <= 0 {
		return nil
	}

	blocks := consensusPool.GetBlockCache().GetCachedBlocks()
	if len(blocks) == 0 {
		return nil
	}

	lookbackBlocks := []*consensus.Block{}
	block := blocks[0]

	for {
		lookbackBlocks = append(lookbackBlocks, block)
		if len(lookbackBlocks) >= t.config.CheckLookback {
			break
		}

		parentRoot := block.GetParentRoot()
		if parentRoot == nil {
			break
		}

		block = consensusPool.GetBlockCache().GetCachedBlockByRoot(*parentRoot)
		if block == nil {
			break
		}
	}

	// reverse, oldest block first
	for i, j := 0, len(lookbackBlocks)-1; i < j; i, j = i+1, j-1 {
		lookbackBlocks[i], lookbackBlocks[j] = lookbackBlocks[j], lookbackBlocks[i]
	}

	return lookbackBlocks
}

// checkBlock reports whether a block reached the configured blob limit. A
// returned error aborts the task.
func (t *Task) checkBlock(ctx context.Context, block *consensus.Block) (bool, error) {
	blockData := block.AwaitBlock(ctx, 2*time.Second)
	if blockData == nil {
		t.logger.Warnf("could not fetch block data for block %v [0x%x]", block.Slot, block.Root)
		return false, nil
	}

	if t.config.ValidatorNamePattern != "" && !t.checkValidatorName(block, blockData) {
		return false, nil
	}

	commitments, err := blockData.BlobKZGCommitments()
	if err != nil {
		t.logger.Debugf("skipping block %v [0x%x], no blob commitments: %v", block.Slot, block.Root, err)
		return false, nil
	}

	limits, err := t.getBlobLimits(ctx)
	if err != nil {
		return false, err
	}

	t.ctx.Outputs.SetVar("blobTarget", limits.Target)
	t.ctx.Outputs.SetVar("blobMax", limits.Max)

	blobCount := len(commitments)
	if blobCount > t.maxBlobCount {
		t.maxBlobCount = blobCount
		t.ctx.Outputs.SetVar("maxBlobCount", blobCount)
	}

	result, threshold, err := limits.check(blobCount, t.config.BlobLimit)
	if err != nil {
		return false, err
	}

	switch result {
	case blobCheckReached:
		t.logger.Infof("block %v [0x%x] reached the %v blob count (want: >= %v, have: %v)", block.Slot, block.Root, t.config.BlobLimit, threshold, blobCount)

		return true, nil
	case blobCheckOverMax:
		if t.config.FailOnOverMax {
			return false, fmt.Errorf("block %v [0x%x] carries more blobs than the max (max: %v, have: %v)", block.Slot, block.Root, limits.Max, blobCount)
		}

		t.logger.Warnf("block %v [0x%x] carries more blobs than the max (max: %v, have: %v)", block.Slot, block.Root, limits.Max, blobCount)
	case blobCheckBelow:
		t.logger.Infof("block %v [0x%x] did not reach the %v blob count (want: >= %v, have: %v)", block.Slot, block.Root, t.config.BlobLimit, threshold, blobCount)
	}

	return false, nil
}

// getBlobLimits reads the blob limits of the fork the execution client is
// currently on. The consensus spec only carries the max, so eth_config
// (EIP-7910) is the only source that has the target as well.
func (t *Task) getBlobLimits(ctx context.Context) (*blobLimits, error) {
	client, err := t.getExecutionClient()
	if err != nil {
		return nil, err
	}

	ethConfig, err := client.GetRPCClient().GetEthConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not query eth_config from client %v: %w", client.GetName(), err)
	}

	if ethConfig == nil || ethConfig.Current == nil {
		// a cancelled request returns no config, keep the context error
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		return nil, fmt.Errorf("client %v returned no current fork config", client.GetName())
	}

	limits, err := parseBlobLimits(ethConfig.Current.BlobSchedule)
	if err != nil {
		return nil, fmt.Errorf("could not read blob limits from client %v: %w", client.GetName(), err)
	}

	return limits, nil
}

// getExecutionClient returns the execution client to read eth_config from.
func (t *Task) getExecutionClient() (*execution.Client, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients := clientPool.GetExecutionPool().GetReadyEndpoints(true)
		if len(clients) == 0 {
			return nil, fmt.Errorf("no ready execution clients found")
		}

		return clients[0], nil
	}

	poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
	if len(poolClients) == 0 {
		return nil, fmt.Errorf("no execution clients found with pattern %v", t.config.ClientPattern)
	}

	return poolClients[0].ExecutionClient, nil
}

func (t *Task) checkValidatorName(block *consensus.Block, blockData *spec.VersionedSignedBeaconBlock) bool {
	proposerIndex, err := blockData.ProposerIndex()
	if err != nil {
		t.logger.Warnf("could not get proposer index for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	validatorName := t.ctx.Scheduler.GetServices().ValidatorNames().GetValidatorName(uint64(proposerIndex))

	matched, err := regexp.MatchString(t.config.ValidatorNamePattern, validatorName)
	if err != nil {
		t.logger.Warnf("could not check validator name for block %v [0x%x]: %v", block.Slot, block.Root, err)
		return false
	}

	if !matched {
		t.logger.Debugf("skipping block %v [0x%x]: unmatched validator name (have: %v, want: %v)", block.Slot, block.Root, validatorName, t.config.ValidatorNamePattern)
	}

	return matched
}

func (t *Task) setMatchingBlocksOutput(blocks []*consensus.Block) {
	blockRoots := make([]string, 0, len(blocks))
	for _, block := range blocks {
		blockRoots = append(blockRoots, block.Root.String())
	}

	t.ctx.Outputs.SetVar("matchingBlockRoots", blockRoots)
}
