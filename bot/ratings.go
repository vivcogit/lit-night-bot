package bot

import (
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const ratingsPerPage = 10

func formatAverageRating(ratings []chatdata.Rating) string {
	return strings.Replace(fmt.Sprintf("%.1f", averageRating(ratings)), ".", ",", 1)
}

func ratingCountLabel(count int) string {
	lastTwo := count % 100
	if lastTwo >= 11 && lastTwo <= 14 {
		return fmt.Sprintf("%d оценок", count)
	}
	switch count % 10 {
	case 1:
		return fmt.Sprintf("%d оценка", count)
	case 2, 3, 4:
		return fmt.Sprintf("%d оценки", count)
	default:
		return fmt.Sprintf("%d оценок", count)
	}
}

func participantCountLabel(count int) string {
	lastTwo := count % 100
	if lastTwo >= 11 && lastTwo <= 14 {
		return fmt.Sprintf("%d участников", count)
	}
	switch count % 10 {
	case 1:
		return fmt.Sprintf("%d участник", count)
	case 2, 3, 4:
		return fmt.Sprintf("%d участника", count)
	default:
		return fmt.Sprintf("%d участников", count)
	}
}

func telegramDisplayName(user *tgbotapi.User) string {
	if user == nil {
		return "Участник"
	}
	firstName := strings.Join(strings.Fields(user.FirstName), " ")
	lastName := strings.Join(strings.Fields(user.LastName), " ")
	name := strings.TrimSpace(strings.Join([]string{firstName, lastName}, " "))
	if name != "" {
		return name
	}
	if user.UserName != "" {
		return "@" + user.UserName
	}
	return "Участник"
}

func renderRatingPanel(book *chatdata.ClubBook, justCompleted bool) string {
	return renderRatingPanelForChat(book, justCompleted, false, 0)
}

func renderRatingPanelForChat(book *chatdata.ClubBook, justCompleted bool, private bool, userID int64) string {
	if private {
		return renderPersonalRatingPanel(book, justCompleted, userID)
	}
	if book.RatingsClosedAt != nil {
		return renderRatingResult(book)
	}
	var text strings.Builder
	if justCompleted {
		text.WriteString("✅ <b>«" + html.EscapeString(book.DisplayName()) + "» обсуждена!</b>\n\n")
	} else {
		text.WriteString("⭐ <b>Оценка книги</b>\n")
		text.WriteString("«" + html.EscapeString(book.DisplayName()) + "»\n\n")
	}
	text.WriteString("Как вам книга?\n\n")
	if len(book.Ratings) == 0 {
		text.WriteString("Средняя оценка: <i>оценок пока нет</i>")
	} else {
		text.WriteString(fmt.Sprintf("Средняя оценка: <b>%s из 10</b>\nОценили: %s", formatAverageRating(book.Ratings), participantCountLabel(len(book.Ratings))))
	}
	return text.String()
}

func renderPersonalRatingPanel(book *chatdata.ClubBook, justCompleted bool, userID int64) string {
	var text strings.Builder
	if justCompleted {
		text.WriteString("✅ <b>«" + html.EscapeString(book.DisplayName()) + "» прочитана!</b>\n\n")
	} else {
		text.WriteString("📚 <b>Личный читательский дневник</b>\n")
		text.WriteString("«" + html.EscapeString(book.DisplayName()) + "»\n\n")
	}
	if rating := book.RatingByUser(userID); rating != nil {
		text.WriteString(fmt.Sprintf("⭐ Моя оценка: <b>%d из 10</b>", rating.Value))
	} else {
		text.WriteString("⭐ Моя оценка: <i>пока не поставлена</i>\n\nКак вам книга?")
	}
	return text.String()
}

func renderRatingResult(book *chatdata.ClubBook) string {
	var text strings.Builder
	text.WriteString("🏁 <b>Сбор оценок завершён</b>\n\n")
	text.WriteString("📖 «" + html.EscapeString(book.DisplayName()) + "»\n")
	if len(book.Ratings) == 0 {
		text.WriteString("⭐ Итоговой оценки нет\n")
		text.WriteString("👥 Никто не проголосовал")
	} else {
		text.WriteString(fmt.Sprintf("⭐ Итоговая оценка: <b>%s из 10</b>\n", formatAverageRating(book.Ratings)))
		text.WriteString("👥 Проголосовали: " + participantCountLabel(len(book.Ratings)))
	}
	return text.String()
}

func ratingPanelButtons(book *chatdata.ClubBook) [][]tgbotapi.InlineKeyboardButton {
	return ratingPanelButtonsForChat(book, false)
}

func ratingPanelButtonsForChat(book *chatdata.ClubBook, private bool) [][]tgbotapi.InlineKeyboardButton {
	if private {
		return personalRatingPanelButtons(book)
	}
	if book.RatingsClosedAt != nil {
		return ratingResultButtons(book)
	}
	firstRow := make([]tgbotapi.InlineKeyboardButton, 0, 5)
	secondRow := make([]tgbotapi.InlineKeyboardButton, 0, 5)
	for value := 1; value <= 10; value++ {
		button := tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(value), GetCallbackParamStr(CBRatingSet, book.ID, strconv.Itoa(value)))
		if value <= 5 {
			firstRow = append(firstRow, button)
		} else {
			secondRow = append(secondRow, button)
		}
	}
	return [][]tgbotapi.InlineKeyboardButton{
		firstRow,
		secondRow,
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👥 Кто как оценил", GetCallbackParamStr(CBRatingList, book.ID, "0"))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить мою оценку", GetCallbackParamStr(CBRatingDeleteRequest, book.ID))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Завершить сбор оценок", GetCallbackParamStr(CBRatingCloseRequest, book.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← К карточке", GetCallbackParamStr(CBRatingBackToBook, book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
		),
	}
}

