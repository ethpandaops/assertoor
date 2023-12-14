package consensus

import (
	"bytes"
	"context"
	"fmt"
	"runtime/debug"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/consensus/rpc"
	"github.com/sirupsen/logrus"
)

func (client *Client) runClientLoop() {
	defer func() {
		if err := recover(); err != nil {
			logrus.WithError(err.(error)).Errorf("uncaught panic in PoolClient.runPoolClientLoop subroutine: %v, stack: %v", err, string(debug.Stack()))
		}
	}()

	for {
		err := client.checkClient()
		waitTime := 10

		if err == nil {
			err = client.runClientLogic()
		}

		if err == nil {
			client.retryCounter = 0
			return
		}

		client.isOnline = false
		client.lastError = err
		client.lastEvent = time.Now()
		client.retryCounter++

		if client.retryCounter > 10 {
			waitTime = 300
		} else if client.retryCounter > 5 {
			waitTime = 60
		}

		client.logger.Warnf("upstream client error: %v, retrying in %v sec...", err, waitTime)
		time.Sleep(time.Duration(waitTime) * time.Second)
	}
}

func (client *Client) checkClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	err := client.rpcClient.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("initialization of attestantio/go-eth2-client failed: %w", err)
	}

	// get node version
	nodeVersion, err := client.rpcClient.GetNodeVersion(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching node version: %v", err)
	}

	client.versionStr = nodeVersion
	client.parseClientVersion(nodeVersion)

	// get & compare genesis
	genesis, err := client.rpcClient.GetGenesis(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching genesis: %v", err)
	}

	err = client.pool.blockCache.SetGenesis(genesis)
	if err != nil {
		return fmt.Errorf("invalid genesis: %v", err)
	}

	// get & compare chain specs
	specs, err := client.rpcClient.GetConfigSpecs(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching specs: %v", err)
	}

	err = client.pool.blockCache.SetClientSpecs(specs)
	if err != nil {
		return fmt.Errorf("invalid node specs: %v", err)
	}

	// check synchronization state
	syncStatus, err := client.rpcClient.GetNodeSyncing(ctx)
	if err != nil {
		return fmt.Errorf("error while fetching synchronization status: %v", err)
	}

	if syncStatus == nil {
		return fmt.Errorf("could not get synchronization status")
	}

	client.isSyncing = syncStatus.IsSyncing
	client.isOptimistic = syncStatus.IsOptimistic

	return nil
}

func (client *Client) runClientLogic() error {
	// get latest header
	err := client.pollClientHead()
	if err != nil {
		return err
	}

	// check latest header / sync status
	if client.isSyncing {
		return fmt.Errorf("beacon node is synchronizing")
	}

	if client.isOptimistic {
		return fmt.Errorf("beacon node is optimistic")
	}

	specs := client.pool.blockCache.GetSpecs()

	finalizedEpoch, _ := client.pool.blockCache.GetFinalizedCheckpoint()
	if client.headSlot < phase0.Slot(finalizedEpoch)*phase0.Slot(specs.SlotsPerEpoch) {
		return fmt.Errorf("beacon node is behind finalized checkpoint (node head: %v, finalized: %v)", client.headSlot, phase0.Slot(finalizedEpoch)*phase0.Slot(specs.SlotsPerEpoch))
	}

	// start event stream
	blockStream := client.rpcClient.NewBlockStream(rpc.StreamBlockEvent | rpc.StreamFinalizedEvent)
	defer blockStream.Close()

	// process events
	client.lastEvent = time.Now()

	for {
		eventTimeout := time.Since(client.lastEvent)
		if eventTimeout > 30*time.Second {
			eventTimeout = 0
		} else {
			eventTimeout = 30*time.Second - eventTimeout
		}

		select {
		case evt := <-blockStream.EventChan:
			now := time.Now()

			switch evt.Event {
			case rpc.StreamBlockEvent:
				err := client.processBlockEvent(evt.Data.(*v1.BlockEvent))
				if err != nil {
					client.logger.Warnf("failed processing block event: %v", err)
				}

			case rpc.StreamFinalizedEvent:
				err := client.processFinalizedEvent(evt.Data.(*v1.FinalizedCheckpointEvent))
				if err != nil {
					client.logger.Warnf("failed processing finalized event: %v", err)
				}
			}

			client.logger.Tracef("event (%v) processing time: %v ms", evt.Event, time.Since(now).Milliseconds())
			client.lastEvent = time.Now()
		case ready := <-blockStream.ReadyChan:
			if client.isOnline != ready {
				client.isOnline = ready
				if ready {
					client.logger.Debug("RPC event stream connected")
				} else {
					client.logger.Debug("RPC event stream disconnected")
				}
			}
		case <-time.After(eventTimeout):
			client.logger.Debug("no head event since 30 secs, polling chain head")

			err := client.pollClientHead()
			if err != nil {
				client.isOnline = false
				return err
			}

			client.lastEvent = time.Now()
		}
	}
}

