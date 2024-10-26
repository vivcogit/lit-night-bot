package bot

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"path/filepath"
	"strconv"
)

func (vb *LitNightBot) getChatDataFilePath(chatId int64) string {
	return filepath.Join(vb.dataPath, strconv.FormatInt(chatId, 10))
}

func (vb *LitNightBot) getChatData(chatId int64) *chatdata.ChatData {
	var cd chatdata.ChatData
	utils.ReadJSONFromFile(vb.getChatDataFilePath(chatId), &cd)

	return &cd
}

func (vb *LitNightBot) setChatData(chatId int64, cd *chatdata.ChatData) {
	utils.WriteJSONToFile(vb.getChatDataFilePath(chatId), cd)
}
