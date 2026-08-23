package bot

import (
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleReply(update *tgbotapi.Update, logger *logrus.Entry) {
	origMsg := update.Message.ReplyToMessage

	logger = logger.WithFields(logrus.Fields{
		"reply_to_message_id": origMsg.MessageID,
		"reply_to_text":       origMsg.Text,
	})
	if origMsg.From != nil {
		logger = logger.WithField("reply_to_user_id", origMsg.From.ID)
	}

	logger.Info("Handling reply to message")

	if lnb.handleHistorySelectionReply(update.Message, origMsg, logger) {
		return
	}

	if lnb.handleReviewReply(update.Message, origMsg.Text, logger) {
		return
	}

	if lnb.handleBookFieldReply(update.Message, origMsg.Text, logger) {
		return
	}

	if origMsg.Text == setDeadlineRequestMessage {
		logger.Info("Processing deadline request")
		lnb.handleCurrentDeadline(update, logger)
		return
	}

	if strings.HasSuffix(origMsg.Text, addBooksToWishlistRequestMessage) {
		logger.Info("Processing add books to wishlist request")
		lnb.handleWishlistAdd(update.Message, logger)
		return
	}

	logger.Warning("Received reply with unrecognized message")
}
