package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (lnb *LitNightBot) handleHistoryAddBook(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.Text, "\n"))

	if len(booknames) == 0 {
		lnb.sendPlainMessage(
			chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
		)
		return
	}

	cd := lnb.getChatData(chatId)

	cd.AddBooksToHistory(booknames)

	lnb.setChatData(chatId, cd)

	var msgText string
	if len(booknames) == 1 {
		msgText = fmt.Sprintf("Книга \"%s\" добавлена в историю.", booknames[0])
	} else {
		msgText = fmt.Sprintf("Книги \"%s\" добавлены в историю.", strings.Join(booknames, "\", \""))
	}
	lnb.sendPlainMessage(chatId, msgText)
}

func (lnb *LitNightBot) handleHistoryRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string) {
	chatId := message.Chat.ID

	cd := lnb.getChatData(chatId)
	_, err := cd.RemoveBookFromHistory(cbParams[0])
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
	lnb.showCleanHistoryPage(chatId, message.MessageID, page)
}

func (lnb *LitNightBot) handleHistoryShow(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	names := cd.GetHistoryBooks()

	if len(names) == 0 {
		lnb.sendPlainMessage(
			chatId,
			"Кажется, список прочитанных книг пока пуст... 😕\n"+
				"Но не переживайте! Начните прямо сейчас, и скоро здесь будут ваши книжные достижения! 📚💪",
		)
		return
	}

	lnb.sendPlainMessage(
		chatId,
		"Вот ваши уже прочитанные книги:\n\n✔ "+strings.Join(names, "\n✔ ")+"\n\nОтличная работа! 👏📖",
	)
}

func (lnb *LitNightBot) handleCleanHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	lnb.showCleanHistoryPage(chatId, -1, 0)
}

func (lnb *LitNightBot) GetCleanHistoryMessage(chatId int64, messageID int, page int) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.getChatData(chatId)

	if len(cd.History) == 0 {
		return "Кажется, список прочитанных книг пока пуст... 😕\n", nil
	}

	booksOnPage, page, isLast := GetBooklistPage(&cd.History, page)

	buttons := GetCleanBooklistButtons(&booksOnPage, page, CBHistoryRemoveBook)

	navButtons := GetPaginationNavButtons(page, isLast, CBHistoryChangePage)

	if len(*navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(*navButtons...))
	}

	messageText := fmt.Sprintf("🗑️ Удаление из истории (страница %d):\n\n", page+1)

	return messageText, buttons
}

func (lnb *LitNightBot) showCleanHistoryPage(chatId int64, messageID int, page int) {
	messageText, buttons := lnb.GetCleanHistoryMessage(chatId, messageID, page)

	if messageID == -1 {
		lnb.sendMessage(chatId, SendMessageParams{text: messageText, buttons: buttons})
	} else {
		lnb.editMessage(chatId, messageID, messageText, buttons)
	}
}
