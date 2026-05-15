package generateapicompatibilitymatrix

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ethpandaops/assertoor/pkg/db"
	"github.com/ethpandaops/assertoor/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

var (
	TaskName       = "generate_api_compatibility_matrix"
	TaskDescriptor = &types.TaskDescriptor{
		Name:        TaskName,
		Description: "Collects results from check_consensus_api tasks and renders a markdown compatibility matrix as a result artifact.",
		Category:    "utility",
		Config:      DefaultConfig(),
		Outputs: []types.TaskOutputDefinition{
			{
				Name:        "matrixMarkdown",
				Type:        "string",
				Description: "The rendered markdown table.",
			},
			{
				Name:        "passRate",
				Type:        "float",
				Description: "Fraction of pass cells over total cells (excluding absent/skipped).",
			},
			{
				Name:        "cellCounts",
				Type:        "object",
				Description: "Counts per cell status: pass, partial, fail, skipped, absent.",
			},
		},
		NewTask: NewTask,
	}
)

type Task struct {
	ctx     *types.TaskContext
	options *types.TaskOptions
	config  Config
	logger  logrus.FieldLogger
}

type matrixRow struct {
	CheckID      string
	CheckTitle   string
	ReferenceURL string
	Cells        map[string]matrixCell // keyed by client-type
}

type matrixCell struct {
	Result     string `json:"result"`
	Note       string `json:"note,omitempty"`
	HTTPStatus int    `json:"httpStatus,omitempty"`
}

func NewTask(ctx *types.TaskContext, options *types.TaskOptions) (types.Task, error) {
	return &Task{
		ctx:     ctx,
		options: options,
		logger:  ctx.Logger.GetLogger(),
	}, nil
}

func (t *Task) Config() interface{} { return t.config }

func (t *Task) Timeout() time.Duration {
	return t.options.Timeout.Duration
}

func (t *Task) LoadConfig() error {
	config := DefaultConfig()

	if t.options.Config != nil {
		if err := t.options.Config.Unmarshal(&config); err != nil {
			return fmt.Errorf("error parsing task config for %v: %w", TaskName, err)
		}
	}
	if err := t.ctx.Vars.ConsumeVars(&config, t.options.ConfigVars); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	t.config = config
	return nil
}

func (t *Task) Execute(_ context.Context) error {
	rows := t.collectRows()
	if len(rows) == 0 {
		t.logger.Warn("no check_consensus_api task outputs found to render")
	}

	// Determine which client-type columns are used.
	allTypes := []string{"lighthouse", "teku", "prysm", "grandine", "nimbus", "lodestar", "caplin"}
	if len(t.config.ClientOrder) > 0 {
		allTypes = t.config.ClientOrder
	}

	usedTypes := map[string]bool{}
	for _, row := range rows {
		for ct := range row.Cells {
			usedTypes[ct] = true
		}
	}
	columnTypes := []string{}
	for _, ct := range allTypes {
		if t.config.ShowAllClientTypes || usedTypes[ct] {
			columnTypes = append(columnTypes, ct)
		}
	}
	// Append any extra types not in the canonical list.
	for ct := range usedTypes {
		seen := false
		for _, c := range columnTypes {
			if c == ct {
				seen = true
				break
			}
		}
		if !seen {
			columnTypes = append(columnTypes, ct)
		}
	}

	markdown, counts, failCount := t.renderMarkdown(rows, columnTypes)

	jsonBytes, _ := json.MarshalIndent(struct {
		Title     string         `json:"title"`
		Generated string         `json:"generated"`
		Columns   []string       `json:"columns"`
		Rows      []rowJSON      `json:"rows"`
		Counts    map[string]int `json:"counts"`
	}{
		Title:     t.config.Title,
		Generated: time.Now().UTC().Format(time.RFC3339),
		Columns:   columnTypes,
		Rows:      rowsToJSON(rows, columnTypes),
		Counts:    counts,
	}, "", "  ")

	t.storeResults(markdown, jsonBytes)

	totalCells := counts["pass"] + counts["partial"] + counts["fail"]
	passRate := 0.0
	if totalCells > 0 {
		passRate = float64(counts["pass"]) / float64(totalCells)
	}
	t.ctx.Outputs.SetVar("matrixMarkdown", markdown)
	t.ctx.Outputs.SetVar("passRate", passRate)
	t.ctx.Outputs.SetVar("cellCounts", counts)

	t.logger.Infof("rendered matrix: %d rows × %d cols (pass=%d partial=%d fail=%d skipped=%d absent=%d)",
		len(rows), len(columnTypes), counts["pass"], counts["partial"], counts["fail"], counts["skipped"], counts["absent"])

	if t.config.FailOnFailures && failCount > 0 {
		t.ctx.SetResult(types.TaskResultFailure)
		return fmt.Errorf("%d cells failed", failCount)
	}

	t.ctx.SetResult(types.TaskResultSuccess)
	return nil
}

type rowJSON struct {
	CheckID      string                `json:"checkId"`
	CheckTitle   string                `json:"checkTitle"`
	ReferenceURL string                `json:"referenceUrl,omitempty"`
	Cells        map[string]matrixCell `json:"cells"`
}

