package generatetransaction

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/erigontech/assertoor/pkg/coordinator/clients/execution"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet/blobtx"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
	ctx                  *types.TaskContext
	options              *types.TaskOptions
	config               Config
	logger               logrus.FieldLogger
	wallet               *wallet.Wallet
	authorizationWallets []*wallet.Wallet

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
	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	// load wallets
	privKey, err := crypto.HexToECDSA(config.PrivateKey)
	if err != nil {
		return err
	}

	t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	// load authorization wallets
	if config.SetCodeTxType && len(config.Authorizations) > 0 {
		t.authorizationWallets = make([]*wallet.Wallet, 0, len(config.Authorizations))

		for _, authorization := range config.Authorizations {
			privKey, err2 := crypto.HexToECDSA(authorization.SignerPrivkey)
			if err2 != nil {
				return err2
			}

			authWallet, err2 := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(privKey)
			if err2 != nil {
				return fmt.Errorf("cannot initialize authorization wallet: %w", err2)
			}

			t.authorizationWallets = append(t.authorizationWallets, authWallet)
		}
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

	if t.config.SetCodeTxType {
		for _, authWallet := range t.authorizationWallets {
			err = authWallet.AwaitReady(ctx)
			if err != nil {
				return fmt.Errorf("cannot load authorization wallet state: %w", err)
			}
		}
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))

	tx, err := t.generateTransaction(ctx)
	if err != nil {
		return err
	}

	if txData, err2 := vars.GeneralizeData(tx); err2 == nil {
		t.ctx.Outputs.SetVar("transaction", txData)
	} else {
		t.logger.Warnf("Failed setting `transaction` output: %v", err2)
	}

	txBytes, err := tx.MarshalBinary()
	if err != nil {
		t.logger.Warnf("Failed setting `transactionHex` output: %v", err)
	} else {
		t.ctx.Outputs.SetVar("transactionHex", hexutil.Encode(txBytes))
	}

	var clients []*execution.Client

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().GetReadyEndpoints(true)
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
	if len(clients) == 0 {
		err = fmt.Errorf("no ready clients available")
	} else {
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
	}

	if err != nil {
		return err
	}

	if t.config.TransactionHashResultVar != "" {
		t.ctx.Vars.SetVar(t.config.TransactionHashResultVar, tx.Hash().Hex())
	}

	t.ctx.Outputs.SetVar("transactionHash", tx.Hash().Hex())

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

		t.ctx.Outputs.SetVar("contractAddress", receipt.ContractAddress.Hex())

		if receiptData, err := vars.GeneralizeData(receipt); err == nil {
			t.ctx.Outputs.SetVar("receipt", receiptData)

			if t.config.TransactionReceiptResultVar != "" {
				t.ctx.Vars.SetVar(t.config.TransactionReceiptResultVar, receiptData)
			}
		} else {
			t.logger.Warnf("Failed setting `receipt` output: %v", err)
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

	for _, authorizationWallet := range t.authorizationWallets {
		// resync nonces of authorization wallets (might be increased by more than one, so we need to resync)
		authorizationWallet.ResyncState()
	}

	return nil
}

func (t *Task) generateTransaction(ctx context.Context) (*ethtypes.Transaction, error) {
	tx, err := t.wallet.BuildTransaction(ctx, func(_ context.Context, nonce uint64, _ bind.SignerFn) (*ethtypes.Transaction, error) {
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

		txAmount := new(big.Int).Set(&t.config.Amount.Value)
		if t.config.RandomAmount {
			n, err := rand.Int(rand.Reader, txAmount)
			if err == nil {
				txAmount = n
			}
		}

		if t.config.Nonce != nil {
			nonce = *t.config.Nonce
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
				GasPrice: &t.config.FeeCap.Value,
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
				BlobFeeCap: uint256.MustFromBig(&t.config.BlobFeeCap.Value),
				GasTipCap:  uint256.MustFromBig(&t.config.TipCap.Value),
				GasFeeCap:  uint256.MustFromBig(&t.config.FeeCap.Value),
				Gas:        t.config.GasLimit,
				To:         *toAddr,
				Value:      uint256.MustFromBig(txAmount),
				Data:       txData,
				BlobHashes: blobHashes,
				Sidecar:    blobSidecar,
			}
		case t.config.SetCodeTxType:
			authList := []ethtypes.SetCodeAuthorization{}

			for idx, authorization := range t.config.Authorizations {
				authEntry := ethtypes.SetCodeAuthorization{
					ChainID: *uint256.NewInt(authorization.ChainID),
					Address: common.HexToAddress(authorization.CodeAddress),
				}

				authWallet := t.authorizationWallets[idx]

				if authorization.Nonce != nil {
					authEntry.Nonce = *authorization.Nonce
				} else {
					authEntry.Nonce = authWallet.UseNextNonce(true)
				}

				authEntry, err := ethtypes.SignSetCode(authWallet.GetPrivateKey(), authEntry)
				if err != nil {
					return nil, err
				}

				authList = append(authList, authEntry)
			}

			txObj = &ethtypes.SetCodeTx{
				ChainID:   uint256.MustFromBig(t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID()),
				Nonce:     nonce,
				GasTipCap: uint256.MustFromBig(&t.config.TipCap.Value),
				GasFeeCap: uint256.MustFromBig(&t.config.FeeCap.Value),
				Gas:       t.config.GasLimit,
				To:        *toAddr,
				Value:     uint256.MustFromBig(txAmount),
				Data:      txData,
				AuthList:  authList,
			}

		default:
			txObj = &ethtypes.DynamicFeeTx{
				ChainID:   t.ctx.Scheduler.GetServices().ClientPool().GetExecutionPool().GetBlockCache().GetChainID(),
				Nonce:     nonce,
				GasTipCap: &t.config.TipCap.Value,
				GasFeeCap: &t.config.FeeCap.Value,
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
