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

	bot, err := bot.NewLitNightBot(token, dataPath, true)

	if err != nil {
		panic(err)
	}

	return bot
}

func main() {
	vb := GetBot()

	vb.InitMenu()
	vb.Start()
}
