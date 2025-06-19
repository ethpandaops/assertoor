package scheduler

import (
	"github.com/erigontech/assertoor/pkg/coordinator/clients"
	"github.com/erigontech/assertoor/pkg/coordinator/db"
	"github.com/erigontech/assertoor/pkg/coordinator/names"
	"github.com/erigontech/assertoor/pkg/coordinator/types"
	"github.com/erigontech/assertoor/pkg/coordinator/wallet"
)

type servicesProvider struct {
	database       *db.Database
	clientPool     *clients.ClientPool
	walletManager  *wallet.Manager
	validatorNames *names.ValidatorNames
}

func NewServicesProvider(database *db.Database, clientPool *clients.ClientPool, walletManager *wallet.Manager, validatorNames *names.ValidatorNames) types.TaskServices {
	return &servicesProvider{
		database:       database,
		clientPool:     clientPool,
		walletManager:  walletManager,
		validatorNames: validatorNames,
	}
}

func (p *servicesProvider) Database() *db.Database {
	return p.database
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
