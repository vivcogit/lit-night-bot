package io

import (
	"errors"
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type IoChatData struct {
	dataPath string
	logger   *logrus.Entry
	mu       sync.RWMutex
}

func (iocd *IoChatData) GetChatDataFilePath(chatId int64) string {
	filePath := filepath.Join(iocd.dataPath, strconv.FormatInt(chatId, 10))
	iocd.logger.Debugf("Chat data file path: %s", filePath)
	return filePath
}

func (iocd *IoChatData) GetChatData(chatId int64) *chatdata.ChatData {
	data, err := iocd.LoadChatData(chatId)
	if err != nil {
		iocd.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to read chat data from file")
		return nil
	}
	return data
}

func (iocd *IoChatData) LoadChatData(chatId int64) (*chatdata.ChatData, error) {
	iocd.mu.RLock()
	defer iocd.mu.RUnlock()

	var cd chatdata.ChatData
	filePath := iocd.GetChatDataFilePath(chatId)

	if err := utils.ReadJSONFromFile(filePath, &cd); err != nil {
		return nil, err
	}

	iocd.logger.WithField("chat_id", chatId).Info("Successfully read chat data")
	return &cd, nil
}

func (iocd *IoChatData) SaveChatData(chatId int64, cd *chatdata.ChatData) error {
	iocd.mu.Lock()
	defer iocd.mu.Unlock()

	filePath := iocd.GetChatDataFilePath(chatId)

	if err := utils.WriteJSONToFile(filePath, cd); err != nil {
		return err
	}

	iocd.logger.WithField("chat_id", chatId).Info("Successfully wrote chat data to file")
	return nil
}

func (iocd *IoChatData) GetOrCreateChatData(chatId int64) *chatdata.ChatData {
	data, err := iocd.LoadChatData(chatId)
	if err == nil {
		return data
	}
	if !errors.Is(err, os.ErrNotExist) {
		iocd.logger.WithField("chat_id", chatId).WithError(err).Error("Refusing to replace unreadable chat data")
		return nil
	}
	data = chatdata.NewChatData()
	if err := iocd.SaveChatData(chatId, data); err != nil {
		iocd.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to create chat data")
		return nil
	}
	return data
}

func (iocd *IoChatData) BackupChatData(chatId int64) (string, error) {
	iocd.mu.Lock()
	defer iocd.mu.Unlock()

	sourcePath := iocd.GetChatDataFilePath(chatId)
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать исходные данные: %w", err)
	}

	backupDir := filepath.Join(iocd.dataPath, "_migration", "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", fmt.Errorf("не удалось создать каталог копий: %w", err)
	}
	if err := utils.SyncDirectory(iocd.dataPath); err != nil {
		return "", fmt.Errorf("не удалось синхронизировать каталог данных: %w", err)
	}
	if err := utils.SyncDirectory(filepath.Dir(backupDir)); err != nil {
		return "", fmt.Errorf("не удалось синхронизировать каталог копий: %w", err)
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("%d-%s.json", chatId, time.Now().Format("20060102-150405.000000000")))
	backup, err := os.OpenFile(backupPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("не удалось создать копию: %w", err)
	}
	complete := false
	defer func() {
		_ = backup.Close()
		if !complete {
			_ = os.Remove(backupPath)
		}
	}()
	if _, err := backup.Write(raw); err != nil {
		return "", fmt.Errorf("не удалось записать копию: %w", err)
	}
	if err := backup.Sync(); err != nil {
		return "", fmt.Errorf("не удалось синхронизировать копию: %w", err)
	}
	if err := backup.Close(); err != nil {
		return "", fmt.Errorf("не удалось закрыть копию: %w", err)
	}
	if err := utils.SyncDirectory(backupDir); err != nil {
		return "", fmt.Errorf("не удалось синхронизировать каталог копий: %w", err)
	}
	complete = true
	return backupPath, nil
}

func (iocd *IoChatData) RestoreChatData(chatId int64, backupPath string) error {
	var data chatdata.ChatData
	if err := utils.ReadJSONFromFile(backupPath, &data); err != nil {
		return err
	}
	return iocd.SaveChatData(chatId, &data)
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
	if err := os.MkdirAll(dataPath, 0o755); err != nil {
		logger.WithError(err).Fatal("Failed to create data directory")
	}
	return &IoChatData{
		logger:   logger,
		dataPath: dataPath,
	}
}
