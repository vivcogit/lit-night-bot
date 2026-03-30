package bot

import (
	chatdata "lit-night-bot/chat-data"
	io "lit-night-bot/io"
	"lit-night-bot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type LitNightBot struct {
	bot    *tgbotapi.BotAPI
	iocd   *io.IoChatData
	logger *logrus.Entry
}

func chatIDFromUpdate(update *tgbotapi.Update, log *logrus.Entry) (chatID int64, ok bool) {
	chat := update.FromChat()
	if chat == nil {
		log.WithField("update_id", update.UpdateID).Error("update has no chat context")
		return 0, false
	}
	return chat.ID, true
}

func (lnb *LitNightBot) getUserLogger(chatID int64, update *tgbotapi.Update) *logrus.Entry {
	user := update.SentFrom()

	fields := logrus.Fields{"chat_id": chatID}
	if user != nil {
		fields["user_id"] = user.ID
		fields["user_name"] = user.UserName
	}
	return lnb.logger.WithFields(fields)
}

func NewLitNightBot(logger *logrus.Entry, token string, iocd *io.IoChatData, isDebug bool) (*LitNightBot, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = isDebug

	logger.WithField("username", bot.Self.UserName).Info("Bot authorized")

	return &LitNightBot{bot, iocd, logger}, nil
}

func (lnb *LitNightBot) handleUpdatesChan(updates *tgbotapi.UpdatesChannel) {
	for update := range *updates {
		go func(update tgbotapi.Update) {
			chatID, ok := chatIDFromUpdate(&update, lnb.logger)
			if !ok {
				return
			}
			logger := lnb.getUserLogger(chatID, &update)

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

func (lnb *LitNightBot) Start() {
	lnb.logger.Info("Starting bot")

	lnb.InitMenu()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := lnb.bot.GetUpdatesChan(updateConfig)

	go lnb.handleUpdatesChan(&updates)
}

func (lnb *LitNightBot) handleStart(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID, ok := chatIDFromUpdate(update, logger)
	if !ok {
		return
	}

	logger.Info("Handling /start command")

	filePath := lnb.iocd.GetChatDataFilePath(chatID)
	exists, _ := utils.CheckFileExists(filePath)

	if !exists {
		var chatData chatdata.ChatData
		lnb.iocd.SetChatData(chatID, &chatData)
	}

	lnb.SendPlainMessage(
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
