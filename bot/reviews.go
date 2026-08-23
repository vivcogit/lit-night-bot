package bot

import (
	"errors"
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const reviewRequestDelay = 15 * time.Minute
const reviewReminderDelay = 24 * time.Hour
const reviewDeliveryClaimLease = 15 * time.Minute
const reviewDeliveryRetryBackoff = 5 * time.Minute

type telegramFailurePolicy int

const (
	telegramFailureAmbiguous telegramFailurePolicy = iota
	telegramFailureRetry
	telegramFailureTerminal
)

type persistenceOutcome int

const (
	persistenceNotCommitted persistenceOutcome = iota
	persistenceDurable
	persistenceCommittedUncertain
)

type reviewDeliveryKey struct {
	chatID int64
	bookID string
	userID int64
}

func isTelegramRateLimit(err error) bool {
	var apiError *tgbotapi.Error
	return errors.As(err, &apiError) && apiError.Code == 429
}

func (lnb *LitNightBot) automatedReviewDeliveryAllowed(key reviewDeliveryKey, at time.Time) bool {
	lnb.reviewRetryMu.Lock()
	defer lnb.reviewRetryMu.Unlock()
	retryAt, exists := lnb.reviewRetryAt[key]
	if !exists {
		return true
	}
	if at.Before(retryAt) {
		return false
	}
	delete(lnb.reviewRetryAt, key)
	return true
}

func (lnb *LitNightBot) rememberAutomatedReviewRetry(key reviewDeliveryKey, retryAt time.Time) {
	lnb.reviewRetryMu.Lock()
	defer lnb.reviewRetryMu.Unlock()
	if lnb.reviewRetryAt == nil {
		lnb.reviewRetryAt = make(map[reviewDeliveryKey]time.Time)
	}
	if existing := lnb.reviewRetryAt[key]; retryAt.After(existing) {
		lnb.reviewRetryAt[key] = retryAt
	}
}

func (lnb *LitNightBot) clearAutomatedReviewRetry(key reviewDeliveryKey) {
	lnb.reviewRetryMu.Lock()
	defer lnb.reviewRetryMu.Unlock()
	delete(lnb.reviewRetryAt, key)
}

func (lnb *LitNightBot) reviewFailureTime(fallback time.Time) time.Time {
	if lnb.reviewNow == nil {
		return fallback
	}
	return lnb.reviewNow()
}

func classifyTelegramFailure(err error, at time.Time) (telegramFailurePolicy, time.Time) {
	var apiError *tgbotapi.Error
	if !errors.As(err, &apiError) {
		return telegramFailureAmbiguous, time.Time{}
	}
	if apiError.Code == 429 {
		delay := time.Duration(apiError.RetryAfter) * time.Second
		if delay < reviewDeliveryRetryBackoff {
			delay = reviewDeliveryRetryBackoff
		}
		return telegramFailureRetry, at.Add(delay)
	}
	if apiError.Code >= 500 {
		return telegramFailureRetry, at.Add(reviewDeliveryRetryBackoff)
	}
	return telegramFailureTerminal, time.Time{}
}

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
	if book == nil || !book.ReviewCollectionOpen() {
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

func (lnb *LitNightBot) sendReviewRequest(chatID int64, data *chatdata.ChatData, book *chatdata.ClubBook, at time.Time, logger *logrus.Entry) (bool, bool) {
	if book == nil {
		return false, true
	}
	deliveryKey := reviewDeliveryKey{chatID: chatID, bookID: strings.ToLower(book.ID)}
	if !lnb.automatedReviewDeliveryAllowed(deliveryKey, at) {
		return false, true
	}
	if !book.ClaimReviewRequest(at, reviewDeliveryClaimLease) {
		return false, true
	}
	if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
		return false, false
	}
	message, err := lnb.SendHTMLMessage(chatID, renderReviewRequest(book), reviewRequestButtons(book.ID))
	if err != nil {
		logger.WithError(err).WithField("book_id", book.ID).Error("Failed to send review request")
		policy, retryAt := classifyTelegramFailure(err, lnb.reviewFailureTime(at))
		switch policy {
		case telegramFailureRetry:
			rateLimited := isTelegramRateLimit(err)
			if rateLimited {
				lnb.rememberAutomatedReviewRetry(deliveryKey, retryAt)
			}
			book.DeferReviewRequest(retryAt)
			if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
				if rateLimited {
					logger.WithFields(logrus.Fields{"book_id": book.ID, "retry_at": retryAt, "retry_not_before_volatile": true}).Error("RetryAfter is protected only until process restart because durable persistence failed")
				}
				return false, false
			}
			lnb.clearAutomatedReviewRetry(deliveryKey)
		case telegramFailureTerminal:
			lnb.clearAutomatedReviewRetry(deliveryKey)
			book.CancelPendingReviewRequest()
			if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
				return false, false
			}
			logger.WithField("book_id", book.ID).Error("Review request delivery was permanently rejected; cancelling pending delivery")
		default:
			logger.WithField("book_id", book.ID).Warn("Review request delivery is ambiguous; keeping claim until lease expires")
		}
		return false, true
	}
	book.MarkReviewRequestSent(at, message.MessageID)
	lnb.clearAutomatedReviewRetry(deliveryKey)
	persisted := lnb.persistReviewState(chatID, data, logger)
	if persisted != persistenceDurable {
		if persisted == persistenceNotCommitted {
			if deleteErr := lnb.removeMessage(chatID, message.MessageID); deleteErr != nil {
				logger.WithError(deleteErr).WithField("book_id", book.ID).Error("Failed to compensate unpersisted review request")
			}
		} else {
			logger.WithField("book_id", book.ID).Error("Review request was committed, but directory durability is uncertain; skipping compensation")
		}
		return false, false
	}
	return true, true
}

