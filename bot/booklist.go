package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetButtonsForBooklist[T chatdata.HasBook](
	booklist *[]T,
	prefix string,
	cbParamsGetter func(uuid string) string,
) [][]tgbotapi.InlineKeyboardButton {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if len(*booklist) == 0 {
		return buttons
	}

	for _, item := range *booklist {
		button := tgbotapi.NewInlineKeyboardButtonData(
			prefix+" "+item.GetBook().Name,
			cbParamsGetter(item.GetBook().UUID),
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

func GetCleanBooklistButtons[T chatdata.HasBook](
	booklist *[]T,
	page int,
	callbackAction CallbackAction,
) [][]tgbotapi.InlineKeyboardButton {
	return GetButtonsForBooklist(
		booklist,
		removePrefix,
		func(uuid string) string {
			return GetCallbackParamStr(callbackAction, uuid, strconv.Itoa(page))
		},
	)
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
