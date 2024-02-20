package scheduler

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
)

type servicesProvider struct {
	clientPool     *clients.ClientPool
	walletManager  *wallet.Manager
	validatorNames *names.ValidatorNames
}

func NewServicesProvider(clientPool *clients.ClientPool, walletManager *wallet.Manager, validatorNames *names.ValidatorNames) types.TaskServices {
	return &servicesProvider{
		clientPool:     clientPool,
		walletManager:  walletManager,
		validatorNames: validatorNames,
	}
}

func (p *servicesProvider) ClientPool() *clients.ClientPool {
	return p.clientPool
}

func (p *servicesProvider) WalletManager() *wallet.Manager {
	return p.walletManager
}

func (p *servicesProvider) ValidatorNames() *names.ValidatorNames {
	return p.validatorNames
}
