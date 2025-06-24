package txloadtool

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/eth/protocols/eth"

	"math/big"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/utils/sentry"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/params"
	"github.com/sirupsen/logrus"
)

// Peer connects to an execution client (a bockchain node) on the p2p network (i.e., the peer of the node)
type Peer struct {
	ctx     context.Context
	taskCtx *types.TaskContext
	logger  logrus.FieldLogger
	node    *execution.Client
	conn    *sentry.Conn
}

// NewPeer creates a new peer
func NewPeer(ctx context.Context, taskCtx *types.TaskContext, logger logrus.FieldLogger, client *execution.Client) *Peer {
	return &Peer{
		ctx:     ctx,
		taskCtx: taskCtx,
		logger:  logger,
		node:    client,
		conn:    nil,
	}
}

// Close closes the connection to the execution node
func (p *Peer) Close() error {
	if p.conn != nil {
		err := p.conn.Close()
		p.conn = nil

		return err
	}

	return nil
}

// Connect establishes a connection to the execution node and performs the handshake
func (p *Peer) Connect() error {
	chainConfig := params.AllDevChainProtocolChanges

	head, err := p.node.GetRPCClient().GetLatestBlock(p.ctx)
	if err != nil {
		p.taskCtx.SetResult(types.TaskResultFailure)
		return err
	}

	chainID, err := p.node.GetRPCClient().GetEthClient().ChainID(p.ctx)
	if err != nil {
		return err
	}

	chainConfig.ChainID = chainID

	genesis, err := p.node.GetRPCClient().GetEthClient().BlockByNumber(p.ctx, new(big.Int).SetUint64(0))
	if err != nil {
		p.logger.Errorf("Failed to fetch genesis block: %v", err)
		p.taskCtx.SetResult(types.TaskResultFailure)

		return err
	}

	conn, err := sentry.GetTCPConn(p.node)
	if err != nil {
		p.logger.Errorf("Failed to get TCP connection: %v", err)
		p.taskCtx.SetResult(types.TaskResultFailure)

		return err
	}

	p.conn = conn
	forkID := forkid.NewID(chainConfig, genesis, head.NumberU64(), head.Time())

	// handshake
	err = p.conn.Peer(chainConfig.ChainID, genesis.Hash(), head.Hash(), forkID, nil)
	if err != nil {
		return err
	}

	p.logger.Infof("Connected to %s", p.node.GetName())

	return nil
}

func (p *Peer) ReadTransactionMessages(timeout time.Duration) (*eth.TransactionsPacket, error) {
	// Check if the connection is nil
	if p.conn == nil {
		p.logger.Errorf("Peer has no active connection, cannot read transaction messages")
		p.taskCtx.SetResult(types.TaskResultFailure)

		return nil, fmt.Errorf("peer has no active connection, cannot read transaction messages")
	}

	txs, err := p.conn.ReadTransactionMessages(timeout)

	return txs, err
}
