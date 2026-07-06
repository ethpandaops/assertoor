package generatebuilderdeposits

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/spamoor/spamoor"
	"github.com/ethpandaops/spamoor/txbuilder"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	e2types "github.com/wealdtech/go-eth2-types/v2"
	util "github.com/wealdtech/go-eth2-util"
)

// domainBuilderDeposit is DOMAIN_BUILDER_DEPOSIT (Gloas/EIP-8282): a dedicated
// domain type (0x0E000000) that separates builder-deposit proofs-of-possession
// from regular validator deposits.
var domainBuilderDeposit = common.BLSDomainType{0x0E, 0x00, 0x00, 0x00}

const outputTypeArray = "array"

var (
	TaskName       = "generate_builder_deposits"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates builder deposits (EIP-8282) and sends them to the network",
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "builderPubkeys",
				Type:        outputTypeArray,
				Description: "Array of builder public keys for the deposits.",
			},
			{
				Name:        "depositTransactions",
				Type:        outputTypeArray,
				Description: "Array of builder deposit transaction hashes.",
			},
			{
				Name:        "depositReceipts",
				Type:        outputTypeArray,
				Description: "Array of builder deposit transaction receipts.",
			},
			{
				Name:        "includedDeposits",
				Type:        "number",
				Description: "Number of builder deposits included on beacon chain (when awaitInclusion is enabled).",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx                 *types.TaskContext
	options             *types.TaskOptions
	config              Config
	logger              logrus.FieldLogger
	builderKeySeed      []byte
	nextIndex           uint64
	lastIndex           uint64
	walletPrivKey       *ecdsa.PrivateKey
	depositContractAddr ethcommon.Address
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

	if config.Mnemonic != "" {
		t.builderKeySeed, err = t.mnemonicToSeed(config.Mnemonic)
		if err != nil {
			return err
		}

		t.logger.Infof("builder key seed: 0x%x", t.builderKeySeed)
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return err
	}

	t.config = config
	t.depositContractAddr = ethcommon.HexToAddress(config.BuilderDepositContract)

	return nil
}

