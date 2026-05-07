package generatebatchdeposits

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	clientpool "github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/clients/consensus"
	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	batchcontract "github.com/ethpandaops/assertoor/pkg/tasks/generate_batch_deposits/batch_contract"
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
	util "github.com/wealdtech/go-eth2-util"
)

const (
	outputTypeArray  = "array"
	outputTypeString = "string"
	outputTypeNumber = "number"
)

var (
	TaskName       = "generate_batch_deposits"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates valid unique deposits and forwards them through a BatchDeposit contract in batchSize-sized transactions. Optionally deploys the contract first.",
		Category:    "validator",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "batchContract",
				Type:        outputTypeString,
				Description: "Address of the BatchDeposit forwarder contract used (deployed by this task if batchContract was empty).",
			},
			{
				Name:        "validatorPubkeys",
				Type:        outputTypeArray,
				Description: "Array of validator public keys for all generated deposits.",
			},
			{
				Name:        "batchTransactions",
				Type:        outputTypeArray,
				Description: "Array of batch transaction hashes (one per submitted batch).",
			},
			{
				Name:        "batchReceipts",
				Type:        outputTypeArray,
				Description: "Array of batch transaction receipts (when awaitReceipt is enabled).",
			},
			{
				Name:        "includedDeposits",
				Type:        outputTypeNumber,
				Description: "Number of deposits included on beacon chain (when awaitInclusion is enabled).",
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
	valkeySeed          []byte
	nextIndex           uint64
	lastIndex           uint64
	walletPrivKey       *ecdsa.PrivateKey
	depositContractAddr ethcommon.Address
	batchContractAddr   ethcommon.Address
	withdrawalCreds     []byte
}

// preparedDeposit holds a single fully-signed deposit ready to be packed into a batch.
type preparedDeposit struct {
	pubkey    common.BLSPubkey
	signature []byte // 96 bytes
	dataRoot  [32]byte
}

// runState carries mutable state through the batched generation loop.
type runState struct {
	totalDeposits    int
	depositsThisSlot int
	pubkeys          []string
	txHashes         []string
	receipts         map[string]*ethtypes.Receipt
	receiptsMtx      sync.Mutex
	pendingWg        sync.WaitGroup
	pendingChan      chan struct{}
	targetCount      int
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

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}

	err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars)
	if err != nil {
		return err
	}

	if valerr := config.Validate(); valerr != nil {
		return valerr
	}

	t.valkeySeed, err = mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
	}

	t.walletPrivKey, err = crypto.HexToECDSA(config.WalletPrivkey)
	if err != nil {
		return err
	}

	creds := ethcommon.FromHex(config.WithdrawalCredentials)
	if len(creds) != 32 {
		return fmt.Errorf("withdrawalCredentials must be 32 bytes, got %d", len(creds))
	}

	t.withdrawalCreds = creds
	t.config = config
	t.depositContractAddr = ethcommon.HexToAddress(config.DepositContract)

	if config.BatchContract != "" {
		t.batchContractAddr = ethcommon.HexToAddress(config.BatchContract)
	}

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex)
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount)
	}

	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	clients, err := t.selectClients(ctx, clientPool)
	if err != nil {
		return err
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	txWallet, err := walletMgr.GetWalletByPrivkey(t.ctx.Scheduler.GetTestRunCtx(), t.walletPrivKey)
	if err != nil {
		return fmt.Errorf("cannot initialize wallet: %w", err)
	}

	t.logger.Infof("wallet: %v [nonce: %v]  %v ETH", txWallet.GetAddress().Hex(), txWallet.GetNonce(), txWallet.GetReadableBalance(18, 0, 4, false, false))

	spamoorClients := make([]*spamoor.Client, len(clients))
	for i, c := range clients {
		spamoorClients[i] = walletMgr.GetClient(c)
	}

	if (t.batchContractAddr == ethcommon.Address{}) {
		t.ctx.ReportProgress(0, "Deploying BatchDeposit contract")

		addr, deployErr := t.deployBatchContract(ctx, txWallet, spamoorClients)
		if deployErr != nil {
			return fmt.Errorf("failed to deploy batch contract: %w", deployErr)
		}

		t.batchContractAddr = addr
		t.logger.Infof("deployed BatchDeposit contract at %s", addr.Hex())
	} else {
		t.logger.Infof("using existing BatchDeposit contract at %s", t.batchContractAddr.Hex())
	}

	t.ctx.Outputs.SetVar("batchContract", t.batchContractAddr.Hex())

	var inclusionSubscription *consensus.Subscription[*consensus.Block]

	if t.config.AwaitInclusion {
		inclusionSubscription = clientPool.GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer inclusionSubscription.Unsubscribe()
	}

	var slotSubscription *consensus.Subscription[*consensus.Block]

	if t.config.LimitPerSlot > 0 {
		slotSubscription = clientPool.GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer slotSubscription.Unsubscribe()
	}

	bound, err := batchcontract.NewBatchDepositContract(t.batchContractAddr, clients[0].GetRPCClient().GetEthClient())
	if err != nil {
		return fmt.Errorf("cannot bind BatchDeposit contract: %w", err)
	}

	state := t.newRunState()

	if err := t.runBatchLoop(ctx, txWallet, bound, spamoorClients, slotSubscription, state); err != nil {
		return err
	}

	if t.config.AwaitReceipt {
		state.pendingWg.Wait()
	}

	t.publishOutputs(state)

	t.ctx.ReportProgress(100, fmt.Sprintf("Submitted %d deposits across %d batches", state.totalDeposits, len(state.txHashes)))

	if t.config.FailOnReject && t.checkReceiptFailures(state) {
		return nil
	}

	if t.config.AwaitInclusion && len(state.pubkeys) > 0 {
		return t.awaitInclusion(ctx, inclusionSubscription, state.pubkeys)
	}

	return nil
}

