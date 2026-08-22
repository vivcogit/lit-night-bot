package bot

import (
	"strings"
	"unicode/utf16"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func selectiveForceReplyConfig(chatID int64, user *tgbotapi.User, text string) tgbotapi.MessageConfig {
	mention := "Участник"
	if user != nil {
		fullName := strings.TrimSpace(strings.Join([]string{user.FirstName, user.LastName}, " "))
		if fullName != "" {
			mention = fullName
		} else if user.UserName != "" {
			mention = "@" + user.UserName
		}
	}

	request := tgbotapi.NewMessage(chatID, mention+", "+text)
	if user != nil {
		request.Entities = []tgbotapi.MessageEntity{{
			Type:   "text_mention",
			Offset: 0,
			Length: len(utf16.Encode([]rune(mention))),
			User:   user,
		}}
	}
	request.ReplyMarkup = tgbotapi.ForceReply{ForceReply: true, Selective: user != nil}
	return request
}
