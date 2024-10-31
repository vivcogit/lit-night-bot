package bot

import (
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type LitNightBot struct {
	bot      *tgbotapi.BotAPI
	dataPath string
	logger   *logrus.Logger
}

func getUpdateChatID(update *tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}
	return 0
}

func getUpdateUserFrom(update *tgbotapi.Update) *tgbotapi.User {
	if update.Message != nil {
		return update.Message.From
	} else if update.CallbackQuery != nil {
		return update.CallbackQuery.From
	}
	return nil
}

func (lnb *LitNightBot) getUserLogger(update *tgbotapi.Update) *logrus.Entry {
	chatID := getUpdateChatID(update)
	user := getUpdateUserFrom(update)

	return lnb.logger.WithFields(logrus.Fields{
		"user_id":   user.ID,
		"user_name": user.UserName,
		"chat_id":   chatID,
	})
}

func NewLitNightBot(token, dataPath string, isDebug bool) (*LitNightBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	logger := logrus.New()
	if isDebug {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&logrus.JSONFormatter{})
	}
	bot.Debug = isDebug

	logger.WithField("username", bot.Self.UserName).Info("Bot authorized")

	return &LitNightBot{bot, dataPath, logger}, nil
}

func (lnb *LitNightBot) Start() {
	lnb.logger.Info("Starting bot")

	lnb.InitMenu()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := lnb.bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		go func(update tgbotapi.Update) {
			logger := lnb.getUserLogger(&update)

			if update.CallbackQuery != nil {
				lnb.handleCallbackQuery(&update, logger)
				return
			}
			if update.Message != nil && update.Message.IsCommand() {
				lnb.handleCommand(&update, logger)
				return
			}
			if update.Message != nil && update.Message.ReplyToMessage != nil {
				lnb.handleReply(&update, logger)
				return
			}
		}(update)
	}
}

func (lnb *LitNightBot) handleStart(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID := getUpdateChatID(update)

	logger.Info("Handling /start command")

	filePath := lnb.getChatDataFilePath(chatID)
	exists, _ := utils.CheckFileExists(filePath)

	if !exists {
		var chatData chatdata.ChatData
		lnb.setChatData(chatID, &chatData)
	}

	lnb.sendPlainMessage(
		chatID,
		"Привет, книжные фанаты! ✨\n"+
			"Я здесь, чтобы сделать ваш клуб ещё лучше!\n"+
			"📚 Теперь вы можете легко управлять списками книг, "+
			"выбирать следующую для чтения и не забывать, что уже обсуждали.\n"+
			"Давайте сделаем чтение ещё увлекательнее вместе!",
	)
	logger.Info("Start message sent")
}

func (lnb *LitNightBot) InitMenu() {
	commands := []tgbotapi.BotCommand{
		{Command: "menu", Description: "показать меню"},
	}

	_, err := lnb.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		lnb.logger.WithError(err).Fatal("Failed to set bot commands")
	}
	lnb.logger.Info("Menu initialized")
}
