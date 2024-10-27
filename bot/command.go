package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Command string

const (
	CmdStart      Command = "start"
	CmdMenu       Command = "menu"
	CmdHistoryAdd Command = "h_add"
)

func (lnb *LitNightBot) handleCommand(update *tgbotapi.Update) {
	cmd := Command(update.Message.Command())
	message := update.Message

	switch cmd {
	case CmdStart:
		lnb.handleStart(message)
	case CmdMenu:
		lnb.handleMenu(message)
	case CmdHistoryAdd:
		lnb.handleHistoryAddBook(message)

	default:
		lnb.sendPlainMessage(update.Message.Chat.ID, "Упс, неизвестная команда, попробуем ещё раз?")
	}
}
