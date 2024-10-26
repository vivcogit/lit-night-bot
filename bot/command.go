package bot

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type Command string

const (
	CmdStart      Command = "start"
	CmdMenu       Command = "menu"
	CmdHistoryAdd Command = "h_add"
)

func (vb *LitNightBot) handleCommand(update *tgbotapi.Update) {
	cmd := Command(update.Message.Command())
	message := update.Message

	switch cmd {
	case CmdStart:
		vb.handleStart(message)
	case CmdMenu:
		vb.handleMenu(message)
	case CmdHistoryAdd:
		vb.handleHistoryAddBook(message)

	default:
		vb.sendMessage(update.Message.Chat.ID, "Упс, неизвестная команда, попробуем ещё раз?", nil)
	}
}
