package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func ratingTestBook() *chatdata.ClubBook {
	return &chatdata.ClubBook{
		ID:      "book0001",
		Title:   "A < B",
		Authors: []string{"X & Y"},
		Status:  chatdata.StatusCompleted,
		Ratings: []chatdata.Rating{
			{UserID: 2, DisplayName: "Яна <tag>", Value: 9},
			{UserID: 1, DisplayName: "Анна & Борис", Value: 8},
		},
	}
}

func TestRatingFormattingAndPluralization(t *testing.T) {
	if got := formatAverageRating([]chatdata.Rating{{Value: 8}, {Value: 9}}); got != "8,5" {
		t.Fatalf("formatAverageRating() = %q", got)
	}
	for count, want := range map[int]string{1: "1 оценка", 2: "2 оценки", 5: "5 оценок", 11: "11 оценок", 21: "21 оценка", 24: "24 оценки"} {
		if got := ratingCountLabel(count); got != want {
			t.Errorf("ratingCountLabel(%d) = %q, want %q", count, got, want)
		}
	}
	for count, want := range map[int]string{1: "1 участник", 2: "2 участника", 5: "5 участников", 11: "11 участников", 21: "21 участник"} {
		if got := participantCountLabel(count); got != want {
			t.Errorf("participantCountLabel(%d) = %q, want %q", count, got, want)
		}
	}
}

func TestRatingPanelAndButtons(t *testing.T) {
	book := ratingTestBook()
	text := renderRatingPanel(book, false)
	for _, fragment := range []string{"A &lt; B", "X &amp; Y", "8,5 из 10", "2 участника"} {
		if !strings.Contains(text, fragment) {
			t.Errorf("panel does not contain %q: %s", fragment, text)
		}
	}
	buttons := ratingPanelButtons(book)
	if len(buttons) != 6 || len(buttons[0]) != 5 || len(buttons[1]) != 5 {
		t.Fatalf("unexpected button layout: %#v", buttons)
	}
	for _, row := range buttons {
		for _, button := range row {
			if button.CallbackData != nil && len(*button.CallbackData) > 64 {
				t.Fatalf("callback exceeds Telegram limit: %q", *button.CallbackData)
			}
		}
	}
}

func TestPersonalDiaryRatingPanelHidesClubData(t *testing.T) {
	book := ratingTestBook()
	text := renderRatingPanelForChat(book, false, true, 1)
	if !strings.Contains(text, "Личный читательский дневник") || !strings.Contains(text, "Моя оценка: <b>8 из 10</b>") {
		t.Fatalf("unexpected personal panel: %s", text)
	}
	for _, forbidden := range []string{"Средняя оценка", "2 участника", "Яна"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("personal panel exposes club data %q: %s", forbidden, text)
		}
	}
	buttons := ratingPanelButtonsForChat(book, true)
	if len(buttons) != 4 {
		t.Fatalf("personal button rows = %d, want 4", len(buttons))
	}
	for _, row := range buttons {
		for _, button := range row {
			if strings.Contains(button.Text, "Кто как") || strings.Contains(button.Text, "Завершить сбор") {
				t.Fatalf("club action leaked into personal diary: %q", button.Text)
			}
		}
	}
}

func TestPersonalDiaryWithoutRating(t *testing.T) {
	text := renderRatingPanelForChat(ratingTestBook(), false, true, 999)
	if !strings.Contains(text, "пока не поставлена") || strings.Contains(text, "8,5") {
		t.Fatalf("unexpected empty personal rating: %s", text)
	}
}

func TestClosedRatingPanelShowsResultAndReopen(t *testing.T) {
	book := ratingTestBook()
	now := time.Date(2026, 8, 22, 20, 0, 0, 0, time.UTC)
	if err := book.CloseRatings(42, "Анна", now); err != nil {
		t.Fatal(err)
	}
	text := renderRatingPanel(book, false)
	if !strings.Contains(text, "Сбор оценок завершён") || !strings.Contains(text, "Итоговая оценка") || !strings.Contains(text, "2 участника") {
		t.Fatalf("unexpected result: %s", text)
	}
	buttons := ratingPanelButtons(book)
	if len(buttons) != 3 || buttons[1][0].Text != "🔓 Возобновить сбор оценок" {
		t.Fatalf("unexpected result buttons: %#v", buttons)
	}
	for _, row := range buttons {
		for _, button := range row {
			if button.Text == "1" || button.Text == "10" {
				t.Fatalf("rating button remained after close: %#v", buttons)
			}
		}
	}
}

func TestRatingsListEscapesSortsAndPaginates(t *testing.T) {
	book := ratingTestBook()
	for index := 3; index <= 12; index++ {
		book.Ratings = append(book.Ratings, chatdata.Rating{UserID: int64(index), DisplayName: string(rune('A' + index)), Value: index%10 + 1})
	}
	text, page, lastPage := renderRatingsList(book, 0)
	if page != 0 || lastPage != 1 {
		t.Fatalf("unexpected first page: page=%d last=%d text=%s", page, lastPage, text)
	}
	second, page, _ := renderRatingsList(book, 99)
	if page != 1 || second == text || !strings.Contains(text+second, "Анна &amp; Борис") || strings.Contains(text+second, "Яна <tag>") {
		t.Fatalf("pagination did not clamp: page=%d text=%s", page, second)
	}
}

func TestTelegramDisplayName(t *testing.T) {
	if got := telegramDisplayName(&tgbotapi.User{FirstName: " Анна ", LastName: " Иванова "}); got != "Анна Иванова" {
		t.Fatalf("full name = %q", got)
	}
	if got := telegramDisplayName(&tgbotapi.User{UserName: "reader"}); got != "@reader" {
		t.Fatalf("username fallback = %q", got)
	}
	if got := telegramDisplayName(nil); got != "Участник" {
		t.Fatalf("nil fallback = %q", got)
	}
}

func TestRatingTimestampCanBeSerialized(t *testing.T) {
	book := ratingTestBook()
	now := time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC)
	if _, err := book.SetRating(99, "Новый участник", 10, now); err != nil {
		t.Fatal(err)
	}
	if rating := book.RatingByUser(99); rating == nil || rating.CreatedAt.IsZero() || rating.UpdatedAt.IsZero() {
		t.Fatalf("timestamps were not set: %#v", rating)
	}
}
