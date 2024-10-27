package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (lnb *LitNightBot) handleWishlistRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string) {
	chatId := message.Chat.ID

	cd := lnb.getChatData(chatId)
	_, err := cd.RemoveBookFromWishlist(cbParams[0])
	lnb.setChatData(chatId, cd)

	if err != nil {
		lnb.sendPlainMessage(chatId, err.Error())
		return
	}

	callbackConfig := tgbotapi.NewCallback(
		cbId,
		"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
	)
	lnb.bot.Send(callbackConfig)

	page, _ := strconv.Atoi(cbParams[1])
	lnb.showCleanWishlistPage(chatId, message.MessageID, page)
}

func (lnb *LitNightBot) handleShowWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
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
}

func (lnb *LitNightBot) handleWishlistClean(message *tgbotapi.Message) {
	lnb.showCleanWishlistPage(message.Chat.ID, -1, 0)
}

func (lnb *LitNightBot) handleWishlistAddRequest(message *tgbotapi.Message) {
	lnb.sendPlainMessage(message.Chat.ID, addBooksToWishlistRequestMessage)
}

func (lnb *LitNightBot) handleWishlistAdd(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))

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
}

func (lnb *LitNightBot) getCleanWishlistMessage(chatId int64, page int) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
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

func (lnb *LitNightBot) showCleanWishlistPage(chatId int64, messageID int, page int) {
	messageText, buttons := lnb.getCleanWishlistMessage(chatId, page)

	if messageID == -1 {
		lnb.sendMessage(chatId, SendMessageParams{text: messageText, buttons: buttons})
	} else {
		lnb.editMessage(chatId, messageID, messageText, buttons)
	}
}
