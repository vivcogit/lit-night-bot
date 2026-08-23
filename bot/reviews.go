package bot

import (
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const reviewRequestDelay = 15 * time.Minute
const reviewReminderDelay = 24 * time.Hour

func truncateUTF16(text string, maxUnits int) string {
	if maxUnits <= 0 {
		return ""
	}
	if len(utf16.Encode([]rune(text))) <= maxUnits {
		return text
	}
	contentLimit := maxUnits - 1 // reserve one unit for the ellipsis
	units := 0
	for index, value := range text {
		width := len(utf16.Encode([]rune{value}))
		if units+width > contentLimit {
			return strings.TrimSpace(text[:index]) + "…"
		}
		units += width
	}
	return "…"
}

func reviewBookName(book *chatdata.ClubBook) string {
	return truncateUTF16(book.DisplayName(), 200)
}

func renderReviewRequest(book *chatdata.ClubBook) string {
	return "✍️ <b>Короткое послесловие о книге</b>\n\n" +
		"«" + html.EscapeString(reviewBookName(book)) + "»\n\n" +
		"Можно опираться на вопросы:\n" +
		"— Что больше всего осталось с вами после обсуждения?\n" +
		"— За что вы поставили такую оценку?\n" +
		"— Кому вы посоветуете или не посоветуете эту книгу?"
}

func reviewRequestButtons(bookID string) [][]tgbotapi.InlineKeyboardButton {
	return reviewRequestButtonsForUser(bookID, 0)
}

func reviewRequestButtonsForUser(bookID string, userID int64) [][]tgbotapi.InlineKeyboardButton {
	params := []string{bookID}
	if userID > 0 {
		params = append(params, strconv.FormatInt(userID, 10))
	}
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✍️ Написать отзыв", GetCallbackParamStr(CBReviewWrite, params...))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⏰ Напомнить завтра", GetCallbackParamStr(CBReviewRemind, params...))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Не буду писать", GetCallbackParamStr(CBReviewSkip, params...))),
	}
}

func (lnb *LitNightBot) reviewCallbackUserAllowed(update *tgbotapi.Update, params []string) bool {
	if len(params) < 2 {
		return true
	}
	expectedUserID, err := strconv.ParseInt(params[1], 10, 64)
	if err == nil && update.CallbackQuery.From != nil && update.CallbackQuery.From.ID == expectedUserID {
		return true
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это действие для другого участника"))
	return false
}

func reviewReplyConfig(chatID int64, book *chatdata.ClubBook, user *tgbotapi.User) tgbotapi.MessageConfig {
	action := "Напишите отзыв одним сообщением. Необязательно отвечать на все вопросы — достаточно нескольких предложений."
	if user != nil && book.ReviewByUser(user.ID) != nil {
		action = "Отправьте новый текст отзыва одним сообщением. Он заменит сохранённый отзыв."
	}
	userID := int64(0)
	if user != nil {
		userID = user.ID
	}
	return selectiveForceReplyConfig(chatID, user, fmt.Sprintf("%s\n\nreview:%s:%d", action, book.ID, userID))
}

func parseReviewPrompt(text string) (bookID string, userID int64, ok bool) {
	for _, line := range strings.Split(text, "\n") {
		parts := strings.Split(strings.TrimSpace(line), ":")
		if len(parts) != 3 || parts[0] != "review" {
			continue
		}
		parsedUserID, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil || strings.TrimSpace(parts[1]) == "" {
			return "", 0, false
		}
		return parts[1], parsedUserID, true
	}
	return "", 0, false
}

func telegramMentionHTML(userID int64, displayName string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "Участник"
	}
	displayName = truncateUTF16(displayName, 100)
	return fmt.Sprintf(`<a href="tg://user?id=%d">%s</a>`, userID, html.EscapeString(displayName))
}

func reviewManageButtons(bookID string, userID int64) [][]tgbotapi.InlineKeyboardButton {
	params := []string{bookID, strconv.FormatInt(userID, 10)}
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить отзыв", GetCallbackParamStr(CBReviewWrite, params...)),
			tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить отзыв", GetCallbackParamStr(CBReviewDelete, params...)),
		),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel))),
	}
}

func (lnb *LitNightBot) requestReview(update *tgbotapi.Update, bookID string, logger *logrus.Entry) {
	message := update.CallbackQuery.Message
	user := update.CallbackQuery.From
	if user == nil || user.IsBot {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Боты не могут оставлять отзывы"))
		return
	}
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(bookID)
	if book == nil || book.Status != chatdata.StatusCompleted || (book.ReviewRequestSentAt == nil && book.ReviewRequestClaimedAt == nil) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор отзывов для этой книги не открыт"))
		return
	}
	request := reviewReplyConfig(message.Chat.ID, book, user)
	if _, err := lnb.bot.Send(request); err != nil {
		logger.WithError(err).Error("Failed to request review")
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось открыть ввод отзыва"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Жду ваш отзыв отдельным сообщением"))
}

