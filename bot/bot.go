package bot

import (
	"fmt"
	io "lit-night-bot/io"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type LitNightBot struct {
	bot               *tgbotapi.BotAPI
	iocd              *io.IoChatData
	logger            *logrus.Entry
	locks             sync.Map
	historySelections sync.Map
	location          *time.Location
}

func (lnb *LitNightBot) chatMutex(chatID int64) *sync.Mutex {
	lockValue, _ := lnb.locks.LoadOrStore(chatID, &sync.Mutex{})
	return lockValue.(*sync.Mutex)
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

func NewLitNightBot(logger *logrus.Entry, token string, iocd *io.IoChatData, isDebug bool, location *time.Location) (*LitNightBot, error) {
	if location == nil {
		return nil, fmt.Errorf("application location is required")
	}
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = isDebug

	logger.WithField("username", bot.Self.UserName).Info("Bot authorized")

	return &LitNightBot{bot: bot, iocd: iocd, logger: logger, location: location}, nil
}

func (lnb *LitNightBot) handleUpdatesChan(updates *tgbotapi.UpdatesChannel) {
	for update := range *updates {
		go func(update tgbotapi.Update) {
			chatID, ok := chatIDFromUpdate(&update, lnb.logger)
			if !ok {
				return
			}
			logger := lnb.getUserLogger(chatID, &update)
			chatLock := lnb.chatMutex(chatID)
			chatLock.Lock()
			defer chatLock.Unlock()

			if !lnb.allowUpdate(&update, logger) {
				return
			}

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

	lnb.iocd.GetOrCreateChatData(chatID)
	if chat := update.FromChat(); chat != nil && chat.IsPrivate() {
		lnb.SendPlainMessage(
			chatID,
			"Привет! ✨\n"+
				"Это твой личный читательский дневник. Здесь можно вести вишлист, "+
				"выбирать текущую книгу, сохранять историю чтения и ставить личные оценки. 📚",
		)
		logger.Info("Personal diary start message sent")
		return
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

	scopes := []tgbotapi.BotCommandScope{
		tgbotapi.NewBotCommandScopeDefault(),
		tgbotapi.NewBotCommandScopeAllPrivateChats(),
		tgbotapi.NewBotCommandScopeAllGroupChats(),
		tgbotapi.NewBotCommandScopeAllChatAdministrators(),
	}
	for _, scope := range scopes {
		if _, err := lnb.bot.Request(tgbotapi.NewDeleteMyCommandsWithScope(scope)); err != nil {
			lnb.logger.WithError(err).WithField("scope", scope.Type).Fatal("Failed to delete old bot commands")
		}
		if _, err := lnb.bot.Request(tgbotapi.NewSetMyCommandsWithScope(scope, commands...)); err != nil {
			lnb.logger.WithError(err).WithField("scope", scope.Type).Fatal("Failed to set bot commands")
		}
	}
	lnb.logger.Info("Menu initialized")
}
