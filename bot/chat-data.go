package bot

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"path/filepath"
	"strconv"
)

func (lnb *LitNightBot) getChatDataFilePath(chatId int64) string {
	return filepath.Join(lnb.dataPath, strconv.FormatInt(chatId, 10))
}

func (lnb *LitNightBot) getChatData(chatId int64) *chatdata.ChatData {
	var cd chatdata.ChatData
	utils.ReadJSONFromFile(lnb.getChatDataFilePath(chatId), &cd)

	return &cd
}

func (lnb *LitNightBot) setChatData(chatId int64, cd *chatdata.ChatData) {
	utils.WriteJSONToFile(lnb.getChatDataFilePath(chatId), cd)
}
