package generateslashings

import (
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// populatedArms names the fork-specific arms set on a versioned slashing.
func populatedArms(versioned *spec.VersionedAttesterSlashing) []string {
	all := []struct {
		name      string
		populated bool
	}{
		{"phase0", versioned.Phase0 != nil},
		{"altair", versioned.Altair != nil},
		{"bellatrix", versioned.Bellatrix != nil},
		{"capella", versioned.Capella != nil},
		{"deneb", versioned.Deneb != nil},
		{"electra", versioned.Electra != nil},
		{"fulu", versioned.Fulu != nil},
		{"gloas", versioned.Gloas != nil},
		{"heze", versioned.Heze != nil},
	}

	arms := []string{}

	for _, arm := range all {
		if arm.populated {
			arms = append(arms, arm.name)
		}
	}

	return arms
}

func TestVersionedAttesterSlashing(t *testing.T) {
	attestation := &phase0.IndexedAttestation{
		AttestingIndices: []uint64{7},
		Data:             &phase0.AttestationData{Slot: 640},
		Signature:        phase0.BLSSignature{0x01},
	}
	slashing := &phase0.AttesterSlashing{
		Attestation1: attestation,
		Attestation2: attestation,
	}

	tests := []struct {
		name    string
		version spec.DataVersion
		wantArm string
	}{
		{name: "Electra", version: spec.DataVersionElectra, wantArm: "electra"},
		{name: "Fulu", version: spec.DataVersionFulu, wantArm: "fulu"},
		{name: "Gloas", version: spec.DataVersionGloas, wantArm: "gloas"},
		{name: "PreElectraRejected", version: spec.DataVersionDeneb},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			versioned, err := versionedAttesterSlashing(test.version, slashing)

			if test.wantArm == "" {
				if err == nil {
					t.Fatalf("versionedAttesterSlashing(%v) = %v, want error", test.version, versioned)
				}

				return
			}

			if err != nil {
				t.Fatalf("versionedAttesterSlashing(%v) failed: %v", test.version, err)
			}

			if versioned.Version != test.version {
				t.Errorf("version = %v, want %v", versioned.Version, test.version)
			}

			// Exactly the arm the version selects must be populated: any other
			// combination makes go-eth2-client submit a null body under an
			// otherwise valid Eth-Consensus-Version header.
			if arms := populatedArms(versioned); len(arms) != 1 || arms[0] != test.wantArm {
				t.Errorf("populated arms = %v, want [%v]", arms, test.wantArm)
			}

			att1, err := versioned.Attestation1()
			if err != nil {
				t.Fatalf("failed reading attestation 1: %v", err)
			}

			indices, err := att1.AttestingIndices()
			if err != nil {
				t.Fatalf("failed reading attesting indices: %v", err)
			}

			if len(indices) != 1 || indices[0] != 7 {
				t.Errorf("attesting indices = %v, want [7]", indices)
			}
		})
	}
}
