package bot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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

func (vb *LitNightBot) handleRemoveFromWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.showRemoveWishlistPage(chatId, -1, 0)
}

func (vb *LitNightBot) GetCleanWishlistMessage(chatId int64, messageID int, page int) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		return "Ваш вишлист пуст, нечего удалять. Добавьте новые книги для удаления.", nil
	}

	booksOnPage, page, isLast := GetBooklistPage(&cd.Wishlist, page)

	buttons := GetCleanBooklistButtons(&booksOnPage, page, CBRemove)

	navButtons := GetPaginationNavButtons(page, isLast, CBRemovePage)

	if len(*navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(*navButtons...))
	}

	messageText := fmt.Sprintf("🗑️ Удаление из вишлиста (страница %d):\n\n", page+1)

	return messageText, buttons
}

func (vb *LitNightBot) showRemoveWishlistPage(chatId int64, messageID int, page int) {
	messageText, buttons := vb.GetCleanWishlistMessage(chatId, messageID, page)

	if messageID == -1 {
		vb.sendMessage(chatId, messageText, buttons)
	} else {
		vb.editMessage(chatId, messageID, messageText, buttons)
	}
}
