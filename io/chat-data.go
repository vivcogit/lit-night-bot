package io

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
)

type IoChatData struct {
	dataPath string
	logger   *logrus.Entry
}

func (iocd *IoChatData) GetChatDataFilePath(chatId int64) string {
	filePath := filepath.Join(iocd.dataPath, strconv.FormatInt(chatId, 10))
	iocd.logger.Debugf("Chat data file path: %s", filePath)
	return filePath
}

func (iocd *IoChatData) GetChatData(chatId int64) *chatdata.ChatData {
	var cd chatdata.ChatData
	filePath := iocd.GetChatDataFilePath(chatId)

	if err := utils.ReadJSONFromFile(filePath, &cd); err != nil {
		iocd.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to read chat data from file")
		return nil
	}

	iocd.logger.WithField("chat_id", chatId).Info("Successfully read chat data")
	return &cd
}

func (iocd *IoChatData) SetChatData(chatId int64, cd *chatdata.ChatData) {
	filePath := iocd.GetChatDataFilePath(chatId)

	if err := utils.WriteJSONToFile(filePath, cd); err != nil {
		iocd.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to write chat data to file")
		return
	}

	iocd.logger.WithField("chat_id", chatId).Info("Successfully wrote chat data to file")
}

func (iocd *IoChatData) GetDatasList() ([]string, error) {
	iocd.logger.Debug("Fetching file list from dataPath")

	entries, err := os.ReadDir(iocd.dataPath)
	if err != nil {
		iocd.logger.WithError(err).Error("Failed to read directory")
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	iocd.logger.Infof("Found %d files in directory", len(files))
	return files, nil
}

func NewIOChatData(logger *logrus.Entry, dataPath string) *IoChatData {
	return &IoChatData{
		logger:   logger,
		dataPath: dataPath,
	}
}
