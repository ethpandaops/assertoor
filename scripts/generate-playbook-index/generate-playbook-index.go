// generate-playbook-index walks a playbook directory tree and writes a
// machine-readable index that lists every playbook's id, name, description,
// version and tags. Other tools (e.g. the assertoor web UI's remote
// library tab) read the index instead of fetching each playbook
// individually.
//
// Usage:
//
//	go run scripts/generate-playbook-index/generate-playbook-index.go <playbooks-dir>
//
// The index is written to <playbooks-dir>/_index.yaml.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// indexFileName is the name of the generated index file; it is also
// skipped when walking the tree so generation is idempotent.
const indexFileName = "_index.yaml"

// PlaybookHeader is the subset of a playbook YAML we care about for
// indexing. We unmarshal into this rather than the full TestConfig to
// avoid pulling assertoor's package dependencies (and to be resilient
// against future schema additions).
type PlaybookHeader struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Version     string   `yaml:"version"`
	Tags        []string `yaml:"tags"`
	Timeout     string   `yaml:"timeout"`
}

// IndexEntry is the schema written to _index.yaml for each playbook.
type IndexEntry struct {
	File        string   `yaml:"file"`
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description,omitempty"`
	Version     string   `yaml:"version,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
}

// Index is the top-level document written to _index.yaml.
type Index struct {
	Generated time.Time    `yaml:"generated"`
	Playbooks []IndexEntry `yaml:"playbooks"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: generate-playbook-index <playbooks-dir>")
		os.Exit(2)
	}

	root := filepath.Clean(os.Args[1])

	info, err := os.Stat(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a directory\n", root)
		os.Exit(1)
	}

	index, warnings, err := buildIndex(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building index: %v\n", err)
		os.Exit(1)
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "Warning:", w)
	}

	outputFile := filepath.Join(root, indexFileName)
	if err := writeIndex(index, outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated playbook index with %d entries -> %s\n", len(index.Playbooks), outputFile)
}

func buildIndex(root string) (*Index, []string, error) {
	index := &Index{
		Generated: time.Now().UTC().Truncate(time.Second),
		Playbooks: []IndexEntry{},
	}

	warnings := make([]string, 0)

	err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if info.IsDir() {
			return nil
		}

		// Only consider .yaml/.yml files, and skip the index file itself.
		base := info.Name()
		if base == indexFileName {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(base))
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("rel path for %s: %w", path, err)
		}
		// Always use forward slashes in the index for portability across
		// platforms — Windows-built indexes should be readable on Linux.
		relPath = filepath.ToSlash(relPath)

		entry, warn, err := loadEntry(path, relPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", relPath, err))
			return nil
		}

		if warn != "" {
			warnings = append(warnings, fmt.Sprintf("%s: %s", relPath, warn))
		}

		index.Playbooks = append(index.Playbooks, entry)

		return nil
	})
	if err != nil {
		return nil, warnings, err
	}

	// Stable order: sort by file path so diffs stay readable.
	sort.Slice(index.Playbooks, func(i, j int) bool {
		return index.Playbooks[i].File < index.Playbooks[j].File
	})

	return index, warnings, nil
}

func loadEntry(absPath, relPath string) (IndexEntry, string, error) {
	data, err := os.ReadFile(absPath)
	if err != nil {
		return IndexEntry{}, "", fmt.Errorf("read: %w", err)
	}

	var header PlaybookHeader
	if err := yaml.Unmarshal(data, &header); err != nil {
		return IndexEntry{}, "", fmt.Errorf("parse: %w", err)
	}

	if header.ID == "" {
		return IndexEntry{}, "", fmt.Errorf("missing required field 'id'")
	}

	if header.Name == "" {
		return IndexEntry{}, "", fmt.Errorf("missing required field 'name'")
	}

	entry := IndexEntry{
		File:        relPath,
		ID:          header.ID,
		Name:        header.Name,
		Description: strings.TrimRight(header.Description, "\n"),
		Version:     header.Version,
		Tags:        header.Tags,
		Timeout:     header.Timeout,
	}

	// Surface playbooks that are missing the new metadata so they
	// can be enriched. Not a hard failure; the entry still lands in
	// the index so the UI lists them.
	var missing []string
	if header.Description == "" {
		missing = append(missing, "description")
	}

	if header.Version == "" {
		missing = append(missing, "version")
	}

	if len(header.Tags) == 0 {
		missing = append(missing, "tags")
	}

	if len(missing) > 0 {
		return entry, fmt.Sprintf("missing metadata fields: %s", strings.Join(missing, ", ")), nil
	}

	return entry, "", nil
}

func writeIndex(index *Index, outputFile string) error {
	out, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer out.Close()

	header := "# Auto-generated playbook index\n" +
		"# Generated: " + index.Generated.Format(time.RFC3339) + "\n" +
		"# DO NOT EDIT MANUALLY - regenerate via `make generate-playbook-index`.\n\n"
	if _, err := out.WriteString(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	enc := yaml.NewEncoder(out)
	enc.SetIndent(2)

	if err := enc.Encode(index); err != nil {
		return fmt.Errorf("encode: %w", err)
	}

	if err := enc.Close(); err != nil {
		return fmt.Errorf("close encoder: %w", err)
	}

	return nil
}
