package bot

import (
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const migrationCompleteText = "✅ Все карточки проверены."

func statusLabel(status chatdata.BookStatus) string {
	switch status {
	case chatdata.StatusWishlist:
		return "📚 В вишлисте"
	case chatdata.StatusReading:
		return "📖 Читаем"
	case chatdata.StatusCompleted:
		return "✅ Прочитана"
	case chatdata.StatusPostponed:
		return "⏸ Отложена"
	case chatdata.StatusUnfinished:
		return "🚫 Не дочитана"
	case chatdata.StatusExcluded:
		return "🗑 Исключена"
	default:
		return string(status)
	}
}

func renderBookCard(book *chatdata.ClubBook) string {
	return renderBookCardForChat(book, false, 0)
}

func renderBookCardForChat(book *chatdata.ClubBook, private bool, userID int64) string {
	var text strings.Builder
	text.WriteString("📖 <b>" + html.EscapeString(book.Title) + "</b>\n")
	if len(book.Authors) > 0 {
		text.WriteString("✍️ " + html.EscapeString(strings.Join(book.Authors, ", ")) + "\n")
	} else {
		text.WriteString("✍️ <i>Автор не указан</i>\n")
	}
	text.WriteString("\n" + statusLabel(book.Status) + "\n")
	if book.CompletedAt != nil {
		text.WriteString("📅 Прочитали: " + book.CompletedAt.Format(DATE_LAYOUT) + "\n")
	}
	if book.StoppedAt != nil {
		text.WriteString("📅 Завершили чтение: " + book.StoppedAt.Format(DATE_LAYOUT) + "\n")
	}
	if book.UnfinishedReason != nil {
		text.WriteString("💬 Причина: " + html.EscapeString(book.UnfinishedReason.DisplayText()) + "\n")
	}
	if book.Deadline != nil {
		text.WriteString("🗓 Дедлайн: " + book.Deadline.Format(DATE_LAYOUT) + "\n")
	}
	if book.Status == chatdata.StatusCompleted {
		if private {
			if rating := book.RatingByUser(userID); rating != nil {
				text.WriteString(fmt.Sprintf("⭐ Моя оценка: %d из 10\n", rating.Value))
			} else {
				text.WriteString("⭐ Моя оценка пока не поставлена\n")
			}
		} else if len(book.Ratings) > 0 {
			text.WriteString(fmt.Sprintf("⭐ %s из 10 · %s\n", formatAverageRating(book.Ratings), ratingCountLabel(len(book.Ratings))))
		} else {
			text.WriteString("⭐ Оценок пока нет\n")
		}
	}
	if !private && book.RatingsClosedAt != nil {
		text.WriteString("🏁 Сбор оценок завершён\n")
	}
	if book.Status == chatdata.StatusCompleted {
		if private {
			if review := book.ReviewByUser(userID); review != nil {
				text.WriteString("💬 Мой отзыв:\n" + html.EscapeString(truncateUTF16(review.Text, 2500)) + "\n")
			} else {
				text.WriteString("💬 Мой отзыв пока не написан\n")
			}
		} else {
			text.WriteString(fmt.Sprintf("💬 Отзывов: %d\n", len(book.Reviews)))
		}
	}
	if book.NeedsReview {
		text.WriteString("\n⚠️ <b>Карточка создана миграцией и требует проверки.</b>\n")
		if book.LegacyName != "" {
			text.WriteString("Было: <i>" + html.EscapeString(book.LegacyName) + "</i>\n")
		}
	}
	return text.String()
}

func bookCardButtons(book *chatdata.ClubBook) [][]tgbotapi.InlineKeyboardButton {
	return bookCardButtonsForChat(book, false, 0)
}

func bookCardButtonsForChat(book *chatdata.ClubBook, private bool, userID int64) [][]tgbotapi.InlineKeyboardButton {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, 5)
	if book.Status == chatdata.StatusCompleted {
		ratingButtonText := "⭐ Оценить / изменить"
		if private {
			if rating := book.RatingByUser(userID); rating != nil {
				ratingButtonText = fmt.Sprintf("⭐ Моя оценка: %d", rating.Value)
			} else {
				ratingButtonText = "⭐ Оценить"
			}
		} else if book.RatingsClosedAt != nil {
			ratingButtonText = "🏁 Итог оценки"
		}
		ratingRow := []tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardButtonData(ratingButtonText, GetCallbackParamStr(CBRatingOpen, book.ID))}
		if !private {
			ratingRow = append(ratingRow, tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("👥 Оценки (%d)", len(book.Ratings)), GetCallbackParamStr(CBRatingList, book.ID, "0")))
		}
		buttons = append(buttons, ratingRow)
		if private {
			if book.ReviewByUser(userID) != nil {
				buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("✏️ Изменить отзыв", GetCallbackParamStr(CBReviewWrite, book.ID, strconv.FormatInt(userID, 10))),
					tgbotapi.NewInlineKeyboardButtonData("🗑 Удалить отзыв", GetCallbackParamStr(CBReviewDelete, book.ID, strconv.FormatInt(userID, 10))),
				))
			} else {
				buttons = append(buttons,
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✍️ Написать отзыв", GetCallbackParamStr(CBReviewWrite, book.ID))),
					tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("⏰ Напомнить завтра об отзыве", GetCallbackParamStr(CBReviewRemind, book.ID))),
				)
			}
		} else {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("💬 Отзывы (%d)", len(book.Reviews)), GetCallbackParamStr(CBReviewList, book.ID, "0")),
			))
		}
	}
	buttons = append(buttons,
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ Название", GetCallbackParamStr(CBBookEditTitle, book.ID)),
			tgbotapi.NewInlineKeyboardButtonData("✏️ Авторы", GetCallbackParamStr(CBBookEditAuthors, book.ID)),
		),
	)
	if len(book.Authors) == 1 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("↔ Поменять название и автора", GetCallbackParamStr(CBBookSwap, book.ID))))
	}
	if book.NeedsReview {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("✅ Карточка верна", GetCallbackParamStr(CBBookApprove, book.ID))))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel))))
	return buttons
}

