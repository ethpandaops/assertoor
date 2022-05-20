package consensus

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"

	eth2client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/http"
)

// Client represents an Ethereum Consensus client.
type Client struct {
	url    string
	client eth2client.Service
	log    logrus.FieldLogger

	state State
}

// NewConsensusClient returns a new Client client.
func NewConsensusClient(log logrus.FieldLogger, url string) Client {
	return Client{
		url:   url,
		log:   log,
		state: NewState(),
	}
}

// State returns the state of the client
func (c *Client) State() State {
	return c.state
}

// Bootstrapped returns true if the client has been bootstrapped.
func (c *Client) Bootstrapped() bool {
	return c.client != nil
}

// Bootstrap bootstraps the client.
func (c *Client) Bootstrap(ctx context.Context) error {
	client, err := http.New(ctx,
		http.WithAddress(c.url),
		http.WithLogLevel(zerolog.WarnLevel),
	)
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

func (c *Client) Start(ctx context.Context) {
	if err := c.tick(ctx); err != nil {
		c.log.WithError(err).Error("tick failed")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * 5):
			if err := c.tick(ctx); err != nil {
				c.log.WithError(err).Error("tick failed")
			}
		}
	}
}

func (c *Client) tick(ctx context.Context) error {
	if _, err := c.IsHealthy(ctx); err != nil {
		c.log.WithError(err).Error("health check failed")
		return err
	}

	if _, err := c.GetSyncStatus(ctx); err != nil {
		c.log.WithError(err).Error("get sync status failed")
	}

	return nil
}

func (c *Client) IsHealthy(ctx context.Context) (bool, error) {
	err := c.GetNodeVersion(ctx)
	if err != nil {
		c.state.Healthy = false

		return false, err
	}

	c.state.Healthy = true

	return true, nil
}

func (c *Client) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	provider, isProvider := c.client.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.NodeSyncingProvider")
	}

	syncing, err := provider.NodeSyncing(ctx)
	if err != nil {
		return nil, err
	}

	status := NewSyncStatus(syncing)

	return &status, nil
}

func (c *Client) GetSpec(ctx context.Context) (*Spec, error) {
	provider, isProvider := c.client.(eth2client.SpecProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.SpecProvider")
	}

	spec, err := provider.Spec(ctx)
	if err != nil {
		return nil, err
	}

	err = c.state.Spec.update(spec)
	if err != nil {
		return nil, err
	}

	return &c.state.Spec, nil
}

func (c *Client) GetChainState(ctx context.Context) (ChainState, error) {
	state := NewChainState()
	for _, name := range CheckpointNames {
		checkpoint, err := c.GetCheckpoint(ctx, name)
		if err != nil {
			c.log.WithError(err).WithField("checkpoint", checkpoint).Error("get checkpoint failed")

			continue
		}

		state[name] = *checkpoint
	}

	return state, nil
}

func (c *Client) GetCheckpoint(ctx context.Context, checkpointName CheckpointName) (*Checkpoint, error) {
	if c.state.Spec.SlotsPerEpoch == 0 {
		return nil, errors.New("slots per epoch not set")
	}

	provider, isProvider := c.client.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		return nil, errors.New("client does not implement eth2client.BeaconBlockHeadersProvider")
	}

	block, err := provider.BeaconBlockHeader(ctx, string(checkpointName))
	if err != nil {
		return nil, err
	}

	if block == nil {
		return nil, errors.New("block is nil")
	}

	if block.Header == nil {
		return nil, errors.New("block header is nil")
	}

	if block.Header.Message == nil {
		return nil, errors.New("block header message is nil")
	}

	checkpoint, exists := c.state.ChainState[checkpointName]
	if !exists {
		return nil, errors.New("checkpoint does not exist")
	}

	checkpoint.Slot = uint64(block.Header.Message.Slot)
	checkpoint.Epoch = uint64(math.Floor(float64(uint64(block.Header.Message.Slot) / c.state.Spec.SlotsPerEpoch))) //nolint:staticcheck // false positive

	return &checkpoint, nil
}

func (c *Client) GetNodeVersion(ctx context.Context) error {
	provider, isProvider := c.client.(eth2client.NodeVersionProvider)
	if !isProvider {
		return errors.New("client does not implement eth2client.NodeVersionProvider")
	}

	_, err := provider.NodeVersion(ctx)
	if err != nil {
		return err
	}

	return nil
}
