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

func (lnb *LitNightBot) handleCallbackQuery(update *tgbotapi.Update) {
	cbAction, cbParams, err := GetCallbackParam(update.CallbackQuery.Data)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	message := update.CallbackQuery.Message
	chatId := message.Chat.ID
	messageId := message.MessageID

	if message.Text == menuText {
		lnb.removeMessage(chatId, messageId)
	}

	switch cbAction {
	case CBCurrentShow:
		lnb.handleCurrent(message)
	case CBCurrentRandom:
		lnb.handleCurrentRandom(message)
	case CBCurrentComplete:
		lnb.handleCurrentComplete(message)
	case CBCurrentChangeDeadlineRequest:
		lnb.handleCurrentDeadlineRequest(message)
	case CBCurrentToHistory:
		lnb.moveCurrentBook(chatId, messageId, true)
	case CBCurrentToWishlist:
		lnb.moveCurrentBook(chatId, messageId, false)
	case CBCurrentAbort:
		lnb.handleCurrentAbort(message)

	case CBWishlistAddBookRequest:
		lnb.handleWishlistAddRequest(message)
	case CBWishlistShow:
		lnb.handleShowWishlist(message)
	case CBWishlistClean:
		lnb.handleWishlistClean(message)
	case CBWishlistChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		lnb.showCleanWishlistPage(chatId, messageId, page)
	case CBWishlistRemoveBook:
		lnb.handleWishlistRemoveBook(message, update.CallbackQuery.ID, cbParams)

	case CBHistoryShow:
		lnb.handleHistoryShow(message)
	case CBHistoryClean:
		lnb.handleCleanHistory(message)
	case CBHistoryChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		lnb.showCleanHistoryPage(chatId, messageId, page)
	case CBHistoryRemoveBook:
		lnb.handleHistoryRemoveBook(message, update.CallbackQuery.ID, cbParams)

	case CBMenuClose, CBCancel:
		lnb.removeMessage(chatId, messageId)

	default:
		log.Printf("Неизвестный callback: %s. Пожалуйста, позаботьтесь об этом, чтобы мы могли помочь вам выбрать следующую книгу! 📚😅", string(cbAction))
	}
}