func (lnb *LitNightBot) handleReviewReply(message *tgbotapi.Message, original string, logger *logrus.Entry) bool {
	bookID, expectedUserID, ok := parseReviewPrompt(original)
	if !ok {
		return false
	}
	if message.From == nil || message.From.ID != expectedUserID {
		lnb.SendPlainMessage(message.Chat.ID, "Этот запрос на отзыв адресован другому участнику.")
		return true
	}
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.SendPlainMessage(message.Chat.ID, "Книга не найдена.")
		return true
	}
	updated, err := book.SetReview(message.From.ID, telegramDisplayName(message.From), message.From.UserName, message.Text, time.Now())
	if err != nil {
		lnb.SendPlainMessage(message.Chat.ID, err.Error())
		return true
	}
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		return true
	}
	action := "отзыв сохранён"
	if updated {
		action = "отзыв обновлён"
	}
	lnb.SendHTMLMessage(message.Chat.ID, "✅ "+telegramMentionHTML(message.From.ID, telegramDisplayName(message.From))+", "+action+".", reviewManageButtons(book.ID, message.From.ID))
	if message.ReplyToMessage != nil {
		lnb.removeMessage(message.Chat.ID, message.ReplyToMessage.MessageID)
	}
	return true
}

func (lnb *LitNightBot) scheduleReviewReminder(update *tgbotapi.Update, bookID string, logger *logrus.Entry) {
	message := update.CallbackQuery.Message
	user := update.CallbackQuery.From
	if user == nil || user.IsBot {
		return
	}
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	dueAt := time.Now().Add(reviewReminderDelay)
	if err := book.SetReviewReminder(user.ID, telegramDisplayName(user), user.UserName, dueAt); err != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, err.Error()))
		return
	}
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось сохранить напоминание"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Напомню завтра в этом чате"))
}

func (lnb *LitNightBot) skipReview(update *tgbotapi.Update, bookID string, logger *logrus.Entry) {
	user := update.CallbackQuery.From
	if user == nil {
		return
	}
	chatID := update.CallbackQuery.Message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	if book.CancelReviewReminder(user.ID) && !lnb.saveChatData(chatID, data, logger) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось отменить напоминание"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Хорошо, напоминать не буду"))
}

func (lnb *LitNightBot) deleteReview(update *tgbotapi.Update, bookID string, logger *logrus.Entry) {
	user := update.CallbackQuery.From
	if user == nil {
		return
	}
	chatID := update.CallbackQuery.Message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(bookID)
	if book == nil || !book.DeleteReview(user.ID) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Ваш отзыв уже удалён"))
		return
	}
	if !lnb.saveChatData(chatID, data, logger) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось удалить отзыв"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отзыв удалён"))
	lnb.removeMessage(chatID, update.CallbackQuery.Message.MessageID)
}

func renderReviewsPage(book *chatdata.ClubBook, page int) (string, int, int) {
	var text strings.Builder
	text.WriteString("💬 <b>Отзывы о книге</b>\n")
	text.WriteString("«" + html.EscapeString(reviewBookName(book)) + "»\n\n")
	if len(book.Reviews) == 0 {
		text.WriteString("Отзывов пока нет.")
		return text.String(), 0, 0
	}
	lastPage := len(book.Reviews) - 1
	if page < 0 {
		page = 0
	}
	if page > lastPage {
		page = lastPage
	}
	review := book.Reviews[page]
	displayName := truncateUTF16(review.DisplayName, 100)
	reviewText := truncateUTF16(review.Text, chatdata.MaxReviewTextUTF16Units)
	text.WriteString(fmt.Sprintf("<b>%d. %s</b>\n%s", page+1, html.EscapeString(displayName), html.EscapeString(reviewText)))
	if lastPage > 0 {
		text.WriteString(fmt.Sprintf("\n\n<i>Отзыв %d из %d</i>", page+1, len(book.Reviews)))
	}
	return strings.TrimSpace(text.String()), page, lastPage
}

func reviewListButtons(bookID string, page int, lastPage int) [][]tgbotapi.InlineKeyboardButton {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, 2)
	if lastPage > 0 {
		navigation := make([]tgbotapi.InlineKeyboardButton, 0, 2)
		if page > 0 {
			navigation = append(navigation, tgbotapi.NewInlineKeyboardButtonData("←", GetCallbackParamStr(CBReviewList, bookID, strconv.Itoa(page-1))))
		}
		if page < lastPage {
			navigation = append(navigation, tgbotapi.NewInlineKeyboardButtonData("→", GetCallbackParamStr(CBReviewList, bookID, strconv.Itoa(page+1))))
		}
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(navigation...))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("← К карточке", GetCallbackParamStr(CBReviewBackToBook, bookID)),
		tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
	))
	return buttons
}

func (lnb *LitNightBot) showReviews(message *tgbotapi.Message, bookID string, page int) error {
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(bookID)
	if book == nil {
		_, err := lnb.editMessage(message.Chat.ID, message.MessageID, "Книга не найдена.", nil)
		return err
	}
	text, page, lastPage := renderReviewsPage(book, page)
	_, err := lnb.editHTMLMessage(message.Chat.ID, message.MessageID, text, reviewListButtons(book.ID, page, lastPage))
	return err
}

