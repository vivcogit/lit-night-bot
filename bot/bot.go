package bot

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type LitNightBot struct {
	bot      *tgbotapi.BotAPI
	dataPath string
}

func (vb *LitNightBot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := vb.bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.CallbackQuery != nil {
			vb.handleCallbackQuery(&update)
			continue
		}

		if update.Message != nil && update.Message.IsCommand() {
			vb.handleCommand(&update)
			continue
		}

		if update.Message != nil && update.Message.ReplyToMessage != nil {
			vb.handleReply(&update)
			continue
		}
	}
}

func (vb *LitNightBot) handleStart(message *tgbotapi.Message) {
	chatId := message.Chat.ID

	filePath := vb.getChatDataFilePath(chatId)
	exists, _ := utils.CheckFileExists(filePath)

	if !exists {
		var chatData chatdata.ChatData
		vb.setChatData(chatId, &chatData)
	}

	vb.sendPlainMessage(
		chatId,
		"Привет, книжные фанаты! ✨\n"+
			"Я здесь, чтобы сделать ваш клуб ещё лучше!\n"+
			"📚 Теперь вы можете легко управлять списками книг, "+
			"выбирать следующую для чтения и не забывать, что уже обсуждали.\n"+
			"Давайте сделаем чтение ещё увлекательнее вместе!",
	)
}

func (vb *LitNightBot) InitMenu() {
	commands := []tgbotapi.BotCommand{
		{
			Command:     string(CmdMenu),
			Description: "показать меню",
		},
	}

	_, err := vb.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Panic(err)
	}
}

func NewLitNightBot(token string, dataPath string, isDebug bool) (*LitNightBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = isDebug
	if isDebug {
		log.Printf("Authorized on account %s", bot.Self.UserName)
	}

	return &LitNightBot{bot, dataPath}, nil
}
