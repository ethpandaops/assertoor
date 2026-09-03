package txmgr

import (
	"slices"
	"testing"

	"github.com/ethpandaops/assertoor/pkg/clients/execution"
	"github.com/ethpandaops/spamoor/spamoor"
)

// spamoor trims group names in its own parser, so the check must trim too:
// otherwise a stray space makes the two disagree and a removal that never
// took effect passes verification — the silent under-isolation this guard
// exists to prevent.
func TestNormalizedGroupsMatchesSpamoorParsing(t *testing.T) {
	config := &execution.ClientConfig{
		Name:   "intake",
		URL:    "http://localhost:8545",
		Groups: []string{" builder ", "", "  ", "-default "},
	}

	got := normalizedGroups(config)
	if want := []string{"builder", "-default"}; !slices.Equal(got, want) {
		t.Fatalf("normalizedGroups = %v, want %v", got, want)
	}

	client, err := spamoor.NewClient(&spamoor.ClientOptions{
		RpcHost: "group(" + got[0] + ")" + config.URL,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !client.HasGroup("builder") {
		t.Error("spamoor did not apply the trimmed group name")
	}

	if err := verifyClientGroups(&execution.ClientConfig{Name: "intake", Groups: []string{" builder "}}, client); err != nil {
		t.Errorf("verification must accept the same trimmed name spamoor applied: %v", err)
	}
}

// A configured group the linked spamoor did not apply must stop startup.
func TestVerifyClientGroupsRejectsAnUnappliedEntry(t *testing.T) {
	client, err := spamoor.NewClient(&spamoor.ClientOptions{RpcHost: "group(builder)http://localhost:8545"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := verifyClientGroups(&execution.ClientConfig{Name: "intake", Groups: []string{"builder"}}, client); err != nil {
		t.Errorf("an applied group must verify: %v", err)
	}

	// The pinned spamoor keeps the client in "default", so a removal that was
	// ignored has to be an error rather than silent under-isolation.
	err = verifyClientGroups(&execution.ClientConfig{Name: "intake", Groups: []string{"-default"}}, client)
	if client.HasGroup("default") && err == nil {
		t.Error("an ignored removal must fail verification")
	}

	if err := verifyClientGroups(&execution.ClientConfig{Name: "intake", Groups: []string{"absent"}}, client); err == nil {
		t.Error("a group the client is not in must fail verification")
	}
}
