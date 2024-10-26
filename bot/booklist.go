package bot

import (
	chatdata "lit-night-bot/chat-data"

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

	button := tgbotapi.NewInlineKeyboardButtonData(
		"Отмена",
		GetCallbackParamStr(CBCancel, "_"),
	)

	inlineRow := tgbotapi.NewInlineKeyboardRow(button)

	return append(buttons, inlineRow)
}
