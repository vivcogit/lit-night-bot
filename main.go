package main

import (
	bot "lit-night-bot/bot"
	"lit-night-bot/cron"
	io "lit-night-bot/io"
	"lit-night-bot/tasks"

	"github.com/sirupsen/logrus"
)

func getBot(logger *logrus.Entry, iocd *io.IoChatData, token string, isDebug bool) *bot.LitNightBot {
	bot, err := bot.NewLitNightBot(logger, token, iocd, isDebug)

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
	config := GetConfig()

	logger := getLogger(config.isDebug)
	iocd := io.NewIOChatData(logger.WithField("entry", "iocd"), config.dataPath)
	lnb := getBot(logger.WithField("entry", "bot"), iocd, config.token, config.isDebug)

	cronTasks := []tasks.Task{
		*tasks.Remind("0 7 * * *", tasks.OneWeekReminderJokes, 7),
		*tasks.Remind("0 7 * * *", tasks.OneDayReminderJokes, 1),
	}

	lnb.Start()
	cron.StartCron(logger.WithField("entry", "cron"), iocd, lnb, &cronTasks)
}