func (t *Task) newRunState() *runState {
	state := &runState{
		pubkeys:  []string{},
		txHashes: []string{},
		receipts: map[string]*ethtypes.Receipt{},
	}

	if t.config.LimitTotal > 0 {
		state.targetCount = t.config.LimitTotal
	} else if t.lastIndex > t.nextIndex {
		state.targetCount = int(t.lastIndex - t.nextIndex) //nolint:gosec // bounded by config
	}

	if t.config.LimitPendingBatches > 0 {
		state.pendingChan = make(chan struct{}, t.config.LimitPendingBatches)
	}

	return state
}

func (t *Task) runBatchLoop(
	ctx context.Context,
	txWallet *spamoor.Wallet,
	bound *batchcontract.BatchDepositContract,
	spamoorClients []*spamoor.Client,
	slotSub *consensus.Subscription[*consensus.Block],
	state *runState,
) error {
	t.ctx.ReportProgress(0, "Starting batched deposit generation")

	for {
		batchSize := t.nextBatchSize(state.totalDeposits)
		if batchSize <= 0 {
			return nil
		}

		if t.config.LimitPerSlot > 0 && state.depositsThisSlot+batchSize > t.config.LimitPerSlot {
			state.depositsThisSlot = 0

			select {
			case <-ctx.Done():
				return nil
			case <-slotSub.Channel():
			}
		}

		prepared, pubkeys, prepErr := t.prepareBatch(batchSize)
		if prepErr != nil {
			return fmt.Errorf("failed to prepare deposit batch: %w", prepErr)
		}

		if state.pendingChan != nil {
			select {
			case <-ctx.Done():
				return nil
			case state.pendingChan <- struct{}{}:
			}
		}

		state.pendingWg.Add(1)

		tx, sendErr := t.submitBatch(ctx, txWallet, bound, spamoorClients, prepared, t.makeOnComplete(state, len(prepared)))
		if sendErr != nil {
			state.pendingWg.Done()

			if state.pendingChan != nil {
				<-state.pendingChan
			}

			return fmt.Errorf("failed sending batch tx: %w", sendErr)
		}

		state.txHashes = append(state.txHashes, tx.Hash().Hex())
		state.pubkeys = append(state.pubkeys, pubkeys...)
		state.totalDeposits += len(prepared)
		state.depositsThisSlot += len(prepared)

		t.reportLoopProgress(state)

		if ctx.Err() != nil {
			return nil
		}

		if t.lastIndex > 0 && t.nextIndex >= t.lastIndex {
			return nil
		}

		if t.config.LimitTotal > 0 && state.totalDeposits >= t.config.LimitTotal {
			return nil
		}
	}
}

