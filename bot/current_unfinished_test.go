package bot

import (
	"strings"
	"testing"
	"time"

	chatdata "lit-night-bot/chat-data"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestPresetUnfinishedReasonIsPersisted(t *testing.T) {
	lnb, storage, _ := newReviewIntegrationBot(t)
	data := chatdata.NewChatData()
	data.Books = []chatdata.ClubBook{{ID: "book0001", Title: "Книга", Status: chatdata.StatusReading, AddedAt: time.Now()}}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	update := &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{
		ID: "reason", From: &tgbotapi.User{ID: 1, FirstName: "Анна"},
		Message: &tgbotapi.Message{MessageID: 10, Chat: &tgbotapi.Chat{ID: -42, Type: "group"}},
	}}
	lnb.chooseUnfinishedReason(update, []string{"book0001", chatdata.UnfinishedReasonNotEngaging}, lnb.logger)
	book := storage.GetChatData(-42).FindBook("book0001")
	if book.Status != chatdata.StatusUnfinished || book.UnfinishedReason == nil || book.UnfinishedReason.Code != chatdata.UnfinishedReasonNotEngaging {
		t.Fatalf("preset reason was not persisted: %#v", book)
	}
}

func TestUnfinishedReasonPromptIsBoundToParticipant(t *testing.T) {
	config := unfinishedReasonReplyConfig(-42, &tgbotapi.User{ID: 7, FirstName: "Борис"}, "book0001", 15)
	bookID, userID, sourceMessageID, ok := parseUnfinishedReasonPrompt(config.Text)
	if !ok || bookID != "book0001" || userID != 7 || sourceMessageID != 15 {
		t.Fatalf("prompt did not round-trip: %q", config.Text)
	}
	markup, ok := config.ReplyMarkup.(tgbotapi.ForceReply)
	if !ok || !markup.Selective {
		t.Fatalf("custom reason prompt is not selective: %#v", config.ReplyMarkup)
	}
}

func TestCustomUnfinishedReasonIsEscapedInCard(t *testing.T) {
	reason, err := chatdata.NewUnfinishedReason(chatdata.UnfinishedReasonOther, `<b>не наша</b>`)
	if err != nil {
		t.Fatal(err)
	}
	book := &chatdata.ClubBook{ID: "book0001", Title: "Книга", Status: chatdata.StatusUnfinished, UnfinishedReason: reason}
	card := renderBookCard(book)
	if strings.Contains(card, `<b>не наша</b>`) || !strings.Contains(card, `&lt;b&gt;не наша&lt;/b&gt;`) {
		t.Fatalf("custom reason was not escaped: %s", card)
	}
}

func TestCustomUnfinishedReasonRejectsAnotherParticipant(t *testing.T) {
	lnb, storage, _ := newReviewIntegrationBot(t)
	data := chatdata.NewChatData()
	data.Books = []chatdata.ClubBook{{ID: "book0001", Title: "Книга", Status: chatdata.StatusReading}}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	original := unfinishedReasonReplyConfig(-42, &tgbotapi.User{ID: 7, FirstName: "Борис"}, "book0001", 15).Text
	message := &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: -42, Type: "group"}, From: &tgbotapi.User{ID: 8, FirstName: "Анна"}, Text: "Не понравилась"}
	if !lnb.handleUnfinishedReasonReply(message, original, lnb.logger) {
		t.Fatal("unfinished reason prompt was not recognized")
	}
	if current := storage.GetChatData(-42).CurrentBook(); current == nil || current.ID != "book0001" {
		t.Fatalf("another participant completed the book: %#v", current)
	}
}
