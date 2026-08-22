package main

import (
	"fmt"
	"os"
	"time"
	_ "time/tzdata"
)

const defaultTimezone = "Europe/Moscow"

type Config struct {
	token    string
	dataPath string
	isDebug  bool
	location *time.Location
}

func GetConfig() *Config {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		panic("failed to retrieve the Telegram token from the environment")
	}

	dataPath := GetDataPath()
	isDebug := os.Getenv("DEBUG") == "1"
	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		timezone = defaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		panic(fmt.Sprintf("failed to load timezone %q: %v", timezone, err))
	}

	return &Config{
		token:    token,
		dataPath: dataPath,
		isDebug:  isDebug,
		location: location,
	}
}

func GetDataPath() string {
	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		panic("failed to retrieve path to storage chats data")
	}
	return dataPath
}
