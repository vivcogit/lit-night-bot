package bot

import (
	"errors"
	chatdata "lit-night-bot/chat-data"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type CallbackAction string

const (
	CBMenuClose           CallbackAction = "m_close"
	CBBookShow            CallbackAction = "book_show"
	CBBookShowReplacing   CallbackAction = "book_replace"
	CBBookEditTitle       CallbackAction = "book_edit_title"
	CBBookEditAuthors     CallbackAction = "book_edit_authors"
	CBBookApprove         CallbackAction = "book_approve"
	CBBookSwap            CallbackAction = "book_swap"
	CBBooksReview         CallbackAction = "books_review"
	CBHistoryYear         CallbackAction = "h_year"
	CBHistoryPick         CallbackAction = "h_pick"
	CBRatingOpen          CallbackAction = "rating_open"
	CBRatingSet           CallbackAction = "rating_set"
	CBRatingList          CallbackAction = "rating_list"
	CBRatingBackToBook    CallbackAction = "rating_book"
	CBRatingDeleteRequest CallbackAction = "rating_del"
	CBRatingDeleteConfirm CallbackAction = "rating_del_ok"
	CBRatingDeleteCancel  CallbackAction = "rating_del_no"
	CBRatingCloseRequest  CallbackAction = "rating_finish"
	CBRatingCloseConfirm  CallbackAction = "rating_finish_ok"
	CBRatingCloseCancel   CallbackAction = "rating_finish_no"
	CBRatingReopen        CallbackAction = "rating_reopen"
	CBReviewWrite         CallbackAction = "review_write"
	CBReviewRemind        CallbackAction = "review_remind"
	CBReviewSkip          CallbackAction = "review_skip"
	CBReviewDelete        CallbackAction = "review_delete"
	CBReviewList          CallbackAction = "review_list"
	CBReviewBackToBook    CallbackAction = "review_book"

	CBCurrentShow                  CallbackAction = "c_show"
	CBCurrentChangeDeadlineRequest CallbackAction = "c_deadline"
	CBCurrentRandom                CallbackAction = "c_random"
	CBCurrentComplete              CallbackAction = "c_complete"
	CBCurrentMarkCompleted         CallbackAction = "c_mark_done"
	CBCurrentMarkUnfinished        CallbackAction = "c_mark_unfinished"
	CBCurrentAbort                 CallbackAction = "c_abort"
	CBCurrentChooseBook            CallbackAction = "c_manual"

	CBWishlistAddBookRequest CallbackAction = "wl_add_req"
	CBWishlistShow           CallbackAction = "wl_show"
	CBWishlistClean          CallbackAction = "wl_clean"
	CBWishlistChangePage     CallbackAction = "wl_clean_page"
	CBWishlistChoose         CallbackAction = "wl_choose"
	CBWishlistChoosePage     CallbackAction = "wl_choose_page"
	CBWishlistRemoveBook     CallbackAction = "wl_rm_book"

	CBHistoryShow          CallbackAction = "h_show"
	CBHistoryClean         CallbackAction = "h_clean"
	CBHistoryChangePage    CallbackAction = "h_clean_page"
	CBHistoryRemoveBook    CallbackAction = "h_rm_book"
	CBHistoryRemoveConfirm CallbackAction = "h_rm_confirm"
	CBHistoryRemoveCancel  CallbackAction = "h_rm_cancel"
	CBStatisticsShow       CallbackAction = "stats_show"

	CBCancel            CallbackAction = "cancel"
	CBCurrentToWishlist CallbackAction = "cur2wish"
	CBCurrentToHistory  CallbackAction = "cur2his"
)

func GetCallbackParamStr(action CallbackAction, params ...string) string {
	return string(action) + callbackParamsDelimeter + strings.Join(params, callbackParamsDelimeter)
}

func GetCallbackParam(callbackData string) (CallbackAction, []string, error) {
	if strings.TrimSpace(callbackData) == "" {
		return "", nil, errors.New("empty callback data")
	}
	cb := strings.Split(callbackData, callbackParamsDelimeter)

	return CallbackAction(cb[0]), cb[1:], nil
}

func shouldRemoveMenuMessage(messageText string, action CallbackAction) bool {
	return messageText == menuText && action != CBBooksReview
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
	if message == nil {
		logger.Warn("Callback query has no message")
		return
	}
	chatId := message.Chat.ID
	messageId := message.MessageID

	if shouldRemoveMenuMessage(message.Text, cbAction) {
		lnb.removeMessage(chatId, messageId)
		logger.Infof("Removed menu message with ID %d", messageId)
	}

	switch cbAction {
	case CBBookShow:
		if len(cbParams) < 1 {
			logger.Warn("Book callback has no ID")
			return
		}
		lnb.handleBookCard(message, cbParams[0], logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBBookShowReplacing:
		if len(cbParams) < 1 {
			logger.Warn("Replacing book callback has no ID")
			return
		}
		lnb.replaceMessageWithBookCard(message, cbParams[0], logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBBookEditTitle, CBBookEditAuthors:
		if len(cbParams) < 1 {
			return
		}
		lnb.requestBookField(message, cbParams[0], cbAction, update.CallbackQuery.From, logger)
	case CBBookApprove:
		if len(cbParams) < 1 {
			return
		}
		lnb.approveBookCard(message, cbParams[0], logger)
	case CBBookSwap:
		if len(cbParams) < 1 {
			return
		}
		lnb.swapBookTitleAndAuthor(message, cbParams[0], logger)
	case CBBooksReview:
		page := 0
		if len(cbParams) > 0 {
			page, _ = strconv.Atoi(cbParams[0])
		}
		lnb.showBooksForReview(chatId, messageId, page, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBHistoryYear:
		if len(cbParams) < 1 {
			logger.Warn("History year callback has no year")
			return
		}
		year, err := strconv.Atoi(cbParams[0])
		if err != nil {
			logger.WithError(err).Warn("Invalid history year")
			return
		}
		lnb.showHistoryYear(chatId, messageId, year, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBHistoryPick:
		if len(cbParams) < 1 || update.CallbackQuery.From == nil {
			return
		}
		year, err := strconv.Atoi(cbParams[0])
		if err != nil {
			logger.WithError(err).Warn("Invalid history selection year")
			return
		}
		lnb.requestHistoryBookNumber(message, year, update.CallbackQuery.From, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBRatingOpen:
		if len(cbParams) < 1 {
			return
		}
		lnb.showRatingPanel(message, cbParams[0], logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBRatingSet:
		if len(cbParams) < 2 {
			return
		}
		value, err := strconv.Atoi(cbParams[1])
		if err != nil {
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Некорректная оценка"))
			return
		}
		lnb.setBookRating(update, cbParams[0], value, logger)
	case CBRatingList:
		if len(cbParams) < 1 {
			return
		}
		page := 0
		if len(cbParams) > 1 {
			page, _ = strconv.Atoi(cbParams[1])
		}
		lnb.showRatingsList(message, cbParams[0], page)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBRatingBackToBook:
		if len(cbParams) < 1 {
			return
		}
		lnb.showBookCardInPlace(message, cbParams[0])
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBRatingDeleteRequest:
		if len(cbParams) < 1 {
			return
		}
		lnb.requestRatingDelete(update, cbParams[0])
	case CBRatingDeleteConfirm:
		lnb.confirmRatingDelete(update, cbParams, logger)
	case CBRatingDeleteCancel:
		lnb.cancelRatingDelete(update, cbParams)
	case CBRatingCloseRequest:
		if len(cbParams) < 1 {
			return
		}
		lnb.requestRatingClose(update, cbParams[0])
	case CBRatingCloseConfirm:
		lnb.confirmRatingClose(update, cbParams, logger)
	case CBRatingCloseCancel:
		lnb.cancelRatingClose(update, cbParams)
	case CBRatingReopen:
		if len(cbParams) < 1 {
			return
		}
		lnb.reopenRatings(update, cbParams[0], logger)
	case CBReviewWrite:
		if len(cbParams) < 1 {
			return
		}
		if !lnb.reviewCallbackUserAllowed(update, cbParams) {
			return
		}
		lnb.requestReview(update, cbParams[0], logger)
	case CBReviewRemind:
		if len(cbParams) < 1 {
			return
		}
		if !lnb.reviewCallbackUserAllowed(update, cbParams) {
			return
		}
		lnb.scheduleReviewReminder(update, cbParams[0], logger)
	case CBReviewSkip:
		if len(cbParams) < 1 {
			return
		}
		if !lnb.reviewCallbackUserAllowed(update, cbParams) {
			return
		}
		lnb.skipReview(update, cbParams[0], logger)
	case CBReviewDelete:
		if len(cbParams) < 1 {
			return
		}
		if !lnb.reviewCallbackUserAllowed(update, cbParams) {
			return
		}
		lnb.deleteReview(update, cbParams[0], logger)
	case CBReviewList:
		if len(cbParams) < 1 {
			return
		}
		page := 0
		if len(cbParams) > 1 {
			page, _ = strconv.Atoi(cbParams[1])
		}
		if err := lnb.showReviews(message, cbParams[0], page); err != nil {
			logger.WithError(err).Warn("Failed to show reviews")
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Не удалось открыть отзывы"))
		} else {
			lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
		}
	case CBReviewBackToBook:
		if len(cbParams) < 1 {
			return
		}
		lnb.showBookCardInPlace(message, cbParams[0])
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBCurrentShow:
		lnb.handleCurrent(update, logger)
	case CBCurrentRandom:
		lnb.handleCurrentRandom(update, logger)
	case CBCurrentComplete:
		lnb.handleCurrentComplete(update, logger)
	case CBCurrentMarkCompleted, CBCurrentMarkUnfinished:
		if len(cbParams) < 1 {
			return
		}
		status := chatdata.StatusCompleted
		if cbAction == CBCurrentMarkUnfinished {
			status = chatdata.StatusUnfinished
		}
		lnb.finishCurrentBook(chatId, messageId, cbParams[0], status, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
	case CBCurrentChangeDeadlineRequest:
		lnb.handleCurrentDeadlineRequest(update, logger)
	case CBCurrentToHistory:
		if len(cbParams) < 1 {
			return
		}
		lnb.moveCurrentBook(chatId, messageId, cbParams[0], true, logger)
	case CBCurrentToWishlist:
		if len(cbParams) < 1 {
			return
		}
		lnb.moveCurrentBook(chatId, messageId, cbParams[0], false, logger)
	case CBCurrentAbort:
		lnb.handleCurrentAbort(update, logger)
	case CBCurrentChooseBook:
		if len(cbParams) < 1 {
			return
		}
		lnb.handleCurrentChoose(update, cbParams[0], logger)

	case CBWishlistChoose:
		lnb.handleWishlistChooseFrom(message, logger)
	case CBWishlistChoosePage:
		if len(cbParams) < 1 {
			return
		}
		page, _ := strconv.Atoi(cbParams[0])
		lnb.showChooseFromWishlistPage(chatId, messageId, page, logger)
	case CBWishlistAddBookRequest:
		lnb.handleWishlistAddRequest(message, update.CallbackQuery.From, logger)
	case CBWishlistShow:
		lnb.handleShowWishlist(message, logger)
	case CBWishlistClean:
		lnb.handleWishlistClean(message, logger)
	case CBWishlistChangePage:
		if len(cbParams) < 1 {
			return
		}
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
		if len(cbParams) < 1 {
			return
		}
		page, _ := strconv.Atoi(cbParams[0])
		logger.Infof("Changing history page to %d", page)
		lnb.showCleanHistoryPage(chatId, messageId, page, logger)
	case CBHistoryRemoveBook:
		logger.Info("Removing book from history")
		lnb.handleHistoryRemoveBook(message, update.CallbackQuery.ID, cbParams, logger)
	case CBHistoryRemoveConfirm:
		lnb.confirmHistoryRemoveBook(message, update.CallbackQuery.ID, cbParams, logger)
	case CBHistoryRemoveCancel:
		page := 0
		if len(cbParams) > 0 {
			page, _ = strconv.Atoi(cbParams[0])
		}
		lnb.showCleanHistoryPage(chatId, messageId, page, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, "Отменено"))
	case CBStatisticsShow:
		lnb.handleStatistics(message, logger)
		lnb.bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))

	case CBMenuClose, CBCancel:
		logger.Info("Closing menu")
		lnb.removeMessage(chatId, messageId)

	default:
		logger.Warnf("Unknown callback: %s. Please address this to help the user select the next book!", string(cbAction))
	}
}
