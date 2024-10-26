package bot

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/rand"
)

func (vb *LitNightBot) sendProgressJokes(chatId int64) {
	rand.Seed(uint64((time.Now().UnixNano())))

	numMessages := rand.Intn(3) + 3

	rand.Shuffle(len(ProgressJokes), func(i, j int) {
		ProgressJokes[i], ProgressJokes[j] = ProgressJokes[j], ProgressJokes[i]
	})

	for i := 0; i < numMessages; i++ {
		vb.sendMessage(chatId, ProgressJokes[i], nil)

		sleepDuration := time.Duration(rand.Intn(1000)+1000) * time.Millisecond
		time.Sleep(sleepDuration)
	}
}

func (vb *LitNightBot) editMessage(chatId int64, msgID int, text string, buttons [][]tgbotapi.InlineKeyboardButton) (tgbotapi.Message, error) {
	editMsg := tgbotapi.NewEditMessageText(chatId, msgID, text)
	var markup tgbotapi.InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		editMsg.ReplyMarkup = &markup
	} else {
		editMsg.ReplyMarkup = nil
	}

	return vb.bot.Send(editMsg)
}

func (vb *LitNightBot) removeMessage(chatId int64, msgId int) (tgbotapi.Message, error) {
	msg := tgbotapi.NewDeleteMessage(chatId, msgId)
	return vb.bot.Send(msg)
}

func (vb *LitNightBot) sendMessage(chatId int64, text string, buttons [][]tgbotapi.InlineKeyboardButton) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatId, text)
	if len(buttons) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	}

	return vb.bot.Send(msg)
}
