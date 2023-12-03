package handlers

import (
	"github.com/ethpandaops/minccino/pkg/coordinator/types"
)

type FrontendHandler struct {
	coordinator types.Coordinator
}

func NewFrontendHandler(coordinator types.Coordinator) *FrontendHandler {
	return &FrontendHandler{
		coordinator: coordinator,
	}
}
