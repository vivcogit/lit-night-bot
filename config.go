package main

import "os"

type Config struct {
	token    string
	dataPath string
	isDebug  bool
}

func GetConfig() *Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		panic("failed to retrieve the Telegram token from the environment")
	}

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		panic("failed to retrieve path to storage chats data")
	}
	isDebug := os.Getenv("DEBUG") == "1"

	return &Config{
		token:    token,
		dataPath: dataPath,
		isDebug:  isDebug,
	}
}
