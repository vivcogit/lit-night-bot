package bot

import (
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func GetPaginationNavButtons(page int, isLast bool, callbackAction CallbackAction) *[]tgbotapi.InlineKeyboardButton {
	var navButtons []tgbotapi.InlineKeyboardButton
	if page > 0 {
		navButtons = append(
			navButtons,
			tgbotapi.NewInlineKeyboardButtonData(
				"⬅",
				GetCallbackParamStr(callbackAction, strconv.Itoa(page-1)),
			),
		)
	}

	navButtons = append(navButtons,
		tgbotapi.NewInlineKeyboardButtonData(
			"Отмена",
			GetCallbackParamStr(CBCancel),
		),
	)

	if !isLast {
		navButtons = append(
			navButtons,
			tgbotapi.NewInlineKeyboardButtonData(
				"➡",
				GetCallbackParamStr(callbackAction, strconv.Itoa(page+1)),
			),
		)
	}

	return &navButtons
}
