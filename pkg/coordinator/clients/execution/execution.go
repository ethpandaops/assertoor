package execution

import (
	"context"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/onrik/ethrpc"
	"github.com/sirupsen/logrus"
)

type Client struct {
	url          string
	log          logrus.FieldLogger
	ethrpcClient *ethrpc.EthRPC
	ethClient    *ethclient.Client
}

func NewExecutionClient(log logrus.FieldLogger, url string) Client {
	return Client{
		url: url,
		log: log,
	}
}

func (c *Client) EthRPC() *ethrpc.EthRPC {
	return c.ethrpcClient
}

func (c *Client) EthClient() *ethclient.Client {
	return c.ethClient
}

func (c *Client) Bootstrapped() bool {
	return c.ethClient != nil && c.ethrpcClient != nil
}

func (c *Client) Bootstrap(ctx context.Context) error {
	client, err := ethclient.Dial(c.url)
	if err != nil {
		return err
	}

	c.ethClient = client
	c.ethrpcClient = ethrpc.New(c.url)

	return nil
}

func (c *Client) IsHealthy(ctx context.Context) (bool, error) {
	_, err := c.SyncStatus(ctx)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (c *Client) SyncStatus(ctx context.Context) (*SyncStatus, error) {
	status, err := c.ethClient.SyncProgress(ctx)
	if err != nil {
		return nil, err
	}

	if status == nil && err == nil {
		// Not syncing
		ss := &SyncStatus{}
		ss.IsSyncing = false

		return ss, nil
	}

	return &SyncStatus{
		IsSyncing:     true,
		CurrentBlock:  status.CurrentBlock,
		HighestBlock:  status.HighestBlock,
		StartingBlock: status.StartingBlock,
	}, nil
}
