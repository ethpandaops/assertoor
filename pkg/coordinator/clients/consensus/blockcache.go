package consensus

import (
	"bytes"
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/mashingan/smapping"
	"github.com/sirupsen/logrus"
)

type BlockCache struct {
	followDistance uint64
	specMutex      sync.RWMutex
	specs          *ChainSpec
	genesisMutex   sync.Mutex
	genesis        *v1.Genesis
	wallclockMutex sync.Mutex
	wallclock      *ethwallclock.EthereumBeaconChain
	finalizedMutex sync.RWMutex
	finalizedEpoch phase0.Epoch
	finalizedRoot  phase0.Root
	cacheMutex     sync.RWMutex
	slotMap        map[phase0.Slot][]*Block
	rootMap        map[phase0.Root]*Block
	maxSlotIdx     int64

	blockDispatcher          Dispatcher[*Block]
	checkpointDispatcher     Dispatcher[*FinalizedCheckpoint]
	wallclockEpochDispatcher Dispatcher[*ethwallclock.Epoch]
	wallclockSlotDispatcher  Dispatcher[*ethwallclock.Slot]
}

func NewBlockCache(followDistance uint64) (*BlockCache, error) {
	if followDistance == 0 {
		return nil, fmt.Errorf("cannot initialize block cache without follow distance")
	}

	cache := BlockCache{
		followDistance: followDistance,
		slotMap:        make(map[phase0.Slot][]*Block),
		rootMap:        make(map[phase0.Root]*Block),
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				logrus.WithError(err.(error)).Errorf("uncaught panic in BlockCache.runCacheCleanup subroutine: %v, stack: %v", err, string(debug.Stack()))
			}
		}()
		cache.runCacheCleanup()
	}()

	return &cache, nil
}

func (cache *BlockCache) SubscribeBlockEvent(capacity int) *Subscription[*Block] {
	return cache.blockDispatcher.Subscribe(capacity)
}

func (cache *BlockCache) SubscribeFinalizedEvent(capacity int) *Subscription[*FinalizedCheckpoint] {
	return cache.checkpointDispatcher.Subscribe(capacity)
}

func (cache *BlockCache) SubscribeWallclockEpochEvent(capacity int) *Subscription[*ethwallclock.Epoch] {
	return cache.wallclockEpochDispatcher.Subscribe(capacity)
}

func (cache *BlockCache) SubscribeWallclockSlotEvent(capacity int) *Subscription[*ethwallclock.Slot] {
	return cache.wallclockSlotDispatcher.Subscribe(capacity)
}

func (cache *BlockCache) notifyBlockReady(block *Block) {
	cache.blockDispatcher.Fire(block)
}

func (cache *BlockCache) SetMinFollowDistance(followDistance uint64) {
	if followDistance > cache.followDistance {
		cache.followDistance = followDistance
	}
}

func (cache *BlockCache) SetGenesis(genesis *v1.Genesis) error {
	cache.genesisMutex.Lock()
	defer cache.genesisMutex.Unlock()

	if cache.genesis != nil {
		if cache.genesis.GenesisTime != genesis.GenesisTime {
			return fmt.Errorf("genesis mismatch: GenesisTime")
		}

		if !bytes.Equal(cache.genesis.GenesisValidatorsRoot[:], genesis.GenesisValidatorsRoot[:]) {
			return fmt.Errorf("genesis mismatch: GenesisValidatorsRoot")
		}
	} else {
		cache.genesis = genesis
	}

	return nil
}

func (cache *BlockCache) GetGenesis() *v1.Genesis {
	return cache.genesis
}

func (cache *BlockCache) SetClientSpecs(specValues map[string]interface{}) error {
	cache.specMutex.Lock()
	defer cache.specMutex.Unlock()

	specs := ChainSpec{}

	err := smapping.FillStructByTags(&specs, specValues, "yaml")
	if err != nil {
		return err
	}

	if cache.specs != nil {
		mismatches := cache.specs.CheckMismatch(&specs)
		if len(mismatches) > 0 {
			return fmt.Errorf("spec mismatch: %v", strings.Join(mismatches, ", "))
		}
	}

	cache.specs = &specs

	return nil
}

func (cache *BlockCache) GetSpecs() *ChainSpec {
	cache.specMutex.RLock()
	defer cache.specMutex.RUnlock()

	return cache.specs
}

func (cache *BlockCache) InitWallclock() {
	cache.wallclockMutex.Lock()
	defer cache.wallclockMutex.Unlock()

	if cache.wallclock != nil {
		return
	}

	specs := cache.GetSpecs()
	if specs == nil || cache.genesis == nil {
		return
	}

	cache.wallclock = ethwallclock.NewEthereumBeaconChain(cache.genesis.GenesisTime, specs.SecondsPerSlot, specs.SlotsPerEpoch)
	cache.wallclock.OnEpochChanged(func(current ethwallclock.Epoch) {
		cache.wallclockEpochDispatcher.Fire(&current)
	})
	cache.wallclock.OnSlotChanged(func(current ethwallclock.Slot) {
		cache.wallclockSlotDispatcher.Fire(&current)
	})
}

