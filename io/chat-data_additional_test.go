package io

import (
	"errors"
	chatdata "lit-night-bot/chat-data"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"testing"
)

func TestGetOrCreateAndListData(t *testing.T) {
	storage := newTestStorage(t)
	created := storage.GetOrCreateChatData(100)
	if created.SchemaVersion != chatdata.CurrentSchemaVersion || created.MigrationComplete {
		t.Fatalf("created data = %#v", created)
	}
	if _, err := os.Stat(storage.GetChatDataFilePath(100)); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(storage.dataPath, "ignored-directory"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storage.dataPath, "200"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := storage.GetDatasList()
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) != 2 || files[0] != "100" || files[1] != "200" {
		t.Fatalf("files = %#v", files)
	}
}

func TestStorageErrors(t *testing.T) {
	storage := newTestStorage(t)
	if _, err := storage.LoadChatData(404); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing chat error = %v, want os.ErrNotExist", err)
	}
	if data := storage.GetChatData(404); data != nil {
		t.Fatalf("missing data = %#v", data)
	}
	if _, err := storage.BackupChatData(404); err == nil {
		t.Fatal("backup of missing chat must fail")
	}
	broken := storage.GetChatDataFilePath(500)
	if err := os.WriteFile(broken, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if data := storage.GetChatData(500); data != nil {
		t.Fatalf("broken data = %#v", data)
	}
	if _, err := storage.LoadChatData(500); err == nil {
		t.Fatal("loading malformed JSON must return an error")
	}
	before, err := os.ReadFile(broken)
	if err != nil {
		t.Fatal(err)
	}
	if data := storage.GetOrCreateChatData(500); data != nil {
		t.Fatalf("malformed JSON was treated as a missing chat: %#v", data)
	}
	after, err := os.ReadFile(broken)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("malformed JSON was overwritten: before %q, after %q", before, after)
	}
	if err := storage.RestoreChatData(500, broken); err == nil {
		t.Fatal("restoring broken backup must fail")
	}
}

func TestConcurrentSavesRemainReadable(t *testing.T) {
	storage := newTestStorage(t)
	var wait sync.WaitGroup
	for index := 0; index < 20; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			data := chatdata.NewChatData()
			data.Books = []chatdata.ClubBook{{ID: "book", Title: string(rune('A' + index)), Status: chatdata.StatusWishlist}}
			if err := storage.SaveChatData(42, data); err != nil {
				t.Errorf("SaveChatData: %v", err)
			}
		}(index)
	}
	wait.Wait()
	data := storage.GetChatData(42)
	if data == nil || len(data.Books) != 1 || data.Books[0].Title == "" {
		t.Fatalf("unreadable final data: %#v", data)
	}
}