func rowsToJSON(rows []*matrixRow, columns []string) []rowJSON {
	out := make([]rowJSON, len(rows))
	for i, r := range rows {
		cells := map[string]matrixCell{}
		for _, c := range columns {
			if cell, ok := r.Cells[c]; ok {
				cells[c] = cell
			} else {
				cells[c] = matrixCell{Result: "absent"}
			}
		}
		out[i] = rowJSON{
			CheckID:      r.CheckID,
			CheckTitle:   r.CheckTitle,
			ReferenceURL: r.ReferenceURL,
			Cells:        cells,
		}
	}
	return out
}

func (t *Task) collectRows() []*matrixRow {
	scheduler := t.ctx.Scheduler
	taskIndices := scheduler.GetAllTasks()

	indexByID := map[string]int{}
	for i, id := range t.config.IncludeCheckIDs {
		indexByID[id] = i
	}

	rows := []*matrixRow{}
	for _, idx := range taskIndices {
		state := scheduler.GetTaskState(idx)
		if state == nil || state.Name() != t.config.SourceTaskName {
			continue
		}
		statusVars := state.GetTaskStatusVars()
		if statusVars == nil {
			continue
		}
		outputsScope := statusVars.GetSubScope("outputs")
		if outputsScope == nil {
			continue
		}

		checkID := stringVar(outputsScope, "checkId")
		if checkID == "" {
			// Skip tasks that didn't run / didn't set checkId yet.
			continue
		}
		if len(t.config.IncludeCheckIDs) > 0 {
			if _, ok := indexByID[checkID]; !ok {
				continue
			}
		}

		title := stringVar(outputsScope, "checkTitle")
		ref := stringVar(outputsScope, "referenceUrl")

		matrixRowRaw := outputsScope.GetVar("matrixRow")
		cells := matrixCellsFromRaw(matrixRowRaw)

		rows = append(rows, &matrixRow{
			CheckID:      checkID,
			CheckTitle:   title,
			ReferenceURL: ref,
			Cells:        cells,
		})
	}

	// If user specified IncludeCheckIDs, reorder.
	if len(t.config.IncludeCheckIDs) > 0 {
		sort.SliceStable(rows, func(a, b int) bool {
			ai, aok := indexByID[rows[a].CheckID]
			bi, bok := indexByID[rows[b].CheckID]
			if !aok {
				return false
			}
			if !bok {
				return true
			}
			return ai < bi
		})
	}

	return rows
}

