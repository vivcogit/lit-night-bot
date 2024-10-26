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
	CBMenuClose CallbackAction = "m_close"

	CBCurrentShow                  CallbackAction = "c_show"
	CBCurrentChangeDeadlineRequest CallbackAction = "c_deadline"
	CBCurrentRandom                CallbackAction = "c_random"
	CBCurrentComplete              CallbackAction = "c_complete"
	CBCurrentAbort                 CallbackAction = "c_abort"

	CBWishlistAddBookRequest CallbackAction = "wl_add_req"
	CBWishlistShow           CallbackAction = "wl_show"
	CBWishlistClean          CallbackAction = "wl_clean"
	CBWishlistChangePage     CallbackAction = "wl_clean_page"
	CBWishlistRemoveBook     CallbackAction = "wl_rm_book"

	CBHistoryShow       CallbackAction = "h_show"
	CBHistoryClean      CallbackAction = "h_clean"
	CBHistoryChangePage CallbackAction = "h_clean_page"
	CBHistoryRemoveBook CallbackAction = "h_rm_book"

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

	message := update.CallbackQuery.Message
	chatId := message.Chat.ID
	messageId := message.MessageID

	if message.Text == menuText {
		vb.removeMessage(chatId, messageId)
	}

	switch cbAction {
	case CBCurrentShow:
		vb.handleCurrent(message)
	case CBCurrentRandom:
		vb.handleCurrentRandom(message)
	case CBCurrentComplete:
		vb.handleCurrentComplete(message)
	case CBCurrentChangeDeadlineRequest:
		vb.handleCurrentDeadlineRequest(message)
	case CBCurrentToHistory:
		vb.moveCurrentBook(chatId, messageId, true)
	case CBCurrentToWishlist:
		vb.moveCurrentBook(chatId, messageId, false)
	case CBCurrentAbort:
		vb.handleCurrentAbort(message)

	case CBWishlistAddBookRequest:
		vb.handleWishlistAddRequest(message)
	case CBWishlistShow:
		vb.handleShowWishlist(message)
	case CBWishlistClean:
		vb.handleWishlistClean(message)
	case CBWishlistChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		vb.showCleanWishlistPage(chatId, messageId, page)
	case CBWishlistRemoveBook:
		vb.handleWishlistRemoveBook(message, update.CallbackQuery.ID, cbParams)

	case CBHistoryShow:
		vb.handleHistoryShow(message)
	case CBHistoryClean:
		vb.handleCleanHistory(message)
	case CBHistoryChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		vb.showCleanWishlistPage(chatId, messageId, page)
	case CBHistoryRemoveBook:
		vb.handleHistoryRemoveBook(message, update.CallbackQuery.ID, cbParams)

	case CBMenuClose, CBCancel:
		vb.removeMessage(chatId, messageId)

	default:
		log.Printf("Неизвестный callback: %s. Пожалуйста, позаботьтесь об этом, чтобы мы могли помочь вам выбрать следующую книгу! 📚😅", string(cbAction))
	}
}