// nextBatchSize returns the largest batch we can still submit, capped by the
// configured BatchSize and remaining limits. Returns 0 (or less) when we are done.
func (t *Task) nextBatchSize(totalDeposits int) int {
	batchSize := t.config.BatchSize

	if t.config.LimitTotal > 0 {
		remaining := t.config.LimitTotal - totalDeposits
		if remaining < batchSize {
			batchSize = remaining
		}
	}

	if t.lastIndex > 0 {
		if t.nextIndex >= t.lastIndex {
			return 0
		}

		idxRemaining := t.lastIndex - t.nextIndex
		if batchSize < 0 || idxRemaining < uint64(batchSize) {
			batchSize = int(idxRemaining) //nolint:gosec // bounded by remaining indices < batchSize
		}
	}

	return batchSize
}

func (t *Task) reportLoopProgress(state *runState) {
	if state.targetCount > 0 {
		progress := float64(state.totalDeposits) / float64(state.targetCount) * 100
		t.ctx.ReportProgress(progress, fmt.Sprintf("Submitted %d/%d deposits across %d batches", state.totalDeposits, state.targetCount, len(state.txHashes)))

		return
	}

	t.ctx.ReportProgress(0, fmt.Sprintf("Submitted %d deposits across %d batches", state.totalDeposits, len(state.txHashes)))
}

func (t *Task) makeOnComplete(state *runState, depositCount int) spamoor.TxCompleteFn {
	return func(tx *ethtypes.Transaction, receipt *ethtypes.Receipt, err error) {
		if state.pendingChan != nil {
			<-state.pendingChan
		}

		state.receiptsMtx.Lock()
		state.receipts[tx.Hash().Hex()] = receipt
		state.receiptsMtx.Unlock()

		switch {
		case receipt != nil:
			t.logger.Infof("batch tx %v confirmed in block %v (nonce: %v, status: %v, deposits: %d)",
				tx.Hash().Hex(), receipt.BlockNumber, tx.Nonce(), receipt.Status, depositCount)
		case err != nil:
			t.logger.Errorf("error awaiting batch tx receipt: %v", err)
		default:
			t.logger.Warnf("no receipt for batch tx: %v", tx.Hash().Hex())
		}

		state.pendingWg.Done()
	}
}

func (t *Task) publishOutputs(state *runState) {
	t.ctx.Outputs.SetVar("validatorPubkeys", state.pubkeys)
	t.ctx.Outputs.SetVar("batchTransactions", state.txHashes)

	receiptList := make([]interface{}, 0, len(state.txHashes))

	for _, txhash := range state.txHashes {
		receipt := state.receipts[txhash]
		if receipt == nil {
			receiptList = append(receiptList, nil)
			continue
		}

		var receiptMap map[string]interface{}

		receiptJSON, jerr := json.Marshal(receipt)
		if jerr != nil {
			t.logger.Errorf("could not marshal batch tx receipt: %v", jerr)

			receiptList = append(receiptList, nil)

			continue
		}

		receiptMap = map[string]interface{}{}

		if uerr := json.Unmarshal(receiptJSON, &receiptMap); uerr != nil {
			t.logger.Errorf("could not unmarshal batch tx receipt: %v", uerr)

			receiptList = append(receiptList, nil)

			continue
		}

		receiptList = append(receiptList, receiptMap)
	}

	t.ctx.Outputs.SetVar("batchReceipts", receiptList)
}

// checkReceiptFailures returns true if a failure was found and the task result was set.
func (t *Task) checkReceiptFailures(state *runState) bool {
	for _, txhash := range state.txHashes {
		receipt := state.receipts[txhash]

		if receipt == nil {
			t.logger.Errorf("no receipt for batch tx: %v", txhash)
			t.ctx.SetResult(types.TaskResultFailure)

			return true
		}

		if receipt.Status == 0 {
			t.logger.Errorf("batch tx failed: %v", txhash)
			t.ctx.SetResult(types.TaskResultFailure)

			return true
		}
	}

	return false
}

