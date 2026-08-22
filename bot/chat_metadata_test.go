package bot

import (
	chatdata "lit-night-bot/chat-data"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestTelegramChatMetadata(t *testing.T) {
	private := &tgbotapi.Chat{ID: 42, Type: "private", FirstName: " Анна ", LastName: " Иванова ", UserName: "reader"}
	metadata := telegramChatMetadata(private)
	if metadata.ID != 42 || metadata.Type != "private" || metadata.Title != "Анна Иванова" || metadata.Username != "reader" {
		t.Fatalf("unexpected private metadata: %#v", metadata)
	}

	group := telegramChatMetadata(&tgbotapi.Chat{ID: -42, Type: "supergroup", Title: "Книжный клуб"})
	if group.ID != -42 || group.Type != "supergroup" || group.Title != "Книжный клуб" {
		t.Fatalf("unexpected group metadata: %#v", group)
	}
}

func TestUpdateChatMetadataWritesOnlyChanges(t *testing.T) {
	data := chatdata.NewChatData()
	chat := &tgbotapi.Chat{ID: -42, Type: "group", Title: "Клуб"}
	if !updateChatMetadata(data, chat) || data.Chat == nil {
		t.Fatal("metadata was not added")
	}
	if updateChatMetadata(data, chat) {
		t.Fatal("unchanged metadata was reported as changed")
	}
	chat.Title = "Новое название"
	if !updateChatMetadata(data, chat) || data.Chat.Title != "Новое название" {
		t.Fatalf("changed title was not saved: %#v", data.Chat)
	}
}
