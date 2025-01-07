package tasks

import (
	"lit-night-bot/bot"
	"lit-night-bot/io"

	"github.com/sirupsen/logrus"
)

type Task struct {
	CB   func(logger *logrus.Entry, iocd *io.IoChatData, lnb *bot.LitNightBot)
	Spec string
}
