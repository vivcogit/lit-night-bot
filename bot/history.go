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

func (lnb *LitNightBot) handleHistoryAddBook(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	lines := strings.Split(update.Message.CommandArguments(), "\n")
	data := lnb.iocd.GetOrCreateChatData(chatID)
	count := 0
	for _, line := range lines {
		title, authors := chatdata.ParseStructuredBook(strings.TrimSpace(line))
		if title == "" {
			continue
		}
		data.AddBook(title, authors, chatdata.StatusCompleted, time.Now())
		count++
	}
	if count == 0 {
		lnb.SendPlainMessage(chatID, "Формат: /h_add Название | Автор")
		return
	}
	if !lnb.saveChatData(chatID, data, logger) {
		return
	}
	lnb.SendPlainMessage(chatID, fmt.Sprintf("✅ В историю добавлено книг: %d", count))
}

func (lnb *LitNightBot) handleHistoryRemoveBook(message *tgbotapi.Message, callbackID string, params []string, logger *logrus.Entry) {
	if len(params) < 2 {
		return
	}
	chatID := message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(params[0])
	if book == nil || !isHistoryBook(book) {
		lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Книга уже удалена"))
		return
	}
	page, _ := strconv.Atoi(params[1])
	if len(book.Ratings) > 0 {
		buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Удалить книгу и оценки", GetCallbackParamStr(CBHistoryRemoveConfirm, book.ID, strconv.Itoa(page))),
			tgbotapi.NewInlineKeyboardButtonData("Отмена", GetCallbackParamStr(CBHistoryRemoveCancel, strconv.Itoa(page))),
		)}
		text := fmt.Sprintf("⚠️ У книги «%s» есть %s. При удалении книги они также будут удалены. Продолжить?", book.DisplayName(), ratingCountLabel(len(book.Ratings)))
		lnb.editMessage(message.Chat.ID, message.MessageID, text, buttons)
		lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Нужно подтверждение"))
		return
	}
	lnb.removeHistoryBook(message, callbackID, book.ID, page, logger)
}

func (lnb *LitNightBot) confirmHistoryRemoveBook(message *tgbotapi.Message, callbackID string, params []string, logger *logrus.Entry) {
	if len(params) < 2 {
		return
	}
	page, _ := strconv.Atoi(params[1])
	lnb.removeHistoryBook(message, callbackID, params[0], page, logger)
}

func (lnb *LitNightBot) removeHistoryBook(message *tgbotapi.Message, callbackID string, bookID string, page int, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	book := data.FindBook(bookID)
	if book == nil || !isHistoryBook(book) {
		lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Книга уже удалена"))
		return
	}
	if _, err := data.RemoveBook(book.ID); err != nil {
		lnb.SendPlainMessage(message.Chat.ID, err.Error())
		return
	}
	if !lnb.saveChatData(message.Chat.ID, data, logger) {
		return
	}
	lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Книга удалена из истории"))
	lnb.showCleanHistoryPage(message.Chat.ID, message.MessageID, page, logger)
}

func completedYear(book chatdata.ClubBook) int {
	return historyBookDate(book).Year()
}

func historyBookDate(book chatdata.ClubBook) time.Time {
	if book.CompletedAt != nil {
		return *book.CompletedAt
	}
	if book.StoppedAt != nil {
		return *book.StoppedAt
	}
	return book.AddedAt
}

func isHistoryBook(book *chatdata.ClubBook) bool {
	return book != nil && (book.Status == chatdata.StatusCompleted || book.Status == chatdata.StatusUnfinished)
}

func historyBooks(data *chatdata.ChatData) []chatdata.ClubBook {
	return data.BooksWithStatus(chatdata.StatusCompleted, chatdata.StatusUnfinished)
}

func (lnb *LitNightBot) handleHistoryShow(message *tgbotapi.Message, logger *logrus.Entry) {
	chatID := message.Chat.ID
	books := historyBooks(lnb.iocd.GetOrCreateChatData(chatID))
	if len(books) == 0 {
		lnb.sendEmptyHistoryMessage(chatID)
		return
	}
	lnb.sendHistoryYear(chatID, -1, time.Now().Year(), logger)
}

func historyBooksForYear(all []chatdata.ClubBook, year int) []chatdata.ClubBook {
	books := make([]chatdata.ClubBook, 0)
	for index := len(all) - 1; index >= 0; index-- {
		book := all[index]
		if completedYear(book) == year {
			books = append(books, book)
		}
	}
	sort.SliceStable(books, func(i, j int) bool {
		left := historyBookDate(books[i])
		right := historyBookDate(books[j])
		return left.After(right)
	})
	return books
}

