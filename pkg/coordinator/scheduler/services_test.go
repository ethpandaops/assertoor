package scheduler

import (
	"testing"

	"github.com/ethpandaops/assertoor/pkg/coordinator/clients"
	"github.com/ethpandaops/assertoor/pkg/coordinator/names"
	"github.com/ethpandaops/assertoor/pkg/coordinator/wallet"
	"github.com/stretchr/testify/assert"
)

func TestNewServicesProvider(t *testing.T) {
	clientPool := &clients.ClientPool{}
	walletManager := &wallet.Manager{}
	validatorNames := &names.ValidatorNames{}

	services := NewServicesProvider(clientPool, walletManager, validatorNames)

	assert.NotNil(t, services)
	assert.Equal(t, clientPool, services.ClientPool())
	assert.Equal(t, walletManager, services.WalletManager())
	assert.Equal(t, validatorNames, services.ValidatorNames())
}

func TestServicesProvider_ClientPool(t *testing.T) {
	clientPool := &clients.ClientPool{}
	services := NewServicesProvider(clientPool, nil, nil)

	assert.Equal(t, clientPool, services.ClientPool())
}

func TestServicesProvider_WalletManager(t *testing.T) {
	walletManager := &wallet.Manager{}
	services := NewServicesProvider(nil, walletManager, nil)

	assert.Equal(t, walletManager, services.WalletManager())
}

func TestServicesProvider_ValidatorNames(t *testing.T) {
	validatorNames := &names.ValidatorNames{}
	services := NewServicesProvider(nil, nil, validatorNames)

	assert.Equal(t, validatorNames, services.ValidatorNames())
}
