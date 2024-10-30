package bot

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type Command string

const (
	CmdStart      Command = "start"
	CmdMenu       Command = "menu"
	CmdHistoryAdd Command = "h_add"
)

func (lnb *LitNightBot) handleCommand(update *tgbotapi.Update, logger *logrus.Entry) {
	message := update.Message
	command := message.Command()

	logger = logger.WithField("command", command)
	logger.Info("Received command")

	switch command {
	case string(CmdStart):
		lnb.handleStart(update, logger)
	case string(CmdMenu):
		lnb.handleMenu(update, logger)
	case string(CmdHistoryAdd):
		lnb.handleHistoryAddBook(update, logger)
	default:
		logger.Warn("Unknown command")
		lnb.sendPlainMessage(message.Chat.ID, "Команда не распознана.")
	}
}
