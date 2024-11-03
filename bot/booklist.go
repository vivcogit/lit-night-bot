package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func GetBooklistPageMessage[T chatdata.HasBook](
	chatId int64,
	page int,
	logger *logrus.Entry,
	booklist *[]T,
	emptyMessage string,
	prefix string,
	callbackItem CallbackAction,
	callbackChangePage CallbackAction,
	title string,
) (string, [][]tgbotapi.InlineKeyboardButton) {
	if len(*booklist) == 0 {
		logger.Info("List is empty")
		return emptyMessage, nil
	}

	booksOnPage, page, isLast := GetBooklistPage(booklist, page)
	buttons := GetButtonsForBooklist(&booksOnPage, prefix, callbackItem, page)
	navButtons := GetPaginationNavButtons(page, isLast, callbackChangePage)

	if len(*navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(*navButtons...))
	}

	messageText := fmt.Sprintf("%s (страница %d):\n\n", title, page+1)
	return messageText, buttons
}

func GetButtonsForBooklist[T chatdata.HasBook](
	booklist *[]T,
	prefix string,
	callbackAction CallbackAction,
	page int,
) [][]tgbotapi.InlineKeyboardButton {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if len(*booklist) == 0 {
		return buttons
	}

	for _, item := range *booklist {
		button := tgbotapi.NewInlineKeyboardButtonData(
			prefix+" "+item.GetBook().Name,
			GetCallbackParamStr(callbackAction, item.GetBook().UUID, strconv.Itoa(page)),
		)

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	return buttons
}

func GetBooklistString[T chatdata.HasBook](booklist *[]T) string {
	var builder strings.Builder

	for i, book := range *booklist {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, book.GetBook().Name))
	}

	return builder.String()
}

func GetBooklistPage[T chatdata.HasBook](booklist *[]T, page int) ([]T, int, bool) {
	totalBooks := len(*booklist)

	maxPage := (totalBooks+BooksPerPage-1)/BooksPerPage - 1

	if page > maxPage {
		page = maxPage
	}

	start := page * BooksPerPage
	end := start + BooksPerPage
	if end > totalBooks {
		end = totalBooks
	}

	isLastPage := end >= totalBooks

	return (*booklist)[start:end], page, isLastPage
}

func (lnb *LitNightBot) displayPage(
	chatId int64, messageID int, messageText string,
	buttons [][]tgbotapi.InlineKeyboardButton, logger *logrus.Entry,
) {
	if messageID == -1 {
		lnb.sendMessage(chatId, SendMessageParams{Text: messageText, Buttons: buttons})
	} else {
		lnb.editMessage(chatId, messageID, messageText, buttons)
	}

	logger.WithFields(logrus.Fields{"message": messageText}).Info("Displayed page")
}
