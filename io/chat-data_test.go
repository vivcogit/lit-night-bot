package io

import (
	chatdata "lit-night-bot/chat-data"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func newTestStorage(t *testing.T) *IoChatData {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	return NewIOChatData(logrus.NewEntry(logger), t.TempDir())
}

type testWriter struct{ t *testing.T }

func (writer testWriter) Write(data []byte) (int, error) {
	writer.t.Log(string(data))
	return len(data), nil
}

func TestSaveBackupAndRestore(t *testing.T) {
	storage := newTestStorage(t)
	data := chatdata.NewChatData()
	data.AddBook("Книга", []string{"Автор"}, chatdata.StatusWishlist, time.Now())
	if err := storage.SaveChatData(42, data); err != nil {
		t.Fatal(err)
	}
	backup, err := storage.BackupChatData(42)
	if err != nil {
		t.Fatal(err)
	}

	data.Books[0].Title = "Изменено"
	if err := storage.SaveChatData(42, data); err != nil {
		t.Fatal(err)
	}
	if err := storage.RestoreChatData(42, backup); err != nil {
		t.Fatal(err)
	}
	restored := storage.GetChatData(42)
	if restored == nil || restored.Books[0].Title != "Книга" {
		t.Fatalf("unexpected restored data: %#v", restored)
	}
}

func TestRatingSurvivesStorageRoundTrip(t *testing.T) {
	storage := newTestStorage(t)
	now := time.Date(2026, time.August, 22, 20, 0, 0, 0, time.UTC)
	data := chatdata.NewChatData()
	book := data.AddBook("Книга", []string{"Автор"}, chatdata.StatusCompleted, now)
	if _, err := book.SetRating(123, "Анна", 9, now); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveChatData(42, data); err != nil {
		t.Fatal(err)
	}

	reloaded := storage.GetChatData(42)
	if reloaded == nil {
		t.Fatal("saved chat was not reloaded")
	}
	rating := reloaded.Books[0].RatingByUser(123)
	if rating == nil || rating.Value != 9 || rating.DisplayName != "Анна" || !rating.CreatedAt.Equal(now) || !rating.UpdatedAt.Equal(now) {
		t.Fatalf("unexpected reloaded rating: %#v", rating)
	}
}

func TestUnfinishedStatusSurvivesStorageRoundTrip(t *testing.T) {
	storage := newTestStorage(t)
	now := time.Date(2026, time.August, 22, 20, 0, 0, 0, time.UTC)
	data := chatdata.NewChatData()
	book := data.AddBook("Не дочитали", nil, chatdata.StatusReading, now.Add(-time.Hour))
	if _, err := data.FinishCurrentBook(book.ID, chatdata.StatusUnfinished, now); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveChatData(42, data); err != nil {
		t.Fatal(err)
	}

	reloaded := storage.GetChatData(42)
	if reloaded == nil || len(reloaded.Books) != 1 || reloaded.Books[0].Status != chatdata.StatusUnfinished || reloaded.Books[0].StoppedAt == nil || !reloaded.Books[0].StoppedAt.Equal(now) || reloaded.Books[0].CompletedAt != nil {
		t.Fatalf("unexpected unfinished book: %#v", reloaded)
	}
}
