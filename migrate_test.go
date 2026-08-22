package main

import (
	io "io"
	chatio "lit-night-bot/io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func migrationTestStorage(t *testing.T) (*chatio.IoChatData, string) {
	t.Helper()
	dir := t.TempDir()
	raw, err := os.ReadFile("testdata/chat-data-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "42"), raw, 0o640); err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return chatio.NewIOChatData(logrus.NewEntry(logger), dir), dir
}

func TestServerMigrationDryRunAndApply(t *testing.T) {
	storage, dir := migrationTestStorage(t)
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	if err := migrateStoredChats(storage, false, 0, now); err != nil {
		t.Fatal(err)
	}
	if data := storage.GetChatData(42); data == nil || !data.IsLegacy() {
		t.Fatal("dry-run changed the source JSON")
	}
	if backups, _ := filepath.Glob(filepath.Join(dir, "_migration", "backups", "*")); len(backups) != 0 {
		t.Fatalf("dry-run created backups: %#v", backups)
	}

	if err := migrateStoredChats(storage, true, 0, now); err != nil {
		t.Fatal(err)
	}
	migrated := storage.GetChatData(42)
	if migrated == nil || migrated.IsLegacy() || !migrated.MigrationComplete || len(migrated.Books) != 47 {
		t.Fatalf("unexpected migrated data: %#v", migrated)
	}
	needsReview := 0
	for _, book := range migrated.Books {
		if book.NeedsReview {
			needsReview++
		}
	}
	if needsReview != 35 {
		t.Fatalf("needs review = %d, want 35", needsReview)
	}
	backups, _ := filepath.Glob(filepath.Join(dir, "_migration", "backups", "42-*.json"))
	if len(backups) != 1 {
		t.Fatalf("backups = %#v", backups)
	}
	info, err := os.Stat(filepath.Join(dir, "42"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("permissions changed to %o", got)
	}
	if err := migrateStoredChats(storage, true, 0, now); err != nil {
		t.Fatal(err)
	}
}
