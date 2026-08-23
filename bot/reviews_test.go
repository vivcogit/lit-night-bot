package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func reviewTestBook() *chatdata.ClubBook {
	now := time.Date(2026, time.August, 23, 18, 0, 0, 0, time.UTC)
	return &chatdata.ClubBook{
		ID: "book0001", Title: "Книга <1>", Authors: []string{"Автор & Co"}, Status: chatdata.StatusCompleted,
		ReviewRequestSentAt: &now,
		Reviews:             []chatdata.Review{{ID: "one", UserID: 1, DisplayName: "Анна <tag>", Text: "Хорошая & важная книга"}},
	}
}

func TestReviewRequestContainsPromptsAndActions(t *testing.T) {
	if reviewRequestDelay != 15*time.Minute {
		t.Fatalf("review request delay = %s, want 15m", reviewRequestDelay)
	}
	book := reviewTestBook()
	text := renderReviewRequest(book)
	for _, fragment := range []string{"Книга &lt;1&gt;", "Что больше всего", "За что вы поставили", "Кому вы посоветуете"} {
		if !strings.Contains(text, fragment) {
			t.Errorf("review request does not contain %q: %s", fragment, text)
		}
	}
	buttons := reviewRequestButtons(book.ID)
	if len(buttons) != 3 || buttons[0][0].Text != "✍️ Написать отзыв" || buttons[1][0].Text != "⏰ Напомнить завтра об отзыве" {
		t.Fatalf("unexpected review buttons: %#v", buttons)
	}
}

func TestReviewReplyPromptIsSelectiveAndBoundToUser(t *testing.T) {
	user := &tgbotapi.User{ID: 77, FirstName: "Анна"}
	config := reviewReplyConfig(-100, reviewTestBook(), user)
	bookID, userID, ok := parseReviewPrompt(config.Text)
	if !ok || bookID != "book0001" || userID != 77 {
		t.Fatalf("prompt was not bound to book and user: %q", config.Text)
	}
	forceReply, ok := config.ReplyMarkup.(tgbotapi.ForceReply)
	if !ok || !forceReply.Selective {
		t.Fatalf("review prompt is not selective: %#v", config.ReplyMarkup)
	}
}

func TestPersonalReviewPromptUsesReadingQuestions(t *testing.T) {
	user := &tgbotapi.User{ID: 77, FirstName: "Анна"}
	config := reviewReplyConfigForChat(77, reviewTestBook(), user, true)
	for _, fragment := range []string{"после чтения", "За что вы поставили", "Кому вы посоветуете"} {
		if !strings.Contains(config.Text, fragment) {
			t.Fatalf("personal review prompt misses %q: %s", fragment, config.Text)
		}
	}
}

func TestReviewReminderMentionsParticipantInGroup(t *testing.T) {
	book := reviewTestBook()
	reminder := chatdata.ReviewReminder{UserID: 55, DisplayName: "Борис & Co"}
	text := renderReviewReminder(book, reminder)
	if !strings.Contains(text, `href="tg://user?id=55"`) || !strings.Contains(text, "Борис &amp; Co") {
		t.Fatalf("reminder has no safe Telegram mention: %s", text)
	}
	buttons := reviewRequestButtonsForUser(book.ID, reminder.UserID)
	action, params, err := GetCallbackParam(*buttons[0][0].CallbackData)
	if err != nil || action != CBReviewWrite || len(params) != 2 || params[1] != "55" {
		t.Fatalf("reminder action is not bound to participant: %q %#v %v", action, params, err)
	}
}

func TestPersonalReviewReminderDoesNotMentionUser(t *testing.T) {
	book := reviewTestBook()
	reminder := chatdata.ReviewReminder{UserID: 55, DisplayName: "Борис"}
	text := renderReviewReminderForChat(book, reminder, true)
	if strings.Contains(text, "tg://user") || !strings.HasPrefix(text, "Напоминаю об отзыве") {
		t.Fatalf("unexpected personal reminder: %s", text)
	}
}

func TestPersonalRatingPanelCombinesRatingAndReview(t *testing.T) {
	book := reviewTestBook()
	book.Reviews = nil
	text := renderPersonalRatingPanel(book, true, 55)
	if !strings.Contains(text, "Моя оценка") || !strings.Contains(text, "Мой отзыв") {
		t.Fatalf("personal completion panel is incomplete: %s", text)
	}
	buttons := personalRatingPanelButtons(book)
	var labels []string
	for _, row := range buttons {
		for _, button := range row {
			labels = append(labels, button.Text)
		}
	}
	joined := strings.Join(labels, "|")
	if !strings.Contains(joined, "✍️ Написать отзыв") || !strings.Contains(joined, "⏰ Напомнить завтра об отзыве") {
		t.Fatalf("personal review actions are missing: %s", joined)
	}
}

