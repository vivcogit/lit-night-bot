package bot

import (
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleWishlistRemoveBook(message *tgbotapi.Message, callbackID string, params []string, logger *logrus.Entry) {
	if len(params) < 2 {
		return
	}
	chatID := message.Chat.ID
	data := lnb.iocd.GetOrCreateChatData(chatID)
	book := data.FindBook(params[0])
	if book == nil || book.Status != chatdata.StatusWishlist {
		lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Книга уже удалена"))
		return
	}
	if _, err := data.RemoveBook(book.ID); err != nil {
		lnb.SendPlainMessage(chatID, err.Error())
		return
	}
	lnb.iocd.SetChatData(chatID, data)
	lnb.bot.Request(tgbotapi.NewCallback(callbackID, "Книга удалена из вишлиста"))
	page, _ := strconv.Atoi(params[1])
	lnb.showCleanWishlistPage(chatID, message.MessageID, page, logger)
}

func (lnb *LitNightBot) handleShowWishlist(message *tgbotapi.Message, logger *logrus.Entry) {
	chatID := message.Chat.ID
	books := lnb.iocd.GetOrCreateChatData(chatID).BooksWithStatus(chatdata.StatusWishlist)
	if len(books) == 0 {
		lnb.SendPlainMessage(chatID, "Вишлист пуст. Добавьте новые книги.")
		return
	}
	var text strings.Builder
	text.WriteString("📚 <b>ВИШЛИСТ</b>\n")
	text.WriteString(fmt.Sprintf("Книг: %d\n\n", len(books)))
	buttons := make([][]tgbotapi.InlineKeyboardButton, 0, len(books))
	for i, book := range books {
		text.WriteString(fmt.Sprintf("%d. %s\n", i+1, html.EscapeString(book.DisplayName())))
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("📖 %d. %s", i+1, truncateButton(book.Title)), GetCallbackParamStr(CBBookShow, book.ID))))
	}
	buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)),
	))
	lnb.SendHTMLMessage(chatID, text.String(), buttons)
}

func (lnb *LitNightBot) handleWishlistClean(message *tgbotapi.Message, logger *logrus.Entry) {
	lnb.showCleanWishlistPage(message.Chat.ID, -1, 0, logger)
}

func wishlistAddRequestConfig(chatID int64, user *tgbotapi.User) tgbotapi.MessageConfig {
	return selectiveForceReplyConfig(chatID, user, addBooksToWishlistRequestMessage)
}

func (lnb *LitNightBot) handleWishlistAddRequest(message *tgbotapi.Message, user *tgbotapi.User, logger *logrus.Entry) {
	request := wishlistAddRequestConfig(message.Chat.ID, user)
	if _, err := lnb.bot.Send(request); err != nil {
		logger.WithError(err).Error("Failed to send wishlist ForceReply request")
	}
}

func (lnb *LitNightBot) handleWishlistAdd(message *tgbotapi.Message, logger *logrus.Entry) {
	chatID := message.Chat.ID
	lines := utils.CleanStrSlice(strings.Split(message.Text, "\n"))
	if len(lines) == 0 {
		lnb.SendPlainMessage(chatID, "Не найдено ни одной книги.")
		return
	}
	data := lnb.iocd.GetOrCreateChatData(chatID)
	added := make([]string, 0, len(lines))
	for _, line := range lines {
		title, authors := chatdata.ParseStructuredBook(line)
		if title == "" {
			continue
		}
		book := data.AddBook(title, authors, chatdata.StatusWishlist, time.Now())
		added = append(added, book.DisplayName())
	}
	if len(added) == 0 {
		lnb.SendPlainMessage(chatID, "Не найдено корректных названий.")
		return
	}
	lnb.iocd.SetChatData(chatID, data)
	lnb.SendPlainMessage(chatID, fmt.Sprintf("✅ Добавлено книг: %d\n%s", len(added), strings.Join(added, "\n")))
}

func (lnb *LitNightBot) getCleanWishlistMessage(chatID int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	books := lnb.iocd.GetOrCreateChatData(chatID).BooksWithStatus(chatdata.StatusWishlist)
	return GetBooklistPageMessage(chatID, page, logger, &books, "Вишлист пус.", removePrefix, CBWishlistRemoveBook, CBWishlistChangePage, "🗑️ Удаление из вишлиста")
}

func (lnb *LitNightBot) showCleanWishlistPage(chatID int64, messageID int, page int, logger *logrus.Entry) {
	text, buttons := lnb.getCleanWishlistMessage(chatID, page, logger)
	lnb.displayPage(chatID, messageID, text, buttons, logger)
}

func (lnb *LitNightBot) handleWishlistChooseFrom(message *tgbotapi.Message, logger *logrus.Entry) {
	lnb.showChooseFromWishlistPage(message.Chat.ID, -1, 0, logger)
}

func (lnb *LitNightBot) getChooseFromWishlistMessage(chatID int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	data := lnb.iocd.GetOrCreateChatData(chatID)
	if warning := lnb.checkCanChooseBook(data); warning != "" {
		return warning, nil
	}
	books := data.BooksWithStatus(chatdata.StatusWishlist)
	return GetBooklistPageMessage(chatID, page, logger, &books, "Вишлист пус.", choosePrefix, CBCurrentChooseBook, CBWishlistChoosePage, "📘 Выбор книги")
}

func (lnb *LitNightBot) showChooseFromWishlistPage(chatID int64, messageID int, page int, logger *logrus.Entry) {
	text, buttons := lnb.getChooseFromWishlistMessage(chatID, page, logger)
	lnb.displayPage(chatID, messageID, text, buttons, logger)
}

func truncateButton(text string) string {
	const limit = 42
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}
