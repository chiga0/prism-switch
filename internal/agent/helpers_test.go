package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	os.WriteFile(path, []byte(`{"key":"value"}`), 0o644)

	if err := backupFile(path); err != nil {
		t.Fatalf("backupFile() error: %v", err)
	}

	// Find the backup file
	entries, _ := os.ReadDir(dir)
	found := false
	for _, e := range entries {
		if e.Name() != "test.json" && len(e.Name()) > len("test.json") {
			found = true
			data, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			if string(data) != `{"key":"value"}` {
				t.Errorf("backup content = %q", string(data))
			}
		}
	}
	if !found {
		t.Error("no backup file created")
	}
}

func TestBackupFileNonExistent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.json")
	if err := backupFile(path); err != nil {
		t.Errorf("backupFile() on nonexistent file should not error, got: %v", err)
	}
}

func TestReadJSONOrWarnValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.json")
	os.WriteFile(path, []byte(`{"env":{"KEY":"val"}}`), 0o644)

	result := readJSONOrWarn(path)
	env, ok := result["env"].(map[string]interface{})
	if !ok {
		t.Fatal("expected env map")
	}
	if env["KEY"] != "val" {
		t.Errorf("KEY = %v", env["KEY"])
	}
}

func TestReadJSONOrWarnCorrupt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.json")
	os.WriteFile(path, []byte(`{invalid json`), 0o644)

	result := readJSONOrWarn(path)
	if len(result) != 0 {
		t.Errorf("expected empty map for corrupt file, got %v", result)
	}

	// Verify backup was created
	entries, _ := os.ReadDir(dir)
	backupFound := false
	for _, e := range entries {
		if e.Name() != "corrupt.json" {
			backupFound = true
		}
	}
	if !backupFound {
		t.Error("backup should be created for corrupt file")
	}
}

func TestReadJSONOrWarnMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	result := readJSONOrWarn(path)
	if len(result) != 0 {
		t.Errorf("expected empty map for missing file, got %v", result)
	}
}

func TestAtomicWriteJSONCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.json")

	data := map[string]string{"hello": "world"}
	if err := atomicWriteJSON(path, data); err != nil {
		t.Fatalf("atomicWriteJSON() error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]string
	json.Unmarshal(content, &parsed)
	if parsed["hello"] != "world" {
		t.Errorf("hello = %q", parsed["hello"])
	}
}

func TestAtomicWriteCreatesDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "file.txt")

	if err := atomicWrite(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("atomicWrite() error: %v", err)
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("perm = %o, want 600", info.Mode().Perm())
	}
}
