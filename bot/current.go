package bot

import (
	"errors"
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"
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
		deadline = current.Deadline.In(lnb.location).Format(DATE_LAYOUT)
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
	date, err := parseDeadlineDate(update.Message.Text, time.Now(), lnb.location)
	if err != nil {
		if errors.Is(err, errDeadlineInPast) {
			lnb.SendPlainMessage(chatID, "Дедлайн не может быть в прошлом.")
			return
		}
		lnb.SendPlainMessage(chatID, "Не удалось разобрать дату. Используйте формат дд.мм.гггг, например 11.02.2027.")
		return
	}
	current.Deadline = &date
	if err := lnb.commitChatData(chatID, data, logger); err != nil {
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
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Обсудили", GetCallbackParamStr(CBCurrentMarkCompleted, bookID))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🚫 Не дочитали / бросили", GetCallbackParamStr(CBCurrentMarkUnfinished, bookID))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBCancel))),
	}
}

func unfinishedReasonButtons(bookID string) [][]tgbotapi.InlineKeyboardButton {
	button := func(text string, code string) tgbotapi.InlineKeyboardButton {
		return tgbotapi.NewInlineKeyboardButtonData(text, GetCallbackParamStr(CBCurrentUnfinishedReason, bookID, code))
	}
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(button("Не увлекла", chatdata.UnfinishedReasonNotEngaging)),
		tgbotapi.NewInlineKeyboardRow(button("Слишком тяжело читается", chatdata.UnfinishedReasonTooDifficult)),
		tgbotapi.NewInlineKeyboardRow(button("Не подошла для клуба", chatdata.UnfinishedReasonNotForClub)),
		tgbotapi.NewInlineKeyboardRow(button("Не успели и не хотим продолжать", chatdata.UnfinishedReasonNoTime)),
		tgbotapi.NewInlineKeyboardRow(button("Другая причина", chatdata.UnfinishedReasonOther)),
		tgbotapi.NewInlineKeyboardRow(button("Без причины", "none")),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBCancel))),
	}
}

func (lnb *LitNightBot) showUnfinishedReasonChoices(message *tgbotapi.Message, expectedID string) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	current := data.CurrentBook()
	if current == nil || !strings.EqualFold(current.ID, expectedID) {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Эта кнопка устарела: текущая книга уже изменилась.", nil)
		return
	}
	lnb.editMessage(message.Chat.ID, message.MessageID, "Почему решили не дочитывать «"+current.DisplayName()+"»?", unfinishedReasonButtons(current.ID))
}

func (lnb *LitNightBot) finishCurrentBook(chatID int64, messageID int, expectedID string, status chatdata.BookStatus, logger *logrus.Entry) {
	_ = lnb.finishCurrentBookWithReason(chatID, messageID, expectedID, status, nil, logger)
}

func (lnb *LitNightBot) finishCurrentBookWithReason(chatID int64, messageID int, expectedID string, status chatdata.BookStatus, reason *chatdata.UnfinishedReason, logger *logrus.Entry) bool {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book, err := data.FinishCurrentBookWithReason(expectedID, status, reason, time.Now())
	if err != nil {
		logger.WithError(err).Warn("Failed to finish current book")
		lnb.editMessage(chatID, messageID, "Эта кнопка устарела: текущая книга уже изменилась.", nil)
		return false
	}
	if err := lnb.commitChatData(chatID, data, logger); err != nil {
		logger.WithError(err).Error("Failed to save final book status")
		lnb.editMessage(chatID, messageID, "Не удалось сохранить новый статус книги. Попробуйте ещё раз.", nil)
		return false
	}
	if status == chatdata.StatusCompleted {
		private, userID := ratingChatContext(data, chatID)
		lnb.editHTMLMessage(chatID, messageID, renderRatingPanelForChat(book, true, private, userID), ratingPanelButtonsForChat(book, private))
		return true
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📖 Открыть карточку", GetCallbackParamStr(CBBookShowReplacing, book.ID)),
		tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
	)}
	text := fmt.Sprintf("🚫 Книга «%s» отмечена как недочитанная и добавлена в историю.", book.DisplayName())
	if reasonText := reason.DisplayText(); reasonText != "" {
		text += "\nПричина: " + reasonText
	}
	lnb.editMessage(chatID, messageID, text, buttons)
	return true
}

