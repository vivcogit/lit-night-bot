package bot

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type CallbackAction string

const (
	CBRemove            CallbackAction = "remove"
	CBRemovePage        CallbackAction = "remove_page"
	CBCancel            CallbackAction = "cancel"
	CBCurrentToWishlist CallbackAction = "cur2wish"
	CBCurrentToHistory  CallbackAction = "cur2his"
)

func GetCallbackParamStr(action CallbackAction, data string) string {
	return string(action) + ":" + data
}

func GetCallbackParam(callbackData string) (CallbackAction, string, error) {
	cb := strings.Split(callbackData, ":")

	if len(cb) != 2 {
		return "", "", errors.New("Неизвестная операция: " + callbackData)
	}

	ca := CallbackAction(cb[0])
	switch ca {
	case CBRemove, CBCancel, CBCurrentToWishlist, CBCurrentToHistory, CBRemovePage:
		return ca, cb[1], nil
	}

	return "", "", errors.New("Неизвестное действие: " + callbackData)
}

func (vb *LitNightBot) handleCallbackQuery(update *tgbotapi.Update) {
	cbAction, cbParam, err := GetCallbackParam(update.CallbackQuery.Data)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	messageId := update.CallbackQuery.Message.MessageID

	switch cbAction {
	case CBRemove:
		cd := vb.getChatData(chatId)
		_, err := cd.RemoveBookFromWishlist(cbParam)
		vb.setChatData(chatId, cd)

		if err != nil {
			vb.sendMessage(chatId, err.Error())
			return
		}

		callbackConfig := tgbotapi.NewCallback(
			update.CallbackQuery.ID,
			"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
		)
		vb.bot.Send(callbackConfig)

		text, buttons := getCleanWishlistMessage(cd)
		vb.editMessage(chatId, messageId, text, buttons)
		return
	case CBRemovePage:
		page, err := strconv.Atoi(cbParam)
		if err != nil {
			vb.sendMessage(chatId, "Ошибка обработки страницы")
			return
		}
		vb.showRemoveWishlistPage(chatId, messageId, page)

	case CBCancel:
		vb.editMessage(chatId, update.CallbackQuery.Message.MessageID, "🤭 Упс! Вы отменили действие! Не переживайте, в следующий раз все получится! 😉", nil)

	case CBCurrentToHistory:
		vb.moveCurrentBook(chatId, update.CallbackQuery.Message.MessageID, true)

	case CBCurrentToWishlist:
		vb.moveCurrentBook(chatId, update.CallbackQuery.Message.MessageID, false)

	default:
		log.Printf("Неизвестный callback: %s. Пожалуйста, позаботьтесь об этом, чтобы мы могли помочь вам выбрать следующую книгу! 📚😅", string(cbAction))
	}
}