func (cache *BlockCache) GetWallclock() *ethwallclock.EthereumBeaconChain {
	cache.wallclockMutex.Lock()
	defer cache.wallclockMutex.Unlock()

	return cache.wallclock
}

func (cache *BlockCache) SetFinalizedCheckpoint(finalizedEpoch phase0.Epoch, finalizedRoot phase0.Root) {
	cache.finalizedMutex.Lock()
	if finalizedEpoch <= cache.finalizedEpoch {
		cache.finalizedMutex.Unlock()
		return
	}

	cache.finalizedEpoch = finalizedEpoch
	cache.finalizedRoot = finalizedRoot
	cache.finalizedMutex.Unlock()

	cache.checkpointDispatcher.Fire(&FinalizedCheckpoint{
		Epoch: finalizedEpoch,
		Root:  finalizedRoot,
	})
}

func (cache *BlockCache) GetFinalizedCheckpoint() (phase0.Epoch, phase0.Root) {
	cache.finalizedMutex.RLock()
	defer cache.finalizedMutex.RUnlock()

	return cache.finalizedEpoch, cache.finalizedRoot
}

func (cache *BlockCache) AddBlock(root phase0.Root, slot phase0.Slot) (*Block, bool) {
	cache.cacheMutex.Lock()
	defer cache.cacheMutex.Unlock()

	if cache.rootMap[root] != nil {
		return cache.rootMap[root], false
	}

	if int64(slot) < cache.maxSlotIdx-int64(cache.followDistance) {
		return nil, false
	}

	cacheBlock := &Block{
		Root:       root,
		Slot:       slot,
		seenMap:    make(map[uint16]*Client),
		headerChan: make(chan bool),
		blockChan:  make(chan bool),
	}
	cache.rootMap[root] = cacheBlock

	if cache.slotMap[slot] == nil {
		cache.slotMap[slot] = []*Block{cacheBlock}
	} else {
		cache.slotMap[slot] = append(cache.slotMap[slot], cacheBlock)
	}

	if int64(slot) > cache.maxSlotIdx {
		cache.maxSlotIdx = int64(slot)
	}

	return cacheBlock, true
}

func (cache *BlockCache) GetCachedBlockByRoot(root phase0.Root) *Block {
	cache.cacheMutex.RLock()
	defer cache.cacheMutex.RUnlock()

	return cache.rootMap[root]
}

func (cache *BlockCache) GetCachedBlocksBySlot(slot phase0.Slot) []*Block {
	cache.cacheMutex.RLock()
	defer cache.cacheMutex.RUnlock()

	blocks := cache.slotMap[slot]
	if len(blocks) == 0 {
		return blocks
	}

	res := make([]*Block, len(blocks))
	copy(res, blocks)

	return res
}

func (cache *BlockCache) GetCachedBlocks() []*Block {
	cache.cacheMutex.RLock()
	defer cache.cacheMutex.RUnlock()

	blocks := []*Block{}
	slots := []phase0.Slot{}

	for slot := range cache.slotMap {
		slots = append(slots, slot)
	}

	sort.Slice(slots, func(a, b int) bool {
		return slots[a] > slots[b]
	})

	for _, slot := range slots {
		blocks = append(blocks, cache.slotMap[slot]...)
	}

	return blocks
}

func (cache *BlockCache) runCacheCleanup() {
	for {
		time.Sleep(30 * time.Second)
		cache.cleanupCache()
	}
}

func (cache *BlockCache) cleanupCache() {
	cache.cacheMutex.Lock()
	defer cache.cacheMutex.Unlock()

	minSlot := cache.maxSlotIdx - int64(cache.followDistance)
	if minSlot <= 0 {
		return
	}

	for slot, blocks := range cache.slotMap {
		if slot >= phase0.Slot(minSlot) {
			continue
		}

		for _, block := range blocks {
			delete(cache.rootMap, block.Root)
		}

		delete(cache.slotMap, slot)
	}
}

func (cache *BlockCache) IsCanonicalBlock(blockRoot, headRoot phase0.Root) bool {
	res, _ := cache.GetBlockDistance(blockRoot, headRoot)
	return res
}

func (cache *BlockCache) GetBlockDistance(blockRoot, headRoot phase0.Root) (linked bool, distance uint64) {
	if bytes.Equal(headRoot[:], blockRoot[:]) {
		return true, 0
	}

	block := cache.GetCachedBlockByRoot(blockRoot)
	if block == nil {
		return false, 0
	}

	blockSlot := block.Slot
	headBlock := cache.GetCachedBlockByRoot(headRoot)
	dist := uint64(0)

	for headBlock != nil {
		if headBlock.Slot < blockSlot {
			return false, 0
		}

		parentRoot := headBlock.GetParentRoot()
		if parentRoot == nil {
			return false, 0
		}

		dist++
		if bytes.Equal(parentRoot[:], blockRoot[:]) {
			return true, dist
		}

		headBlock = cache.GetCachedBlockByRoot(*parentRoot)
	}

	return false, 0
}