func (lnb *LitNightBot) saveReviewState(chatID int64, data *chatdata.ChatData) error {
	if lnb.reviewStateSaver != nil {
		return lnb.reviewStateSaver(chatID, data)
	}
	return lnb.iocd.SaveChatData(chatID, data)
}

func (lnb *LitNightBot) persistReviewState(chatID int64, data *chatdata.ChatData, logger *logrus.Entry) persistenceOutcome {
	var lastErr error
	committed := false
	for attempt := 1; attempt <= 3; attempt++ {
		if err := lnb.saveReviewState(chatID, data); err == nil {
			return persistenceDurable
		} else {
			lastErr = err
			committed = committed || utils.IsPostCommitDurabilityError(err)
			logger.WithError(err).WithField("attempt", attempt).Error("Failed to persist review delivery state")
		}
	}
	logger.WithError(lastErr).Error("Review delivery state remains claimed to prevent duplicate messages")
	if committed {
		return persistenceCommittedUncertain
	}
	lnb.SendPlainMessage(chatID, dataStorageErrorText)
	return persistenceNotCommitted
}

func (lnb *LitNightBot) sendPendingReviewRequests(chatID int64, data *chatdata.ChatData, at time.Time, dueOnly bool, logger *logrus.Entry) (int, bool) {
	sent := 0
	for index := range data.Books {
		book := &data.Books[index]
		if book.ReviewRequestDueAt == nil || book.ReviewRequestSentAt != nil {
			continue
		}
		if dueOnly && book.ReviewRequestDueAt.After(at) {
			continue
		}
		wasSent, usable := lnb.sendReviewRequest(chatID, data, book, at, logger)
		if !usable {
			return sent, false
		}
		if wasSent {
			sent++
		}
	}
	return sent, true
}

func renderReviewReminder(book *chatdata.ClubBook, reminder chatdata.ReviewReminder) string {
	return telegramMentionHTML(reminder.UserID, reminder.DisplayName) + ", напоминаю об отзыве о книге «" + html.EscapeString(reviewBookName(book)) + "». Хотите написать его сейчас?"
}

