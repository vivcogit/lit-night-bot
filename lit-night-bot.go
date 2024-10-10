package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type LitNightBot struct {
	bot      *tgbotapi.BotAPI
	dataPath string
}

type Book struct {
	Name string `json:"name"`
}

type HistoryItem struct {
	Book Book      `json:"book"`
	Date time.Time `json:"date"`
}

type ChatData struct {
	Wishlist []Book        `json:"wishlist"`
	History  []HistoryItem `json:"history"`
	Current  Book          `json:"current_book"`
}

func (vb *LitNightBot) getCallbackParamStr(action, data string) string {
	return action + ":" + data
}

func (vb *LitNightBot) getCallbackParam(callbackData string) (string, string, error) {
	cb := strings.Split(callbackData, ":")

	if len(cb) != 2 {
		return "", "", errors.New("Неизвестная операция: " + callbackData)
	}

	return cb[0], cb[1], nil
}

func (vb *LitNightBot) getChatDataFilePath(chatId int64) string {
	return filepath.Join(vb.dataPath, strconv.FormatInt(chatId, 10))
}

func (vb *LitNightBot) getChatData(chatId int64) *ChatData {
	var cd ChatData
	ReadJSONFromFile(vb.getChatDataFilePath(chatId), &cd)

	return &cd
}

func (vb *LitNightBot) setChatData(chatId int64, cd *ChatData) {
	WriteJSONToFile(vb.getChatDataFilePath(chatId), cd)
}

func (vb *LitNightBot) sendMessage(chatId int64, text string) {
	vb.bot.Send(tgbotapi.NewMessage(chatId, text))
}

func (vb *LitNightBot) handleCallbackQuery(callback *tgbotapi.CallbackQuery) {
	cbAction, cbParam, err := vb.getCallbackParam(callback.Data)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	chatId := callback.Message.Chat.ID

	switch cbAction {
	case "remove":
		vb.handleRemove(chatId, cbParam)
		callbackConfig := tgbotapi.NewCallback(callback.ID, "Действие выполнено")
		if _, err := vb.bot.Send(callbackConfig); err != nil {
			log.Printf("Error sending callback response: %v", err)
		}
		return
	default:
		log.Printf("Неизвестный callback " + cbAction)
	}

}

func (vb *LitNightBot) handleStart(chatId int64) {
	filePath := vb.getChatDataFilePath(chatId)
	exists, _ := CheckFileExists(filePath)

	if !exists {
		var chatData ChatData
		vb.setChatData(chatId, &chatData)
	}

	vb.sendMessage(chatId,
		"Привет, книжные фанаты! ✨\n"+
			"Я здесь, чтобы сделать ваш клуб ещё лучше!\n"+
			"📚 Теперь вы можете легко управлять списками книг, "+
			"выбирать следующую для чтения и не забывать, что уже обсуждали.\n"+
			"Давайте сделаем чтение ещё увлекательнее вместе!",
	)
}

func (vb *LitNightBot) handleList(chatId int64) {
	cd := vb.getChatData(chatId)

	var names []string
	for _, book := range cd.Wishlist {
		names = append(names, book.Name)
	}

	msg := strings.Join(names, "\n")
	fmt.Println(msg, names, cd)
	if msg == "" {
		vb.sendMessage(chatId, "Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
			"Самое время добавить новые книги и продолжить наши литературные приключения!")
		return
	}
	vb.sendMessage(chatId, msg)
}

func (vb *LitNightBot) handleAdd(chatId int64, book string) {
	if book == "" {
		vb.sendMessage(chatId, "Для добавления книги в список нужно указать её в команде add, например:\n/add Моя первая книга")
		return
	}
	cd := vb.getChatData(chatId)

	cd.Wishlist = append(cd.Wishlist, Book{book})

	vb.setChatData(chatId, cd)

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена в список", book))
}