func (t *Task) selectClients(ctx context.Context, clientPool *clientpool.ClientPool) ([]*execution.Client, error) {
	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		clients := clientPool.GetExecutionPool().AwaitReadyEndpoints(ctx, true)
		if len(clients) == 0 {
			return nil, ctx.Err()
		}

		return clients, nil
	}

	poolClients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
	if len(poolClients) == 0 {
		return nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
	}

	out := make([]*execution.Client, len(poolClients))
	for i, c := range poolClients {
		out[i] = c.ExecutionClient
	}

	return out, nil
}

// deployBatchContract deploys a fresh BatchDeposit contract bound to the configured deposit contract
// and waits for the receipt before returning the deployed address.
func (t *Task) deployBatchContract(ctx context.Context, txWallet *spamoor.Wallet, spamoorClients []*spamoor.Client) (ethcommon.Address, error) {
	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	parsed, err := batchcontract.BatchDepositContractMetaData.GetAbi()
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("cannot parse abi: %w", err)
	}

	bytecode := ethcommon.FromHex(batchcontract.BatchDepositContractMetaData.Bin)

	constructorArgs, err := parsed.Pack("", t.depositContractAddr)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("cannot pack constructor args: %w", err)
	}

	deployData := make([]byte, 0, len(bytecode)+len(constructorArgs))
	deployData = append(deployData, bytecode...)
	deployData = append(deployData, constructorArgs...)

	txData, err := txbuilder.DynFeeTx(&txbuilder.TxMetadata{
		GasFeeCap: uint256.MustFromBig(big.NewInt(t.config.DepositTxFeeCap)),
		GasTipCap: uint256.MustFromBig(big.NewInt(t.config.DepositTxTipCap)),
		Gas:       2_000_000,
		To:        nil,
		Value:     uint256.NewInt(0),
		Data:      deployData,
	})
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("cannot build deploy tx: %w", err)
	}

	tx, err := txWallet.BuildDynamicFeeTx(txData)
	if err != nil {
		return ethcommon.Address{}, fmt.Errorf("cannot sign deploy tx: %w", err)
	}

	receipt, err := walletMgr.GetTxPool().SendAndAwaitTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      spamoorClients[0],
		ClientList:  spamoorClients,
		SubmitCount: len(spamoorClients),
		Rebroadcast: true,
		LogFn: func(client *spamoor.Client, retry int, rebroadcast int, err error) {
			if err != nil {
				return
			}

			logEntry := t.logger.WithField("client", client.GetName())
			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted batch contract deploy tx (nonce: %v, attempt: %v)", tx.Nonce(), retry)
		},
	})
	if err != nil {
		txWallet.MarkSkippedNonce(tx.Nonce())
		return ethcommon.Address{}, fmt.Errorf("deploy tx failed: %w", err)
	}

	if receipt == nil {
		return ethcommon.Address{}, errors.New("deploy tx receipt was nil")
	}

	if receipt.Status == 0 {
		return ethcommon.Address{}, fmt.Errorf("deploy tx reverted (hash %s)", tx.Hash().Hex())
	}

	if (receipt.ContractAddress == ethcommon.Address{}) {
		return ethcommon.Address{}, errors.New("deploy receipt has no contract address")
	}

	return receipt.ContractAddress, nil
}

