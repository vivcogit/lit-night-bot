package tasks

import (
	"lit-night-bot/bot"
	"lit-night-bot/io"
	"time"

	"github.com/sirupsen/logrus"
)

func Reviews(spec string) *Task {
	return &Task{
		CB: func(logger *logrus.Entry, _ *io.IoChatData, lnb *bot.LitNightBot) {
			lnb.ProcessDueReviews(time.Now(), logger)
		},
		Spec: spec,
	}
}
