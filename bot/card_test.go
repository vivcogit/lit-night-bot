package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestRenderBookCardEscapesHTML(t *testing.T) {
	book := &chatdata.ClubBook{
		ID:      "book0001",
		Title:   "A < B & C",
		Authors: []string{"O'Connor & Co"},
		Status:  chatdata.StatusWishlist,
	}
	text := renderBookCard(book)
	if strings.Contains(text, "A < B") || strings.Contains(text, "Connor & Co") {
		t.Fatalf("unsafe HTML: %s", text)
	}
	if !strings.Contains(text, "A &lt; B &amp; C") {
		t.Fatalf("escaped title missing: %s", text)
	}
}

func TestPersonalBookCardShowsOnlyOwnRating(t *testing.T) {
	book := ratingTestBook()
	text := renderBookCardForChat(book, true, 1)
	if !strings.Contains(text, "⭐ Моя оценка: 8 из 10") || strings.Contains(text, "8,5") || strings.Contains(text, "2 оценки") {
		t.Fatalf("unexpected personal card: %s", text)
	}
	buttons := bookCardButtonsForChat(book, true, 1)
	if len(buttons[0]) != 1 || buttons[0][0].Text != "⭐ Моя оценка: 8" {
		t.Fatalf("unexpected personal rating button: %#v", buttons[0])
	}
}

func TestPersonalBookCardShowsReviewInline(t *testing.T) {
	book := ratingTestBook()
	book.Reviews = []chatdata.Review{{UserID: 1, DisplayName: "Анна", Text: "Важная <книга> & хороший финал"}}
	text := renderBookCardForChat(book, true, 1)
	if !strings.Contains(text, "💬 Мой отзыв:\nВажная &lt;книга&gt; &amp; хороший финал") {
		t.Fatalf("personal review is not rendered safely in card: %s", text)
	}
	labels := buttonLabels(bookCardButtonsForChat(book, true, 1))
	if strings.Contains(labels, "💬 Мой отзыв") || !strings.Contains(labels, "✏️ Изменить отзыв") || !strings.Contains(labels, "🗑 Удалить отзыв") {
		t.Fatalf("personal card still requires a separate review page: %s", labels)
	}
	buttons := bookCardButtonsForChat(book, true, 1)
	action, params, err := GetCallbackParam(*buttons[1][0].CallbackData)
	if err != nil || action != CBReviewWrite || len(params) != 3 || params[2] != reviewSourceCard {
		t.Fatalf("card edit action lost its source context: %q %#v %v", action, params, err)
	}
}

func TestBookFieldForceReplyTargetsCallbackUser(t *testing.T) {
	user := &tgbotapi.User{ID: 88, FirstName: "Ольга"}
	request := bookFieldRequestConfig(-100123, user, "book_title", "book0001", 321, "Введите новое название.")

	if len(request.Entities) != 1 || request.Entities[0].User == nil || request.Entities[0].User.ID != user.ID {
		t.Fatalf("unexpected mention: %#v", request.Entities)
	}
	field, id, sourceMessageID, ok := parseBookFieldPrompt(request.Text)
	if !ok || field != "book_title" || id != "book0001" || sourceMessageID != 321 {
		t.Fatalf("unexpected prompt metadata: %q %q %d %v", field, id, sourceMessageID, ok)
	}
}

func TestBooklistRejectsNegativePage(t *testing.T) {
	books := []chatdata.ClubBook{{ID: "1", Title: "One"}}
	pageBooks, page, last := GetBooklistPage(&books, -10)
	if page != 0 || len(pageBooks) != 1 || !last {
		t.Fatalf("unexpected page: books=%d page=%d last=%v", len(pageBooks), page, last)
	}
}

func TestReviewBookCardAlwaysOpensInPlace(t *testing.T) {
	book := &chatdata.ClubBook{NeedsReview: true}
	if !shouldEditBookCardInPlace(book) {
		t.Fatal("a card awaiting review must replace the review list message")
	}
	book.NeedsReview = false
	if shouldEditBookCardInPlace(book) {
		t.Fatal("a regular card must retain the normal send behavior")
	}
}