//nolint:gocyclo // ignore
func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex)
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount)
	}

	var subscription *consensus.Subscription[*consensus.Block]
	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	// Subscribe early so we don't miss the block containing the builder deposit.
	// As with EIP-6110 deposits, the request is surfaced in the same beacon block
	// as the EL transaction is dequeued.
	var inclusionSubscription *consensus.Subscription[*consensus.Block]
	if t.config.AwaitInclusion {
		inclusionSubscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer inclusionSubscription.Unsubscribe()
	}

	var pendingChan chan bool

	pendingWg := sync.WaitGroup{}

	if t.config.LimitPending > 0 {
		pendingChan = make(chan bool, t.config.LimitPending)
	}

	perSlotCount := 0
	totalCount := 0

	targetCount := 0
	if t.config.LimitTotal > 0 {
		targetCount = t.config.LimitTotal
	} else if t.lastIndex > 0 {
		targetCount = int(t.lastIndex - t.nextIndex) //nolint:gosec // G115: difference is bounded by config values
	}

	t.ctx.ReportProgress(0, "Starting builder deposit generation")

	depositTransactions := []string{}
	builderPubkeys := []string{}
	depositReceipts := map[string]*ethtypes.Receipt{}
	depositReceiptsMtx := sync.Mutex{}

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		if pendingChan != nil {
			select {
			case <-ctx.Done():
				return nil
			case pendingChan <- true:
			}
		}

		pendingWg.Add(1)

		pubkey, tx, err := t.generateBuilderDeposit(ctx, accountIdx, func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
			if pendingChan != nil {
				<-pendingChan
			}

			depositReceiptsMtx.Lock()

			depositReceipts[tx.Hash().Hex()] = receipt

			depositReceiptsMtx.Unlock()

			switch {
			case receipt != nil:
				t.logger.Infof("builder deposit %v confirmed in block %v (nonce: %v, status: %v)", tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status)
			case err != nil:
				t.logger.Errorf("error awaiting builder deposit transaction receipt: %v", err.Error())
			default:
				t.logger.Warnf("no receipt for builder deposit transaction: %v", tx.Hash().Hex())
			}

			pendingWg.Done()
		})
		if err != nil {
			t.logger.Errorf("error generating builder deposit: %v", err.Error())
			// Note: onComplete callback is still called by spamoor even on error,
			// so we don't call pendingWg.Done() here
		} else {
			perSlotCount++
			totalCount++

			builderPubkeys = append(builderPubkeys, pubkey)
			depositTransactions = append(depositTransactions, tx.Hash().Hex())

			if targetCount > 0 {
				progress := float64(totalCount) / float64(targetCount) * 100
				t.ctx.ReportProgress(progress, fmt.Sprintf("Generated %d/%d builder deposits", totalCount, targetCount))
			} else {
				t.ctx.ReportProgress(0, fmt.Sprintf("Generated %d builder deposits", totalCount))
			}
		}

		if t.lastIndex > 0 && t.nextIndex >= t.lastIndex {
			break
		}

		if t.config.LimitTotal > 0 && totalCount >= t.config.LimitTotal {
			break
		}

		if t.config.LimitPerSlot > 0 && perSlotCount >= t.config.LimitPerSlot {
			// await next block
			perSlotCount = 0

			select {
			case <-ctx.Done():
				return nil
			case <-subscription.Channel():
			}
		} else if ctx.Err() != nil {
			return nil
		}
	}

	if t.config.AwaitReceipt {
		pendingWg.Wait()
	}

	t.ctx.Outputs.SetVar("builderPubkeys", builderPubkeys)
	t.ctx.Outputs.SetVar("depositTransactions", depositTransactions)

	receiptList := []interface{}{}

	for _, txhash := range depositTransactions {
		var receiptMap map[string]interface{}

		receipt := depositReceipts[txhash]
		if receipt != nil {
			receiptJSON, err := json.Marshal(receipt)
			if err == nil {
				receiptMap = map[string]interface{}{}

				if err = json.Unmarshal(receiptJSON, &receiptMap); err != nil {
					t.logger.Errorf("could not unmarshal transaction receipt for result var: %v", err)

					receiptMap = nil
				}
			} else {
				t.logger.Errorf("could not marshal transaction receipt for result var: %v", err)
			}
		}

		receiptList = append(receiptList, receiptMap)
	}

	t.ctx.Outputs.SetVar("depositReceipts", receiptList)

	t.ctx.ReportProgress(100, fmt.Sprintf("Completed generating %d builder deposits", totalCount))

	if t.config.FailOnReject {
		for _, txhash := range depositTransactions {
			if depositReceipts[txhash] == nil {
				t.logger.Errorf("no receipt for builder deposit transaction: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}

			if depositReceipts[txhash].Status == 0 {
				t.logger.Errorf("builder deposit transaction failed: %v", txhash)
				t.ctx.SetResult(types.TaskResultFailure)

				break
			}
		}
	}

	if t.config.AwaitInclusion && len(builderPubkeys) > 0 {
		err := t.awaitInclusion(ctx, inclusionSubscription, builderPubkeys, totalCount)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Task) awaitInclusion(ctx context.Context, blockSubscription *consensus.Subscription[*consensus.Block], builderPubkeys []string, totalCount int) error {
	pendingPubkeys := make(map[string]bool, len(builderPubkeys))
	for _, pubkey := range builderPubkeys {
		pendingPubkeys[pubkey] = true
	}

	includedCount := 0
	t.ctx.Outputs.SetVar("includedDeposits", includedCount)

	t.logger.Infof("waiting for %d builder deposits to be included in beacon blocks", len(pendingPubkeys))
	t.ctx.ReportProgress(50, fmt.Sprintf("Awaiting inclusion: 0/%d builder deposits included", len(pendingPubkeys)))

	for len(pendingPubkeys) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case block := <-blockSubscription.Channel():
			blockData := block.AwaitBlock(ctx, 2*time.Second)
			if blockData == nil {
				continue
			}

			// Builder deposit requests only exist on Gloas+ and live in the
			// separate execution payload envelope.
			if blockData.Version >= spec.DataVersionGloas {
				payload := block.AwaitPayload(ctx, 2*time.Second)
				if payload != nil && payload.Gloas != nil && payload.Gloas.Message.ExecutionRequests != nil {
					for _, depositReq := range payload.Gloas.Message.ExecutionRequests.BuilderDeposits {
						pubkeyStr := depositReq.Pubkey.String()
						if pendingPubkeys[pubkeyStr] {
							delete(pendingPubkeys, pubkeyStr)

							includedCount++
						}
					}
				}
			}

			t.ctx.Outputs.SetVar("includedDeposits", includedCount)

			if includedCount > 0 {
				inclusionProgress := float64(includedCount) / float64(totalCount) * 50
				t.ctx.ReportProgress(50+inclusionProgress,
					fmt.Sprintf("Awaiting inclusion: %d/%d builder deposits included", includedCount, totalCount))

				t.logger.Infof("builder deposits included in block %d (%d/%d)", block.Slot, includedCount, totalCount)
			}
		}
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("All %d builder deposits included on beacon chain", totalCount))

	return nil
}

