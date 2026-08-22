package cron

import (
	"lit-night-bot/bot"
	"lit-night-bot/io"
	"lit-night-bot/tasks"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

func StartCron(logger *logrus.Entry, iocd *io.IoChatData, lnb *bot.LitNightBot, tasks *[]tasks.Task, location *time.Location) {
	c := newCron(location)

	for _, task := range *tasks {
		taskLogger := logger.WithField("spec", task.Spec)
		taskLogger.Info("register task")

		c.AddFunc(task.Spec, func() {
			taskLogger.Info("run task")
			task.CB(taskLogger, iocd, lnb)
		})
	}

	c.Start()

	defer c.Stop()
	select {}
}

func newCron(location *time.Location) *cron.Cron {
	return cron.New(cron.WithLocation(location))
}