func TestCompletingPersonalBookOpensReviewWithoutDelayedRequest(t *testing.T) {
	lnb, storage, _ := newReviewIntegrationBot(t)
	data := chatdata.NewChatData()
	data.Chat = &chatdata.ChatMetadata{ID: 42, Type: "private", Title: "Личный дневник"}
	data.Books = []chatdata.ClubBook{{ID: "book0001", Title: "Книга", Status: chatdata.StatusReading}}
	if err := storage.SaveChatData(42, data); err != nil {
		t.Fatal(err)
	}
	if !lnb.finishCurrentBookWithReason(42, 10, "book0001", chatdata.StatusCompleted, nil, lnb.logger) {
		t.Fatal("personal book was not completed")
	}
	book := storage.GetChatData(42).FindBook("book0001")
	if !book.ReviewCollectionOpen() || book.ReviewCollectionOpenedAt == nil {
		t.Fatalf("personal review collection was not opened: %#v", book)
	}
	if book.ReviewRequestDueAt != nil || book.ReviewRequestSentAt != nil {
		t.Fatalf("personal chat scheduled a delayed group request: %#v", book)
	}
}

func TestRenderReviewsEscapesUserContent(t *testing.T) {
	text, page, lastPage := renderReviewsPage(reviewTestBook(), 0)
	if page != 0 || lastPage != 0 {
		t.Fatalf("unexpected pagination: page=%d last=%d", page, lastPage)
	}
	for _, fragment := range []string{"Анна &lt;tag&gt;", "Хорошая &amp; важная"} {
		if !strings.Contains(text, fragment) {
			t.Errorf("review list does not contain %q: %s", fragment, text)
		}
	}
	if strings.Contains(text, "Анна <tag>") {
		t.Fatalf("review list contains unsafe HTML: %s", text)
	}
}

func TestReviewsArePaginatedOnePerTelegramMessage(t *testing.T) {
	book := reviewTestBook()
	book.Reviews = append(book.Reviews,
		chatdata.Review{ID: "two", UserID: 2, DisplayName: "Борис", Text: strings.Repeat("б", 3000)},
		chatdata.Review{ID: "three", UserID: 3, DisplayName: "Вера", Text: "Третий отзыв"},
	)
	text, page, lastPage := renderReviewsPage(book, 1)
	if page != 1 || lastPage != 2 || !strings.Contains(text, "Борис") || strings.Contains(text, "Анна") || !strings.Contains(text, "Отзыв 2 из 3") {
		t.Fatalf("unexpected second review page: page=%d last=%d text=%s", page, lastPage, text)
	}
	buttons := reviewListButtons(book.ID, page, lastPage)
	if len(buttons) != 2 || len(buttons[0]) != 2 {
		t.Fatalf("pagination buttons are missing: %#v", buttons)
	}
	_, clampedPage, _ := renderReviewsPage(book, 99)
	if clampedPage != 2 {
		t.Fatalf("page was not clamped: %d", clampedPage)
	}
}

func TestUTF16TruncationReservesSpaceForEllipsis(t *testing.T) {
	got := truncateUTF16(strings.Repeat("😀", 10), 7)
	if len([]rune(got)) != 4 || !strings.HasSuffix(got, "…") {
		t.Fatalf("unexpected truncation: %q", got)
	}
}

func TestRatingResultNextBookButtonsRespectEmptyWishlist(t *testing.T) {
	book := reviewTestBook()
	empty := chatdata.NewChatData()
	empty.Books = append(empty.Books, *book)
	buttons := ratingResultButtonsWithNextBook(book, empty)
	if buttons[0][0].Text != "➕ Добавить в вишлист" {
		t.Fatalf("empty wishlist action = %q", buttons[0][0].Text)
	}
	if text := renderRatingResultWithNextBook(book, empty); !strings.Contains(text, "Вишлист пуст") {
		t.Fatalf("empty wishlist is not explained: %s", text)
	}

	withWishlist := chatdata.NewChatData()
	withWishlist.Books = append(withWishlist.Books, *book, chatdata.ClubBook{ID: "wish0001", Title: "Другая", Status: chatdata.StatusWishlist})
	buttons = ratingResultButtonsWithNextBook(book, withWishlist)
	if buttons[0][0].Text != "🎲 Случайная книга" || buttons[1][0].Text != "📘 Выбрать книгу" {
		t.Fatalf("wishlist choice actions are missing: %#v", buttons)
	}
	if text := renderRatingResultWithNextBook(book, withWishlist); !strings.Contains(text, "случайно или из вишлиста") {
		t.Fatalf("next-book choice is not explained: %s", text)
	}
	for _, row := range buttons[:2] {
		for _, button := range row {
			if button.Text == "➕ Добавить в вишлист" {
				t.Fatal("add button must be hidden for non-empty wishlist")
			}
		}
	}
}