func personalRatingPanelButtons(book *chatdata.ClubBook) [][]tgbotapi.InlineKeyboardButton {
	firstRow := make([]tgbotapi.InlineKeyboardButton, 0, 5)
	secondRow := make([]tgbotapi.InlineKeyboardButton, 0, 5)
	for value := 1; value <= 10; value++ {
		button := tgbotapi.NewInlineKeyboardButtonData(strconv.Itoa(value), GetCallbackParamStr(CBRatingSet, book.ID, strconv.Itoa(value)))
		if value <= 5 {
			firstRow = append(firstRow, button)
		} else {
			secondRow = append(secondRow, button)
		}
	}
	return [][]tgbotapi.InlineKeyboardButton{
		firstRow,
		secondRow,
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить мою оценку", GetCallbackParamStr(CBRatingDeleteRequest, book.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← К карточке", GetCallbackParamStr(CBRatingBackToBook, book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
		),
	}
}

func ratingResultButtons(book *chatdata.ClubBook) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("👥 Кто как оценил", GetCallbackParamStr(CBRatingList, book.ID, "0"))),
		tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🔓 Возобновить сбор оценок", GetCallbackParamStr(CBRatingReopen, book.ID))),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("← К карточке", GetCallbackParamStr(CBRatingBackToBook, book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
		),
	}
}

func ratingResultButtonsWithNextBook(book *chatdata.ClubBook, data *chatdata.ChatData) [][]tgbotapi.InlineKeyboardButton {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, 6)
	if data != nil && data.CurrentBook() == nil {
		if len(data.BooksWithStatus(chatdata.StatusWishlist)) == 0 {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("➕ Добавить в вишлист", GetCallbackParamStr(CBWishlistAddBookRequest)),
			))
		} else {
			buttons = append(buttons,
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("🎲 Случайная книга", GetCallbackParamStr(CBCurrentRandom))),
				tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📘 Выбрать книгу", GetCallbackParamStr(CBWishlistChoose))),
			)
		}
	}
	buttons = append(buttons, ratingResultButtons(book)...)
	return buttons
}

