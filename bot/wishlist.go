package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getCleanWishlistMessage(cd *chatdata.ChatData) (string, [][]tgbotapi.InlineKeyboardButton) {
	buttons := GetButtonsForBooklist(
		&cd.Wishlist,
		"❌",
		func(uuid string) string {
			return GetCallbackParamStr(CBRemove, uuid)
		},
	)

	if len(cd.Wishlist) == 0 {
		return "Список книг пуст, удалять нечего", buttons
	}

	return "Вот ваши книги в списке:", buttons
}

func getWishlistMessage(books []string) string {
	var formattedList strings.Builder
	formattedList.WriteString("📚 Ваши книги в вишлисте:\n\n")

	for i, book := range books {
		formattedList.WriteString(fmt.Sprintf("%d. %s\n", i+1, book))
	}

	formattedList.WriteString("\n🎉 Не забудьте выбрать книгу для чтения!")

	return formattedList.String()
}

func (vb *LitNightBot) handleWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	names := cd.GetWishlistBooks()

	if len(names) == 0 {
		vb.sendMessage(chatId, "Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
			"Самое время добавить новые книги и продолжить наши литературные приключения!")
		return
	}

	vb.sendMessage(chatId, getWishlistMessage(names))
}

func (vb *LitNightBot) handleRemoveFromWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.showRemoveWishlistPage(chatId, -1, 0)
}

func (vb *LitNightBot) showRemoveWishlistPage(chatId int64, messageID int, page int) {
	cd := vb.getChatData(chatId)
	books := cd.GetWishlistBooks()

	if len(books) == 0 {
		vb.sendMessage(chatId, "Ваш вишлист пуст, нечего удалять. Добавьте новые книги для удаления.")
		return
	}

	start := page * BooksPerPage
	end := start + BooksPerPage
	if end > len(books) {
		end = len(books)
	}

	booksOnPage := books[start:end]
	messageText := fmt.Sprintf("🗑️ Удаление книг (страница %d):\n\n", page+1)
	var buttons [][]tgbotapi.InlineKeyboardButton
	for i, book := range booksOnPage {
		button := tgbotapi.NewInlineKeyboardButtonData(
			book,
			GetCallbackParamStr(CBRemove, strconv.Itoa(start+i)),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(
			navButtons,
			tgbotapi.NewInlineKeyboardButtonData(
				"⬅ Назад",
				GetCallbackParamStr(CBRemovePage, strconv.Itoa(page-1)),
			),
		)
	}
	if end < len(books) {
		navButtons = append(
			navButtons,
			tgbotapi.NewInlineKeyboardButtonData(
				"Вперед ➡",
				GetCallbackParamStr(CBRemovePage, strconv.Itoa(page+1)),
			),
		)
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(navButtons...))
	}

	if messageID == -1 {
		msg := tgbotapi.NewMessage(chatId, messageText)
		if len(buttons) > 0 {
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		}

		vb.bot.Send(msg)
	} else {
		vb.editMessage(chatId, messageID, messageText, buttons)
	}
}
