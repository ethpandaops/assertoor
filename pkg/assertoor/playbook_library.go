package assertoor

import (
	"context"

	"github.com/ethpandaops/assertoor/pkg/playbooklibrary"
)

// coordinatorLocalTestProvider adapts the Coordinator's database lookups
// to the playbooklibrary.LocalTestProvider interface, letting the
// library service compare remote playbooks against locally registered
// tests by id.
type coordinatorLocalTestProvider struct {
	coordinator *Coordinator
}

// FindLocalYaml returns the stored YAML and display name for a locally
// registered test with the given id. When no matching test exists it
// returns empty strings with a nil error (per the LocalTestProvider
// contract). Tests without a stored YamlSource (e.g. older legacy
// registrations) also return empty strings; the library treats those
// the same as "absent" for comparison purposes.
func (p *coordinatorLocalTestProvider) FindLocalYaml(_ context.Context, testID string) (yaml, name string, err error) {
	if p.coordinator == nil || p.coordinator.database == nil {
		return "", "", nil
	}

	cfg, lookupErr := p.coordinator.database.GetTestConfig(testID)
	if lookupErr != nil {
		// "not found" is the common case and should not be reported as
		// an error to the library. We can't easily distinguish "not
		// found" from a real DB error without sql.ErrNoRows plumbing,
		// so log + swallow.
		p.coordinator.Logger().WithError(lookupErr).Debugf("local test lookup failed for %q", testID)
		return "", "", nil
	}

	if cfg == nil {
		return "", "", nil
	}

	if cfg.YamlSource == "" {
		// Test exists but no captured YAML — we can't compare; signal
		// absence so the UI doesn't show a misleading "same" badge.
		return "", cfg.Name, nil
	}

	return cfg.YamlSource, cfg.Name, nil
}

// Compile-time assertion that the adapter satisfies the library's
// LocalTestProvider contract.
var _ playbooklibrary.LocalTestProvider = (*coordinatorLocalTestProvider)(nil)
