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

func (bc *BeaconClient) GetGenesis() (*v1.Genesis, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
