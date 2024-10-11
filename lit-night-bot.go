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
	"golang.org/x/exp/rand"
)

type LitNightBot struct {
	bot      *tgbotapi.BotAPI
	dataPath string
}

var progressJokes = []string{
	"Думаю... думаю... кажется, нашел книгу, которая смотрит на меня в ответ!",
	"Дайте мне пару секунд, книги устраивают бой за ваше внимание!",
	"Секунду... мне нужно спросить у всех персонажей, кто готов к встрече.",
	"Так, так, так... какая из книг станет звездой сегодняшнего вечера?",
	"Магический шар предсказаний вращается... и... почти готов!",
	"Дайте мне немного времени, книги все еще спорят, кто из них лучше.",
	"Выбираю... кажется, одна из книг шепчет мне на ухо!",
	"Загружаю данные... о, кажется, одна книга уже подмигнула мне!",
	"Книги так и прыгают на полку, дайте мне минутку их успокоить!",
	"Книги играют в прятки, но я вот-вот их найду!",
	"Загружаю данные... 99%, 99%, 99%... Ой, опять зависло на 99%.",
	"Ищу ответ в 42-страничной инструкции... почти нашел!",
	"Анализирую отражение луны в глазах Шрека.",
	"Отправляю запрос котам. Ответ от кота: 'мяу'. Перевожу...",
	"Провожу многоходовочку, как Шерлок в последнем сезоне.",
	"Запускаю квантовый процессор. Ой, это тостер, секунду...",
	"Пересчитываю уток в Майнкрафт... Подождите немного.",
	"Секунду... нахожу нужную инфу в файлах 'Мемы 2012-го'.",
	"Проверяю, хватит ли колбасы для этого вычисления.",
	"Сейчас, обнуляю счетчик дня без багов... Ой, он снова сломался.",
}

func (vb *LitNightBot) sendProgressJokes(chatId int64) {
	rand.Seed(uint64((time.Now().UnixNano())))

	numMessages := rand.Intn(3) + 3

	rand.Shuffle(len(progressJokes), func(i, j int) {
		progressJokes[i], progressJokes[j] = progressJokes[j], progressJokes[i]
	})

	for i := 0; i < numMessages; i++ {
		vb.sendMessage(chatId, progressJokes[i])

		sleepDuration := time.Duration(rand.Intn(1000)+1000) * time.Millisecond
		time.Sleep(sleepDuration)
	}
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
		vb.removeBookFromWishList(chatId, cbParam)

		callbackConfig := tgbotapi.NewCallback(callback.ID, "Действие выполнено")
		if _, err := vb.bot.Send(callbackConfig); err != nil {
			log.Printf("Error sending callback response: %v", err)
		}
		return
	default:
		log.Printf("Неизвестный callback " + cbAction)
	}

}

