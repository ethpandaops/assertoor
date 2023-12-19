package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	eth2client "github.com/attestantio/go-eth2-client"
	v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/http"
	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/rs/zerolog"
	"github.com/sirupsen/logrus"
)

var logger = logrus.StandardLogger().WithField("module", "rpc")

type BeaconClient struct {
	name      string
	endpoint  string
	headers   map[string]string
	clientSvc eth2client.Service
}

// NewBeaconClient is used to create a new beacon client
func NewBeaconClient(name, url string, headers map[string]string) (*BeaconClient, error) {
	client := &BeaconClient{
		name:     name,
		endpoint: url,
		headers:  headers,
	}

	return client, nil
}

func (bc *BeaconClient) Initialize(ctx context.Context) error {
	if bc.clientSvc != nil {
		return nil
	}

	cliParams := []http.Parameter{
		http.WithAddress(bc.endpoint),
		http.WithTimeout(10 * time.Minute),
		http.WithLogLevel(zerolog.Disabled),
		// TODO (when upstream PR is merged)
		// http.WithConnectionCheck(false),
	}

	// set extra endpoint headers
	if bc.headers != nil && len(bc.headers) > 0 {
		cliParams = append(cliParams, http.WithExtraHeaders(bc.headers))
	}

	clientSvc, err := http.New(ctx, cliParams...)
	if err != nil {
		return err
	}

	bc.clientSvc = clientSvc

	return nil
}

func (bc *BeaconClient) GetGenesis(ctx context.Context) (*v1.Genesis, error) {
	provider, isProvider := bc.clientSvc.(eth2client.GenesisProvider)
	if !isProvider {
		return nil, fmt.Errorf("get genesis not supported")
	}

	result, err := provider.Genesis(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetNodeSyncing(ctx context.Context) (*v1.SyncState, error) {
	provider, isProvider := bc.clientSvc.(eth2client.NodeSyncingProvider)
	if !isProvider {
		return nil, fmt.Errorf("get node syncing not supported")
	}

	result, err := provider.NodeSyncing(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetNodeSyncStatus(ctx context.Context) (*SyncStatus, error) {
	syncState, err := bc.GetNodeSyncing(ctx)
	if err != nil {
		return nil, err
	}

	syncStatus := NewSyncStatus(syncState)

	return &syncStatus, nil
}

func (bc *BeaconClient) GetNodeVersion(ctx context.Context) (string, error) {
	provider, isProvider := bc.clientSvc.(eth2client.NodeVersionProvider)
	if !isProvider {
		return "", fmt.Errorf("get node version not supported")
	}

	result, err := provider.NodeVersion(ctx)
	if err != nil {
		return "", err
	}

	return result, nil
}

func (bc *BeaconClient) GetConfigSpecs(ctx context.Context) (map[string]interface{}, error) {
	provider, isProvider := bc.clientSvc.(eth2client.SpecProvider)
	if !isProvider {
		return nil, fmt.Errorf("get specs not supported")
	}

	result, err := provider.Spec(ctx)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetLatestBlockHead(ctx context.Context) (*v1.BeaconBlockHeader, error) {
	provider, isProvider := bc.clientSvc.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon block headers not supported")
	}

	result, err := provider.BeaconBlockHeader(ctx, "head")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetFinalityCheckpoints(ctx context.Context) (*v1.Finality, error) {
	provider, isProvider := bc.clientSvc.(eth2client.FinalityProvider)
	if !isProvider {
		return nil, fmt.Errorf("get finality not supported")
	}

	result, err := provider.Finality(ctx, "head")
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetBlockHeaderByBlockroot(ctx context.Context, blockroot phase0.Root) (*v1.BeaconBlockHeader, error) {
	provider, isProvider := bc.clientSvc.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon block headers not supported")
	}

	result, err := provider.BeaconBlockHeader(ctx, fmt.Sprintf("0x%x", blockroot))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetBlockHeaderBySlot(ctx context.Context, slot phase0.Slot) (*v1.BeaconBlockHeader, error) {
	provider, isProvider := bc.clientSvc.(eth2client.BeaconBlockHeadersProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon block headers not supported")
	}

	result, err := provider.BeaconBlockHeader(ctx, fmt.Sprintf("%d", slot))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetBlockBodyByBlockroot(ctx context.Context, blockroot phase0.Root) (*spec.VersionedSignedBeaconBlock, error) {
	provider, isProvider := bc.clientSvc.(eth2client.SignedBeaconBlockProvider)
	if !isProvider {
		return nil, fmt.Errorf("get signed beacon block not supported")
	}

	result, err := provider.SignedBeaconBlock(ctx, fmt.Sprintf("0x%x", blockroot))
	if err != nil {
		if strings.HasPrefix(err.Error(), "GET failed with status 404") {
			return nil, nil
		}

		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetState(ctx context.Context, stateRef string) (*spec.VersionedBeaconState, error) {
	provider, isProvider := bc.clientSvc.(eth2client.BeaconStateProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon state not supported")
	}

	result, err := provider.BeaconState(ctx, stateRef)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetStateValidators(ctx context.Context, stateRef string) (map[phase0.ValidatorIndex]*v1.Validator, error) {
	provider, isProvider := bc.clientSvc.(eth2client.ValidatorsProvider)
	if !isProvider {
		return nil, fmt.Errorf("get validators not supported")
	}

	result, err := provider.Validators(ctx, stateRef, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetProposerDuties(ctx context.Context, epoch uint64) ([]*v1.ProposerDuty, error) {
	provider, isProvider := bc.clientSvc.(eth2client.ProposerDutiesProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon committees not supported")
	}

	result, err := provider.ProposerDuties(ctx, phase0.Epoch(epoch), nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetCommitteeDuties(ctx context.Context, stateRef string, epoch uint64) ([]*v1.BeaconCommittee, error) {
	provider, isProvider := bc.clientSvc.(eth2client.BeaconCommitteesProvider)
	if !isProvider {
		return nil, fmt.Errorf("get beacon committees not supported")
	}

	result, err := provider.BeaconCommitteesAtEpoch(ctx, stateRef, phase0.Epoch(epoch))
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) GetForkState(ctx context.Context, stateRef string) (*phase0.Fork, error) {
	provider, isProvider := bc.clientSvc.(eth2client.ForkProvider)
	if !isProvider {
		return nil, fmt.Errorf("get fork not supported")
	}

	result, err := provider.Fork(ctx, stateRef)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (bc *BeaconClient) SubmitBLSToExecutionChanges(ctx context.Context, blsChanges []*capella.SignedBLSToExecutionChange) error {
	submitter, isOk := bc.clientSvc.(eth2client.BLSToExecutionChangesSubmitter)
	if !isOk {
		return fmt.Errorf("submit bls to execution changes not supported")
	}

	err := submitter.SubmitBLSToExecutionChanges(ctx, blsChanges)
	if err != nil {
		return err
	}

	return nil
}

func (bc *BeaconClient) SubmitVoluntaryExits(ctx context.Context, exit *phase0.SignedVoluntaryExit) error {
	submitter, isOk := bc.clientSvc.(eth2client.VoluntaryExitSubmitter)
	if !isOk {
		return fmt.Errorf("submit voluntary exit not supported")
	}

	err := submitter.SubmitVoluntaryExit(ctx, exit)
	if err != nil {
		return err
	}

	return nil
}
