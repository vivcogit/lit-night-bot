package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

func (lnb *LitNightBot) handleCurrent(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	current := lnb.iocd.GetOrCreateChatData(chatID).CurrentBook()
	if current == nil {
		lnb.SendPlainMessage(chatID, "Похоже, у вас пока нет выбранной книги.")
		return
	}
	deadline := "не задан"
	if current.Deadline != nil {
		deadline = current.Deadline.Format(DATE_LAYOUT)
	}
	lnb.SendPlainMessage(chatID, fmt.Sprintf("Сейчас вы читаете «%s» 📖\nДедлайн: %s", current.DisplayName(), deadline))
}

func (lnb *LitNightBot) handleCurrentDeadlineNoBook(chatID int64) {
	lnb.SendPlainMessage(chatID, "Сначала выберите текущую книгу, а затем назначьте дедлайн. 📖")
}

func (lnb *LitNightBot) handleCurrentDeadlineRequest(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	if lnb.iocd.GetOrCreateChatData(chatID).CurrentBook() == nil {
		lnb.handleCurrentDeadlineNoBook(chatID)
		return
	}
	lnb.SendPlainMessage(chatID, setDeadlineRequestMessage)
}

func (lnb *LitNightBot) handleCurrentDeadline(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	data := lnb.iocd.GetOrCreateChatData(chatID)
	current := data.CurrentBook()
	if current == nil {
		lnb.handleCurrentDeadlineNoBook(chatID)
		return
	}
	date, err := time.ParseInLocation(DATE_LAYOUT, update.Message.Text, time.Local)
	if err != nil {
		lnb.SendPlainMessage(chatID, "Не удалось разобрать дату. Используйте формат дд.мм.гггг, например 11.02.2027.")
		return
	}
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if date.Before(today) {
		lnb.SendPlainMessage(chatID, "Дедлайн не может быть в прошлом.")
		return
	}
	current.Deadline = &date
	if err := lnb.iocd.SaveChatData(chatID, data); err != nil {
		logger.WithError(err).Error("Failed to save deadline")
		lnb.SendPlainMessage(chatID, "Не удалось сохранить дедлайн. Попробуйте ещё раз.")
		return
	}
	lnb.SendPlainMessage(chatID, fmt.Sprintf("✅ Дедлайн установлен на %s.", date.Format(DATE_LAYOUT)))
}

func (lnb *LitNightBot) handleCurrentComplete(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	current := lnb.iocd.GetOrCreateChatData(chatID).CurrentBook()
	if current == nil {
		lnb.SendPlainMessage(chatID, "Сейчас нет книги в процессе чтения.")
		return
	}
	lnb.sendMessage(chatID, SendMessageParams{
		Text:    fmt.Sprintf("Как завершили чтение «%s»?", current.DisplayName()),
		Buttons: currentCompletionButtons(current.ID),
	})
}

func currentCompletionButtons(bookID string) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Прочитали", GetCallbackParamStr(CBCurrentMarkCompleted, bookID))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚫 Не дочитали / бросили", GetCallbackParamStr(CBCurrentMarkUnfinished, bookID))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBCancel))),
	}
}

func (lnb *LitNightBot) finishCurrentBook(chatID int64, messageID int, expectedID string, status chatdata.BookStatus, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book, err := data.FinishCurrentBook(expectedID, status, time.Now())
	if err != nil {
		logger.WithError(err).Warn("Failed to finish current book")
		lnb.editMessage(chatID, messageID, "Эта кнопка устарела: текущая книга уже изменилась.", nil)
		return
	}
	if err := lnb.iocd.SaveChatData(chatID, data); err != nil {
		logger.WithError(err).Error("Failed to save final book status")
		lnb.editMessage(chatID, messageID, "Не удалось сохранить новый статус книги. Попробуйте ещё раз.", nil)
		return
	}
	if status == chatdata.StatusCompleted {
		private, userID := ratingChatContext(data, chatID)
		lnb.editHTMLMessage(chatID, messageID, renderRatingPanelForChat(book, true, private, userID), ratingPanelButtonsForChat(book, private))
		return
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📖 Открыть карточку", GetCallbackParamStr(CBBookShowReplacing, book.ID)),
		tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
	)}
	lnb.editMessage(chatID, messageID, fmt.Sprintf("🚫 Книга «%s» отмечена как недочитанная и добавлена в историю.", book.DisplayName()), buttons)
}

