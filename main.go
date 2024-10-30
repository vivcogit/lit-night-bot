package main

import (
	bot "lit-night-bot/bot"
	"os"
)

func GetBot() *bot.LitNightBot {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		panic("failed to retrieve the Telegram token from the environment")
	}

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		panic("failed to retrieve path to storage chats data")
	}

	isDebug := os.Getenv("DEBUG") == "1"

	bot, err := bot.NewLitNightBot(token, dataPath, isDebug)

	if err != nil {
		panic(err)
	}

	return bot
}

func main() {
	lnb := GetBot()

	lnb.Start()
}
