package generateapicompatibilitymatrix

type Config struct {
	// Display ----------------------------------------------------------------
	Title       string `yaml:"title" json:"title" desc:"Title displayed at the top of the matrix."`
	Description string `yaml:"description" json:"description" desc:"Free-form markdown description above the table."`

	// Selection --------------------------------------------------------------
	// If non-empty, only check_consensus_api tasks whose checkId is in this
	// list contribute rows (ordered by this list). Default: all completed
	// check_consensus_api tasks in scheduler index order.
	IncludeCheckIDs []string `yaml:"includeCheckIds" json:"includeCheckIds" desc:"Restrict matrix rows to these checkIds, in order."`

	// SourceTaskName lets the user re-use this aggregator for future tasks
	// that emit a 'matrixRow' output under a different task name. Default
	// 'check_consensus_api'.
	SourceTaskName string `yaml:"sourceTaskName" json:"sourceTaskName" desc:"Task name to pick matrix-row outputs from. Default: check_consensus_api."`

	// Display ----------------------------------------------------------------
	// Client column ordering. If empty, defaults to lighthouse, teku, prysm,
	// grandine, nimbus, lodestar, caplin, unknown — and unused columns are
	// omitted unless 'showAllClientTypes' is true.
	ClientOrder        []string `yaml:"clientOrder" json:"clientOrder" desc:"Client-type column ordering."`
	ShowAllClientTypes bool     `yaml:"showAllClientTypes" json:"showAllClientTypes" desc:"If true, render all client-type columns even when unused."`

	EmojiPass    string `yaml:"emojiPass" json:"emojiPass" desc:"Emoji for pass. Default ✅."`
	EmojiPartial string `yaml:"emojiPartial" json:"emojiPartial" desc:"Emoji for partial. Default 🟡."`
	EmojiFail    string `yaml:"emojiFail" json:"emojiFail" desc:"Emoji for fail. Default ❌."`
	EmojiSkipped string `yaml:"emojiSkipped" json:"emojiSkipped" desc:"Emoji for skipped. Default ⚪."`
	EmojiAbsent  string `yaml:"emojiAbsent" json:"emojiAbsent" desc:"Emoji for absent (no result for this client-type). Default -."`

	IncludeFootnotes bool `yaml:"includeFootnotes" json:"includeFootnotes" desc:"If true, append numbered footnotes for cells with notes."`
	IncludeLegend    bool `yaml:"includeLegend" json:"includeLegend" desc:"If true, render a legend below the table."`

	FailOnFailures bool `yaml:"failOnFailures" json:"failOnFailures" desc:"If true, fail the task when any cell has fail status."`
}

func DefaultConfig() Config {
	return Config{
		Title:            "API Compatibility Matrix",
		SourceTaskName:   "check_consensus_api",
		EmojiPass:        "✅",
		EmojiPartial:     "🟡",
		EmojiFail:        "❌",
		EmojiSkipped:     "⚪",
		EmojiAbsent:      "—",
		IncludeFootnotes: true,
		IncludeLegend:    true,
	}
}

func (c *Config) Validate() error {
	if c.SourceTaskName == "" {
		c.SourceTaskName = "check_consensus_api"
	}
	if c.EmojiPass == "" {
		c.EmojiPass = "✅"
	}
	if c.EmojiPartial == "" {
		c.EmojiPartial = "🟡"
	}
	if c.EmojiFail == "" {
		c.EmojiFail = "❌"
	}
	if c.EmojiSkipped == "" {
		c.EmojiSkipped = "⚪"
	}
	if c.EmojiAbsent == "" {
		c.EmojiAbsent = "—"
	}
	return nil
}
