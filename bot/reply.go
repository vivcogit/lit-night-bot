package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (lnb *LitNightBot) handleReply(update *tgbotapi.Update) {
	origMsg := update.Message.ReplyToMessage

	if origMsg.Text == setDeadlineRequestMessage {
		lnb.handleCurrentDeadline(update.Message)
		return
	}

	if origMsg.Text == addBooksToWishlistRequestMessage {
		lnb.handleWishlistAdd(update.Message)
		return
	}
}
