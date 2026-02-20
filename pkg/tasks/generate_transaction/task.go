package generatetransaction

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	mrand "math/rand"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/ethpandaops/spamoor/txbuilder"
	"github.com/holiman/uint256"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_transaction"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates normal transaction, sends it to the network and checks the receipt",
		Category:    "transaction",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "transaction",
				Type:        "object",
				Description: "The generated transaction object.",
			},
			{
				Name:        "transactionHex",
				Type:        "string",
				Description: "The transaction encoded as hex.",
			},
			{
				Name:        "transactionHash",
				Type:        "string",
				Description: "The transaction hash.",
			},
			{
				Name:        "contractAddress",
				Type:        "string",
				Description: "The deployed contract address (if contract deployment).",
			},
			{
				Name:        "receipt",
				Type:        "object",
				Description: "The transaction receipt (if awaitReceipt is enabled).",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx                  *types.TaskContext
	options              *types.TaskOptions
	config               Config
	logger               logrus.FieldLogger
	wallet               *spamoor.Wallet
	authorizationWallets []*spamoor.Wallet

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
	t.ctx.ReportProgress(0, "Preparing wallet...")

	privKey, err := crypto.HexToECDSA(t.config.PrivateKey)
	if err != nil {
		return err
	}

	t.wallet, err = t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.ctx.Scheduler.GetTestRunCtx(), privKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	// load authorization wallets
	if t.config.SetCodeTxType && len(t.config.Authorizations) > 0 {
		t.authorizationWallets = make([]*spamoor.Wallet, 0, len(t.config.Authorizations))

		for _, authorization := range t.config.Authorizations {
			privKey, err2 := crypto.HexToECDSA(authorization.SignerPrivkey)
			if err2 != nil {
				return err2
			}

			authWallet, err2 := t.ctx.Scheduler.GetServices().WalletManager().GetWalletByPrivkey(t.ctx.Scheduler.GetTestRunCtx(), privKey)
			if err2 != nil {
				return fmt.Errorf("cannot initialize authorization wallet: %w", err2)
			}

			t.authorizationWallets = append(t.authorizationWallets, authWallet)
		}
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", t.wallet.GetAddress().Hex(), t.wallet.GetNonce(), t.wallet.GetReadableBalance(18, 0, 4, false, false))

	t.ctx.ReportProgress(0, "Waiting for ready clients...")

	var clients []*execution.Client

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().AwaitReadyEndpoints(ctx, true)
		if len(clients) == 0 {
			return ctx.Err()
		}
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

	t.ctx.ReportProgress(0, "Generating transaction...")

	tx, err := t.generateTransaction()
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

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()
	spamoorClients := make([]*spamoor.Client, len(clients))
	for i, c := range clients {
		spamoorClients[i] = walletMgr.GetClient(c)
	}

	for i := 0; i < len(spamoorClients); i++ {
		clientIdx := i % len(spamoorClients)

		t.logger.WithFields(logrus.Fields{
			"client": spamoorClients[clientIdx].GetName(),
		}).Infof("sending tx: %v", tx.Hash().Hex())

		err = walletMgr.GetTxPool().SendTransaction(ctx, t.wallet, tx, &spamoor.SendTransactionOptions{
			Client:             spamoorClients[clientIdx],
			ClientList:         spamoorClients,
			ClientsStartOffset: i,
		})
		if err == nil {
			break
		}
	}

	if err != nil {
		t.wallet.MarkSkippedNonce(tx.Nonce())
		return err
	}

	if t.config.TransactionHashResultVar != "" {
		t.ctx.Vars.SetVar(t.config.TransactionHashResultVar, tx.Hash().Hex())
	}

	t.ctx.Outputs.SetVar("transactionHash", tx.Hash().Hex())

	if t.config.AwaitReceipt {
		var receipt *ethtypes.Receipt

		receipt, err = t.ctx.Scheduler.GetServices().WalletManager().GetTxPool().AwaitTransaction(ctx, t.wallet, tx)
		if err != nil {
			t.logger.Warnf("failed waiting for tx receipt: %v", err)
			return fmt.Errorf("failed waiting for tx receipt: %w", err)
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

		receiptData, generalizeErr := vars.GeneralizeData(receipt)
		if generalizeErr == nil {
			t.ctx.Outputs.SetVar("receipt", receiptData)

			if t.config.TransactionReceiptResultVar != "" {
				t.ctx.Vars.SetVar(t.config.TransactionReceiptResultVar, receiptData)
			}
		} else {
			t.logger.Warnf("Failed setting `receipt` output: %v", generalizeErr)
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
		client := t.ctx.Scheduler.GetServices().WalletManager().GetReadyClient()

		err = authorizationWallet.UpdateWallet(ctx, client, true)
		if err != nil {
			return fmt.Errorf("cannot update authorization wallet: %w", err)
		}
	}

	t.ctx.ReportProgress(100, fmt.Sprintf("Transaction completed: %s", tx.Hash().Hex()))

	return nil
}

//nolint:gocyclo // transaction generation has multiple branches for different tx types
func (t *Task) generateTransaction() (*ethtypes.Transaction, error) {
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

	txData := []byte{}
	if t.transactionData != nil {
		txData = t.transactionData
	}

	var txObj *ethtypes.Transaction

	switch {
	case t.config.LegacyTxType:
		txData, err := txbuilder.LegacyTx(&txbuilder.TxMetadata{
			GasFeeCap: uint256.MustFromBig(&t.config.FeeCap.Value),
			Gas:       t.config.GasLimit,
			To:        toAddr,
			Value:     uint256.MustFromBig(txAmount),
			Data:      txData,
		})
		if err != nil {
			return nil, err
		}

		if t.config.Nonce != nil {
			txObj, err = t.wallet.ReplaceLegacyTx(txData, *t.config.Nonce)
		} else {
			txObj, err = t.wallet.BuildLegacyTx(txData)
		}

		if err != nil {
			return nil, err
		}

	case t.config.BlobTxType:
		if toAddr == nil {
			return nil, fmt.Errorf("contract deployment not supported with blob transactions")
		}

		blobCount := t.config.BlobSidecars

		// Parse blobData formats:
		//   Old format (per-sidecar): "refs;refs;refs" — semicolons separate sidecar groups
		//   New format (all sidecars): "ref,ref,ref" — commas separate refs, applied to all sidecars
		// Within each group, commas separate individual refs.
		// "identifier" and "label" are replaced with a unique blob label.
		var blobDataGroups []string

		if t.config.BlobData != "" {
			if strings.Contains(t.config.BlobData, ";") {
				blobDataGroups = strings.Split(t.config.BlobData, ";")
				blobCount = uint64(len(blobDataGroups))
			} else {
				// Same refs for all sidecars
				for range blobCount {
					blobDataGroups = append(blobDataGroups, t.config.BlobData)
				}
			}
		}

		blobRefs := make([][]string, blobCount)

		for i := uint64(0); i < blobCount; i++ {
			blobLabel := fmt.Sprintf("0x1611BB0000%08dFF%02dFF%04dFEED", 0, i, 0)

			if i < uint64(len(blobDataGroups)) {
				blobRefs[i] = []string{}

				for _, blob := range strings.Split(blobDataGroups[i], ",") {
					if blob == "identifier" || blob == "label" {
						blob = blobLabel
					}

					blobRefs[i] = append(blobRefs[i], blob)
				}
			} else {
				specialBlob := mrand.Intn(50) //nolint:gosec // weak random is fine here
				switch specialBlob {
				case 0: // special blob commitment - all 0x0
					blobRefs[i] = []string{"0x0"}
				case 1, 2: // reuse well known blob
					blobRefs[i] = []string{"repeat:0x42:1337"}
				case 3, 4: // duplicate commitment
					if i == 0 {
						blobRefs[i] = []string{blobLabel, "random"}
					} else {
						blobRefs[i] = []string{"copy:0"}
					}
				default: // random blob data
					blobRefs[i] = []string{blobLabel, "random:full"}
				}
			}
		}

		blobTx, err := txbuilder.BuildBlobTx(&txbuilder.TxMetadata{
			GasFeeCap:  uint256.MustFromBig(&t.config.FeeCap.Value),
			GasTipCap:  uint256.MustFromBig(&t.config.TipCap.Value),
			BlobFeeCap: uint256.MustFromBig(&t.config.BlobFeeCap.Value),
			Gas:        t.config.GasLimit,
			To:         toAddr,
			Value:      uint256.NewInt(0),
			Data:       txData,
		}, blobRefs)
		if err != nil {
			return nil, err
		}

		if t.config.Nonce != nil {
			txObj, err = t.wallet.ReplaceBlobTx(blobTx, *t.config.Nonce)
		} else {
			txObj, err = t.wallet.BuildBlobTx(blobTx)
		}

		if err != nil {
			return nil, err
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
				authEntry.Nonce = authWallet.GetNextNonce()
			}

			authEntry, err := ethtypes.SignSetCode(authWallet.GetPrivateKey(), authEntry)
			if err != nil {
				return nil, err
			}

			authList = append(authList, authEntry)
		}

		setCodeTx, err := txbuilder.SetCodeTx(&txbuilder.TxMetadata{
			GasFeeCap:  uint256.MustFromBig(&t.config.FeeCap.Value),
			GasTipCap:  uint256.MustFromBig(&t.config.TipCap.Value),
			BlobFeeCap: uint256.MustFromBig(&t.config.BlobFeeCap.Value),
			Gas:        t.config.GasLimit,
			To:         toAddr,
			Value:      uint256.NewInt(0),
			AuthList:   authList,
			Data:       txData,
		})
		if err != nil {
			return nil, err
		}

		if t.config.Nonce != nil {
			txObj, err = t.wallet.ReplaceSetCodeTx(setCodeTx, *t.config.Nonce)
		} else {
			txObj, err = t.wallet.BuildSetCodeTx(setCodeTx)
		}

		if err != nil {
			return nil, err
		}

	default:
		txData, err := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
			GasTipCap: uint256.MustFromBig(&t.config.TipCap.Value),
			GasFeeCap: uint256.MustFromBig(&t.config.FeeCap.Value),
			Gas:       t.config.GasLimit,
			To:        toAddr,
			Value:     uint256.MustFromBig(txAmount),
			Data:      txData,
		})
		if err != nil {
			return nil, err
		}

		if t.config.Nonce != nil {
			txObj, err = t.wallet.ReplaceDynamicFeeTx(txData, *t.config.Nonce)
		} else {
			txObj, err = t.wallet.BuildDynamicFeeTx(txData)
		}

		if err != nil {
			return nil, err
		}
	}

	return txObj, nil
}