func historyYears(all []chatdata.ClubBook) ([]int, map[int]int) {
	counts := make(map[int]int)
	for _, book := range all {
		counts[completedYear(book)]++
	}
	years := make([]int, 0, len(counts))
	for year := range counts {
		years = append(years, year)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, counts
}

func renderHistoryYear(year int, books []chatdata.ClubBook, total int, currentYear int, hasArchive bool) string {
	return renderHistoryYearForChat(year, books, total, currentYear, hasArchive, false, 0)
}

func renderHistoryYearForChat(year int, books []chatdata.ClubBook, total int, currentYear int, hasArchive bool, private bool, userID int64) string {
	var text strings.Builder
	if private {
		text.WriteString("📚 <b>МОЙ ЧИТАТЕЛЬСКИЙ ДНЕВНИК</b>\n")
	} else {
		text.WriteString("📚 <b>ИСТОРИЯ КНИЖНОГО КЛУБА</b>\n")
	}
	if private {
		text.WriteString(fmt.Sprintf("<i>Всего книг в дневнике: %d</i>\n\n", total))
	} else {
		text.WriteString(fmt.Sprintf("<i>Всего книг в истории: %d</i>\n\n", total))
	}
	text.WriteString(fmt.Sprintf("<b>📖 %d · %d книг</b>\n\n", year, len(books)))
	if len(books) == 0 {
		if private {
			text.WriteString("В этом году вы пока не завершили ни одной книги.\n")
		} else {
			text.WriteString("В этом году клуб пока не завершил ни одной книги.\n")
		}
	} else {
		for index, book := range books {
			statusIcon := "✅"
			if book.Status == chatdata.StatusUnfinished {
				statusIcon = "🚫"
			}
			text.WriteString(fmt.Sprintf("%d. %s <b>%s</b>", index+1, statusIcon, html.EscapeString(book.Title)))
			if len(book.Authors) > 0 {
				text.WriteString(" — " + html.EscapeString(strings.Join(book.Authors, ", ")))
			}
			text.WriteString("\n")
			if private {
				if rating := book.RatingByUser(userID); rating != nil {
					text.WriteString(fmt.Sprintf("   ⭐ Моя оценка: %d\n", rating.Value))
				}
			} else if len(book.Ratings) > 0 {
				text.WriteString(fmt.Sprintf("   ⭐ %s · %s\n", formatAverageRating(book.Ratings), ratingCountLabel(len(book.Ratings))))
			}
		}
	}
	if hasArchive && year == currentYear {
		text.WriteString("\n<b>Архив:</b>")
	}
	return text.String()
}

func (lnb *LitNightBot) historyYearButtons(year int, books []chatdata.ClubBook, years []int, counts map[int]int, currentYear int) [][]tgbotapi.InlineKeyboardButton {
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, len(years)+3)
	appendArchiveYears := func() {
		for _, archiveYear := range years {
			if archiveYear == currentYear || archiveYear == year {
				continue
			}
			buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%d · %d книг", archiveYear, counts[archiveYear]), GetCallbackParamStr(CBHistoryYear, strconv.Itoa(archiveYear)))))
		}
	}
	if year == currentYear {
		appendArchiveYears()
	}
	if len(books) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("📖 Открыть карточку", GetCallbackParamStr(CBHistoryPick, strconv.Itoa(year)))))
	}
	if year != currentYear {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("← Вернуться к %d", currentYear), GetCallbackParamStr(CBHistoryYear, strconv.Itoa(currentYear)))))
		appendArchiveYears()
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel))))
	return buttons
}

func (lnb *LitNightBot) showHistoryYear(chatID int64, messageID int, year int, logger *logrus.Entry) {
	lnb.sendHistoryYear(chatID, messageID, year, logger)
}

func (lnb *LitNightBot) sendHistoryYear(chatID int64, messageID int, year int, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	all := historyBooks(data)
	books := historyBooksForYear(all, year)
	years, counts := historyYears(all)
	currentYear := time.Now().Year()
	hasArchive := false
	for _, candidate := range years {
		if candidate != currentYear {
			hasArchive = true
			break
		}
	}
	private, userID := ratingChatContext(data, chatID)
	text := renderHistoryYearForChat(year, books, len(all), currentYear, hasArchive, private, userID)
	buttons := lnb.historyYearButtons(year, books, years, counts, currentYear)
	if messageID < 0 {
		lnb.SendHTMLMessage(chatID, text, buttons)
		return
	}
	lnb.editHTMLMessage(chatID, messageID, text, buttons)
}

