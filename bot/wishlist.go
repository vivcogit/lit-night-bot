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

	cd := lnb.iocd.GetChatData(chatId)
	bookID := cbParams[0]

	_, err := cd.RemoveBookFromWishlist(bookID)
	lnb.iocd.SetChatData(chatId, cd)

	if err != nil {
		logger.WithField("book_id", bookID).WithError(err).Error("Failed to remove book from wishlist")
		lnb.SendPlainMessage(chatId, err.Error())
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

	cd := lnb.iocd.GetChatData(chatId)

	if len(cd.Wishlist) == 0 {
		logger.Info("Wishlist is empty")
		lnb.SendPlainMessage(
			chatId,
			"Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
				"Самое время добавить новые книги и продолжить наши литературные приключения!",
		)
		return
	}

	lnb.SendPlainMessage(
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
	lnb.SendPlainMessage(message.Chat.ID, addBooksToWishlistRequestMessage)
}

func (lnb *LitNightBot) handleWishlistAdd(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))
	logger.WithField("booknames", booknames).Info("Adding books to wishlist")

	cd := lnb.iocd.GetChatData(chatId)
	cd.AddBooksToWishlist(booknames)

	lnb.iocd.SetChatData(chatId, cd)

	var textMessage string
	if len(booknames) == 1 {
		textMessage = fmt.Sprintf("Книга \"%s\" добавлена.", booknames[0])
	} else {
		textMessage = fmt.Sprintf("Книги \"%s\" добавлены.", strings.Join(booknames, "\", \""))
	}

	lnb.SendPlainMessage(chatId, textMessage)
	logger.Info("Books added to wishlist")
}

func (lnb *LitNightBot) getCleanWishlistMessage(chatId int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.iocd.GetChatData(chatId)
	return GetBooklistPageMessage(
		chatId, page, logger,
		&cd.Wishlist,
		"Ваш вишлист пуст, нечего удалять. Добавьте новые книги для удаления.",
		removePrefix,
		CBWishlistRemoveBook,
		CBWishlistChangePage,
		"🗑️ Удаление из вишлиста",
	)
}

func (lnb *LitNightBot) showCleanWishlistPage(chatId int64, messageID int, page int, logger *logrus.Entry) {
	messageText, buttons := lnb.getCleanWishlistMessage(chatId, page, logger)
	lnb.displayPage(chatId, messageID, messageText, buttons, logger)
}

func (lnb *LitNightBot) handleWishlistChooseFrom(message *tgbotapi.Message, logger *logrus.Entry) {
	logger.Info("Handling wishlist choose from")
	lnb.showChooseFromWishlistPage(message.Chat.ID, -1, 0, logger)
}

func (lnb *LitNightBot) getChooseFromWishlistMessage(chatId int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.iocd.GetChatData(chatId)
	return GetBooklistPageMessage(
		chatId, page, logger,
		&cd.Wishlist,
		"Ваш вишлист пуст, нечего выбирать. Добавьте новые книги чтобы было из чего выбрать.",
		choosePrefix,
		CBCurrentChooseBook,
		CBWishlistChoosePage,
		"📘 Выбор книги из вишлиста",
	)
}

func (lnb *LitNightBot) showChooseFromWishlistPage(chatId int64, messageID int, page int, logger *logrus.Entry) {
	messageText, buttons := lnb.getChooseFromWishlistMessage(chatId, page, logger)
	lnb.displayPage(chatId, messageID, messageText, buttons, logger)
}