func stringVar(vars types.Variables, key string) string {
	v := vars.GetVar(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func matrixCellsFromRaw(raw interface{}) map[string]matrixCell {
	out := map[string]matrixCell{}
	m, ok := raw.(map[string]interface{})
	if !ok {
		return out
	}
	for ct, v := range m {
		c, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		cell := matrixCell{}
		if r, ok := c["result"].(string); ok {
			cell.Result = r
		}
		if n, ok := c["note"].(string); ok {
			cell.Note = n
		}
		if hs, ok := c["httpStatus"].(float64); ok {
			cell.HTTPStatus = int(hs)
		}
		if hs, ok := c["httpStatus"].(int); ok {
			cell.HTTPStatus = hs
		}
		if hs, ok := c["httpStatus"].(int64); ok {
			cell.HTTPStatus = int(hs)
		}
		out[ct] = cell
	}
	return out
}

// shortClientLabel renders client-type column headers as compact labels.
func shortClientLabel(t string) string {
	switch t {
	case "lighthouse":
		return "LH"
	case "lodestar":
		return "Lo"
	case "nimbus":
		return "Ni"
	case "prysm":
		return "Pr"
	case "teku":
		return "Tk"
	case "grandine":
		return "Gr"
	case "caplin":
		return "Ca"
	}
	if t == "" {
		return "?"
	}
	// Fallback: two-character title-case label.
	upper := strings.ToUpper(t[:1])
	if len(t) >= 2 {
		return upper + t[1:2]
	}
	return upper
}

func (t *Task) renderMarkdown(rows []*matrixRow, columns []string) (string, map[string]int, int) {
	counts := map[string]int{"pass": 0, "partial": 0, "fail": 0, "skipped": 0, "absent": 0}

	var b strings.Builder
	if t.config.Title != "" {
		fmt.Fprintf(&b, "# %s\n\n", t.config.Title)
	}
	if t.config.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", t.config.Description)
	}

	// Header.
	b.WriteString("| # | Endpoint / Event |")
	for _, c := range columns {
		fmt.Fprintf(&b, " %s |", shortClientLabel(c))
	}
	b.WriteString("\n")
	b.WriteString("|---|---|")
	for range columns {
		b.WriteString("---|")
	}
	b.WriteString("\n")

	// Track footnotes.
	type footnote struct {
		idx  int
		text string
	}
	footnotes := []footnote{}
	footnoteMap := map[string]int{}

	footnoteRef := func(note string) string {
		if !t.config.IncludeFootnotes || note == "" {
			return ""
		}
		if idx, ok := footnoteMap[note]; ok {
			return superscript(idx)
		}
		idx := len(footnotes) + 1
		footnoteMap[note] = idx
		footnotes = append(footnotes, footnote{idx: idx, text: note})
		return superscript(idx)
	}

	for i, row := range rows {
		title := row.CheckTitle
		if title == "" {
			title = row.CheckID
		}
		// escape pipes in title
		title = strings.ReplaceAll(title, "|", "\\|")
		fmt.Fprintf(&b, "| %d | %s |", i+1, formatRowTitle(title))

		for _, c := range columns {
			cell, ok := row.Cells[c]
			if !ok || cell.Result == "" {
				counts["absent"]++
				fmt.Fprintf(&b, " %s |", t.config.EmojiAbsent)
				continue
			}
			counts[cell.Result]++

			emoji := emojiFor(cell.Result, t.config)
			footRef := ""
			if cell.Result == "partial" || cell.Result == "fail" {
				footRef = footnoteRef(cell.Note)
			}
			fmt.Fprintf(&b, " %s%s |", emoji, footRef)
		}
		b.WriteString("\n")
	}

	if t.config.IncludeFootnotes && len(footnotes) > 0 {
		b.WriteString("\n")
		for _, fn := range footnotes {
			fmt.Fprintf(&b, "%d. %s\n", fn.idx, fn.text)
		}
	}

	if t.config.IncludeLegend {
		b.WriteString("\n**Legend:** ")
		fmt.Fprintf(&b, "%s pass · %s partial · %s fail · %s skipped · %s absent\n",
			t.config.EmojiPass, t.config.EmojiPartial, t.config.EmojiFail, t.config.EmojiSkipped, t.config.EmojiAbsent)
	}

	return b.String(), counts, counts["fail"]
}

// formatRowTitle renders the row title with the leading HTTP-method+path
// (or SSE topic) wrapped in code-ticks and any trailing descriptor kept
// outside the code span. Falls back to the raw title when no path is
// detected.
func formatRowTitle(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return title
	}
	// SSE row, e.g. "SSE execution_payload"
	if strings.HasPrefix(t, "SSE ") {
		parts := strings.SplitN(t, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " `" + parts[1] + "`"
		}
	}
	// HTTP row: "<METHOD> /path[ (suffix)]"
	for _, m := range []string{"GET", "POST", "PUT", "DELETE", "PATCH"} {
		prefix := m + " "
		if strings.HasPrefix(t, prefix) {
			rest := t[len(prefix):]
			suffixStart := strings.Index(rest, " (")
			if suffixStart < 0 {
				return "`" + t + "`"
			}
			pathPart := rest[:suffixStart]
			suffix := rest[suffixStart:]
			return "`" + m + " " + pathPart + "`" + suffix
		}
	}
	return "`" + t + "`"
}

func emojiFor(result string, cfg Config) string {
	switch result {
	case "pass":
		return cfg.EmojiPass
	case "partial":
		return cfg.EmojiPartial
	case "fail":
		return cfg.EmojiFail
	case "skipped":
		return cfg.EmojiSkipped
	}
	return cfg.EmojiAbsent
}

func superscript(n int) string {
	digits := []rune("⁰¹²³⁴⁵⁶⁷⁸⁹")
	if n == 0 {
		return string(digits[0])
	}
	var out []rune
	for n > 0 {
		out = append([]rune{digits[n%10]}, out...)
		n /= 10
	}
	return string(out)
}

// storeResults writes the rendered markdown into the DB as both a "summary"
// (single, shown inline in the UI) and as a result file (matrix.md +
// matrix.json) downloadable from the task pane.
func (t *Task) storeResults(markdown string, jsonBytes []byte) {
	database := t.ctx.Scheduler.GetServices().Database()
	runID := t.ctx.Scheduler.GetTestRunID()
	taskID := uint64(t.ctx.Index)

	err := database.RunTransaction(func(tx *sqlx.Tx) error {
		if err := database.UpsertTaskResult(tx, &db.TaskResult{
			RunID:  runID,
			TaskID: taskID,
			Type:   "summary",
			Index:  0,
			Name:   "matrix.md",
			Size:   uint64(len(markdown)),
			Data:   []byte(markdown),
		}); err != nil {
			return fmt.Errorf("store summary: %w", err)
		}
		if err := database.UpsertTaskResult(tx, &db.TaskResult{
			RunID:  runID,
			TaskID: taskID,
			Type:   "result",
			Index:  0,
			Name:   "matrix.md",
			Size:   uint64(len(markdown)),
			Data:   []byte(markdown),
		}); err != nil {
			return fmt.Errorf("store markdown: %w", err)
		}
		if err := database.UpsertTaskResult(tx, &db.TaskResult{
			RunID:  runID,
			TaskID: taskID,
			Type:   "result",
			Index:  1,
			Name:   "matrix.json",
			Size:   uint64(len(jsonBytes)),
			Data:   jsonBytes,
		}); err != nil {
			return fmt.Errorf("store json: %w", err)
		}
		return nil
	})
	if err != nil {
		t.logger.WithError(err).Error("failed storing matrix result files")
	}
}
