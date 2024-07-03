package checkconsensusblockproposals

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/electra"
)

func getExecutionDepositRequests(v *spec.VersionedSignedBeaconBlock) ([]*electra.DepositRequest, error) {
	switch v.Version {
	case spec.DataVersionPhase0:
		return nil, errors.New("phase0 block does not have execution deposit requests")
	case spec.DataVersionAltair:
		return nil, errors.New("altair block does not have execution deposit requests")
	case spec.DataVersionBellatrix:
		return nil, errors.New("bellatrix block does not have execution deposit requests")
	case spec.DataVersionCapella:
		return nil, errors.New("capella block does not have execution deposit requests")
	case spec.DataVersionDeneb:
		return nil, errors.New("deneb block does not have execution deposit requests")
	case spec.DataVersionElectra:
		if v.Electra == nil ||
			v.Electra.Message == nil ||
			v.Electra.Message.Body == nil ||
			v.Electra.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no electra block")
		}

		return v.Electra.Message.Body.ExecutionPayload.DepositRequests, nil
	default:
		return nil, errors.New("unknown version")
	}
}

// ExecutionWithdrawalRequests returs the execution withdrawal requests for the block.
func getExecutionWithdrawalRequests(v *spec.VersionedSignedBeaconBlock) ([]*electra.WithdrawalRequest, error) {
	switch v.Version {
	case spec.DataVersionPhase0:
		return nil, errors.New("phase0 block does not have execution withdrawal requests")
	case spec.DataVersionAltair:
		return nil, errors.New("altair block does not have execution withdrawal requests")
	case spec.DataVersionBellatrix:
		return nil, errors.New("bellatrix block does not have execution withdrawal requests")
	case spec.DataVersionCapella:
		return nil, errors.New("capella block does not have execution withdrawal requests")
	case spec.DataVersionDeneb:
		return nil, errors.New("deneb block does not have execution withdrawal requests")
	case spec.DataVersionElectra:
		if v.Electra == nil ||
			v.Electra.Message == nil ||
			v.Electra.Message.Body == nil ||
			v.Electra.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no electra block")
		}

		return v.Electra.Message.Body.ExecutionPayload.WithdrawalRequests, nil
	default:
		return nil, errors.New("unknown version")
	}
}

// ExecutionConsolidationRequests returs the execution consolidation requests for the block.
func getExecutionConsolidationRequests(v *spec.VersionedSignedBeaconBlock) ([]*electra.ConsolidationRequest, error) {
	switch v.Version {
	case spec.DataVersionPhase0:
		return nil, errors.New("phase0 block does not have execution consolidation requests")
	case spec.DataVersionAltair:
		return nil, errors.New("altair block does not have execution consolidation requests")
	case spec.DataVersionBellatrix:
		return nil, errors.New("bellatrix block does not have execution consolidation requests")
	case spec.DataVersionCapella:
		return nil, errors.New("capella block does not have execution consolidation requests")
	case spec.DataVersionDeneb:
		return nil, errors.New("deneb block does not have execution consolidation requests")
	case spec.DataVersionElectra:
		if v.Electra == nil ||
			v.Electra.Message == nil ||
			v.Electra.Message.Body == nil ||
			v.Electra.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no electra block")
		}

		return v.Electra.Message.Body.ExecutionPayload.ConsolidationRequests, nil
	default:
		return nil, errors.New("unknown version")
	}
}
