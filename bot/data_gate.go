package bot

import (
	"errors"
	chatdata "lit-night-bot/chat-data"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const serverMigrationRequiredText = "🔒 Данные бота ещё не обновлены.\n\nАдминистратору сервера нужно выполнить команду migrate. До этого функции бота недоступны."
const newerSchemaRequiredText = "🔒 Данные созданы более новой версией бота.\n\nОбновите приложение перед продолжением."
const dataStorageErrorText = "⚠️ Не удалось сохранить данные. Изменение не применено; попробуйте ещё раз позже."
const dataReadErrorText = "⚠️ Данные чата временно недоступны. Файл не был изменён; обратитесь к администратору."

func (lnb *LitNightBot) saveChatData(chatID int64, data *chatdata.ChatData, logger *logrus.Entry) bool {
	if err := lnb.iocd.SaveChatData(chatID, data); err != nil {
		logger.WithError(err).Error("Failed to save chat data")
		lnb.SendPlainMessage(chatID, dataStorageErrorText)
		return false
	}
	return true
}

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
	data, err := lnb.iocd.LoadChatData(chatID)
	if errors.Is(err, os.ErrNotExist) {
		data = chatdata.NewChatData()
		updateChatMetadata(data, update.FromChat())
		return lnb.saveChatData(chatID, data, logger)
	}
	if err != nil {
		logger.WithError(err).Error("Failed to load chat data")
		lnb.SendPlainMessage(chatID, dataReadErrorText)
		return false
	}
	if data.IsFutureSchema() {
		if update.CallbackQuery != nil {
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Нужно обновить бота"))
		}
		lnb.SendPlainMessage(chatID, newerSchemaRequiredText)
		return false
	}
	if !data.IsLegacy() {
		if updateChatMetadata(data, update.FromChat()) {
			if !lnb.saveChatData(chatID, data, logger) {
				return false
			}
		}
		return true
	}
	if update.CallbackQuery != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Нужна серверная миграция"))
	}
	lnb.SendPlainMessage(chatID, serverMigrationRequiredText)
	return false
}