// prepareBatch deterministically generates `count` deposits, each with a unique BLS keypair
// derived from the configured mnemonic. The signatures are produced using the genesis fork
// version so the consensus layer accepts them as valid, forcing a real (worst-case) BLS
// verification per deposit.
func (t *Task) prepareBatch(count int) ([]preparedDeposit, []string, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()

	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	if genesis == nil {
		return nil, nil, errors.New("consensus genesis info not available yet")
	}

	domain := common.ComputeDomain(common.DOMAIN_DEPOSIT, common.Version(genesis.GenesisForkVersion), common.Root{})

	depositAmountGwei := new(big.Int).SetUint64(t.config.DepositAmount)
	depositAmountGwei.Mul(depositAmountGwei, big.NewInt(1_000_000_000))

	if !depositAmountGwei.IsUint64() {
		return nil, nil, fmt.Errorf("deposit amount too large: %v ETH", t.config.DepositAmount)
	}

	prepared := make([]preparedDeposit, 0, count)
	pubkeyStrs := make([]string, 0, count)

	for i := 0; i < count; i++ {
		accountIdx := t.nextIndex
		t.nextIndex++

		pd, err := t.prepareSingle(accountIdx, domain, depositAmountGwei.Uint64())
		if err != nil {
			return nil, nil, err
		}

		prepared = append(prepared, pd)
		pubkeyStrs = append(pubkeyStrs, pd.pubkey.String())
	}

	return prepared, pubkeyStrs, nil
}

func (t *Task) prepareSingle(accountIdx uint64, domain common.BLSDomain, amountGwei uint64) (preparedDeposit, error) {
	keyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPriv, err := util.PrivateKeyFromSeedAndPath(t.valkeySeed, keyPath)
	if err != nil {
		return preparedDeposit{}, fmt.Errorf("failed generating validator key %v: %w", keyPath, err)
	}

	pub := common.BLSPubkey{}
	copy(pub[:], validatorPriv.PublicKey().Marshal())

	depositData := common.DepositData{
		Pubkey:                pub,
		WithdrawalCredentials: tree.Root(t.withdrawalCreds),
		Amount:                common.Gwei(amountGwei),
		Signature:             common.BLSSignature{},
	}

	msgRoot := depositData.ToMessage().HashTreeRoot(tree.GetHashFn())
	signingRoot := common.ComputeSigningRoot(msgRoot, domain)

	var secKey hbls.SecretKey
	if err := secKey.Deserialize(validatorPriv.Marshal()); err != nil {
		return preparedDeposit{}, fmt.Errorf("cannot convert validator priv key: %w", err)
	}

	sig := secKey.SignHash(signingRoot[:])
	copy(depositData.Signature[:], sig.Serialize())

	dataRoot := depositData.HashTreeRoot(tree.GetHashFn())

	pd := preparedDeposit{
		pubkey:    pub,
		signature: depositData.Signature[:],
	}
	copy(pd.dataRoot[:], dataRoot[:])

	return pd, nil
}

// submitBatch packs the prepared deposits into a single batchDeposit() call and sends the tx.
func (t *Task) submitBatch(
	ctx context.Context,
	txWallet *spamoor.Wallet,
	bound *batchcontract.BatchDepositContract,
	spamoorClients []*spamoor.Client,
	prepared []preparedDeposit,
	onComplete spamoor.TxCompleteFn,
) (*ethtypes.Transaction, error) {
	count := len(prepared)

	pubkeys := make([]byte, 0, count*48)
	signatures := make([]byte, 0, count*96)
	dataRoots := make([][32]byte, 0, count)

	for i := range prepared {
		pubkeys = append(pubkeys, prepared[i].pubkey[:]...)
		signatures = append(signatures, prepared[i].signature...)
		dataRoots = append(dataRoots, prepared[i].dataRoot)
	}

	amountWei := new(big.Int).SetUint64(t.config.DepositAmount)
	amountWei.Mul(amountWei, big.NewInt(1_000_000_000_000_000_000))

	totalValue := new(big.Int).Mul(amountWei, big.NewInt(int64(count)))

	txMeta := &txbuilder.TxMetadata{
		GasFeeCap: uint256.MustFromBig(big.NewInt(t.config.DepositTxFeeCap)),
		GasTipCap: uint256.MustFromBig(big.NewInt(t.config.DepositTxTipCap)),
		Gas:       t.config.BatchTxGasLimit,
		Value:     uint256.MustFromBig(totalValue),
	}

	tx, err := txWallet.BuildBoundTx(ctx, txMeta, func(opts *bind.TransactOpts) (*ethtypes.Transaction, error) {
		return bound.BatchDeposit(opts, pubkeys, signatures, dataRoots, t.withdrawalCreds, amountWei)
	})
	if err != nil {
		return nil, fmt.Errorf("cannot build batch tx: %w", err)
	}

	walletMgr := t.ctx.Scheduler.GetServices().WalletManager()

	err = walletMgr.GetTxPool().SendTransaction(ctx, txWallet, tx, &spamoor.SendTransactionOptions{
		Client:      spamoorClients[0],
		ClientList:  spamoorClients,
		SubmitCount: len(spamoorClients),
		Rebroadcast: true,
		OnComplete:  onComplete,
		LogFn: func(client *spamoor.Client, retry int, rebroadcast int, err error) {
			if err != nil {
				return
			}

			logEntry := t.logger.WithFields(logrus.Fields{
				"client": client.GetName(),
				"size":   count,
			})

			if rebroadcast > 0 {
				logEntry = logEntry.WithField("rebroadcast", rebroadcast)
			}

			logEntry.Infof("submitted batch deposit tx (nonce: %v, attempt: %v)", tx.Nonce(), retry)
		},
	})
	if err != nil {
		txWallet.MarkSkippedNonce(tx.Nonce())
		return nil, err
	}

	return tx, nil
}