func unfinishedReasonReplyConfig(chatID int64, user *tgbotapi.User, bookID string, sourceMessageID int) tgbotapi.MessageConfig {
	userID := int64(0)
	if user != nil {
		userID = user.ID
	}
	text := fmt.Sprintf("Напишите общую причину одним сообщением.\n\nunfinished_reason:%s:%d:%d", bookID, userID, sourceMessageID)
	return selectiveForceReplyConfig(chatID, user, text)
}

func parseUnfinishedReasonPrompt(text string) (string, int64, int, bool) {
	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(strings.TrimSpace(line), ":")
		if len(parts) != 4 || parts[0] != "unfinished_reason" {
			continue
		}
		userID, userErr := strconv.ParseInt(parts[2], 10, 64)
		sourceMessageID, sourceErr := strconv.Atoi(parts[3])
		if parts[1] == "" || userErr != nil || sourceErr != nil {
			return "", 0, 0, false
		}
		return parts[1], userID, sourceMessageID, true
	}
	return "", 0, 0, false
}

func (lnb *LitNightBot) chooseUnfinishedReason(update *tgbotapi.Update, params []string, logger *logrus.Entry) {
	if len(params) < 2 || update.CallbackQuery.From == nil {
		return
	}
	bookID, code := params[0], params[1]
	current := lnb.iocd.GetOrCreateChatData(update.CallbackQuery.Message.Chat.ID).CurrentBook()
	if current == nil || !strings.EqualFold(current.ID, bookID) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Текущая книга уже изменилась"))
		return
	}
	if code == chatdata.UnfinishedReasonOther {
		request := unfinishedReasonReplyConfig(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.From, bookID, update.CallbackQuery.Message.MessageID)
		if _, err := lnb.bot.Send(request); err != nil {
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось открыть ввод причины"))
			return
		}
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Жду причину отдельным сообщением"))
		return
	}
	var reason *chatdata.UnfinishedReason
	if code != "none" {
		var err error
		reason, err = chatdata.NewUnfinishedReason(code, "")
		if err != nil {
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Некорректная причина"))
			return
		}
	}
	lnb.finishCurrentBookWithReason(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID, bookID, chatdata.StatusUnfinished, reason, logger)
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

func (lnb *LitNightBot) handleUnfinishedReasonReply(message *tgbotapi.Message, original string, logger *logrus.Entry) bool {
	bookID, expectedUserID, sourceMessageID, ok := parseUnfinishedReasonPrompt(original)
	if !ok {
		return false
	}
	if message.From == nil || message.From.ID != expectedUserID {
		lnb.SendPlainMessage(message.Chat.ID, "Этот запрос причины адресован другому участнику.")
		return true
	}
	reason, err := chatdata.NewUnfinishedReason(chatdata.UnfinishedReasonOther, message.Text)
	if err != nil {
		lnb.SendPlainMessage(message.Chat.ID, err.Error())
		return true
	}
	if lnb.finishCurrentBookWithReason(message.Chat.ID, sourceMessageID, bookID, chatdata.StatusUnfinished, reason, logger) && message.ReplyToMessage != nil {
		lnb.removeMessage(message.Chat.ID, message.ReplyToMessage.MessageID)
	}
	return true
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
	now := time.Now().In(lnb.location)
	deadline := automaticDeadline(now, lnb.location)
	book.Status = chatdata.StatusReading
	book.StartedAt = &now
	book.Deadline = &deadline
	if !lnb.saveChatData(chatID, data, logger) {
		return false
	}
	lnb.SendPlainMessage(chatID, fmt.Sprintf("📖 Текущая книга: «%s»\nАвтоматический дедлайн: %s", book.DisplayName(), deadline.Format(DATE_LAYOUT)))
	_, _ = lnb.sendPendingReviewRequests(chatID, data, time.Now(), false, logger)
	return true
}

var errDeadlineInPast = errors.New("deadline is in the past")

func parseDeadlineDate(input string, now time.Time, location *time.Location) (time.Time, error) {
	if location == nil {
		return time.Time{}, errors.New("application location is required")
	}
	date, err := time.ParseInLocation(DATE_LAYOUT, input, location)
	if err != nil {
		return time.Time{}, err
	}
	now = now.In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if date.Before(today) {
		return time.Time{}, errDeadlineInPast
	}
	return date, nil
}

func automaticDeadline(now time.Time, location *time.Location) time.Time {
	return now.In(location).AddDate(0, 0, 14)
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
