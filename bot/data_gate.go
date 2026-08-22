package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const serverMigrationRequiredText = "🔒 Данные бота ещё не обновлены.\n\nАдминистратору сервера нужно выполнить команду migrate. До этого функции бота недоступны."

func telegramChatMetadata(chat *tgbotapi.Chat) *chatdata.ChatMetadata {
	if chat == nil {
		return nil
	}
	title := strings.TrimSpace(chat.Title)
	if chat.IsPrivate() {
		firstName := strings.Join(strings.Fields(chat.FirstName), " ")
		lastName := strings.Join(strings.Fields(chat.LastName), " ")
		title = strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
		if title == "" && chat.UserName != "" {
			title = "@" + chat.UserName
		}
		if title == "" {
			title = "Личный читательский дневник"
		}
	}
	return &chatdata.ChatMetadata{ID: chat.ID, Type: chat.Type, Title: title, Username: chat.UserName}
}

func updateChatMetadata(data *chatdata.ChatData, chat *tgbotapi.Chat) bool {
	metadata := telegramChatMetadata(chat)
	if data == nil || metadata == nil {
		return false
	}
	if data.Chat != nil {
		if metadata.Title == "" {
			metadata.Title = data.Chat.Title
		}
		if metadata.Username == "" {
			metadata.Username = data.Chat.Username
		}
		if *data.Chat == *metadata {
			return false
		}
	}
	data.Chat = metadata
	return true
}

func (lnb *LitNightBot) allowUpdate(update *tgbotapi.Update, logger *logrus.Entry) bool {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return false
	}
	data := lnb.iocd.GetChatData(chatID)
	if data == nil {
		data = chatdata.NewChatData()
		updateChatMetadata(data, update.FromChat())
		lnb.iocd.SetChatData(chatID, data)
		return true
	}
	if !data.IsLegacy() && data.MigrationComplete {
		if updateChatMetadata(data, update.FromChat()) {
			lnb.iocd.SetChatData(chatID, data)
		}
		return true
	}
	if update.CallbackQuery != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Нужна серверная миграция"))
	}
	lnb.SendPlainMessage(chatID, serverMigrationRequiredText)
	return false
}