func renderRatingResultWithNextBook(book *chatdata.ClubBook, data *chatdata.ChatData) string {
	text := renderRatingResult(book)
	if data == nil || data.CurrentBook() != nil {
		return text
	}
	if len(data.BooksWithStatus(chatdata.StatusWishlist)) == 0 {
		return text + "\n\n📚 Вишлист пуст. Добавьте книги, когда будете готовы, и вернитесь к выбору позже."
	}
	return text + "\n\nВыберите следующую книгу случайно или из вишлиста."
}

func renderRatingsList(book *chatdata.ClubBook, page int) (string, int, int) {
	ratings := append([]chatdata.Rating(nil), book.Ratings...)
	sort.SliceStable(ratings, func(i, j int) bool {
		return strings.ToLower(ratings[i].DisplayName) < strings.ToLower(ratings[j].DisplayName)
	})
	lastPage := 0
	if len(ratings) > 0 {
		lastPage = (len(ratings) - 1) / ratingsPerPage
	}
	if page < 0 {
		page = 0
	}
	if page > lastPage {
		page = lastPage
	}
	var text strings.Builder
	text.WriteString("⭐ <b>Оценки книги</b>\n")
	text.WriteString("«" + html.EscapeString(book.DisplayName()) + "»\n\n")
	if len(ratings) == 0 {
		text.WriteString("Оценок пока нет.")
		return text.String(), page, lastPage
	}
	text.WriteString(fmt.Sprintf("Средняя оценка: <b>%s из 10</b>\nВсего: %s\n\n", formatAverageRating(ratings), ratingCountLabel(len(ratings))))
	start := page * ratingsPerPage
	end := start + ratingsPerPage
	if end > len(ratings) {
		end = len(ratings)
	}
	for index := start; index < end; index++ {
		text.WriteString(fmt.Sprintf("%d. %s — <b>%d</b>\n", index+1, html.EscapeString(ratings[index].DisplayName), ratings[index].Value))
	}
	return strings.TrimSpace(text.String()), page, lastPage
}

func ratingsListButtons(book *chatdata.ClubBook, page int, lastPage int) [][]tgbotapi.InlineKeyboardButton {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, 3)
	if lastPage > 0 {
		row := make([]tgbotapi.InlineKeyboardButton, 0, 2)
		if page > 0 {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("←", GetCallbackParamStr(CBRatingList, book.ID, strconv.Itoa(page-1))))
		}
		if page < lastPage {
			row = append(row, tgbotapi.NewInlineKeyboardButtonData("→", GetCallbackParamStr(CBRatingList, book.ID, strconv.Itoa(page+1))))
		}
		buttons = append(buttons, row)
	}
	backLabel := "← Оценить"
	if book.RatingsClosedAt != nil {
		backLabel = "← К итогу"
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData(backLabel, GetCallbackParamStr(CBRatingOpen, book.ID)),
		tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
	))
	return buttons
}

func ratingChatContext(data *chatdata.ChatData, chatID int64) (private bool, userID int64) {
	if data.IsPrivateChat(chatID) {
		return true, chatID
	}
	return false, 0
}

func (lnb *LitNightBot) sendCompletionRatingPanel(chatID int64, book *chatdata.ClubBook) {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	private, userID := ratingChatContext(data, chatID)
	lnb.SendHTMLMessage(chatID, renderRatingPanelForChat(book, true, private, userID), ratingPanelButtonsForChat(book, private))
}

func (lnb *LitNightBot) showRatingPanel(message *tgbotapi.Message, bookID string, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Книга не найдена.", nil)
		return
	}
	if book.Status != chatdata.StatusCompleted {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Оценить книгу можно после завершения чтения.", nil)
		return
	}
	private, userID := ratingChatContext(data, message.Chat.ID)
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderRatingPanelForChat(book, false, private, userID), ratingPanelButtonsForChat(book, private))
}

