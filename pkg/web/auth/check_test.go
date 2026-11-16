package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethpandaops/service-authenticatoor/pkg/issuer"
)

const (
	testIssuer   = "https://auth.test.example"
	testAudience = "test.example"
	testHost     = "assertoor.test.example"
)

// TestHasFeature runs real tokens through the verifier and the feature
// check: the services claim is what an authenticatoor would mint.
func TestHasFeature(t *testing.T) {
	t.Parallel()

	key, err := issuer.GenerateRSAKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	newIssuer := func(services string) *issuer.RS256Issuer {
		iss, issErr := issuer.NewRS256Issuer(issuer.Config{
			ActiveKey:       key,
			Issuer:          testIssuer,
			DefaultAudience: []string{testAudience},
			Scope:           "*." + testAudience,
			Services:        services,
			TTL:             time.Minute,
		})
		if issErr != nil {
			t.Fatalf("new issuer: %v", issErr)
		}

		return iss
	}

	jwks, err := newIssuer("").JWKS()
	if err != nil {
		t.Fatalf("jwks: %v", err)
	}

	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"issuer": testIssuer, "jwks_uri": srv.URL + "/jwks.json"})
	})
	mux.HandleFunc("/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(jwks)
	})

	handler, err := NewAuthHandler(context.Background(), nil, srv.URL, testAudience)
	if err != nil {
		t.Fatalf("new auth handler: %v", err)
	}

	tests := []struct {
		name         string
		services     string
		wantService  bool // passes the verifier's "assertoor" service gate
		wantRegistry bool
		wantPprof    bool
	}{
		{"no services claim (authenticatoor without grants)", "", true, true, true},
		{"staff: every service and feature", "+all,+all:all", true, true, true},
		{"basic access only", "-all,+assertoor", true, false, false},
		{"test_registry granted", "-all,+assertoor,+assertoor:test_registry", true, true, false},
		{"all assertoor features", "-all,+assertoor,+assertoor:all", true, true, true},
		{"all but pprof", "-all,+assertoor,+assertoor:all:-pprof", true, true, false},
		{"faucet only", "-all,+faucet,+faucet:drop_100eth", false, false, false},
		{"no grant at all", "-all", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tokenStr, _, err := newIssuer(tt.services).Issue("alice@example.com", "alice@example.com", nil)
			if err != nil {
				t.Fatalf("issue: %v", err)
			}

			token := handler.CheckAuthToken("Bearer "+tokenStr, testHost)
			if got := token != nil && token.Valid; got != tt.wantService {
				t.Fatalf("service gate: valid = %v, want %v", got, tt.wantService)
			}

			if got := handler.HasFeature(token, FeatureTestRegistry); got != tt.wantRegistry {
				t.Errorf("HasFeature(%s) = %v, want %v", FeatureTestRegistry, got, tt.wantRegistry)
			}

			if got := handler.HasFeature(token, FeaturePprof); got != tt.wantPprof {
				t.Errorf("HasFeature(%s) = %v, want %v", FeaturePprof, got, tt.wantPprof)
			}
		})
	}
}

func TestHasFeatureOpenMode(t *testing.T) {
	t.Parallel()

	handler, err := NewAuthHandler(context.Background(), nil, "", "")
	if err != nil {
		t.Fatalf("new auth handler: %v", err)
	}

	token := handler.CheckAuthToken("", testHost)
	if !handler.HasFeature(token, FeatureTestRegistry) || !handler.HasFeature(token, FeaturePprof) {
		t.Fatal("open mode must allow every feature")
	}
}