func (client *Client) processBlockEvent(evt *v1.BlockEvent) error {
	err := client.processBlock(evt.Block, evt.Slot, nil, "streamed")
	if err != nil {
		return fmt.Errorf("could not process block: %v", err)
	}

	client.headMutex.Lock()
	client.headSlot = evt.Slot
	client.headRoot = evt.Block
	client.headMutex.Unlock()
	client.pool.resetHeadForkCache()

	return nil
}

func (client *Client) processFinalizedEvent(evt *v1.FinalizedCheckpointEvent) error {
	client.logger.Debugf("received finalization_checkpoint event: finalized %v [0x%x]", evt.Epoch, evt.Block)
	return client.setFinalizedHead(evt.Epoch, evt.Block)
}

func (client *Client) pollClientHead() error {
	ctx, cancel := context.WithTimeout(client.clientCtx, 10*time.Second)
	defer cancel()

	latestHeader, err := client.rpcClient.GetLatestBlockHead(ctx)
	if err != nil {
		return fmt.Errorf("could not get latest header: %v", err)
	}

	if latestHeader == nil {
		return fmt.Errorf("could not find latest header")
	}

	err = client.processBlock(latestHeader.Root, latestHeader.Header.Message.Slot, latestHeader.Header, "polled")
	if err != nil {
		return fmt.Errorf("could not get process block: %v", err)
	}

	finalityCheckpoint, err := client.rpcClient.GetFinalityCheckpoints(ctx)
	if err != nil {
		return fmt.Errorf("could not get finality checkpoint: %v", err)
	}

	return client.setFinalizedHead(finalityCheckpoint.Finalized.Epoch, finalityCheckpoint.Finalized.Root)
}

func (client *Client) processBlock(root phase0.Root, slot phase0.Slot, header *phase0.SignedBeaconBlockHeader, source string) error {
	cachedBlock, isNewBlock := client.pool.blockCache.AddBlock(root, slot)
	if cachedBlock == nil {
		return fmt.Errorf("could not add block to cache %v [0x%x]", slot, root)
	}

	cachedBlock.SetSeenBy(client)

	if isNewBlock {
		client.logger.Infof("received cl block %v [0x%x] %v", slot, root, source)
	} else {
		client.logger.Debugf("received known cl block %v [0x%x] %v", slot, root, source)
	}

	err := cachedBlock.EnsureHeader(func() (*phase0.SignedBeaconBlockHeader, error) {
		if header != nil {
			return header, nil
		}
		ctx, cancel := context.WithTimeout(client.clientCtx, 10*time.Second)
		defer cancel()
		header, err := client.rpcClient.GetBlockHeaderByBlockroot(ctx, cachedBlock.Root)
		if err != nil {
			return nil, err
		}
		return header.Header, nil
	})
	if err != nil {
		return err
	}

	loaded, err := cachedBlock.EnsureBlock(func() (*spec.VersionedSignedBeaconBlock, error) {
		ctx, cancel := context.WithTimeout(client.clientCtx, 10*time.Second)
		defer cancel()

		block, err2 := client.rpcClient.GetBlockBodyByBlockroot(ctx, cachedBlock.Root)
		if err2 != nil {
			return nil, err2
		}
		return block, nil
	})
	if err != nil {
		return err
	}

	if loaded {
		client.pool.blockCache.notifyBlockReady(cachedBlock)
	}

	client.headMutex.Lock()
	if bytes.Equal(client.headRoot[:], root[:]) {
		client.headMutex.Unlock()
		return nil
	}

	client.headSlot = slot
	client.headRoot = root
	client.headMutex.Unlock()

	client.pool.resetHeadForkCache()

	err = client.blockDispatcher.Fire(cachedBlock)
	if err != nil {
		return fmt.Errorf("error in client block dispatcher: %w", err)
	}

	return nil
}

func (client *Client) setFinalizedHead(epoch phase0.Epoch, root phase0.Root) error {
	client.headMutex.Lock()
	if bytes.Equal(client.finalizedRoot[:], root[:]) {
		client.headMutex.Unlock()
		return nil
	}

	client.finalizedEpoch = epoch
	client.finalizedRoot = root
	client.headMutex.Unlock()

	client.pool.blockCache.SetFinalizedCheckpoint(epoch, root)

	err := client.checkpointDispatcher.Fire(&FinalizedCheckpoint{
		Epoch: epoch,
		Root:  root,
	})
	if err != nil {
		return fmt.Errorf("error in client checkpoint dispatcher: %w", err)
	}

	return nil
}