func (lnb *LitNightBot) setBookRating(update *tgbotapi.Update, bookID string, value int, logger *logrus.Entry) {
	message := update.CallbackQuery.Message
	user := update.CallbackQuery.From
	if user == nil || user.IsBot {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Боты не могут оценивать книги"))
		return
	}
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	private := message.Chat.IsPrivate()
	if private && book.RatingsClosedAt != nil {
		book.ReopenRatings()
	}
	previous, err := book.SetRating(user.ID, telegramDisplayName(user), value, time.Now())
	if err != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, err.Error()))
		return
	}
	if err := lnb.iocd.SaveChatData(message.Chat.ID, data); err != nil {
		logger.WithError(err).Error("Failed to save rating")
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось сохранить оценку"))
		return
	}
	answer := fmt.Sprintf("Ваша оценка принята: %d из 10", value)
	if previous != nil {
		if *previous == value {
			answer = fmt.Sprintf("Ваша оценка уже %d из 10", value)
		} else {
			answer = fmt.Sprintf("Оценка изменена: %d → %d", *previous, value)
		}
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, answer))
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderRatingPanelForChat(book, false, private, user.ID), ratingPanelButtonsForChat(book, private))
}

func (lnb *LitNightBot) showRatingsList(message *tgbotapi.Message, bookID string, page int) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Книга не найдена.", nil)
		return
	}
	if data.IsPrivateChat(message.Chat.ID) {
		lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderPersonalRatingPanel(book, false, message.Chat.ID), personalRatingPanelButtons(book))
		return
	}
	text, page, lastPage := renderRatingsList(book, page)
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, text, ratingsListButtons(book, page, lastPage))
}

func (lnb *LitNightBot) requestRatingDelete(update *tgbotapi.Update, bookID string) {
	message := update.CallbackQuery.Message
	user := update.CallbackQuery.From
	if user == nil {
		return
	}
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	if !message.Chat.IsPrivate() && book.RatingsClosedAt != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор оценок завершён — сначала возобновите его"))
		return
	}
	if book.RatingByUser(user.ID) == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Вы ещё не оценивали эту книгу"))
		return
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Удалить", GetCallbackParamStr(CBRatingDeleteConfirm, book.ID, strconv.FormatInt(user.ID, 10), strconv.Itoa(message.MessageID))),
		tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBRatingDeleteCancel, strconv.FormatInt(user.ID, 10))),
	)}
	text := fmt.Sprintf("🗑 <b>%s</b>, удалить вашу оценку книге «%s»?", html.EscapeString(telegramDisplayName(user)), html.EscapeString(book.DisplayName()))
	lnb.SendHTMLMessage(message.Chat.ID, text, buttons)
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

func (lnb *LitNightBot) confirmRatingDelete(update *tgbotapi.Update, params []string, logger *logrus.Entry) {
	if len(params) < 3 || update.CallbackQuery.From == nil {
		return
	}
	expectedUserID, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil || expectedUserID != update.CallbackQuery.From.ID {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это подтверждение для другого участника"))
		return
	}
	sourceMessageID, _ := strconv.Atoi(params[2])
	chatID := update.CallbackQuery.Message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(params[0])
	private := update.CallbackQuery.Message.Chat.IsPrivate()
	if book != nil && !private && book.RatingsClosedAt != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор оценок завершён — сначала возобновите его"))
		return
	}
	if book == nil || !book.DeleteRating(expectedUserID) {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Оценка уже удалена"))
		return
	}
	if err := lnb.iocd.SaveChatData(chatID, data); err != nil {
		logger.WithError(err).Error("Failed to delete rating")
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось удалить оценку"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Оценка удалена"))
	lnb.removeMessage(chatID, update.CallbackQuery.Message.MessageID)
	if sourceMessageID > 0 {
		lnb.editHTMLMessage(chatID, sourceMessageID, renderRatingPanelForChat(book, false, private, expectedUserID), ratingPanelButtonsForChat(book, private))
	}
}

