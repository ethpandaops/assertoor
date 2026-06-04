package runpython

import (
	"fmt"
	"os"
)

// resultFile is a tiny helper that owns a path inside the task's temp
// dir and reads its contents back at cleanup.
type resultFile struct {
	filePath string
}

func newResultFile(filePath string) (*resultFile, error) {
	if err := os.WriteFile(filePath, []byte{}, 0o600); err != nil {
		return nil, fmt.Errorf("failed to create result file: %w", err)
	}

	return &resultFile{filePath: filePath}, nil
}

func (f *resultFile) FilePath() string {
	return f.filePath
}

func (f *resultFile) Cleanup() ([]byte, error) {
	data, err := os.ReadFile(f.filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	if err := os.Remove(f.filePath); err != nil {
		return nil, fmt.Errorf("failed to remove temp file: %w", err)
	}

	return data, nil
}
