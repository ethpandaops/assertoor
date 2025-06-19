package generateblschanges

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/erigontech/assertoor/pkg/coordinator/clients/consensus"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/vars"
	hbls "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/ztyp/tree"
	"github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	util "github.com/wealdtech/go-eth2-util"
)

var (
	TaskName       = "generate_bls_changes"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Generates bls changes and sends them to the network",
		Config:      DefaultConfig(),
		NewTask:     NewTask,
	}
)

type Task struct {
	ctx        *types.TaskContext
	options    *types.TaskOptions
	config     Config
	logger     logrus.FieldLogger
	withdrSeed []byte
	nextIndex  uint64
	lastIndex  uint64
	targetAddr common.Eth1Address
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

	t.withdrSeed, err = t.mnemonicToSeed(config.Mnemonic)
	if err != nil {
		return err
	}

	err = t.targetAddr.UnmarshalText([]byte(config.TargetAddress))
	if err != nil {
		return fmt.Errorf("cannot decode execution addr: %w", err)
	}

	t.config = config

	return nil
}

func (t *Task) Execute(ctx context.Context) error {
	if t.config.StartIndex > 0 {
		t.nextIndex = uint64(t.config.StartIndex) //nolint:gosec // no overflow possible
	}

	if t.config.IndexCount > 0 {
		t.lastIndex = t.nextIndex + uint64(t.config.IndexCount) //nolint:gosec // no overflow possible
	}

	var subscription *consensus.Subscription[*consensus.Block]
	if t.config.LimitPerSlot > 0 {
		subscription = t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetBlockCache().SubscribeBlockEvent(10)
		defer subscription.Unsubscribe()
	}

	perSlotCount := 0
	totalCount := 0
	blsChangesList := []interface{}{}

	for {
		accountIdx := t.nextIndex
		t.nextIndex++

		blsChange, err := t.generateBlsChange(ctx, accountIdx)
		if err != nil {
			t.logger.Errorf("error generating bls change: %v", err.Error())
		} else {
			blsChangesList = append(blsChangesList, blsChange)
			t.ctx.Outputs.SetVar("blsChanges", blsChangesList)
			t.ctx.SetResult(types.TaskResultSuccess)

			perSlotCount++
			totalCount++
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
		} else if err := ctx.Err(); err != nil {
			return err
		}
	}

	t.ctx.Outputs.SetVar("blsChanges", blsChangesList)

	return nil
}

func (t *Task) generateBlsChange(ctx context.Context, accountIdx uint64) (interface{}, error) {
	clientPool := t.ctx.Scheduler.GetServices().ClientPool()
	validatorKeyPath := fmt.Sprintf("m/12381/3600/%d/0/0", accountIdx)

	validatorPrivkey, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, validatorKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed generating validator key %v: %w", validatorKeyPath, err)
	}

	validatorSet := t.ctx.Scheduler.GetServices().ClientPool().GetConsensusPool().GetValidatorSet()

	var validator *v1.Validator

	validatorPubkey := validatorPrivkey.PublicKey().Marshal()

	t.logger.Debugf("generated validator pubkey %v: 0x%x", validatorKeyPath, validatorPubkey)

	for _, val := range validatorSet {
		if bytes.Equal(val.Validator.PublicKey[:], validatorPubkey) {
			validator = val
			break
		}
	}

	if validator == nil {
		return nil, fmt.Errorf("validator not found")
	}

	if validator.Validator.WithdrawalCredentials[0] != 0x00 {
		return nil, fmt.Errorf("validator %v does not have 0x00 withdrawal creds", validator.Index)
	}

	withdrAccPath := fmt.Sprintf("m/12381/3600/%d/0", accountIdx)

	withdr, err := util.PrivateKeyFromSeedAndPath(t.withdrSeed, withdrAccPath)
	if err != nil {
		return nil, fmt.Errorf("failed generating key %v: %w", withdrAccPath, err)
	}

	var withdrPub common.BLSPubkey

	copy(withdrPub[:], withdr.PublicKey().Marshal())

	t.logger.Debugf("generated withdrawal pubkey %v: 0x%x", withdrAccPath, withdrPub[:])

	msg := common.BLSToExecutionChange{
		ValidatorIndex:     common.ValidatorIndex(validator.Index),
		FromBLSPubKey:      withdrPub,
		ToExecutionAddress: t.targetAddr,
	}

	var secKey hbls.SecretKey

	err = secKey.Deserialize(withdr.Marshal())
	if err != nil {
		return nil, fmt.Errorf("failed converting validator priv key: %w", err)
	}

	msgRoot := msg.HashTreeRoot(tree.GetHashFn())
	genesis := clientPool.GetConsensusPool().GetBlockCache().GetGenesis()
	dom := common.ComputeDomain(common.DOMAIN_BLS_TO_EXECUTION_CHANGE, common.Version(genesis.GenesisForkVersion), tree.Root(genesis.GenesisValidatorsRoot))
	signingRoot := common.ComputeSigningRoot(msgRoot, dom)
	sig := secKey.SignHash(signingRoot[:])

	var signedMsg common.SignedBLSToExecutionChange

	signedMsg.BLSToExecutionChange = msg
	copy(signedMsg.Signature[:], sig.Serialize())

	signedChangeJSON, err := json.Marshal(&signedMsg)
	if err != nil {
		return nil, err
	}

	var blsChangeRes interface{}
	if blsc, err2 := vars.GeneralizeData(signedMsg); err2 == nil {
		blsChangeRes = blsc
	} else {
		t.logger.Warnf("Failed encoding blschange output: %v", err2)
	}

	t.ctx.Outputs.SetVar("latestBlsChange", blsChangeRes)

	var client *consensus.Client

	if t.config.ClientPattern == "" && t.config.ExcludeClientPattern == "" {
		client = clientPool.GetConsensusPool().AwaitReadyEndpoint(ctx, consensus.AnyClient)
		if client == nil {
			return nil, ctx.Err()
		}
	} else {
		clients := clientPool.GetClientsByNamePatterns(t.config.ClientPattern, t.config.ExcludeClientPattern)
		if len(clients) == 0 {
			return nil, fmt.Errorf("no client found with pattern %v", t.config.ClientPattern)
		}

		client = clients[0].ConsensusClient
	}

	blsChange := &capella.SignedBLSToExecutionChange{}

	err = json.Unmarshal(signedChangeJSON, blsChange)
	if err != nil {
		return nil, err
	}

	t.logger.WithField("client", client.GetName()).Infof("sending bls change for validator %v", validator.Index)

	err = client.GetRPCClient().SubmitBLSToExecutionChanges(ctx, []*capella.SignedBLSToExecutionChange{blsChange})
	if err != nil {
		return nil, err
	}

	return blsChangeRes, nil
}

func (t *Task) mnemonicToSeed(mnemonic string) (seed []byte, err error) {
	mnemonic = strings.TrimSpace(mnemonic)
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, errors.New("mnemonic is not valid")
	}

	return bip39.NewSeed(mnemonic, ""), nil
}