func (lnb *LitNightBot) handleCurrentRandom(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	data := lnb.iocd.GetOrCreateChatData(chatID)
	if message := lnb.checkCanChooseBook(data); message != "" {
		lnb.SendPlainMessage(chatID, message)
		return
	}
	wishlist := data.BooksWithStatus(chatdata.StatusWishlist)
	lnb.sendProgressJokes(chatID)
	book := wishlist[rand.Intn(len(wishlist))]
	lnb.setCurrentBook(chatID, data, book.ID, logger)
}

func (lnb *LitNightBot) handleCurrentChoose(update *tgbotapi.Update, id string, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	data := lnb.iocd.GetOrCreateChatData(chatID)
	if message := lnb.checkCanChooseBook(data); message != "" {
		lnb.SendPlainMessage(chatID, message)
		return
	}
	book := data.FindBook(id)
	if book == nil || book.Status != chatdata.StatusWishlist {
		lnb.SendPlainMessage(chatID, "Книга больше недоступна для выбора.")
		return
	}
	if !lnb.setCurrentBook(chatID, data, id, logger) {
		return
	}
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		lnb.removeMessage(chatID, update.CallbackQuery.Message.MessageID)
	}
}

func (lnb *LitNightBot) checkCanChooseBook(data *chatdata.ChatData) string {
	if current := data.CurrentBook(); current != nil {
		return fmt.Sprintf("Вы уже читаете «%s». Сначала завершите или отмените её.", current.DisplayName())
	}
	if len(data.BooksWithStatus(chatdata.StatusWishlist)) == 0 {
		return "Вишлист пуст. Сначала добавьте книги."
	}
	return ""
}

func (lnb *LitNightBot) setCurrentBook(chatID int64, data *chatdata.ChatData, id string, logger *logrus.Entry) bool {
	book := data.FindBook(id)
	if book == nil || book.Status != chatdata.StatusWishlist {
		lnb.SendPlainMessage(chatID, "Книга не найдена в вишлисте.")
		return false
	}
	now := time.Now()
	deadline := now.Add(14 * 24 * time.Hour)
	book.Status = chatdata.StatusReading
	book.StartedAt = &now
	book.Deadline = &deadline
	if !lnb.saveChatData(chatID, data, logger) {
		return false
	}
	lnb.SendPlainMessage(chatID, fmt.Sprintf("📖 Текущая книга: «%s»\nАвтоматический дедлайн: %s", book.DisplayName(), deadline.Format(DATE_LAYOUT)))
	return true
}

func (lnb *LitNightBot) handleCurrentAbort(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	current := lnb.iocd.GetOrCreateChatData(chatID).CurrentBook()
	if current == nil {
		lnb.SendPlainMessage(chatID, "Текущей книги нет.")
		return
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🚫 Не дочитали", GetCallbackParamStr(CBCurrentToHistory, current.ID)),
		tgbotapi.NewInlineKeyboardButtonData("🕑 В вишлист", GetCallbackParamStr(CBCurrentToWishlist, current.ID)),
		tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBCancel)),
	)}
	lnb.sendMessage(chatID, SendMessageParams{Text: fmt.Sprintf("Что сделать с «%s»?", current.DisplayName()), Buttons: buttons})
}

func (lnb *LitNightBot) moveCurrentBook(chatID int64, messageID int, expectedID string, moveToHistory bool, logger *logrus.Entry) {
	if moveToHistory {
		lnb.finishCurrentBook(chatID, messageID, expectedID, chatdata.StatusUnfinished, logger)
		return
	}
	data := lnb.iocd.GetOrCreateChatData(chatID)
	current := data.CurrentBook()
	if current == nil || current.ID != expectedID {
		lnb.editMessage(chatID, messageID, "Эта кнопка устарела: текущая книга уже изменилась.", nil)
		return
	}
	name := current.DisplayName()
	current.Deadline = nil
	current.Status = chatdata.StatusWishlist
	current.StartedAt = nil
	if !lnb.saveChatData(chatID, data, logger) {
		return
	}
	destination := "вишлист"
	lnb.editMessage(chatID, messageID, fmt.Sprintf("Книга «%s» перемещена в %s.", name, destination), nil)
}
