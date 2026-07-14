package runnetworkdisruption

import (
	"testing"

	"github.com/ethpandaops/assertoor/pkg/vars"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidate(t *testing.T) {
	isolation := map[string]any{testKeyName: testNameBlackout, testKeyTarget: map[string]any{testKeyNodeIndex: 1}}

	tests := []struct {
		name    string
		mutate  func(c *Config)
		wantErr string
	}{
		{
			name: "valid set with isolation",
			mutate: func(c *Config) {
				c.Isolations = []map[string]any{isolation}
			},
		},
		{
			name:    "missing url",
			mutate:  func(c *Config) { c.DisruptoorURL = "" },
			wantErr: "disruptoorUrl is required",
		},
		{
			name: "non-http url",
			mutate: func(c *Config) {
				c.DisruptoorURL = "ftp://disruptoor:7700"
				c.Isolations = []map[string]any{isolation}
			},
			wantErr: "scheme must be http or https",
		},
		{
			name:    "set without entries",
			mutate:  func(_ *Config) {},
			wantErr: "use action: clear instead",
		},
		{
			name: "clear with entries",
			mutate: func(c *Config) {
				c.Action = ActionClear
				c.Isolations = []map[string]any{isolation}
			},
			wantErr: "clear does not take",
		},
		{
			name:   "clear without entries",
			mutate: func(c *Config) { c.Action = ActionClear },
		},
		{
			name:    "update without entries or removeNames",
			mutate:  func(c *Config) { c.Action = ActionUpdate },
			wantErr: "update requires at least one",
		},
		{
			name: "update with removeNames only",
			mutate: func(c *Config) {
				c.Action = ActionUpdate
				c.RemoveNames = []string{testNameBlackout}
			},
		},
		{
			name: "removeNames with set",
			mutate: func(c *Config) {
				c.Isolations = []map[string]any{isolation}
				c.RemoveNames = []string{testNameBlackout}
			},
			wantErr: "removeNames is only valid with action: update",
		},
		{
			name: "unknown action",
			mutate: func(c *Config) {
				c.Action = "heal"
			},
			wantErr: "invalid action",
		},
		{
			name: "action is case-insensitive",
			mutate: func(c *Config) {
				c.Action = "CLEAR"
			},
		},
		{
			name: "entry without name",
			mutate: func(c *Config) {
				c.Isolations = []map[string]any{{testKeyTarget: map[string]any{testKeyNodeIndex: 1}}}
			},
			wantErr: "isolations[0]: name is required",
		},
		{
			name: "duplicate entry names",
			mutate: func(c *Config) {
				c.Partitions = []map[string]any{
					{testKeyName: "x", testKeyGroups: []any{}},
					{testKeyName: "x", testKeyGroups: []any{}},
				}
			},
			wantErr: `partitions[1]: duplicate name "x"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultConfig()
			config.DisruptoorURL = "http://disruptoor:7700"
			tt.mutate(&config)

			err := config.Validate()
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestConfigFromConfigVars exercises the playbook usage pattern: the
// isolations list is built by a jq expression in configVars and lands in the
// Config via the vars system's YAML round-trip.
func TestConfigFromConfigVars(t *testing.T) {
	taskVars := vars.NewVariables(nil)
	taskVars.SetVar("disruptoorApiUrl", "http://disruptoor:7700")
	taskVars.SetVar("targetParticipantIndex", 1)

	config := DefaultConfig()
	require.NoError(t, taskVars.ConsumeVars(&config, map[string]string{
		"disruptoorUrl": "disruptoorApiUrl",
		"isolations": `| [{
			name: "assertoor-blackout-target-cl",
			target: {"node-index": (.targetParticipantIndex | tonumber), "client-type": "beacon"},
			scope: ["cl_p2p", "el_p2p", "include_control"]
		}]`,
	}))
	require.NoError(t, config.Validate())

	assert.Equal(t, "http://disruptoor:7700", config.DisruptoorURL)
	require.Len(t, config.Isolations, 1)
	assert.Equal(t, "assertoor-blackout-target-cl", config.Isolations[0][testKeyName])
	assert.Equal(t, map[string]any{testKeyNodeIndex: 1, "client-type": "beacon"}, config.Isolations[0][testKeyTarget])
}

func TestConfigValidateTrimsTrailingSlash(t *testing.T) {
	config := DefaultConfig()
	config.DisruptoorURL = "http://disruptoor:7700/"
	config.Action = ActionClear

	require.NoError(t, config.Validate())
	assert.Equal(t, "http://disruptoor:7700", config.DisruptoorURL)
}