func (vb *LitNightBot) handleStart(message *tgbotapi.Message) {
	chatId := message.Chat.ID

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

func (vb *LitNightBot) handleList(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	var names []string
	for _, item := range cd.Wishlist {
		names = append(names, item.Book.Name)
	}

	msg := strings.Join(names, "\n")

	if msg == "" {
		vb.sendMessage(chatId, "Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
			"Самое время добавить новые книги и продолжить наши литературные приключения!")
		return
	}
	vb.sendMessage(chatId, msg)
}

func (vb *LitNightBot) handleCurrent(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	var msg string

	if cd.Current.Book.Name == "" {
		msg = "Похоже, у вас пока нет выбранной книги. Как насчёт выбрать что-нибудь интересное для чтения?"
	} else {
		msg = fmt.Sprintf("В данный момент вы читаете книгу \"%s\". Как вам она? Делитесь впечатлениями!", cd.Current.Book.Name)
	}

	vb.sendMessage(chatId, msg)
}

func (vb *LitNightBot) handleCurrentRandom(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	if cd.Current.Book.Name != "" {
		vb.sendMessage(chatId,
			fmt.Sprintf("Вы уже читаете \"%s\"\n"+
				"Эта книга не заслуживает такого обращения!\n"+
				"Но если вы хотите новую, давайте найдем ее вместе!\n"+
				"Но сначала скажите ей об отмене",
				cd.Current.Book.Name),
		)
		return
	}

	if len(cd.Wishlist) == 0 {
		vb.sendMessage(chatId, "Ваш вишлист пуст! Добавьте книги, чтобы я мог выбрать одну для вас.")
		return
	}

	go func() {
		vb.sendProgressJokes(chatId)

		randomIndex := rand.Intn(len(cd.Wishlist))

		cd.Current = CurrentBook{cd.Wishlist[randomIndex].Book}
		cd.RemoveBookFromWishlist(cd.Wishlist[randomIndex].Book.Name)

		vb.setChatData(chatId, cd)

		vb.sendMessage(chatId, fmt.Sprintf("Тадааам! Вот ваша книга: \"%s\". Приятного чтения!", cd.Current.Book.Name))
	}()
}

func (vb *LitNightBot) handleAdd(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.sendMessage(chatId, "Эй, книжный искатель! 📚✨ Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде add, например:\n/add Моя первая книга")
		return
	}

	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) >= 10 {
		vb.sendMessage(chatId,
			"Ой-ой! Похоже, ваш вишлист уже полон книг! 📚✨\nЧтобы добавить новую, давайте попрощаемся с одной из них.")
		return
	}

	cd.AddBookToWishlist(bookname)

	vb.setChatData(chatId, cd)

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена в список", bookname))
}

func (vb *LitNightBot) handleAddHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.sendMessage(chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
		)
		return
	}

	cd := vb.getChatData(chatId)

	cd.AddBookToHistory(bookname)

	vb.setChatData(chatId, cd)

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена в список", bookname))
}

func (vb *LitNightBot) handleRemove(message *tgbotapi.Message) {
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.handleEmptyRemove(message)
		return
	}

	chatId := message.Chat.ID
	vb.removeBookFromWishList(chatId, bookname)
}

func (vb *LitNightBot) removeBookFromWishList(chatId int64, bookname string) {
	cd := vb.getChatData(chatId)
	err := cd.RemoveBookFromWishlist(bookname)
	vb.setChatData(chatId, cd)

	if err != nil {
		vb.sendMessage(chatId, err.Error())
		return
	}

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" удалена из списка", bookname))
}

func (vb *LitNightBot) handleEmptyRemove(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) == 0 {
		vb.sendMessage(chatId, "Список книг пуст, удалять нечего")
		return
	}

	var inlineButtons [][]tgbotapi.InlineKeyboardButton
	for _, item := range cd.Wishlist {
		bookname := item.Book.Name
		button := tgbotapi.NewInlineKeyboardButtonData(
			"❌ "+bookname,
			vb.getCallbackParamStr("remove", bookname),
		)

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
		{
			Command:     "add_history",
			Description: "добавить в прочитанные",
		},
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
		{
			Command:     "current_complete",
			Description: "пометить книгу прочитанной",
		},
		{
			Command:     "current_random",
			Description: "выбрать рандомом из списка",
		},
		// {
		// 	Command:     "current_set",
		// 	Description: "выбрать книгу вручную",
		// },
		// {
		// 	Command:     "current_abort", // спрашивать про удаление или возврат в вишлист
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

	switch update.Message.Command() {
	case "start":
		vb.handleStart(update.Message)
	case "list":
		vb.handleList(update.Message)
	case "add": // TODO сохранять автора
		vb.handleAdd(update.Message)
	case "current":
		vb.handleCurrent(update.Message)
	case "current_random":
		vb.handleCurrentRandom(update.Message)
	case "remove":
		vb.handleRemove(update.Message)
	case "history-add":
		vb.handleAddHistory(update.Message)
	default:
		vb.sendMessage(update.Message.Chat.ID, "Unknown command")
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
