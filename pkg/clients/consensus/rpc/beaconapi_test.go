package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/all"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// attesterSlashing builds a versioned slashing the same way generate_slashings
// does, so this test covers that construction too.
func attesterSlashing(t *testing.T, version spec.DataVersion) *spec.VersionedAttesterSlashing {
	t.Helper()

	attestation := &all.IndexedAttestation{
		Version:          version,
		AttestingIndices: []uint64{42},
		Data:             &phase0.AttestationData{Slot: 640, Source: &phase0.Checkpoint{}, Target: &phase0.Checkpoint{}},
	}

	versioned, err := (&all.AttesterSlashing{
		Version:      version,
		Attestation1: attestation,
		Attestation2: attestation,
	}).ToVersioned()
	if err != nil {
		t.Fatalf("failed building %v attester slashing: %v", version, err)
	}

	return versioned
}

// TestSubmitAttesterSlashing asserts that slashing submission goes through
// go-eth2-client's versioned pool endpoint, that the Eth-Consensus-Version
// header follows the slashing's fork rather than a hardcoded value, and that a
// pre-electra slashing fails on the fork instead of on the endpoint.
func TestSubmitAttesterSlashing(t *testing.T) {
	var (
		postCount int
		gotPath   string
		gotHeader string
		gotBody   []byte
	)

	srv := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		switch r.URL.Path {
		case "/eth/v1/node/version":
			_, _ = w.Write([]byte(`{"data":{"version":"test"}}`))
		case "/eth/v1/node/syncing":
			_, _ = w.Write([]byte(`{"data":{"is_syncing":false,"is_optimistic":false,"el_offline":false,"head_slot":"640","sync_distance":"0"}}`))
		case "/eth/v2/beacon/pool/attester_slashings":
			postCount++
			gotPath = r.URL.Path
			gotHeader = r.Header.Get("Eth-Consensus-Version")
			gotBody, _ = io.ReadAll(r.Body)
		default:
			w.WriteHeader(nethttp.StatusNotFound)
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, err := NewBeaconClient("test", srv.URL, nil)
	if err != nil {
		t.Fatalf("failed creating beacon client: %v", err)
	}

	if err = client.Initialize(ctx); err != nil {
		t.Fatalf("failed initializing beacon client: %v", err)
	}

	tests := []struct {
		name       string
		version    spec.DataVersion
		wantHeader string
	}{
		{name: "Electra", version: spec.DataVersionElectra, wantHeader: "electra"},
		{name: "Fulu", version: spec.DataVersionFulu, wantHeader: "fulu"},
		{name: "Gloas", version: spec.DataVersionGloas, wantHeader: "gloas"},
		{name: "PreElectraRejected", version: spec.DataVersionDeneb},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			postCount, gotPath, gotHeader, gotBody = 0, "", "", nil

			err := client.SubmitAttesterSlashing(ctx, attesterSlashing(t, test.version))

			if test.wantHeader == "" {
				if !errors.Is(err, ErrPreElectraSlashing) {
					t.Fatalf("error = %v, want ErrPreElectraSlashing", err)
				}

				if postCount != 0 {
					t.Errorf("submissions = %d, want 0: a rejected fork must not reach the node", postCount)
				}

				return
			}

			if err != nil {
				t.Fatalf("failed submitting attester slashing: %v", err)
			}

			if postCount != 1 {
				t.Fatalf("submissions = %d, want 1", postCount)
			}

			if gotPath != "/eth/v2/beacon/pool/attester_slashings" {
				t.Errorf("path = %q, want the v2 pool endpoint", gotPath)
			}

			if gotHeader != test.wantHeader {
				t.Errorf("Eth-Consensus-Version = %q, want %q", gotHeader, test.wantHeader)
			}

			var body map[string]json.RawMessage
			if err := json.Unmarshal(gotBody, &body); err != nil {
				t.Fatalf("failed parsing submitted body: %v", err)
			}

			// A version/arm mismatch marshals to a null body under an
			// otherwise valid header, so assert the payload is really there.
			for _, key := range []string{"attestation_1", "attestation_2"} {
				if raw, ok := body[key]; !ok || string(raw) == "null" {
					t.Errorf("submitted body %v = %q, want the fork-specific attestation", key, string(raw))
				}
			}
		})
	}
}