type historySelectionKey struct {
	ChatID    int64
	MessageID int
}

type historySelection struct {
	UserID  int64
	Year    int
	BookIDs []string
}

func (lnb *LitNightBot) requestHistoryBookNumber(message *tgbotapi.Message, year int, user *tgbotapi.User, logger *logrus.Entry) {
	if user == nil {
		logger.Warn("History selection callback has no user")
		return
	}
	all := historyBooks(lnb.iocd.GetOrCreateChatData(message.Chat.ID))
	books := historyBooksForYear(all, year)
	if len(books) == 0 {
		lnb.SendPlainMessage(message.Chat.ID, "В этом году нет книг для выбора.")
		return
	}
	bookIDs := make([]string, len(books))
	for index, book := range books {
		bookIDs[index] = book.ID
	}
	lnb.sendHistorySelectionPrompt(message.Chat.ID, historySelection{UserID: user.ID, Year: year, BookIDs: bookIDs}, user, logger)
}

func historySelectionPromptConfig(chatID int64, selection historySelection, user *tgbotapi.User) tgbotapi.MessageConfig {
	text := fmt.Sprintf("Введите номер книги из списка за %d год.\n\nДоступные номера: 1–%d", selection.Year, len(selection.BookIDs))
	return selectiveForceReplyConfig(chatID, user, text)
}

func (lnb *LitNightBot) sendHistorySelectionPrompt(chatID int64, selection historySelection, user *tgbotapi.User, logger *logrus.Entry) {
	request := historySelectionPromptConfig(chatID, selection, user)
	sent, err := lnb.bot.Send(request)
	if err != nil {
		logger.WithError(err).Error("Failed to request history book number")
		return
	}
	key := historySelectionKey{ChatID: chatID, MessageID: sent.MessageID}
	lnb.historySelections.Store(key, selection)
}

func (lnb *LitNightBot) handleHistorySelectionReply(message *tgbotapi.Message, original *tgbotapi.Message, logger *logrus.Entry) bool {
	key := historySelectionKey{ChatID: message.Chat.ID, MessageID: original.MessageID}
	value, exists := lnb.historySelections.Load(key)
	if !exists {
		return false
	}
	selection := value.(historySelection)
	number, err := strconv.Atoi(strings.TrimSpace(message.Text))
	if err != nil || number < 1 || number > len(selection.BookIDs) {
		lnb.historySelections.Delete(key)
		lnb.SendPlainMessage(message.Chat.ID, fmt.Sprintf("Нужен номер от 1 до %d.", len(selection.BookIDs)))
		lnb.sendHistorySelectionPrompt(message.Chat.ID, selection, message.From, logger)
		return true
	}
	lnb.historySelections.Delete(key)
	lnb.handleBookCard(message, selection.BookIDs[number-1], logger)
	return true
}

func averageRating(ratings []chatdata.Rating) float64 {
	if len(ratings) == 0 {
		return 0
	}
	total := 0
	for _, rating := range ratings {
		total += rating.Value
	}
	return float64(total) / float64(len(ratings))
}

func (lnb *LitNightBot) handleCleanHistory(message *tgbotapi.Message, logger *logrus.Entry) {
	lnb.showCleanHistoryPage(message.Chat.ID, -1, 0, logger)
}

func (lnb *LitNightBot) sendEmptyHistoryMessage(chatID int64) {
	if data := lnb.iocd.GetOrCreateChatData(chatID); data.IsPrivateChat(chatID) {
		lnb.SendPlainMessage(chatID, "Читательский дневник пока пуст. Завершите первую книгу, и она появится здесь. 📚")
		return
	}
	lnb.SendPlainMessage(chatID, "История пока пуста. Завершите первую книгу, и она появится здесь. 📚")
}

func (lnb *LitNightBot) getCleanHistoryMessage(chatID int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	books := historyBooks(lnb.iocd.GetOrCreateChatData(chatID))
	return GetBooklistPageMessage(chatID, page, logger, &books, "История пуста.", removePrefix, CBHistoryRemoveBook, CBHistoryChangePage, "🗑️ Удаление из истории")
}

func (lnb *LitNightBot) showCleanHistoryPage(chatID int64, messageID int, page int, logger *logrus.Entry) {
	text, buttons := lnb.getCleanHistoryMessage(chatID, page, logger)
	lnb.displayPage(chatID, messageID, text, buttons, logger)
}