func (lnb *LitNightBot) sendDueReviewReminders(chatID int64, data *chatdata.ChatData, at time.Time, logger *logrus.Entry) (int, bool) {
	sent := 0
	for bookIndex := range data.Books {
		book := &data.Books[bookIndex]
		for reminderIndex := 0; reminderIndex < len(book.ReviewReminders); {
			reminder := &book.ReviewReminders[reminderIndex]
			reminderUserID := reminder.UserID
			if reminder.DueAt.After(at) {
				reminderIndex++
				continue
			}
			if book.ReviewByUser(reminderUserID) != nil {
				book.ReviewReminders = append(book.ReviewReminders[:reminderIndex], book.ReviewReminders[reminderIndex+1:]...)
				if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
					return sent, false
				}
				continue
			}
			advance, wasSent, usable := func() (bool, bool, bool) {
				deliveryKey := reviewDeliveryKey{chatID: chatID, bookID: strings.ToLower(book.ID), userID: reminderUserID}
				if !lnb.automatedReviewDeliveryAllowed(deliveryKey, at) {
					return true, false, true
				}
				if !reminder.ClaimDelivery(at, reviewDeliveryClaimLease) {
					return true, false, true
				}
				if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
					return false, false, false
				}
				message, err := lnb.SendHTMLMessage(chatID, renderReviewReminder(book, *reminder), reviewRequestButtonsForUser(book.ID, reminderUserID))
				if err != nil {
					logger.WithError(err).WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID}).Error("Failed to send review reminder")
					policy, retryAt := classifyTelegramFailure(err, lnb.reviewFailureTime(at))
					switch policy {
					case telegramFailureRetry:
						rateLimited := isTelegramRateLimit(err)
						if rateLimited {
							lnb.rememberAutomatedReviewRetry(deliveryKey, retryAt)
						}
						reminder.ReleaseDeliveryClaim()
						reminder.DueAt = retryAt
						if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
							if rateLimited {
								logger.WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID, "retry_at": retryAt, "retry_not_before_volatile": true}).Error("Reminder RetryAfter is protected only until process restart because durable persistence failed")
							}
							return false, false, false
						}
						lnb.clearAutomatedReviewRetry(deliveryKey)
						return true, false, true
					case telegramFailureTerminal:
						lnb.clearAutomatedReviewRetry(deliveryKey)
						book.ReviewReminders = append(book.ReviewReminders[:reminderIndex], book.ReviewReminders[reminderIndex+1:]...)
						if lnb.persistReviewState(chatID, data, logger) != persistenceDurable {
							return false, false, false
						}
						logger.WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID}).Error("Review reminder was permanently rejected; cancelling reminder")
						return false, false, true
					default:
						logger.WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID}).Warn("Review reminder delivery is ambiguous; keeping claim until lease expires")
						return true, false, true
					}
				}
				book.ReviewReminders = append(book.ReviewReminders[:reminderIndex], book.ReviewReminders[reminderIndex+1:]...)
				lnb.clearAutomatedReviewRetry(deliveryKey)
				persisted := lnb.persistReviewState(chatID, data, logger)
				if persisted != persistenceDurable {
					if persisted == persistenceNotCommitted {
						if deleteErr := lnb.removeMessage(chatID, message.MessageID); deleteErr != nil {
							logger.WithError(deleteErr).WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID}).Error("Failed to compensate unpersisted review reminder")
						}
					} else {
						logger.WithFields(logrus.Fields{"book_id": book.ID, "user_id": reminderUserID}).Error("Review reminder was committed, but directory durability is uncertain; skipping compensation")
					}
					return false, false, false
				}
				return false, true, true
			}()
			if !usable {
				return sent, false
			}
			if wasSent {
				sent++
			}
			if advance {
				reminderIndex++
			}
		}
	}
	return sent, true
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
		data, loadErr := lnb.iocd.LoadChatData(chatID)
		if loadErr != nil {
			logger.WithError(loadErr).WithField("chat_id", chatID).Error("Failed to load chat for review processing")
			chatLock.Unlock()
			continue
		}
		if data.SchemaVersion != chatdata.CurrentSchemaVersion {
			logger.WithFields(logrus.Fields{"chat_id": chatID, "schema_version": data.SchemaVersion}).Warn("Skipping review processing for unsupported schema")
			chatLock.Unlock()
			continue
		}
		if validateErr := data.ValidateV2(); validateErr != nil {
			logger.WithError(validateErr).WithField("chat_id", chatID).Error("Skipping review processing for invalid data")
			chatLock.Unlock()
			continue
		}
		_, usable := lnb.sendPendingReviewRequests(chatID, data, at, true, logger)
		if usable {
			_, _ = lnb.sendDueReviewReminders(chatID, data, at, logger)
		}
		chatLock.Unlock()
	}
}
