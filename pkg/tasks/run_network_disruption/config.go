package runnetworkdisruption

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/helper"
)

// Action selects what the task does against the disruptoor API.
type Action string

const (
	// ActionSet replaces the entire disruptoor state with the configured entries.
	ActionSet Action = "set"
	// ActionUpdate merges the configured entries into the current state by name.
	ActionUpdate Action = "update"
	// ActionClear heals all active disruptions.
	ActionClear Action = "clear"
)

// Config holds the task configuration for driving a disruptoor instance:
// applying network partitions, isolations, and shaping rules to a
// Kurtosis-launched devnet, or healing them again.
//
// Partition, isolation, and shaping entries are passed through to disruptoor
// verbatim (wire format of PUT /v1/state, see the disruptoor JSON schema).
// Assertoor only checks that every entry carries a name — everything else is
// validated server-side so new disruptoor fields work without assertoor
// changes.
type Config struct {
	DisruptoorURL   string           `yaml:"disruptoorUrl" json:"disruptoorUrl" require:"A" desc:"Base URL of the disruptoor HTTP API (e.g. http://disruptoor:7700)."`
	Action          Action           `yaml:"action" json:"action" desc:"Action to perform: set (replace the whole disruptoor state), update (merge entries by name into the current state), clear (heal everything)."`
	Partitions      []map[string]any `yaml:"partitions" json:"partitions,omitempty" desc:"Disruptoor partition entries: each splits the enclave into 2+ disjoint groups."`
	Isolations      []map[string]any `yaml:"isolations" json:"isolations,omitempty" desc:"Disruptoor isolation entries: each cuts the containers matched by its target selector off from the rest of the enclave (the counterparty group is computed by disruptoor; a multi-container target is isolated as a group)."`
	Shaping         []map[string]any `yaml:"shaping" json:"shaping,omitempty" desc:"Disruptoor shaping entries: per-target delay/jitter/loss/bandwidth degradation."`
	RemoveNames     []string         `yaml:"removeNames" json:"removeNames,omitempty" desc:"Entry names to remove from the current state before merging (update action only)."`
	AwaitAPITimeout helper.Duration  `yaml:"awaitApiTimeout" json:"awaitApiTimeout" desc:"How long to wait for the disruptoor API to report healthy before acting (0 = act immediately)."`
	PollInterval    helper.Duration  `yaml:"pollInterval" json:"pollInterval" desc:"Interval between health probes while waiting for the API."`
	RequestTimeout  helper.Duration  `yaml:"requestTimeout" json:"requestTimeout" desc:"Timeout for a single HTTP request."`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		Action:          ActionSet,
		AwaitAPITimeout: helper.Duration{Duration: 30 * time.Second},
		PollInterval:    helper.Duration{Duration: 2 * time.Second},
		RequestTimeout:  helper.Duration{Duration: 10 * time.Second},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.DisruptoorURL == "" {
		return fmt.Errorf("disruptoorUrl is required")
	}

	parsed, err := url.Parse(c.DisruptoorURL)
	if err != nil {
		return fmt.Errorf("invalid disruptoorUrl %q: %w", c.DisruptoorURL, err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("invalid disruptoorUrl %q: scheme must be http or https", c.DisruptoorURL)
	}

	c.DisruptoorURL = strings.TrimRight(c.DisruptoorURL, "/")

	c.Action = Action(strings.ToLower(string(c.Action)))

	entryCount := len(c.Partitions) + len(c.Isolations) + len(c.Shaping)

	switch c.Action {
	case ActionSet:
		if entryCount == 0 {
			return fmt.Errorf("set with no partitions/isolations/shaping would clear everything; use action: clear instead")
		}
	case ActionUpdate:
		if entryCount == 0 && len(c.RemoveNames) == 0 {
			return fmt.Errorf("update requires at least one partition/isolation/shaping entry or removeNames")
		}
	case ActionClear:
		if entryCount > 0 {
			return fmt.Errorf("clear does not take partition/isolation/shaping entries")
		}
	default:
		return fmt.Errorf("invalid action %q, must be one of: set, update, clear", c.Action)
	}

	if len(c.RemoveNames) > 0 && c.Action != ActionUpdate {
		return fmt.Errorf("removeNames is only valid with action: update")
	}

	if err := validateEntryNames("partitions", c.Partitions); err != nil {
		return err
	}

	if err := validateEntryNames("isolations", c.Isolations); err != nil {
		return err
	}

	if err := validateEntryNames("shaping", c.Shaping); err != nil {
		return err
	}

	if c.AwaitAPITimeout.Duration > 0 && c.PollInterval.Duration <= 0 {
		return fmt.Errorf("pollInterval must be positive")
	}

	if c.RequestTimeout.Duration <= 0 {
		return fmt.Errorf("requestTimeout must be positive")
	}

	return nil
}

// validateEntryNames checks that every entry in a passthrough list carries a
// unique, non-empty name. Names are what disruptoor keys entries on and what
// the update action merges by, so they must be present client-side.
func validateEntryNames(kind string, entries []map[string]any) error {
	seen := make(map[string]bool, len(entries))

	for i, entry := range entries {
		name, ok := entry["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("%s[%d]: name is required", kind, i)
		}

		if seen[name] {
			return fmt.Errorf("%s[%d]: duplicate name %q", kind, i, name)
		}

		seen[name] = true
	}

	return nil
}
