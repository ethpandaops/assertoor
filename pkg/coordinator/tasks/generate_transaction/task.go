package generatetransaction

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet/blobtx"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_transaction"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates normal transaction, sends it to the network and checks the receipt",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
	wallet  *wallet.Wallet

	targetAddr      common.Address
	transactionData []byte
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Name() string {
	return TaskName
}

func (t *Task) Description() string {
	return TaskDescriptor.Description
}

func (t *Task) Title() string {
	return t.ctx.Vars.ResolvePlaceholders(t.options.Title)
}

func (t *Task) Config() interface{} {
	return t.config
}

func (t *Task) Logger() logrus.FieldLogger {
	return t.logger
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
	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	// load wallets
	privKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}

	t.wallet, err = t.ctx.Scheduler.GetCoordinator().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	// parse target addr
	if config.TargetAddress != "" {
		err = t.targetAddr.UnmarshalText([]byte(config.TargetAddress))
		if err != nil {
			return fmt.Errorf("cannot decode execution addr: %w", err)
		}
	}

	// parse transaction data
	if config.CallData != "" {
		t.transactionData = common.FromHex(config.CallData)
	}

	t.config = config

	return nil
}

//nolint:gocyclo // ignore
func (t *Task) Execute(ctx context.Context) error {
	err := t.wallet.AwaitReady(ctx)
	if err != nil {
		return fmt.Errorf("cannot load wallet state: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := t.generateTransaction(ctx)
	if err != nil {
		return err
	}

	var clients []*execution.Client

	clientPool := t.ctx.Scheduler.GetCoordinator().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints()
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			return fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	err = nil

	for i := 0; i < len(clients); i++ {
		client := clients[i%len(clients)]

		t.logger.WithFields(logrus.Fields{
			"client": client.GetName(),
		}).Infof("sending tx: %v", tx.Hash().Hex())

		err = client.GetRPCClient().SendTransaction(ctx, tx)
		if err == nil {
			break
		}
	}

	if err != nil {
		return err
	}

	if t.config.TransactionHashResultVar != "" {
		t.ctx.Vars.SetVar(t.config.TransactionHashResultVar, tx.Hash().Hex())
	}

	if t.config.AwaitReceipt {
		receipt, err := t.wallet.AwaitTransaction(ctx, tx)
		if err != nil {
			t.logger.Warnf("failed waiting for tx receipt: %v", err)
			return fmt.Errorf("failed waiting for tx receipt: %v", err)
		}

		if receipt == nil {
			return fmt.Errorf("tx receipt not found")
		}

		t.logger.Infof("transaction %v confirmed (nonce: %v, status: %v)", tx.Hash().Hex(), tx.Nonce(), receipt.Status)

		if t.config.FailOnSuccess && receipt.Status > 0 {
			return fmt.Errorf("transaction succeeded, but expected rejection")
		}

		if t.config.FailOnReject && receipt.Status == 0 {
			return fmt.Errorf("transaction rejected, but expected success")
		}

		if t.config.ContractAddressResultVar != "" {
			t.ctx.Vars.SetVar(t.config.ContractAddressResultVar, receipt.ContractAddress.Hex())
		}

		if t.config.TransactionReceiptResultVar != "" {
			receiptJSON, err := json.Marshal(receipt)
			if err == nil {
				receiptMap := map[string]interface{}{}
				err = json.Unmarshal(receiptJSON, &receiptMap)

				if err == nil {
					t.ctx.Vars.SetVar(t.config.TransactionReceiptResultVar, receiptMap)
				} else {
					t.logger.Errorf("could not unmarshal transaction receipt for result var: %v", err)
				}
			} else {
				t.logger.Errorf("could not marshal transaction receipt for result var: %v", err)
			}
		}

		if len(t.config.ExpectEvents) > 0 {
			for _, expectedEvent := range t.config.ExpectEvents {
				foundEvent := false

				for _, log := range receipt.Logs {
					if expectedEvent.Topic0 != "" && (len(log.Topics) < 1 || !bytes.Equal(common.FromHex(expectedEvent.Topic0), log.Topics[0][:])) {
						continue
					}

					if expectedEvent.Topic1 != "" && (len(log.Topics) < 2 || !bytes.Equal(common.FromHex(expectedEvent.Topic1), log.Topics[1][:])) {
						continue
					}

					if expectedEvent.Topic2 != "" && (len(log.Topics) < 3 || !bytes.Equal(common.FromHex(expectedEvent.Topic2), log.Topics[2][:])) {
						continue
					}

					if expectedEvent.Data != "" && !bytes.Equal(common.FromHex(expectedEvent.Data), log.Data) {
						continue
					}

					foundEvent = true

					break
				}

				if !foundEvent {
					return fmt.Errorf("expected event not fired: %v", expectedEvent)
				}
			}
		}
	}

	return nil
}

func (t *Task) generateTransaction(ctx context.Context) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(ctx context.Context, nonce uint64, signer bind.SignerFn) (*ethtypes.Transaction, error) {
		var toAddr *common.Address

		if !t.config.ContractDeployment {
			addr := t.wallet.GetAddress()
			if t.config.RandomTarget {
				addrBytes := make([]byte, 20)
				//nolint:errcheck // ignore
				rand.Read(addrBytes)
				addr = common.Address(addrBytes)
			} else if t.config.TargetAddress != "" {
				addr = t.targetAddr
			}

			toAddr = &addr
		}

		txAmount := new(big.Int).Set(t.config.Amount)
		if t.config.RandomAmount {
			n, err := rand.Int(rand.Reader, txAmount)
			if err == nil {
				txAmount = n
			}
		}

		txData := []byte{}
		if t.transactionData != nil {
			txData = t.transactionData
		}

		var txObj ethtypes.TxData

		switch {
		case t.config.LegacyTxType:
			txObj = &ethtypes.LegacyTx{
				Nonce:    nonce,
				GasPrice: t.config.FeeCap,
				Gas:      t.config.GasLimit,
				To:       toAddr,
				Value:    txAmount,
				Data:     txData,
			}
		case t.config.BlobTxType:
			if toAddr == nil {
				return nil, fmt.Errorf("contract deployment not supported with blob transactions")
			}

			blobData := t.config.BlobData
			if blobData == "" {
				blobData = "identifier"
			}

			blobHashes, blobSidecar, err := blobtx.GenerateBlobSidecar(strings.Split(blobData, ";"), 0, 0)
			if err != nil {
				return nil, err
			}

			txObj = &ethtypes.BlobTx{
				Nonce:      nonce,
				BlobFeeCap: uint256.MustFromBig(t.config.BlobFeeCap),
				GasTipCap:  uint256.MustFromBig(t.config.TipCap),
				GasFeeCap:  uint256.MustFromBig(t.config.FeeCap),
				Gas:        t.config.GasLimit,
				To:         *toAddr,
				Value:      uint256.MustFromBig(txAmount),
				Data:       txData,
				BlobHashes: blobHashes,
				Sidecar:    blobSidecar,
			}
		default:
			txObj = &ethtypes.DynamicFeeTx{
				ChainID:   t.ctx.Scheduler.GetCoordinator().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
				Nonce:     nonce,
				GasTipCap: t.config.TipCap,
				GasFeeCap: t.config.FeeCap,
				Gas:       t.config.GasLimit,
				To:        toAddr,
				Value:     txAmount,
				Data:      txData,
			}
		}
		return ethtypes.NewTx(txObj), nil
	})
	if err != nil {
		return nil, err
	}

	return tx, nil
}