func (vb *LitNightBot) handleRemove(chatId int64, bookname string) {
	if bookname == "" {
		vb.sendMessage(chatId, "Для удаления книги из списка нужно указать её в команде remove, например:\n/remove Моя первая книга")
		return
	}

	cd := vb.getChatData(chatId)

	index := -1
	for i, b := range cd.Wishlist {
		if strings.EqualFold(b.Name, bookname) {
			index = i
			break
		}
	}

	if index == -1 {
		vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" не найдена в списке", bookname))
		return
	}

	cd.Wishlist = append(cd.Wishlist[:index], cd.Wishlist[index+1:]...)

	vb.setChatData(chatId, cd)
	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" удалена из списка", bookname))
}

// TODO add pagination
func (vb *LitNightBot) handleEmptyRemove(chatId int64) {
	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		vb.sendMessage(chatId, "Список книг пуст, удалять нечего")
		return
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	for _, book := range cd.Wishlist {
		button := tgbotapi.NewInlineKeyboardButtonData("❌ "+book.Name, vb.getCallbackParamStr("remove", book.Name))

		inlineRow := tgbotapi.NewInlineKeyboardRow(button)

		inlineButtons = append(inlineButtons, inlineRow)
	}

	msg := tgbotapi.NewMessage(chatId, "Вот ваши книги в списке:")

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineButtons...)
	msg.ReplyMarkup = keyboard

	vb.bot.Send(msg)
}

func (vb *LitNightBot) Init() {
	commands := []tgbotapi.BotCommand{
		{
			Command:     "list",
			Description: "просмотр списка",
		},
		{
			Command:     "add",
			Description: "добавление книг в список, мультидобавление по строкам",
		},
		{
			Command:     "remove",
			Description: "удаление из списка",
		},
		// {
		// 	Command:     "history",
		// 	Description: "просмотр прочитанных",
		// },
		// {
		// 	Command:     "add_history",
		// 	Description: "добавить в прочитанные",
		// },
		// {
		// 	Command:     "remove_history",
		// 	Description: "удалить из прочитанных",
		// },
		{
			Command:     "current",
			Description: "отобразить текущую книгу",
		},
		// {
		// 	Command:     "current_deadline",
		// 	Description: "назначить срок дедлайна по текущей книге с опциональным напоминанием",
		// },
		// {
		// 	Command:     "current_complete",
		// 	Description: "пометить книгу прочитанной",
		// },
		// {
		// 	Command:     "current_random",
		// 	Description: "выбрать рандомом из списка",
		// },
		// {
		// 	Command:     "current_set",
		// 	Description: "выбрать книгу вручную",
		// },
		// {
		// 	Command:     "current_abort",
		// 	Description: "отменить текущую книгу (вернуть в список?)",
		// },
		// {
		// 	Command:     "help",
		// 	Description: "вывод справки",
		// },
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

	chatId := update.Message.Chat.ID
	cmdArg := update.Message.CommandArguments()

	switch update.Message.Command() {
	case "start":
		vb.handleStart(chatId)
	case "list":
		vb.handleList(chatId)
	case "add":
		vb.handleAdd(chatId, cmdArg)
	case "remove":
		if cmdArg == "" {
			vb.handleEmptyRemove(chatId)
			return
		}
		vb.handleRemove(chatId, cmdArg)
	default:
		vb.sendMessage(chatId, "Unknown command")
	}
}

func (vb *LitNightBot) Start() {
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates := vb.bot.GetUpdatesChan(updateConfig)

	for update := range updates {
		if update.CallbackQuery != nil {
			vb.handleCallbackQuery(update.CallbackQuery)
			continue
		}

		if update.Message != nil {
			vb.handleMessage(&update)
		}

	}
}

func NewLitNightBot() LitNightBot {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		panic("failed to retrieve the Telegram token from the environment")
	}

	dataPath := os.Getenv("DATA_PATH")
	if dataPath == "" {
		panic("failed to retrieve path to storage chats data")
	}

	bot, err := GetBot(token, true)

	if err != nil {
		panic(err)
	}

	return LitNightBot{bot, dataPath}
}

func GetBot(token string, isDebug bool) (*tgbotapi.BotAPI, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	bot.Debug = isDebug
	log.Printf("Authorized on account %s", bot.Self.UserName)
	return bot, nil
}