func (t *Task) awaitInclusion(ctx context.Context, sub *consensus.Subscription[*consensus.Block], pubkeys []string) error {
	pending := make(map[string]bool, len(pubkeys))
	for _, pk := range pubkeys {
		pending[pk] = true
	}

	includedCount := 0
	t.ctx.Outputs.SetVar("includedDeposits", includedCount)

	total := len(pending)
	t.logger.Infof("waiting for %d deposits to be included in beacon blocks", total)
	t.ctx.ReportProgress(50, fmt.Sprintf("Awaiting inclusion: 0/%d", total))

	for len(pending) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case block := <-sub.Channel():
			t.scanBlockForDeposits(ctx, block, pending, &includedCount)

			t.ctx.Outputs.SetVar("includedDeposits", includedCount)

			if includedCount > 0 {
				progress := 50 + (float64(includedCount)/float64(total))*50
				t.ctx.ReportProgress(progress, fmt.Sprintf("Awaiting inclusion: %d/%d", includedCount, total))

				t.logger.Infof("deposits included in block %d (%d/%d)", block.Slot, includedCount, total)
			}
		}
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	t.ctx.ReportProgress(100, fmt.Sprintf("All %d deposits included on beacon chain", total))

	return nil
}

// scanBlockForDeposits checks a beacon block (and its execution payload envelope on Gloas+)
// for deposit pubkeys we are waiting on, removing matches from `pending` and bumping `included`.
func (t *Task) scanBlockForDeposits(ctx context.Context, block *consensus.Block, pending map[string]bool, included *int) {
	blockData := block.AwaitBlock(ctx, 2*time.Second)
	if blockData == nil {
		return
	}

	if deposits, err := blockData.Deposits(); err == nil {
		for _, deposit := range deposits {
			pubkeyStr := deposit.Data.PublicKey.String()
			if pending[pubkeyStr] {
				delete(pending, pubkeyStr)

				*included++
			}
		}
	}

	if blockData.Version >= spec.DataVersionGloas {
		payload := block.AwaitPayload(ctx, 2*time.Second)
		if payload != nil {
			payloadData := payload.Gloas
			if payloadData != nil && payloadData.Message.ExecutionRequests != nil {
				for _, depositReq := range payloadData.Message.ExecutionRequests.Deposits {
					pubkeyStr := depositReq.Pubkey.String()
					if pending[pubkeyStr] {
						delete(pending, pubkeyStr)

						*included++
					}
				}
			}
		}

		return
	}

	execRequests, err := blockData.ExecutionRequests()
	if err != nil || execRequests == nil {
		return
	}

	for _, depositReq := range execRequests.Deposits {
		pubkeyStr := depositReq.Pubkey.String()
		if pending[pubkeyStr] {
			delete(pending, pubkeyStr)

			*included++
		}
	}
}

func mnemonicToSeed(mnemonic string) ([]byte, error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
