package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// backupFile creates a timestamped backup of a file before overwriting.
func backupFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // file doesn't exist, nothing to back up
	}
	backupPath := fmt.Sprintf("%s.prism-backup-%s", path, time.Now().Format("20060102-150405"))
	return os.WriteFile(backupPath, data, 0o644)
}

// readJSONOrWarn reads a JSON file into a map. If the file exists but is corrupt,
// it prints a warning to stderr and returns an empty map.
func readJSONOrWarn(path string) map[string]interface{} {
	result := make(map[string]interface{})
	data, err := os.ReadFile(path)
	if err != nil {
		return result
	}
	if err := json.Unmarshal(data, &result); err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s is corrupt (%v), backing up and overwriting\n", path, err)
		_ = backupFile(path)
		return make(map[string]interface{})
	}
	return result
}

func atomicWriteJSON(path string, v interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	data = append(data, '\n')
	return atomicWrite(path, data, 0o644)
}

func atomicWrite(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".prism-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}
