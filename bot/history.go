package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"sort"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleHistoryAddBook(update *tgbotapi.Update, logger *logrus.Entry) {
	chatId, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}
	booknames := utils.CleanStrSlice(strings.Split(update.Message.Text, "\n"))

	if len(booknames) == 0 {
		lnb.SendPlainMessage(
			chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
		)
		logger.Info("Empty command")
		return
	}

	cd := lnb.iocd.GetChatData(chatId)
	cd.AddBooksToHistory(booknames)
	lnb.iocd.SetChatData(chatId, cd)

	logger.WithFields(logrus.Fields{
		"books": booknames,
	}).Info("Books added to history")

	var msgText string
	if len(booknames) == 1 {
		msgText = fmt.Sprintf("Книга \"%s\" добавлена в историю.", booknames[0])
	} else {
		msgText = fmt.Sprintf("Книги \"%s\" добавлены в историю.", strings.Join(booknames, "\", \""))
	}
	lnb.SendPlainMessage(chatId, msgText)
}

func (lnb *LitNightBot) handleHistoryRemoveBook(message *tgbotapi.Message, cbId string, cbParams []string, logger *logrus.Entry) {
	chatId := message.Chat.ID
	cd := lnb.iocd.GetChatData(chatId)

	bookName := cbParams[0]
	_, err := cd.RemoveBookFromHistory(bookName)
	lnb.iocd.SetChatData(chatId, cd)

	if err != nil {
		lnb.SendPlainMessage(chatId, err.Error())
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

func formatHistoryItems(items *[]chatdata.HistoryItem) string {
	currentYear := time.Now().Year()
	grouped := make(map[int][]string)

	for _, item := range *items {
		year := item.Date.Year()
		grouped[year] = append(grouped[year], item.Book.Name)
	}

	var years []int
	for year := range grouped {
		years = append(years, year)
	}

	sort.Sort(sort.Reverse(sort.IntSlice(years)))

	var sb strings.Builder
	for _, year := range years {
		if year != currentYear {
			sb.WriteString(fmt.Sprintf("%d\n", year))
		}
		for _, name := range grouped[year] {
			sb.WriteString(fmt.Sprintf("- %s\n", name))
		}
	}

	return sb.String()
}

func (lnb *LitNightBot) handleHistoryShow(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	cd := lnb.iocd.GetChatData(chatId)
	names := cd.GetHistoryBooks()

	if len(names) == 0 {
		lnb.sendEmptyHistoryMessage(chatId)
		return
	}

	lnb.SendPlainMessage(
		chatId,
		"Вот ваши уже прочитанные книги:\n\n"+
			formatHistoryItems(&cd.History)+
			"\nОтличная работа! 👏📖",
	)

	logger.Info("Displayed book history")
}

func (lnb *LitNightBot) handleCleanHistory(message *tgbotapi.Message, logger *logrus.Entry) {
	chatId := message.Chat.ID
	lnb.showCleanHistoryPage(chatId, -1, 0, logger)

	logger.Info("Displayed clean history page")
}

func (lnb *LitNightBot) sendEmptyHistoryMessage(chatId int64) {
	lnb.SendPlainMessage(
		chatId,
		"Кажется, список прочитанных книг пока пуст... 😕\n"+
			"Но не переживайте! Начните прямо сейчас, и скоро здесь будут ваши книжные достижения! 📚💪",
	)
}

func (lnb *LitNightBot) getCleanHistoryMessage(chatId int64, page int, logger *logrus.Entry) (string, [][]tgbotapi.InlineKeyboardButton) {
	cd := lnb.iocd.GetChatData(chatId)
	return GetBooklistPageMessage(
		chatId, page, logger,
		&cd.History,
		"Кажется, список прочитанных книг пока пуст... 😕\n",
		removePrefix,
		CBHistoryRemoveBook,
		CBHistoryChangePage,
		"🗑️ Удаление из истории",
	)
}

func (lnb *LitNightBot) showCleanHistoryPage(chatId int64, messageID int, page int, logger *logrus.Entry) {
	messageText, buttons := lnb.getCleanHistoryMessage(chatId, page, logger)
	lnb.displayPage(chatId, messageID, messageText, buttons, logger)
}
