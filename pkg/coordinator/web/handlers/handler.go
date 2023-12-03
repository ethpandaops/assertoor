package handlers

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/clients"
)

type FrontendHandler struct {
	pool *clients.ClientPool
}

func NewFrontendHandler(clientPool *clients.ClientPool) *FrontendHandler {
	return &FrontendHandler{
		pool: clientPool,
	}
}