func (t *Task) generateBuilderDeposit(ctx context.Context, accountIdx uint64, onComplete spamoor.TxCompleteFn) (string, *ethtypes.Transaction, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	builderSet := clientPool.GetConsensusPool().GetBuilderSet()

	var builderPubkey []byte

	var builderPrivkey *e2types.BLSPrivateKey

	if t.builderKeySeed != nil {
		builderKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

		builderPriv, err := util.PrivateKeyFromSeedAndPath(t.builderKeySeed, builderKeyPath)
		if err != nil {
			return "", nil, fmt.Errorf("failed generating builder key %v: %w", builderKeyPath, err)
		}

		builderPrivkey = builderPriv
		builderPubkey = builderPrivkey.PublicKey().Marshal()

		t.logger.Debugf("generated builder pubkey %v: 0x%x", builderKeyPath, builderPubkey)
	} else {
		builderPubkey = ethcommon.FromHex(t.config.PublicKey)
	}

	var existingBuilder *consensus.BuilderInfo

	for i := range builderSet {
		if bytes.Equal(builderSet[i].Builder.PublicKey[:], builderPubkey) {
			existingBuilder = builderSet[i]
			break
		}
	}

	if t.builderKeySeed != nil && existingBuilder != nil {
		t.logger.Warnf("builder already exists on chain (index: %v)", existingBuilder.Index)
	} else if t.builderKeySeed == nil && existingBuilder == nil {
		t.logger.Warnf("builder not found on chain for top-up deposit")
	}

	var pub common.BLSPubkey

	copy(pub[:], builderPubkey)

	withdrCreds, err := t.builderWithdrawalCredentials()
	if err != nil {
		return "", nil, err
	}

	// Convert deposit amount from ETH to Gwei using big.Int to avoid overflow.
	depositAmountGwei := new(big.Int).SetUint64(t.config.DepositAmount)
	depositAmountGwei.Mul(depositAmountGwei, big.NewInt(1000000000))

	if !depositAmountGwei.IsUint64() {
		return "", nil, fmt.Errorf("deposit amount too large: %v ETH", t.config.DepositAmount)
	}

	depositData := common.DepositData{
		Pubkey:                pub,
		WithdrawalCredentials: tree.Root(withdrCreds),
		Amount:                common.Gwei(depositAmountGwei.Uint64()),
		Signature:             common.BLSSignature{},
	}

	if signErr := t.signDepositData(&depositData, builderPrivkey); signErr != nil {
		return "", nil, signErr
	}

	// Build the raw 184-byte builder deposit calldata (EIP-8282, no function selector):
	// 0-48:   pubkey
	// 48-80:  withdrawal credentials
	// 80-88:  amount (8 bytes, big-endian gwei)
	// 88-184: signature
	txData := make([]byte, 184)
	copy(txData[0:48], depositData.Pubkey[:])
	copy(txData[48:80], depositData.WithdrawalCredentials[:])
	binary.BigEndian.PutUint64(txData[80:88], uint64(depositData.Amount))
	copy(txData[88:184], depositData.Signature[:])

	// select clients
	var clients []*execution.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients = clientPool.GetExecutionPool().AwaitReadyEndpoints(ctx, true)
		if len(clients) == 0 {
			return "", nil, ctx.Err()
		}
	} else {
		poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(poolClients) == 0 {
			return "", nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		clients = make([]*execution.Client, len(poolClients))
		for i, c := range poolClients {
			clients[i] = c.ExecutionClient
		}
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	spamoorClients := make([]*spamoor.Client, len(clients))
	for i, c := range clients {
		spamoorClients[i] = walletMgr.GetClient(c)
	}

	txWallet, err := walletMgr.GetWalletByPrivkey(t.ctx.Scheduler.GetTestRunCtx(), t.walletPrivKey)
	if err != nil {
		return "", nil, fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", txWallet.GetAddress().Hex(), txWallet.GetNonce(), txWallet.GetReadableBalance(18, 0, 4, false, false))

	// The contract requires msg.value - fee >= amount * 1 gwei, so send the deposit
	// amount (in wei) plus a small buffer to cover the request fee.
	amountWei := new(big.Int).Mul(depositAmountGwei, big.NewInt(1000000000))

	txValue := new(big.Int).Set(amountWei)
	if t.config.TxFeeBuffer != nil {
		txValue.Add(txValue, t.config.TxFeeBuffer)
	}

	dynFeeTx, err := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
		GasTipCap: uint256.MustFromBig(t.config.TxTipCap),
		GasFeeCap: uint256.MustFromBig(t.config.TxFeeCap),
		Gas:       t.config.TxGasLimit,
		To:        &t.depositContractAddr,
		Value:     uint256.MustFromBig(txValue),
		Data:      txData,
	})
	if err != nil {
		return "", nil, fmt.Errorf("cannot build builder deposit tx data: %w", err)
	}

	tx, err := txWallet.BuildDynamicFeeTx(dynFeeTx)
	if err != nil {
		return "", nil, fmt.Errorf("cannot build builder deposit transaction: %w", err)
	}

	err = walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      spamoorClients[0],
		ClientList:  spamoorClients,
		Rebroadcast: true,
		OnComplete:  onComplete,
		LogFn: func(client *spamoor.Client, retry int, rebroadcast int, err error) {
			if err != nil {
				return
			}

			logEntry := t.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
			})

			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted builder deposit transaction (account idx: %v, nonce: %v, attempt: %v)", accountIdx, tx.Nonce(), retry)
		},
	})
	if err != nil {
		txWallet.MarkSkippedNonce(tx.Nonce())
		return "", nil, fmt.Errorf("failed sending builder deposit transaction: %w", err)
	}

	return pub.String(), tx, nil
}

