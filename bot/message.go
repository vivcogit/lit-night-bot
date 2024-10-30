package bot

import (
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

func (lnb *LitNightBot) sendProgressJokes(chatId int64) {
	numMessages := rand.Intn(3) + 3
	rand.Seed(uint64(time.Now().UnixNano()))

	rand.Shuffle(len(ProgressJokes), func(i, j int) {
		ProgressJokes[i], ProgressJokes[j] = ProgressJokes[j], ProgressJokes[i]
	})

	for i := 0; i < numMessages; i++ {
		_, err := lnb.sendPlainMessage(chatId, ProgressJokes[i])
		if err != nil {
			lnb.logger.WithError(err).WithField("chat_id", chatId).Error("Failed to send progress joke")
		}

		sleepDuration := time.Duration(rand.Intn(1000)+600) * time.Millisecond
		time.Sleep(sleepDuration)
	}
}

func (lnb *LitNightBot) editMessage(chatId int64, msgID int, text string, buttons [][]tgbotapi.InlineKeyboardButton) (tgbotapi.Message, error) {
	editMsg := tgbotapi.NewEditMessageText(chatId, msgID, text)
	if len(buttons) > 0 {
		replyMarkup := tgbotapi.NewInlineKeyboardMarkup(buttons...)
		editMsg.ReplyMarkup = &replyMarkup
	} else {
		editMsg.ReplyMarkup = nil
	}

	lnb.logger.WithFields(logrus.Fields{
		"chat_id": chatId,
		"msg_id":  msgID,
		"text":    text,
		"buttons": buttons,
	}).Info("Editing message")

	return lnb.bot.Send(editMsg)
}

func (lnb *LitNightBot) removeMessage(chatId int64, msgId int) error {
	lnb.logger.WithFields(logrus.Fields{
		"chat_id": chatId,
		"msg_id":  msgId,
	}).Info("Removing message")

	_, err := lnb.bot.Send(tgbotapi.NewDeleteMessage(chatId, msgId))
	if err != nil {
		lnb.logger.WithError(err).WithField("msg_id", msgId).Error("Failed to delete message")
	}
	return err
}

type SendMessageParams struct {
	Text    string
	Buttons [][]tgbotapi.InlineKeyboardButton
	ReplyTo int
}

func (lnb *LitNightBot) sendMessage(chatId int64, params SendMessageParams) (tgbotapi.Message, error) {
	msg := tgbotapi.NewMessage(chatId, params.Text)
	if len(params.Buttons) > 0 {
		msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(params.Buttons...)
	}
	if params.ReplyTo != 0 {
		msg.ReplyToMessageID = params.ReplyTo
	}

	lnb.logger.WithFields(logrus.Fields{
		"chat_id":  chatId,
		"text":     params.Text,
		"buttons":  params.Buttons,
		"reply_to": params.ReplyTo,
	}).Info("Sending message")

	return lnb.bot.Send(msg)
}

func (lnb *LitNightBot) sendPlainMessage(chatId int64, text string) (tgbotapi.Message, error) {
	return lnb.sendMessage(chatId, SendMessageParams{Text: text})
}
