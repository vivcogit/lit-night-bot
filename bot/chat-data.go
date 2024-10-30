package bot

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"path/filepath"
	"strconv"
)

func (lnb *LitNightBot) getChatDataFilePath(chatId int64) string {
	filePath := filepath.Join(lnb.dataPath, strconv.FormatInt(chatId, 10))
	lnb.logger.Debugf("Chat data file path: %s", filePath)
	return filePath
}

func (lnb *LitNightBot) getChatData(chatId int64) *chatdata.ChatData {
	var cd chatdata.ChatData
	filePath := lnb.getChatDataFilePath(chatId)

	if err := utils.ReadJSONFromFile(filePath, &cd); err != nil {
		lnb.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to read chat data from file")
		return nil
	}

	lnb.logger.WithField("chat_id", chatId).Info("Successfully read chat data")
	return &cd
}

func (lnb *LitNightBot) setChatData(chatId int64, cd *chatdata.ChatData) {
	filePath := lnb.getChatDataFilePath(chatId)

	if err := utils.WriteJSONToFile(filePath, cd); err != nil {
		lnb.logger.WithField("chat_id", chatId).WithError(err).Error("Failed to write chat data to file")
		return
	}

	lnb.logger.WithField("chat_id", chatId).Info("Successfully wrote chat data to file")
}