func cardChatContext(chat *tgbotapi.Chat) (private bool, userID int64) {
	if chat != nil && chat.IsPrivate() {
		return true, chat.ID
	}
	return false, 0
}

func (lnb *LitNightBot) showBookCardInPlace(message *tgbotapi.Message, id string) {
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(id)
	if book == nil {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Книга не найдена.", nil)
		return
	}
	private, userID := cardChatContext(message.Chat)
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
}

func shouldEditBookCardInPlace(book *chatdata.ClubBook) bool {
	return book.NeedsReview
}

func (lnb *LitNightBot) swapBookTitleAndAuthor(message *tgbotapi.Message, id string, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(id)
	if book == nil || len(book.Authors) != 1 {
		lnb.SendPlainMessage(message.Chat.ID, "Не удалось поменять поля местами.")
		return
	}
	book.Title, book.Authors[0] = book.Authors[0], book.Title
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		return
	}
	private, userID := cardChatContext(message.Chat)
	lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
}

func (lnb *LitNightBot) handleBookCard(message *tgbotapi.Message, id string, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(id)
	if book == nil {
		lnb.SendPlainMessage(message.Chat.ID, "Книга не найдена.")
		return
	}
	if shouldEditBookCardInPlace(book) {
		private, userID := cardChatContext(message.Chat)
		lnb.editHTMLMessage(message.Chat.ID, message.MessageID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
		return
	}
	private, userID := cardChatContext(message.Chat)
	lnb.SendHTMLMessage(message.Chat.ID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
}

func (lnb *LitNightBot) replaceMessageWithBookCard(message *tgbotapi.Message, id string, logger *logrus.Entry) {
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(id)
	if book == nil {
		lnb.editMessage(message.Chat.ID, message.MessageID, "Книга не найдена.", nil)
		return
	}
	private, userID := cardChatContext(message.Chat)
	if _, err := lnb.SendHTMLMessage(message.Chat.ID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID)); err != nil {
		logger.WithError(err).Error("Failed to send replacing book card")
		return
	}
	if err := lnb.removeMessage(message.Chat.ID, message.MessageID); err != nil {
		logger.WithError(err).Warn("Failed to remove source message after showing book card")
	}
}

func bookFieldRequestConfig(chatID int64, user *tgbotapi.User, prefix string, id string, sourceMessageID int, prompt string) tgbotapi.MessageConfig {
	text := fmt.Sprintf("%s\n\n%s:%s:%d", prompt, prefix, id, sourceMessageID)
	return selectiveForceReplyConfig(chatID, user, text)
}

func parseBookFieldPrompt(original string) (field string, id string, sourceMessageID int, ok bool) {
	for _, line := range strings.Split(original, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(parts) < 2 || (parts[0] != "book_title" && parts[0] != "book_authors") {
			continue
		}
		if len(parts) == 3 {
			sourceMessageID, _ = strconv.Atoi(parts[2])
		}
		return parts[0], parts[1], sourceMessageID, true
	}
	return "", "", 0, false
}

func (lnb *LitNightBot) requestBookField(message *tgbotapi.Message, id string, action CallbackAction, user *tgbotapi.User, logger *logrus.Entry) {
	book := lnb.iocd.GetOrCreateChatData(message.Chat.ID).FindBook(id)
	if book == nil {
		lnb.SendPlainMessage(message.Chat.ID, "Книга не найдена.")
		return
	}
	prefix := "book_title"
	prompt := "Введите новое название."
	if action == CBBookEditAuthors {
		prefix = "book_authors"
		prompt = "Введите авторов через точку с запятой. Ответ «-» очистит авторов."
	}
	request := bookFieldRequestConfig(message.Chat.ID, user, prefix, id, message.MessageID, prompt)
	lnb.bot.Send(request)
}

func (lnb *LitNightBot) handleBookFieldReply(message *tgbotapi.Message, original string, logger *logrus.Entry) bool {
	field, id, sourceMessageID, ok := parseBookFieldPrompt(original)
	if !ok {
		return false
	}
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(id)
	if book == nil {
		lnb.SendPlainMessage(message.Chat.ID, "Книга не найдена.")
		return true
	}
	value := strings.TrimSpace(message.Text)
	if field == "book_title" {
		if value == "" {
			lnb.SendPlainMessage(message.Chat.ID, "Название не может быть пустым.")
			return true
		}
		book.Title = value
	} else if value == "-" || value == "" {
		book.Authors = []string{}
	} else {
		book.Authors = []string{}
		for _, author := range strings.Split(value, ";") {
			if author = strings.TrimSpace(author); author != "" {
				book.Authors = append(book.Authors, author)
			}
		}
	}
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		return true
	}
	private, userID := cardChatContext(message.Chat)
	if sourceMessageID > 0 {
		lnb.editHTMLMessage(message.Chat.ID, sourceMessageID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
	} else {
		lnb.SendHTMLMessage(message.Chat.ID, renderBookCardForChat(book, private, userID), bookCardButtonsForChat(book, private, userID))
	}
	if message.ReplyToMessage != nil {
		lnb.removeMessage(message.Chat.ID, message.ReplyToMessage.MessageID)
	}
	return true
}

func (lnb *LitNightBot) approveBookCard(message *tgbotapi.Message, id string, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(id)
	if book == nil {
		lnb.SendPlainMessage(message.Chat.ID, "Книга не найдена.")
		return
	}
	book.NeedsReview = false
	remaining := 0
	for _, candidate := range data.Books {
		if candidate.NeedsReview {
			remaining++
		}
	}
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		return
	}
	if remaining == 0 {
		lnb.editMessage(message.Chat.ID, message.MessageID, migrationCompleteText, nil)
		return
	}
	lnb.showBooksForReview(message.Chat.ID, message.MessageID, 0, logger)
}

