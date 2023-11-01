package consensus

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/beacon/pkg/human"
	"github.com/sirupsen/logrus"
)

// Client represents an Ethereum Consensus client.
type Client struct {
	url  string
	log  logrus.FieldLogger
	node beacon.Node
}

// NewConsensusClient returns a new Client client.
func NewConsensusClient(log logrus.FieldLogger, url string) Client {
	return Client{
		url: url,
		log: log,
	}
}

// Bootstrapped returns true if the client has been bootstrapped.
func (c *Client) Bootstrapped() bool {
	return c.node != nil
}

// Ready returns true if the client is ready to be used.
func (c *Client) Ready() bool {
	return c.Bootstrapped() && c.node.Healthy()
}

func (c *Client) Node() beacon.Node {
	return c.node
}

// Bootstrap bootstraps the client.
func (c *Client) Bootstrap(ctx context.Context, opts ...*beacon.Options) error {
	op := beacon.DefaultOptions().
		DisableEmptySlotDetection().
		DisableEmptySlotDetection().
		DisablePrometheusMetrics()

	op.HealthCheck.Interval = human.Duration{1 * time.Second}

	if len(opts) > 0 {
		op = opts[0]
	}

	node := beacon.NewNode(c.log, &beacon.Config{
		Name: "beacon_node",
		Addr: c.url,
	}, "sync_test_coordinator", *op)

	if err := node.Start(ctx); err != nil {
		return err
	}

	return nil
}

func (c *Client) IsHealthy(ctx context.Context) (bool, error) {
	return c.node.Healthy(), nil
}

func (c *Client) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	state, err := c.node.FetchSyncStatus(ctx)
	if err != nil {
		return nil, err
	}

	status := NewSyncStatus(state)

	return &status, nil
}

func (c *Client) GetSpec(ctx context.Context) (*Spec, error) {
	return c.GetSpec(ctx)
}

func (c *Client) GetCheckpoint(ctx context.Context, checkpointName CheckpointName) (*phase0.Checkpoint, error) {
	finality, err := c.node.FetchFinality(ctx, "head")
	if err != nil {
		return nil, err
	}

	if finality == nil {
		return nil, errors.New("finality is nil")
	}

	if checkpointName == Finalized {
		return finality.Finalized, nil
	}

	if checkpointName == Justified {
		return finality.Justified, nil
	}

	return nil, fmt.Errorf("unknown checkpoint name %s", checkpointName)
}

func (c *Client) GetNodeVersion(ctx context.Context) error {
	return c.GetNodeVersion(ctx)
}
