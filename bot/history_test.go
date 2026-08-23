package bot

import (
	"encoding/json"
	chatdata "lit-night-bot/chat-data"
	"os"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func migratedHistoryFixture(t *testing.T) []chatdata.ClubBook {
	t.Helper()
	raw, err := os.ReadFile("../testdata/chat-data-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var legacy chatdata.ChatData
	if err := json.Unmarshal(raw, &legacy); err != nil {
		t.Fatal(err)
	}
	migrated, _, err := chatdata.MigrateV1(&legacy, time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return migrated.BooksWithStatus(chatdata.StatusCompleted)
}

func TestHistoryShowsWholeSelectedYear(t *testing.T) {
	all := migratedHistoryFixture(t)
	years, counts := historyYears(all)
	if len(years) != 3 || years[0] != 2026 || counts[2026] != 10 || counts[2025] != 15 || counts[2024] != 18 {
		t.Fatalf("unexpected years: %#v %#v", years, counts)
	}

	books := historyBooksForYear(all, 2026)
	text := renderHistoryYear(2026, books, len(all), 2026, true)
	if !strings.Contains(text, "<b>📖 2026 · 10 книг</b>") {
		t.Fatalf("year heading missing: %s", text)
	}
	if !strings.Contains(text, "1. ✅ <b>Synthetic 2026-10</b>") || !strings.Contains(text, "10. ✅ <b>Synthetic 2026-01</b>") {
		t.Fatalf("full current year is not rendered: %s", text)
	}
	if !strings.Contains(text, "<b>Архив:</b>") {
		t.Fatal("archive label missing")
	}
	if len([]rune(text)) > 4096 {
		t.Fatalf("current fixture exceeds Telegram message limit: %d", len([]rune(text)))
	}
}

func TestHistoryShowsUnfinishedBooksWithSeparateStatus(t *testing.T) {
	ended := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	books := []chatdata.ClubBook{
		{ID: "done", Title: "Прочитана", Status: chatdata.StatusCompleted, AddedAt: ended, CompletedAt: &ended},
		{ID: "dropped", Title: "Брошена", Status: chatdata.StatusUnfinished, AddedAt: ended, StoppedAt: &ended},
	}
	text := renderHistoryYear(2026, books, 2, 2026, false)
	if !strings.Contains(text, "Всего книг в истории: 2") || !strings.Contains(text, "✅ <b>Прочитана</b>") || !strings.Contains(text, "🚫 <b>Брошена</b>") {
		t.Fatalf("statuses are missing: %s", text)
	}
	data := chatdata.NewChatData()
	data.Books = books
	if got := historyBooks(data); len(got) != 2 {
		t.Fatalf("historyBooks() = %#v", got)
	}
}

func TestHistoryEscapesBookFields(t *testing.T) {
	books := []chatdata.ClubBook{{Title: "A < B", Authors: []string{"X & Y"}, Ratings: []chatdata.Rating{{Value: 8}, {Value: 9}}}}
	text := renderHistoryYear(2026, books, 1, 2026, false)
	if strings.Contains(text, "A < B") || strings.Contains(text, "X & Y") {
		t.Fatalf("unsafe HTML: %s", text)
	}
	if !strings.Contains(text, "⭐ 8,5 · 2 оценки") {
		t.Fatalf("rating is missing: %s", text)
	}
}

func TestPersonalHistoryShowsOnlyOwnRating(t *testing.T) {
	books := []chatdata.ClubBook{{
		Title:  "Книга",
		Status: chatdata.StatusCompleted,
		Ratings: []chatdata.Rating{
			{UserID: 1, DisplayName: "Анна", Value: 7},
			{UserID: 2, DisplayName: "Борис", Value: 10},
		},
	}}
	text := renderHistoryYearForChat(2026, books, 1, 2026, false, true, 1)
	if !strings.Contains(text, "МОЙ ЧИТАТЕЛЬСКИЙ ДНЕВНИК") || !strings.Contains(text, "⭐ Моя оценка: 7") {
		t.Fatalf("unexpected personal history: %s", text)
	}
	if strings.Contains(text, "8,5") || strings.Contains(text, "2 оценки") {
		t.Fatalf("personal history exposes aggregate: %s", text)
	}
}

func TestHistoryRemovalProtectsReviewsUntilConfirmed(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := chatdata.NewChatData()
	data.Chat = &chatdata.ChatMetadata{ID: -42, Type: "group", Title: "Клуб"}
	data.Books = []chatdata.ClubBook{{
		ID: "done0001", Title: "Книга", Status: chatdata.StatusCompleted, CompletedAt: &now,
		ReviewRequestSentAt: &now,
		Reviews:             []chatdata.Review{{ID: "review1", UserID: 1, DisplayName: "Анна", Text: "Отзыв", CreatedAt: now}},
	}}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	message := &tgbotapi.Message{MessageID: 77, Chat: &tgbotapi.Chat{ID: -42, Type: "group"}}

	lnb.handleHistoryRemoveBook(message, "remove", []string{"done0001", "0"}, lnb.logger)

	if storage.GetChatData(-42).FindBook("done0001") == nil {
		t.Fatal("review-only book was deleted without confirmation")
	}
	if texts := recorder.snapshot(); countTextsContaining(texts, "1 отзыв") != 1 {
		t.Fatalf("confirmation does not mention the review: %#v", texts)
	}

	cancel := &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "cancel", Data: GetCallbackParamStr(CBHistoryRemoveCancel, "0"), From: &tgbotapi.User{ID: 1}, Message: message,
	}}
	lnb.handleCallbackQuery(cancel, lnb.logger)
	if storage.GetChatData(-42).FindBook("done0001") == nil {
		t.Fatal("cancel removed the book")
	}

	lnb.confirmHistoryRemoveBook(message, "confirm", []string{"done0001", "0"}, lnb.logger)
	if storage.GetChatData(-42).FindBook("done0001") != nil {
		t.Fatal("confirmed removal kept the book")
	}
}

func TestHistoryRemovalListsMixedAssociatedData(t *testing.T) {
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	book := &chatdata.ClubBook{
		Status:              chatdata.StatusCompleted,
		Ratings:             []chatdata.Rating{{UserID: 1, Value: 8}, {UserID: 2, Value: 9}},
		RatingsClosedAt:     &now,
		ReviewRequestSentAt: &now,
		Reviews:             []chatdata.Review{{UserID: 1}},
		ReviewReminders:     []chatdata.ReviewReminder{{UserID: 2}, {UserID: 3}},
	}
	joined := strings.Join(historyRemovalDetails(book), "|")
	for _, expected := range []string{"2 оценки", "итог сбора оценок", "1 отзыв", "2 напоминания", "данные сбора отзывов"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("removal details miss %q: %s", expected, joined)
		}
	}
}
