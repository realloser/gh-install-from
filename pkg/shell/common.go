package shell

import (
	"fmt"
	"os"
	"strings"
)

// appendIfNotPresent appends line to file if marker is not already present
func appendIfNotPresent(filePath, line, marker string) error {
	content, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read file: %w", err)
	}
	text := string(content)
	if strings.Contains(text, marker) {
		return nil // already present, idempotent
	}
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	return nil
}
