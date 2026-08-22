package bot

import (
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestHistorySelectionForceReplyTargetsCallbackUser(t *testing.T) {
	user := &tgbotapi.User{ID: 77, FirstName: "Мария"}
	selection := historySelection{UserID: user.ID, Year: 2025, BookIDs: []string{"one", "two", "three"}}
	request := historySelectionPromptConfig(-100123, selection, user)

	if !strings.Contains(request.Text, "Введите номер книги из списка за 2025 год") {
		t.Fatalf("unexpected prompt: %q", request.Text)
	}
	if len(request.Entities) != 1 || request.Entities[0].User == nil || request.Entities[0].User.ID != user.ID {
		t.Fatalf("unexpected mention: %#v", request.Entities)
	}
	forceReply, ok := request.ReplyMarkup.(tgbotapi.ForceReply)
	if !ok || !forceReply.ForceReply || !forceReply.Selective {
		t.Fatalf("unexpected ForceReply: %#v", request.ReplyMarkup)
	}
}
