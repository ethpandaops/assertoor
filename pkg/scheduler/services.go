package scheduler

import (
	"github.com/ethpandaops/assertoor/pkg/clients"
	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/events"
	"github.com/ethpandaops/assertoor/pkg/names"
	"github.com/ethpandaops/assertoor/pkg/txmgr"
	"github.com/ethpandaops/assertoor/pkg/types"
)

type servicesProvider struct {
	database       *db.Database
	clientPool     *clients.ClientPool
	walletManager  *txmgr.Spamoor
	validatorNames *names.ValidatorNames
	eventBus       *events.EventBus
}

func NewServicesProvider(database *db.Database, clientPool *clients.ClientPool, walletManager *txmgr.Spamoor, validatorNames *names.ValidatorNames, eventBus *events.EventBus) types.TaskServices {
	return &servicesProvider{
		database:       database,
		clientPool:     clientPool,
		walletManager:  walletManager,
		validatorNames: validatorNames,
		eventBus:       eventBus,
	}
}

func (p *servicesProvider) Database() *db.Database {
	return p.database
}

func (p *servicesProvider) ClientPool() *clients.ClientPool {
	return p.clientPool
}

func (p *servicesProvider) WalletManager() *txmgr.Spamoor {
	return p.walletManager
}

func (p *servicesProvider) ValidatorNames() *names.ValidatorNames {
	return p.validatorNames
}

func (p *servicesProvider) EventBus() *events.EventBus {
	return p.eventBus
}
