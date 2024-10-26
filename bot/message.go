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
		vb.sendMessage(chatId, ProgressJokes[i])

		sleepDuration := time.Duration(rand.Intn(1000)+1000) * time.Millisecond
		time.Sleep(sleepDuration)
	}
}

func (vb *LitNightBot) editMessage(chatId int64, msgID int, text string, buttons [][]tgbotapi.InlineKeyboardButton) error {
	editMsg := tgbotapi.NewEditMessageText(chatId, msgID, text)
	var markup tgbotapi.InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		editMsg.ReplyMarkup = &markup
	} else {
		editMsg.ReplyMarkup = nil
	}
	_, err := vb.bot.Send(editMsg)

	return err
}

func (vb *LitNightBot) sendMessage(chatId int64, text string) {
	vb.bot.Send(tgbotapi.NewMessage(chatId, text))
}
