package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

func (lnb *LitNightBot) handleReply(update *tgbotapi.Update, logger *logrus.Entry) {
	origMsg := update.Message.ReplyToMessage

	logger = logger.WithFields(logrus.Fields{
		"reply_to_message_id": origMsg.MessageID,
		"reply_to_user_id":    origMsg.From.ID,
		"reply_to_text":       origMsg.Text,
	})

	logger.Info("Handling reply to message")

	if origMsg.Text == setDeadlineRequestMessage {
		logger.Info("Processing deadline request")
		lnb.handleCurrentDeadline(update, logger)
		return
	}

	if origMsg.Text == addBooksToWishlistRequestMessage {
		logger.Info("Processing add books to wishlist request")
		lnb.handleWishlistAdd(update.Message, logger)
		return
	}

	logger.Warning("Received reply with unrecognized message")
}
