package bot

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/rand"
)

func (lnb *LitNightBot) sendProgressJokes(chatId int64) {
	rand.Seed(uint64((time.Now().UnixNano())))

	numMessages := rand.Intn(3) + 3

	rand.Shuffle(len(ProgressJokes), func(i, j int) {
		ProgressJokes[i], ProgressJokes[j] = ProgressJokes[j], ProgressJokes[i]
	})

	for i := 0; i < numMessages; i++ {
		lnb.sendPlainMessage(chatId, ProgressJokes[i])

		sleepDuration := time.Duration(rand.Intn(1000)+1000) * time.Millisecond
		time.Sleep(sleepDuration)
	}
}

func (lnb *LitNightBot) editMessage(chatId int64, msgID int, text string, buttons [][]tgbotapi.InlineKeyboardButton) (tgbotapi.Message, error) {
	editMsg := tgbotapi.NewEditMessageText(chatId, msgID, text)
	var markup tgbotapi.InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		editMsg.ReplyMarkup = &markup
	} else {
		editMsg.ReplyMarkup = nil
	}

	return lnb.bot.Send(editMsg)
}

func (lnb *LitNightBot) removeMessage(chatId int64, msgId int) (tgbotapi.Message, error) {
	msg := tgbotapi.NewDeleteMessage(chatId, msgId)
	return lnb.bot.Send(msg)
}

type SendMessageParams struct {
	text    string
	buttons [][]tgbotapi.InlineKeyboardButton
	replyTo int
}

func (lnb *LitNightBot) sendPlainMessage(chatId int64, text string) (tgbotapi.Message, error) {
	return lnb.sendMessage(chatId, SendMessageParams{text: text})
}

func (lnb *LitNightBot) sendMessage(chatId int64, params SendMessageParams) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatId, params.text)

	if len(params.buttons) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(params.buttons...)
	}

	if params.replyTo != 0 {
		msg.ReplyToMessageID = params.replyTo
	}

	return lnb.bot.Send(msg)
}
