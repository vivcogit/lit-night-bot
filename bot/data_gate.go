package bot

import (
	"errors"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const serverMigrationRequiredText = "🔒 Данные бота ещё не обновлены.\n\nАдминистратору сервера нужно выполнить команду migrate. До этого функции бота недоступны."
const newerSchemaRequiredText = "🔒 Данные созданы более новой версией бота.\n\nОбновите приложение перед продолжением."
const cardReviewRequiredText = "⚠️ Сначала проверьте карточки, созданные при миграции. Откройте /menu → «Проверить карточки»."
const dataStorageErrorText = "⚠️ Не удалось сохранить данные. Изменение не применено; попробуйте ещё раз позже."
const dataReadErrorText = "⚠️ Данные чата временно недоступны. Файл не был изменён; обратитесь к администратору."

func (lnb *LitNightBot) saveChatData(chatID int64, data *chatdata.ChatData, logger *logrus.Entry) bool {
	if err := lnb.commitChatData(chatID, data, logger); err != nil {
		logger.WithError(err).Error("Failed to save chat data")
		lnb.SendPlainMessage(chatID, dataStorageErrorText)
		return false
	}
	return true
}

func (lnb *LitNightBot) commitChatData(chatID int64, data *chatdata.ChatData, logger *logrus.Entry) error {
	err := lnb.iocd.SaveChatData(chatID, data)
	if utils.IsPostCommitDurabilityError(err) {
		logger.WithError(err).Error("Chat data was committed, but directory durability is uncertain")
		return nil
	}
	return err
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

func isCardReviewUpdate(data *chatdata.ChatData, update *tgbotapi.Update) bool {
	if data == nil || update == nil {
		return false
	}
	if update.Message != nil {
		if update.Message.IsCommand() {
			command := update.Message.Command()
			return command == "menu" || command == "start"
		}
		if update.Message.ReplyToMessage != nil {
			_, bookID, _, ok := parseBookFieldPrompt(update.Message.ReplyToMessage.Text)
			book := data.FindBook(bookID)
			return ok && book != nil && book.NeedsReview
		}
	}
	if update.CallbackQuery == nil {
		return false
	}
	action, params, err := GetCallbackParam(update.CallbackQuery.Data)
	if err != nil {
		return false
	}
	switch action {
	case CBBooksReview, CBCancel:
		return true
	case CBBookShow, CBBookEditTitle, CBBookEditAuthors, CBBookSwap, CBBookApprove:
		if len(params) == 0 {
			return false
		}
		book := data.FindBook(params[0])
		return book != nil && book.NeedsReview
	default:
		return false
	}
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
		if data.MigrationComplete {
			logger.Error("Current schema contains an invalid legacy migration sentinel")
			lnb.SendPlainMessage(chatID, dataReadErrorText)
			return false
		}
		if data.HasBooksNeedingReview() && !isCardReviewUpdate(data, update) {
			if update.CallbackQuery != nil {
				lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сначала проверьте карточки"))
			}
			lnb.SendPlainMessage(chatID, cardReviewRequiredText)
			return false
		}
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