func (lnb *LitNightBot) sendReviewRequest(chatID int64, data *chatdata.ChatData, book *chatdata.ClubBook, at time.Time, logger *logrus.Entry) bool {
	if book == nil || !book.ClaimReviewRequest(at) {
		return false
	}
	if !lnb.persistReviewState(chatID, data, logger) {
		book.ReleaseReviewRequestClaim()
		return false
	}
	message, err := lnb.SendHTMLMessage(chatID, renderReviewRequest(book), reviewRequestButtons(book.ID))
	if err != nil {
		logger.WithError(err).WithField("book_id", book.ID).Error("Failed to send review request")
		book.ReleaseReviewRequestClaim()
		lnb.persistReviewState(chatID, data, logger)
		return false
	}
	book.MarkReviewRequestSent(at, message.MessageID)
	if !lnb.persistReviewState(chatID, data, logger) {
		return false
	}
	return true
}

func (lnb *LitNightBot) persistReviewState(chatID int64, data *chatdata.ChatData, logger *logrus.Entry) bool {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := lnb.iocd.SaveChatData(chatID, data); err == nil {
			return true
		} else {
			lastErr = err
			logger.WithError(err).WithField("attempt", attempt).Error("Failed to persist review delivery state")
		}
	}
	lnb.SendPlainMessage(chatID, dataStorageErrorText)
	logger.WithError(lastErr).Error("Review delivery state remains claimed to prevent duplicate messages")
	return false
}

func (lnb *LitNightBot) sendPendingReviewRequests(chatID int64, data *chatdata.ChatData, at time.Time, dueOnly bool, logger *logrus.Entry) int {
	sent := 0
	for index := range data.Books {
		book := &data.Books[index]
		if book.ReviewRequestDueAt == nil || book.ReviewRequestSentAt != nil {
			continue
		}
		if dueOnly && book.ReviewRequestDueAt.After(at) {
			continue
		}
		if lnb.sendReviewRequest(chatID, data, book, at, logger) {
			sent++
		}
	}
	return sent
}

func renderReviewReminder(book *chatdata.ClubBook, reminder chatdata.ReviewReminder) string {
	return telegramMentionHTML(reminder.UserID, reminder.DisplayName) + ", напоминаю об отзыве о книге «" + html.EscapeString(reviewBookName(book)) + "». Хотите написать его сейчас?"
}

func (lnb *LitNightBot) sendDueReviewReminders(chatID int64, data *chatdata.ChatData, at time.Time, logger *logrus.Entry) int {
	sent := 0
	for bookIndex := range data.Books {
		book := &data.Books[bookIndex]
		for reminderIndex := 0; reminderIndex < len(book.ReviewReminders); {
			reminder := &book.ReviewReminders[reminderIndex]
			if reminder.DueAt.After(at) {
				reminderIndex++
				continue
			}
			if book.ReviewByUser(reminder.UserID) != nil {
				book.ReviewReminders = append(book.ReviewReminders[:reminderIndex], book.ReviewReminders[reminderIndex+1:]...)
				lnb.persistReviewState(chatID, data, logger)
				continue
			}
			if reminder.DeliveryClaimedAt != nil {
				reminderIndex++
				continue
			}
			claimedAt := at
			reminder.DeliveryClaimedAt = &claimedAt
			if !lnb.persistReviewState(chatID, data, logger) {
				reminder.DeliveryClaimedAt = nil
				return sent
			}
			if _, err := lnb.SendHTMLMessage(chatID, renderReviewReminder(book, *reminder), reviewRequestButtonsForUser(book.ID, reminder.UserID)); err != nil {
				logger.WithError(err).WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminder.UserID}).Error("Failed to send review reminder")
				reminder.DeliveryClaimedAt = nil
				lnb.persistReviewState(chatID, data, logger)
				reminderIndex++
				continue
			}
			sent++
			book.ReviewReminders = append(book.ReviewReminders[:reminderIndex], book.ReviewReminders[reminderIndex+1:]...)
			if !lnb.persistReviewState(chatID, data, logger) {
				return sent
			}
		}
	}
	return sent
}

func (lnb *LitNightBot) ProcessDueReviews(at time.Time, logger *logrus.Entry) {
	files, err := lnb.iocd.GetDatasList()
	if err != nil {
		logger.WithError(err).Error("Failed to list chats for review processing")
		return
	}
	for _, file := range files {
		chatID, err := strconv.ParseInt(file, 10, 64)
		if err != nil {
			continue
		}
		chatLock := lnb.chatMutex(chatID)
		chatLock.Lock()
		data := lnb.iocd.GetChatData(chatID)
		if data == nil {
			chatLock.Unlock()
			continue
		}
		lnb.sendPendingReviewRequests(chatID, data, at, true, logger)
		lnb.sendDueReviewReminders(chatID, data, at, logger)
		chatLock.Unlock()
	}
}
