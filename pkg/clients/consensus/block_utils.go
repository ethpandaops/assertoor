package consensus

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/gloas"
)

func GetExecutionExtraData(v *spec.VersionedSignedBeaconBlock) ([]byte, error) {
	//nolint:exhaustive // ignore
	switch v.Version {
	case spec.DataVersionBellatrix:
		if v.Bellatrix == nil || v.Bellatrix.Message == nil || v.Bellatrix.Message.Body == nil || v.Bellatrix.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no bellatrix block")
		}

		return v.Bellatrix.Message.Body.ExecutionPayload.ExtraData, nil
	case spec.DataVersionCapella:
		if v.Capella == nil || v.Capella.Message == nil || v.Capella.Message.Body == nil || v.Capella.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no capella block")
		}

		return v.Capella.Message.Body.ExecutionPayload.ExtraData, nil
	case spec.DataVersionDeneb:
		if v.Deneb == nil || v.Deneb.Message == nil || v.Deneb.Message.Body == nil || v.Deneb.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no deneb block")
		}

		return v.Deneb.Message.Body.ExecutionPayload.ExtraData, nil
	case spec.DataVersionElectra:
		if v.Electra == nil || v.Electra.Message == nil || v.Electra.Message.Body == nil || v.Electra.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no electra block")
		}

		return v.Electra.Message.Body.ExecutionPayload.ExtraData, nil
	case spec.DataVersionFulu:
		if v.Fulu == nil || v.Fulu.Message == nil || v.Fulu.Message.Body == nil || v.Fulu.Message.Body.ExecutionPayload == nil {
			return nil, errors.New("no fulu block")
		}

		return v.Fulu.Message.Body.ExecutionPayload.ExtraData, nil
	case spec.DataVersionGloas:
		return nil, errors.New("gloas extra data is in separate payload")
	default:
		return nil, errors.New("unknown version")
	}
}

// GetPayloadExtraData returns the extra data from a gloas execution payload envelope.
func GetPayloadExtraData(payload *gloas.SignedExecutionPayloadEnvelope) ([]byte, error) {
	if payload == nil || payload.Message == nil || payload.Message.Payload == nil {
		return nil, errors.New("no payload")
	}

	return payload.Message.Payload.ExtraData, nil
}

func GetBlockBody(v *spec.VersionedSignedBeaconBlock) any {
	//nolint:exhaustive // ignore
	switch v.Version {
	case spec.DataVersionPhase0:
		return v.Phase0
	case spec.DataVersionAltair:
		return v.Altair
	case spec.DataVersionBellatrix:
		return v.Bellatrix
	case spec.DataVersionCapella:
		return v.Capella
	case spec.DataVersionDeneb:
		return v.Deneb
	case spec.DataVersionElectra:
		return v.Electra
	case spec.DataVersionFulu:
		return v.Fulu
	case spec.DataVersionGloas:
		return v.Gloas
	default:
		return nil
	}
}
