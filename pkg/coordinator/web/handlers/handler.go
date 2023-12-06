package handlers

import (
	"github.com/ethpandaops/assertoor/pkg/coordinator/types"
)

type FrontendHandler struct {
	coordinator types.Coordinator
}

func NewFrontendHandler(coordinator types.Coordinator) *FrontendHandler {
	return &FrontendHandler{
		coordinator: coordinator,
	}
}