// builderWithdrawalCredentials returns the 32-byte withdrawal credentials for the
// builder deposit. Builder deposits require 0xB0-prefixed credentials.
func (t *Task) builderWithdrawalCredentials() ([]byte, error) {
	if t.config.TopUpDeposit {
		// Credentials are ignored by the consensus layer for top-ups.
		return ethcommon.FromHex("0x0000000000000000000000000000000000000000000000000000000000000000"), nil
	}

	if t.config.WithdrawalCredentials != "" {
		creds := ethcommon.FromHex(t.config.WithdrawalCredentials)
		if len(creds) != 32 {
			return nil, fmt.Errorf("withdrawalCredentials must be 32 bytes, got %d", len(creds))
		}

		return creds, nil
	}

	// Derive 0xB0 credentials from the builder address, or the funding wallet address.
	var addr ethcommon.Address
	if t.config.BuilderAddress != "" {
		addr = ethcommon.HexToAddress(t.config.BuilderAddress)
	} else {
		addr = crypto.PubkeyToAddress(t.walletPrivKey.PublicKey)
	}

	creds := make([]byte, 32)
	creds[0] = 0xB0
	copy(creds[12:], addr[:])

	return creds, nil
}

func (t *Task) signDepositData(depositData *common.DepositData, builderPrivkey *e2types.BLSPrivateKey) error {
	if t.config.TopUpDeposit {
		return nil
	}

	useInvalid, err := t.shouldUseInvalidSignature()
	if err != nil {
		return err
	}

	if useInvalid {
		if _, err := cryptorand.Read(depositData.Signature[:]); err != nil {
			return fmt.Errorf("failed to generate random invalid signature: %w", err)
		}

		t.logger.Debugf("generated builder deposit with invalid (random) signature for pubkey 0x%x", depositData.Pubkey)

		return nil
	}

	msgRoot := depositData.ToMessage().HashTreeRoot(tree.GetHashFn())

	var secKey hbls.SecretKey
	if err := secKey.Deserialize(builderPrivkey.Marshal()); err != nil {
		return fmt.Errorf("cannot convert builder priv key")
	}

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(domainBuilderDeposit, common.Version(genesis.GenesisForkVersion), common.Root{})
	msg := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(msg[:])
	copy(depositData.Signature[:], sig.Serialize())

	return nil
}

func (t *Task) shouldUseInvalidSignature() (bool, error) {
	if t.config.InvalidSigPercent <= 0 {
		return false, nil
	}

	var b [1]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return false, fmt.Errorf("failed to read random byte: %w", err)
	}

	return int(b[0])%100 < t.config.InvalidSigPercent, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
