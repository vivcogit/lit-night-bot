package bot

import (
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
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

func (lnb *LitNightBot) handleCallbackQuery(update *tgbotapi.Update, logger *logrus.Entry) {
	logger = logger.WithField("callback_data", update.CallbackQuery.Data)
	logger.Info("Handling callback query")

	cbAction, cbParams, err := GetCallbackParam(update.CallbackQuery.Data)

	if err != nil {
		logger.Error("Error parsing callback parameters: ", err)
		return
	}

	message := update.CallbackQuery.Message
	chatId := message.Chat.ID
	messageId := message.MessageID

	if message.Text == menuText {
		lnb.removeMessage(chatId, messageId)
		logger.Infof("Removed menu message with ID %d", messageId)
	}

	switch cbAction {
	case CBCurrentShow:
		lnb.handleCurrent(update, logger)
	case CBCurrentRandom:
		lnb.handleCurrentRandom(update, logger)
	case CBCurrentComplete:
		lnb.handleCurrentComplete(update, logger)
	case CBCurrentChangeDeadlineRequest:
		lnb.handleCurrentDeadlineRequest(update, logger)
	case CBCurrentToHistory:
		lnb.moveCurrentBook(chatId, messageId, true, logger)
	case CBCurrentToWishlist:
		lnb.moveCurrentBook(chatId, messageId, false, logger)
	case CBCurrentAbort:
		lnb.handleCurrentAbort(update, logger)

	case CBWishlistAddBookRequest:
		lnb.handleWishlistAddRequest(message, logger)
	case CBWishlistShow:
		lnb.handleShowWishlist(message, logger)
	case CBWishlistClean:
		lnb.handleWishlistClean(message, logger)
	case CBWishlistChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		lnb.showCleanWishlistPage(chatId, messageId, page, logger)
	case CBWishlistRemoveBook:
		lnb.handleWishlistRemoveBook(message, update.CallbackQuery.ID, cbParams, logger)

	case CBHistoryShow:
		lnb.handleHistoryShow(message, logger)
	case CBHistoryClean:
		logger.Info("Cleaning history")
		lnb.handleCleanHistory(message, logger)
	case CBHistoryChangePage:
		page, _ := strconv.Atoi(cbParams[0])
		logger.Infof("Changing history page to %d", page)
		lnb.showCleanHistoryPage(chatId, messageId, page, logger)
	case CBHistoryRemoveBook:
		logger.Info("Removing book from history")
		lnb.handleHistoryRemoveBook(message, update.CallbackQuery.ID, cbParams, logger)

	case CBMenuClose, CBCancel:
		logger.Info("Closing menu")
		lnb.removeMessage(chatId, messageId)

	default:
		logger.Warnf("Unknown callback: %s. Please address this to help the user select the next book!", string(cbAction))
	}
}
