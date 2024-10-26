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

		if update.Message != nil {
			vb.handleMessage(&update)
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

	vb.sendMessage(chatId,
		"Привет, книжные фанаты! ✨\n"+
			"Я здесь, чтобы сделать ваш клуб ещё лучше!\n"+
			"📚 Теперь вы можете легко управлять списками книг, "+
			"выбирать следующую для чтения и не забывать, что уже обсуждали.\n"+
			"Давайте сделаем чтение ещё увлекательнее вместе!",
		nil,
	)
}

func (vb *LitNightBot) InitMenu() {
	commands := []tgbotapi.BotCommand{
		{
			Command:     string(UAList),
			Description: "просмотр списка",
		},
		{
			Command:     string(UAAdd),
			Description: "добавление книг в список, мультидобавление по строкам",
		},
		{
			Command:     string(UARemove),
			Description: "удаление из списка",
		},
		{
			Command:     string(UAHistory),
			Description: "просмотр прочитанных",
		},
		{
			Command:     string(UAHistoryAdd),
			Description: "добавить в прочитанные",
		},
		{
			Command:     string(UAHistoryRemove),
			Description: "удалить из прочитанных",
		},
		{
			Command:     string(UACurrent),
			Description: "отобразить текущую книгу",
		},
		{
			Command:     string(UACurrentDeadline),
			Description: "назначить срок дедлайна по текущей книге с опциональным напоминанием",
		},
		{
			Command:     string(UACurrentComplete),
			Description: "пометить книгу прочитанной",
		},
		{
			Command:     string(UACurrentRandom),
			Description: "выбрать рандомом из списка",
		},
		{
			Command:     string(UACurrentAbort),
			Description: "отменить выбор книги",
		},
	}

	_, err := vb.bot.Request(tgbotapi.NewSetMyCommands(commands...))
	if err != nil {
		log.Panic(err)
	}
}

func (vb *LitNightBot) handleMessage(update *tgbotapi.Update) {
	if !update.Message.IsCommand() {
		return
	}

	cmd := UserAction(update.Message.Command())
	switch cmd {
	case UAStart:
		vb.handleStart(update.Message)
	case UAList:
		vb.handleShowWishlist(update.Message)
	case UAAdd: // TODO сохранять добавителя
		vb.handleAdd(update.Message)
	case UACurrent:
		vb.handleCurrent(update.Message)
	case UACurrentSet: // TODO remove?
		vb.handleCurrentSet(update.Message)
	case UACurrentRandom:
		vb.handleCurrentRandom(update.Message)
	case UACurrentAbort:
		vb.handleCurrentAbort(update.Message)
	case UACurrentComplete:
		vb.handleCurrentComplete(update.Message)
	case UACurrentDeadline:
		vb.handleCurrentDeadline(update.Message)
	case UARemove:
		vb.handleRemoveFromWishlist(update.Message)
	case UAHistory:
		vb.handleHistoryList(update.Message)
	case UAHistoryAdd:
		vb.handleAddHistory(update.Message)
	case UAHistoryRemove:
		vb.handleRemoveHistory(update.Message)
	default:
		vb.sendMessage(update.Message.Chat.ID, "Упс, неизвестная команда, попробуем ещё раз?", nil)
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