func (lnb *LitNightBot) showBooksForReview(chatID int64, messageID int, page int, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	books := make([]chatdata.ClubBook, 0)
	for _, book := range data.Books {
		if book.NeedsReview {
			books = append(books, book)
		}
	}
	if len(books) == 0 {
		lnb.editMessage(chatID, messageID, migrationCompleteText, nil)
		return
	}
	maxPage := (len(books) - 1) / BooksPerPage
	if page < 0 {
		page = 0
	}
	if page > maxPage {
		page = maxPage
	}
	start := page * BooksPerPage
	end := start + BooksPerPage
	if end > len(books) {
		end = len(books)
	}
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, end-start+1)
	for i := start; i < end; i++ {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("⚠️ %d. %s", i+1, truncateButton(books[i].Title)), GetCallbackParamStr(CBBookShow, books[i].ID))))
	}
	nav := make([]tgbotapi.InlineKeyboardButton, 0, 2)
	if page > 0 {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("⬅", GetCallbackParamStr(CBBooksReview, strconv.Itoa(page-1))))
	}
	if page < maxPage {
		nav = append(nav, tgbotapi.NewInlineKeyboardButtonData("➡", GetCallbackParamStr(CBBooksReview, strconv.Itoa(page+1))))
	}
	if len(nav) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(nav...))
	}
	buttons = append(buttons, getMenuButton("Закрыть", CBCancel))
	text := fmt.Sprintf("⚠️ ПРОВЕРКА КАРТОЧЕК\n\nОсталось: %d\n\nОткройте карточку, исправьте название и авторов, затем нажмите «Карточка верна».", len(books))
	lnb.editMessage(chatID, messageID, text, buttons)
}
