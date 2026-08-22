package main

import (
	bot "lit-night-bot/bot"
	"lit-night-bot/cron"
	io "lit-night-bot/io"
	"lit-night-bot/tasks"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

func getBot(logger *logrus.Entry, iocd *io.IoChatData, token string, isDebug bool, location *time.Location) *bot.LitNightBot {
	bot, err := bot.NewLitNightBot(logger, token, iocd, isDebug, location)

	if err != nil {
		panic(err)
	}

	return bot
}

func getLogger(isDebug bool) *logrus.Entry {
	logger := logrus.New()
	if isDebug {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}

	return logger.WithField("project", "lit-night-bot")
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		os.Exit(runMigrationCommand(os.Args[2:]))
	}
	config := GetConfig()

	logger := getLogger(config.isDebug)
	iocd := io.NewIOChatData(logger.WithField("entry", "iocd"), config.dataPath)
	lnb := getBot(logger.WithField("entry", "bot"), iocd, config.token, config.isDebug, config.location)

	cronTasks := []tasks.Task{
		*tasks.Remind("0 7 * * *", tasks.OneWeekReminderJokes, 7, config.location),
		*tasks.Remind("0 7 * * *", tasks.OneDayReminderJokes, 1, config.location),
	}

	lnb.Start()
	cron.StartCron(logger.WithField("entry", "cron"), iocd, lnb, &cronTasks, config.location)
}
