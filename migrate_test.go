package main

import (
	"bytes"
	"encoding/json"
	"errors"
	io "io"
	chatdata "lit-night-bot/chat-data"
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

func TestServerMigrationUpgradesV2WithoutLosingBooks(t *testing.T) {
	dir := t.TempDir()
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	storage := chatio.NewIOChatData(logrus.NewEntry(logger), dir)
	v2 := &chatdata.ChatData{
		SchemaVersion: 2, MigrationComplete: true,
		Books: []chatdata.ClubBook{{ID: "book0001", Title: "Книга", Status: chatdata.StatusWishlist}},
	}
	raw, err := json.Marshal(v2)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "42"), raw, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := migrateStoredChats(storage, true, 42, time.Now()); err != nil {
		t.Fatal(err)
	}
	migrated := storage.GetChatData(42)
	if migrated == nil || migrated.SchemaVersion != chatdata.CurrentSchemaVersion || migrated.MigrationComplete || len(migrated.Books) != 1 || migrated.Books[0].Title != "Книга" {
		t.Fatalf("v2 data was not safely migrated: %#v", migrated)
	}
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
	if migrated == nil || migrated.IsLegacy() || migrated.MigrationComplete || len(migrated.Books) != 47 {
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

func TestServerMigrationRefusesConcurrentWriter(t *testing.T) {
	storage, dir := migrationTestStorage(t)
	before, err := os.ReadFile(filepath.Join(dir, "42"))
	if err != nil {
		t.Fatal(err)
	}
	dataLock, err := storage.TryAcquireDataDirectoryLock()
	if err != nil {
		t.Fatal(err)
	}
	err = migrateStoredChats(storage, true, 0, time.Now())
	if !errors.Is(err, chatio.ErrDataDirectoryLocked) {
		t.Fatalf("migration lock error = %v, want ErrDataDirectoryLocked", err)
	}
	after, readErr := os.ReadFile(filepath.Join(dir, "42"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("blocked migration changed the source file")
	}
	if backups, _ := filepath.Glob(filepath.Join(dir, "_migration", "backups", "*")); len(backups) != 0 {
		t.Fatalf("blocked migration created backups: %#v", backups)
	}
	if err := dataLock.Close(); err != nil {
		t.Fatal(err)
	}
	if err := migrateStoredChats(storage, true, 0, time.Now()); err != nil {
		t.Fatalf("migration did not resume after lock release: %v", err)
	}
}

func TestMigrationCommandRequiresStoppedBotConfirmation(t *testing.T) {
	if code := runMigrationCommand([]string{"--apply"}); code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
}
