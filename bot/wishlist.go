package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (vb *LitNightBot) handleWishlistRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string) {
	chatId := message.Chat.ID

	cd := vb.getChatData(chatId)
	_, err := cd.RemoveBookFromWishlist(cbParams[0])
	vb.setChatData(chatId, cd)

	if err != nil {
		vb.sendMessage(chatId, err.Error(), nil)
		return
	}

	callbackConfig := tgbotapi.NewCallback(
		cbId,
		"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
	)
	vb.bot.Send(callbackConfig)

	page, _ := strconv.Atoi(cbParams[1])
	vb.showCleanWishlistPage(chatId, message.MessageID, page)
}

func (vb *LitNightBot) handleShowWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		vb.sendMessage(
			chatId,
			"Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
				"Самое время добавить новые книги и продолжить наши литературные приключения!",
			nil,
		)
		return
	}

	vb.sendMessage(
		chatId,
		"📚 Ваши книги в вишлисте:\n\n"+GetBooklistString(&cd.Wishlist),
		nil,
	)
}

func (vb *LitNightBot) handleWishlistClean(message *tgbotapi.Message) {
	vb.showCleanWishlistPage(message.Chat.ID, -1, 0)
}

func (vb *LitNightBot) handleWishlistAddRequest(message *tgbotapi.Message) {
	vb.sendMessage(message.Chat.ID, addBooksToWishlistRequestMessage, nil)
}

func (vb *LitNightBot) handleWishlistAdd(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))

	cd := vb.getChatData(chatId)

	cd.AddBooksToWishlist(booknames)

	vb.setChatData(chatId, cd)

	var textMessage string
	if len(booknames) == 1 {
		textMessage = fmt.Sprintf("Книга \"%s\" добавлена.", booknames[0])
	} else {
		textMessage = fmt.Sprintf("Книги \"%s\" добавлены.", strings.Join(booknames, "\", \""))
	}

	vb.sendMessage(chatId, textMessage, nil)
}

func (vb *LitNightBot) getCleanWishlistMessage(chatId int64, page int) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := vb.getChatData(chatId)

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

func (vb *LitNightBot) showCleanWishlistPage(chatId int64, messageID int, page int) {
	messageText, buttons := vb.getCleanWishlistMessage(chatId, page)

	if messageID == -1 {
		vb.sendMessage(chatId, messageText, buttons)
	} else {
		vb.editMessage(chatId, messageID, messageText, buttons)
	}
}
