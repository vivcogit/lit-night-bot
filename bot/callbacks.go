package bot

import (
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

func GetCallbackParamStr(action CallbackAction, params ...string) string {
	return string(action) + callbackParamsDelimeter + strings.Join(params, callbackParamsDelimeter)
}

func GetCallbackParam(callbackData string) (CallbackAction, []string, error) {
	cb := strings.Split(callbackData, callbackParamsDelimeter)

	return CallbackAction(cb[0]), cb[1:], nil
}

func (vb *LitNightBot) handleCallbackQuery(update *tgbotapi.Update) {
	cbAction, cbParams, err := GetCallbackParam(update.CallbackQuery.Data)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	messageId := update.CallbackQuery.Message.MessageID

	switch cbAction {
	case CBRemove:
		cd := vb.getChatData(chatId)
		_, err := cd.RemoveBookFromWishlist(cbParams[0])
		vb.setChatData(chatId, cd)

		if err != nil {
			vb.sendMessage(chatId, err.Error(), nil)
			return
		}

		callbackConfig := tgbotapi.NewCallback(
			update.CallbackQuery.ID,
			"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
		)
		vb.bot.Send(callbackConfig)

		page, _ := strconv.Atoi(cbParams[1])
		vb.showRemoveWishlistPage(chatId, messageId, page)
		return
	case CBRemovePage:
		page, err := strconv.Atoi(cbParams[0])
		if err != nil {
			vb.sendMessage(chatId, "Ошибка обработки страницы", nil)
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
