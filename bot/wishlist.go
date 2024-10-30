package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleWishlistRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string, logger *logrus.Entry) {
	chatId := message.Chat.ID
	logger.Info("Handling wishlist remove book")

	cd := lnb.getChatData(chatId)
	bookID := cbParams[0]

	_, err := cd.RemoveBookFromWishlist(bookID)
	lnb.setChatData(chatId, cd)

	if err != nil {
		logger.WithField("book_id", bookID).WithError(err).Error("Failed to remove book from wishlist")
		lnb.sendPlainMessage(chatId, err.Error())
		return
	}

	callbackConfig := tgbotapi.NewCallback(
		cbId,
		"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
	)
	lnb.bot.Send(callbackConfig)

	logger.WithField("book_id", bookID).Info("Book removed from wishlist")

	page, _ := strconv.Atoi(cbParams[1])
	lnb.showCleanWishlistPage(chatId, message.MessageID, page, logger)
}

func (lnb *LitNightBot) handleShowWishlist(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	logger.Info("Handling show wishlist")

	cd := lnb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		logger.Info("Wishlist is empty")
		lnb.sendPlainMessage(
			chatId,
			"Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
				"Самое время добавить новые книги и продолжить наши литературные приключения!",
		)
		return
	}

	lnb.sendPlainMessage(
		chatId,
		"📚 Ваши книги в вишлисте:\n\n"+GetBooklistString(&cd.Wishlist),
	)
	logger.Info("Displayed wishlist books")
}

func (lnb *LitNightBot) handleWishlistClean(message *tgbotapi.Message, logger *logrus.Entry) {
	logger.Info("Handling wishlist clean")
	lnb.showCleanWishlistPage(message.Chat.ID, -1, 0, logger)
}

func (lnb *LitNightBot) handleWishlistAddRequest(message *tgbotapi.Message, logger *logrus.Entry) {
	logger.Info("Handling request to add book to wishlist")
	lnb.sendPlainMessage(message.Chat.ID, addBooksToWishlistRequestMessage)
}

func (lnb *LitNightBot) handleWishlistAdd(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))
	logger.WithField("booknames", booknames).Info("Adding books to wishlist")

	cd := lnb.getChatData(chatId)
	cd.AddBooksToWishlist(booknames)

	lnb.setChatData(chatId, cd)

	var textMessage string
	if len(booknames) == 1 {
		textMessage = fmt.Sprintf("Книга \"%s\" добавлена.", booknames[0])
	} else {
		textMessage = fmt.Sprintf("Книги \"%s\" добавлены.", strings.Join(booknames, "\", \""))
	}

	lnb.sendPlainMessage(chatId, textMessage)
	logger.Info("Books added to wishlist")
}

func (lnb *LitNightBot) getCleanWishlistMessage(chatId int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		logger.Info("Wishlist is empty")
		return "Ваш вишлист пуст, нечего удалять. Добавьте новые книги для удаления.", nil
	}

	booksOnPage, page, isLast := GetBooklistPage(&cd.Wishlist, page)

	buttons := GetCleanBooklistButtons(&booksOnPage, page, CBWishlistRemoveBook)

	navButtons := GetPaginationNavButtons(page, isLast, CBWishlistChangePage)

	if len(*navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(*navButtons...))
	}

	messageText := fmt.Sprintf("🗑️ Удаление из вишлиста (страница %d):\n\n", page+1)

	return messageText, buttons
}

func (lnb *LitNightBot) showCleanWishlistPage(chatId int64, messageID int, page int, logger *logrus.Entry) {
	logger.WithField("messageID", messageID).WithField("page", page).Info("Showing clean wishlist page")
	messageText, buttons := lnb.getCleanWishlistMessage(chatId, page, logger)

	if messageID == -1 {
		lnb.sendMessage(chatId, SendMessageParams{Text: messageText, Buttons: buttons})
	} else {
		lnb.editMessage(chatId, messageID, messageText, buttons)
	}
}
