package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleHistoryAddBook(update *tgbotapi.Update, logger *logrus.Entry) {
	chatId := getUpdateChatID(update)
	booknames := utils.CleanStrSlice(strings.Split(update.Message.Text, "\n"))

	if len(booknames) == 0 {
		lnb.sendPlainMessage(
			chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
		)
		logger.Info("Empty command")
		return
	}

	cd := lnb.getChatData(chatId)
	cd.AddBooksToHistory(booknames)
	lnb.setChatData(chatId, cd)

	logger.WithFields(logrus.Fields{
		"books": booknames,
	}).Info("Books added to history")

	var msgText string
	if len(booknames) == 1 {
		msgText = fmt.Sprintf("Книга \"%s\" добавлена в историю.", booknames[0])
	} else {
		msgText = fmt.Sprintf("Книги \"%s\" добавлены в историю.", strings.Join(booknames, "\", \""))
	}
	lnb.sendPlainMessage(chatId, msgText)
}

func (lnb *LitNightBot) handleHistoryRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string, logger *logrus.Entry) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	bookName := cbParams[0]
	_, err := cd.RemoveBookFromHistory(bookName)
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
	lnb.showCleanHistoryPage(chatId, message.MessageID, page, logger)

	logger.WithFields(logrus.Fields{
		"book": bookName,
		"page": page,
	}).Info("Book removed from history")
}

func (lnb *LitNightBot) handleHistoryShow(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)
	names := cd.GetHistoryBooks()

	if len(names) == 0 {
		lnb.sendEmptyHistoryMessage(chatId)
		return
	}

	lnb.sendPlainMessage(
		chatId,
		"Вот ваши уже прочитанные книги:\n\n✔ "+strings.Join(names, "\n✔ ")+"\n\nОтличная работа! 👏📖",
	)

	logger.Info("Displayed book history")
}

func (lnb *LitNightBot) handleCleanHistory(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	lnb.showCleanHistoryPage(chatId, -1, 0, logger)

	logger.Info("Displayed clean history page")
}

func (lnb *LitNightBot) sendEmptyHistoryMessage(chatId int64) {
	lnb.sendPlainMessage(
		chatId,
		"Кажется, список прочитанных книг пока пуст... 😕\n"+
			"Но не переживайте! Начните прямо сейчас, и скоро здесь будут ваши книжные достижения! 📚💪",
	)
}

func (lnb *LitNightBot) GetCleanHistoryMessage(chatId int64, messageID int, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
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

func (lnb *LitNightBot) showCleanHistoryPage(chatId int64, messageID int, page int, logger *logrus.Entry) {
	messageText, buttons := lnb.GetCleanHistoryMessage(chatId, messageID, page, logger)

	if messageID == -1 {
		lnb.sendMessage(chatId, SendMessageParams{Text: messageText, Buttons: buttons})
	} else {
		lnb.editMessage(chatId, messageID, messageText, buttons)
	}

	logger.WithFields(logrus.Fields{"page": page}).Info("Displayed clean history page")
}
