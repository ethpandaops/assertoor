package execution

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Block struct {
	Hash       common.Hash
	Number     uint64
	blockMutex sync.Mutex
	blockChan  chan bool
	block      *types.Block
	seenMutex  sync.RWMutex
	seenChan   chan bool
	seenMap    map[uint16]*Client
}

func (block *Block) GetSeenBy() []*Client {
	block.seenMutex.RLock()
	defer block.seenMutex.RUnlock()

	clients := []*Client{}
	for _, client := range block.seenMap {
		clients = append(clients, client)
	}

	sort.Slice(clients, func(a, b int) bool {
		return clients[a].clientIdx < clients[b].clientIdx
	})

	return clients
}

func (block *Block) SetSeenBy(client *Client) {
	block.seenMutex.Lock()
	defer block.seenMutex.Unlock()
	block.seenMap[client.clientIdx] = client

	if block.seenChan != nil {
		close(block.seenChan)
	}
}

func (block *Block) AwaitSeenBy(ctx context.Context, client *Client) bool {
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		seen, seenChan := block.checkSeenBy(client)
		if seen {
			return true
		}

		select {
		case <-seenChan:
		case <-ctx.Done():
			return false
		}
	}
}

func (block *Block) checkSeenBy(client *Client) (seen bool, seenChan chan bool) {
	block.seenMutex.RLock()
	defer block.seenMutex.RUnlock()

	_, ok := block.seenMap[client.clientIdx]
	if !ok && block.seenChan == nil {
		block.seenChan = make(chan bool)
	}

	return ok, block.seenChan
}

func (block *Block) GetBlock() *types.Block {
	return block.block
}

func (block *Block) AwaitBlock(ctx context.Context, timeout time.Duration) *types.Block {
	if ctx == nil {
		ctx = context.Background()
	}

	select {
	case <-block.blockChan:
	case <-time.After(timeout):
	case <-ctx.Done():
	}

	return block.block
}

func (block *Block) GetParentHash() *common.Hash {
	if block.block == nil {
		return nil
	}

	return &block.block.Header().ParentHash
}

func (block *Block) EnsureBlock(loadBlock func() (*types.Block, error)) (bool, error) {
	if block.block != nil {
		return false, nil
	}

	block.blockMutex.Lock()
	defer block.blockMutex.Unlock()

	if block.block != nil {
		return false, nil
	}

	blockBody, err := loadBlock()
	if err != nil {
		return false, err
	}

	block.block = blockBody
	close(block.blockChan)

	return true, nil
}
