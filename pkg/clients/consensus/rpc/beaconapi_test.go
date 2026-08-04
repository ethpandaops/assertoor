package rpc

import (
	"context"
	"encoding/json"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
)

// TestSubmitAttesterSlashing asserts that slashing submission goes through
// go-eth2-client's versioned pool endpoint and that the Eth-Consensus-Version
// header follows the slashing's fork rather than a hardcoded value.
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

	attestation := &gloas.IndexedAttestation{
		AttestingIndices: []uint64{42},
		Data:             &phase0.AttestationData{Slot: 640, Source: &phase0.Checkpoint{}, Target: &phase0.Checkpoint{}},
	}

	err = client.SubmitAttesterSlashing(ctx, &spec.VersionedAttesterSlashing{
		Version: spec.DataVersionGloas,
		Gloas: &gloas.AttesterSlashing{
			Attestation1: attestation,
			Attestation2: attestation,
		},
	})
	if err != nil {
		t.Fatalf("failed submitting attester slashing: %v", err)
	}

	if postCount != 1 {
		t.Errorf("submissions = %d, want 1", postCount)
	}

	if gotPath != "/eth/v2/beacon/pool/attester_slashings" {
		t.Errorf("path = %q, want the v2 pool endpoint", gotPath)
	}

	if gotHeader != "gloas" {
		t.Errorf("Eth-Consensus-Version = %q, want %q", gotHeader, "gloas")
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(gotBody, &body); err != nil {
		t.Fatalf("failed parsing submitted body: %v", err)
	}

	for _, key := range []string{"attestation_1", "attestation_2"} {
		if raw, ok := body[key]; !ok || string(raw) == "null" {
			t.Errorf("submitted body %v = %q, want the fork-specific attestation", key, string(raw))
		}
	}
}
