package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

func (vb *LitNightBot) handleReply(update *tgbotapi.Update) {
	origMsg := update.Message.ReplyToMessage

	if origMsg.Text == setDeadlineRequestMessage {
		vb.handleCurrentDeadline(update.Message)
		return
	}

	if origMsg.Text == addBooksToWishlistRequestMessage {
		vb.handleWishlistAdd(update.Message)
		return
	}
}