func (lnb *LitNightBot) requestRatingClose(update *tgbotapi.Update, bookID string) {
	message := update.CallbackQuery.Message
	if message.Chat.IsPrivate() {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "В личном дневнике сбор оценок не требуется"))
		return
	}
	user := update.CallbackQuery.From
	if user == nil {
		return
	}
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	if book.RatingsClosedAt != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор оценок уже завершён"))
		return
	}
	if len(book.Ratings) == 0 {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Пока нет оценок"))
		return
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("✅ Показать итог", GetCallbackParamStr(CBRatingCloseConfirm, book.ID, strconv.FormatInt(user.ID, 10), strconv.Itoa(message.MessageID))),
		tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBRatingCloseCancel, strconv.FormatInt(user.ID, 10))),
	)}
	text := "<b>Все участники успели поставить оценки?</b>\n\nБот покажет итог и скроет кнопки оценки.\nПри необходимости сбор оценок можно возобновить."
	lnb.SendHTMLMessage(message.Chat.ID, text, buttons)
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

func (lnb *LitNightBot) confirmRatingClose(update *tgbotapi.Update, params []string, logger *logrus.Entry) {
	if len(params) < 3 || update.CallbackQuery.From == nil {
		return
	}
	user := update.CallbackQuery.From
	if update.CallbackQuery.Message.Chat.IsPrivate() {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "В личном дневнике сбор оценок не требуется"))
		return
	}
	expectedUserID, err := strconv.ParseInt(params[1], 10, 64)
	if err != nil || expectedUserID != user.ID {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это подтверждение для другого участника"))
		return
	}
	sourceMessageID, _ := strconv.Atoi(params[2])
	chatID := update.CallbackQuery.Message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(params[0])
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	now := time.Now()
	if err := book.CloseRatings(user.ID, telegramDisplayName(user), now); err != nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, err.Error()))
		return
	}
	book.ScheduleReviewRequest(now.Add(reviewRequestDelay))
	if err := lnb.iocd.SaveChatData(chatID, data); err != nil {
		logger.WithError(err).Error("Failed to close ratings")
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось сохранить итог"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Итог готов"))
	lnb.removeMessage(chatID, update.CallbackQuery.Message.MessageID)
	if sourceMessageID > 0 {
		lnb.editHTMLMessage(chatID, sourceMessageID, renderRatingResultWithNextBook(book, data), ratingResultButtonsWithNextBook(book, data))
	}
}

func (lnb *LitNightBot) cancelRatingClose(update *tgbotapi.Update, params []string) {
	if len(params) < 1 || update.CallbackQuery.From == nil {
		return
	}
	expectedUserID, err := strconv.ParseInt(params[0], 10, 64)
	if err != nil || expectedUserID != update.CallbackQuery.From.ID {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это подтверждение для другого участника"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отменено"))
	lnb.removeMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)
}

func (lnb *LitNightBot) reopenRatings(update *tgbotapi.Update, bookID string, logger *logrus.Entry) {
	message := update.CallbackQuery.Message
	if message.Chat.IsPrivate() {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "В личном дневнике сбор оценок всегда открыт"))
		return
	}
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Книга не найдена"))
		return
	}
	if !book.ReopenRatings() {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор оценок уже открыт"))
		return
	}
	book.CancelPendingReviewRequest()
	if err := lnb.iocd.SaveChatData(message.Chat.ID, data); err != nil {
		logger.WithError(err).Error("Failed to reopen ratings")
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось возобновить сбор оценок"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Сбор оценок возобновлён"))
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderRatingPanel(book, false), ratingPanelButtons(book))
}

func (lnb *LitNightBot) cancelRatingDelete(update *tgbotapi.Update, params []string) {
	if len(params) < 1 || update.CallbackQuery.From == nil {
		return
	}
	expectedUserID, err := strconv.ParseInt(params[0], 10, 64)
	if err != nil || expectedUserID != update.CallbackQuery.From.ID {
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Это подтверждение для другого участника"))
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отменено"))
	lnb.removeMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Message.MessageID)
}
