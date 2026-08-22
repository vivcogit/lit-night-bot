package bot

import (
	chatdata "lit-night-bot/chat-data"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func TestChatIDFromUpdate(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	update := &tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: -100123}}}
	if id, ok := chatIDFromUpdate(update, logger); !ok || id != -100123 {
		t.Fatalf("chatIDFromUpdate() = %d, %v", id, ok)
	}
	if id, ok := chatIDFromUpdate(&tgbotapi.Update{}, logger); ok || id != 0 {
		t.Fatalf("empty update = %d, %v", id, ok)
	}
}

func TestMenuDependsOnCurrentBook(t *testing.T) {
	data := chatdata.NewChatData()
	empty := getCurrentBookMenu(data)
	if len(empty) != 2 || empty[0][0].Text != "🎲 Случайная книга" {
		t.Fatalf("empty current menu = %#v", empty)
	}
	data.Books = append(data.Books, chatdata.ClubBook{ID: "current", Title: "Book", Status: chatdata.StatusReading})
	current := getCurrentBookMenu(data)
	if len(current) != 4 || current[0][0].Text != "📖 Текущая книга" {
		t.Fatalf("current menu = %#v", current)
	}
}

func TestSelectiveForceReplyWithoutUser(t *testing.T) {
	request := selectiveForceReplyConfig(42, nil, "введите ответ")
	if request.Text != "Участник, введите ответ" || len(request.Entities) != 0 {
		t.Fatalf("request = %#v", request)
	}
	forceReply, ok := request.ReplyMarkup.(tgbotapi.ForceReply)
	if !ok || !forceReply.ForceReply || forceReply.Selective {
		t.Fatalf("ForceReply = %#v", request.ReplyMarkup)
	}
}
