package utils

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type jsonFixture struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestWriteAndReadJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "data.json")
	if err := WriteJSONToFile(path, jsonFixture{Name: "first", Count: 1}); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteJSONToFile(path, jsonFixture{Name: "second", Count: 2}); err != nil {
		t.Fatal(err)
	}

	var got jsonFixture
	if err := ReadJSONFromFile(path, &got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "second" || got.Count != 2 {
		t.Fatalf("read data = %#v", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "\n  \"name\"") {
		t.Fatalf("JSON is not indented: %s", raw)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("JSON permissions = %v, %v; want 0600", info, err)
	}
	if info, err := os.Stat(filepath.Dir(path)); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("directory permissions = %v, %v; want 0700", info, err)
	}
	tempFiles, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".chat-data-*.tmp"))
	if err != nil || len(tempFiles) != 0 {
		t.Fatalf("temporary files leaked: %#v, %v", tempFiles, err)
	}
}

func TestWriteJSONEncodingErrorLeavesNoDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := WriteJSONToFile(path, map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("unsupported JSON value must fail")
	}
	if exists, err := CheckFileExists(path); err != nil || exists {
		t.Fatalf("failed write created destination: exists=%v err=%v", exists, err)
	}
	tempFiles, _ := filepath.Glob(filepath.Join(dir, ".chat-data-*.tmp"))
	if len(tempFiles) != 0 {
		t.Fatalf("temporary files leaked: %#v", tempFiles)
	}
}

func TestWriteJSONReportsPostCommitDurabilityError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := WriteJSONToFile(path, jsonFixture{Name: "old"}); err != nil {
		t.Fatal(err)
	}
	syncErr := errors.New("directory sync failed")
	err := writeJSONToFile(path, jsonFixture{Name: "new"}, func(string) error { return syncErr })
	var durabilityErr *PostCommitDurabilityError
	if !errors.As(err, &durabilityErr) || !errors.Is(err, syncErr) {
		t.Fatalf("error = %v, want PostCommitDurabilityError", err)
	}
	var got jsonFixture
	if readErr := ReadJSONFromFile(path, &got); readErr != nil {
		t.Fatal(readErr)
	}
	if got.Name != "new" {
		t.Fatalf("visible file = %#v, want committed replacement", got)
	}
}

func TestReadJSONErrors(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	var value jsonFixture
	if err := ReadJSONFromFile(missing, &value); err == nil || !strings.Contains(err.Error(), "открытии") {
		t.Fatalf("missing file error = %v", err)
	}
	broken := filepath.Join(dir, "broken.json")
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ReadJSONFromFile(broken, &value); err == nil || !strings.Contains(err.Error(), "разборе JSON") {
		t.Fatalf("broken JSON error = %v", err)
	}
}

func TestCheckFileExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if exists, err := CheckFileExists(path); err != nil || exists {
		t.Fatalf("missing: exists=%v err=%v", exists, err)
	}
	if err := os.WriteFile(path, []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	if exists, err := CheckFileExists(path); err != nil || !exists {
		t.Fatalf("existing: exists=%v err=%v", exists, err)
	}
}
