package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (vb *LitNightBot) handleHistoryAddBook(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))

	if len(booknames) == 0 {
		vb.sendMessage(chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
			nil,
		)
		return
	}

	cd := vb.getChatData(chatId)

	cd.AddBooksToHistory(booknames)

	vb.setChatData(chatId, cd)

	if len(booknames) == 1 {
		vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена в историю.", booknames[0]), nil)
	} else {
		vb.sendMessage(chatId, fmt.Sprintf("Книги \"%s\" добавлены в историю.", strings.Join(booknames, "\", \"")), nil)
	}
}

func (vb *LitNightBot) handleHistoryRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string) {
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
	vb.showCleanHistoryPage(chatId, message.MessageID, page)
}

func (vb *LitNightBot) handleHistoryShow(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	names := cd.GetHistoryBooks()

	if len(names) == 0 {
		vb.sendMessage(chatId,
			"Кажется, список прочитанных книг пока пуст... 😕\n"+
				"Но не переживайте! Начните прямо сейчас, и скоро здесь будут ваши книжные достижения! 📚💪",
			nil,
		)
		return
	}

	vb.sendMessage(
		chatId,
		"Вот ваши уже прочитанные книги:\n\n✔ "+strings.Join(names, "\n✔ ")+"\nОтличная работа! 👏📖",
		nil,
	)
}

func (vb *LitNightBot) handleCleanHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.showCleanHistoryPage(chatId, -1, 0)
}

func (vb *LitNightBot) GetCleanHistoryMessage(chatId int64, messageID int, page int) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := vb.getChatData(chatId)

	if len(cd.History) == 0 {
		return "Кажется, список прочитанных книг пока пуст... 😕\n", nil
	}

	booksOnPage, page, isLast := GetBooklistPage(&cd.History, page)

	buttons := GetCleanBooklistButtons(&booksOnPage, page, CBWishlistRemoveBook)

	navButtons := GetPaginationNavButtons(page, isLast, CBHistoryChangePage)

	if len(*navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(*navButtons...))
	}

	messageText := fmt.Sprintf("🗑️ Удаление из истории (страница %d):\n\n", page+1)

	return messageText, buttons
}

func (vb *LitNightBot) showCleanHistoryPage(chatId int64, messageID int, page int) {
	messageText, buttons := vb.GetCleanHistoryMessage(chatId, messageID, page)

	if messageID == -1 {
		vb.sendMessage(chatId, messageText, buttons)
	} else {
		vb.editMessage(chatId, messageID, messageText, buttons)
	}
}
