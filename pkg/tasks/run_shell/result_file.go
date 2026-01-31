package runshell

import (
	"fmt"
	"os"
)

type resultFile struct {
	filePath string
}

func newResultFile(filePath string) (*resultFile, error) {
	// Touch the file
	if err := os.WriteFile(filePath, []byte{}, 0o600); err != nil {
		return nil, fmt.Errorf("failed to create result file: %w", err)
	}

	return &resultFile{
		filePath: filePath,
	}, nil
}

func (f *resultFile) FilePath() string {
	return f.filePath
}

func (f *resultFile) Cleanup() ([]byte, error) {
	// Read file contents
	data, err := os.ReadFile(f.filePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read result file: %w", err)
	}

	// Remove temp file
	if err := os.Remove(f.filePath); err != nil {
		return nil, fmt.Errorf("failed to remove temp file: %w", err)
	}

	return data, nil
}
