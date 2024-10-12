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
		vb.removeBookFromWishlist(chatId, cbParam)

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

func (vb *LitNightBot) handleWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	names := cd.GetWishlistBooks()

	if len(names) == 0 {
		vb.sendMessage(chatId, "Все книги из очереди уже прочитаны, и сейчас список пуст.\n"+
			"Самое время добавить новые книги и продолжить наши литературные приключения!")
		return
	}

	vb.sendMessage(chatId, "Вот что ждёт вас в ближайшее время:\n\n"+strings.Join(names, "\n")+"\n\nГотовы начать? 📖✨")
}

func (vb *LitNightBot) handleHistoryList(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	names := cd.GetHistoryBooks()

	if len(names) == 0 {
		vb.sendMessage(chatId,
			"Кажется, список прочитанных книг пока пуст... 😕\n"+
				"Но не переживайте! Начните прямо сейчас, и скоро здесь будут ваши книжные достижения! 📚💪",
		)
		return
	}

	vb.sendMessage(chatId, "Вот ваши уже прочитанные книги:\n\n"+strings.Join(names, "\n")+"\nОтличная работа! 👏📖")
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

func (vb *LitNightBot) handleCurrentSet(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.sendMessage(chatId, "/current-set <bookname>")
		return
	}

	cd := vb.getChatData(chatId)

	if cd.Current.Book.Name != "" {
		vb.sendMessage(chatId,
			fmt.Sprintf("О, кажется, вы уже читаете \"%s\"! 📖\n"+
				"Может, сначала завершим эту книгу, прежде чем начать новое приключение? 😉",
				cd.Current.Book.Name,
			))
		return
	}

	err := cd.RemoveBookFromWishlist(bookname)
	cd.SetCurrentBook(bookname)
	vb.setChatData(chatId, cd)

	if err != nil && len(cd.Wishlist) > 0 {
		vb.sendMessage(
			chatId,
			"Кажется, выбранная вами книга не из вашего вишлиста. 📚\n"+
				"Может, в следующий раз стоит выбрать что-то из списка желаемого чтения? 😄",
		)
		return
	}

	vb.sendMessage(
		chatId,
		fmt.Sprintf(
			"Отличный выбор! Теперь ваша новая книга для чтения — \"%s\". 📚✨\n"+
				"Удачного чтения, и не забудьте вернуться для обсуждения! 😉",
			bookname,
		),
	)
}

func (vb *LitNightBot) handleCurrentComplete(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	currentBook := cd.Current.Book.Name
	if currentBook == "" {
		vb.sendMessage(
			chatId,
			"Хмм... Похоже, у вас ещё нет книги в процессе чтения.\n"+
				"Давайте выберем что-нибудь интересное и погрузимся в новые страницы! 📚✨",
		)
		return
	}

	cd.AddBookToHistory(currentBook)
	cd.Current = CurrentBook{}

	vb.setChatData(chatId, cd)

	vb.sendMessage(
		chatId,
		fmt.Sprintf(
			"Ура! Книга \"%s\" прочитана! 🎉\n"+
				"Надеюсь, она оставила вам море впечатлений.\n"+
				"Готовы к следующему литературному приключению?",
			currentBook,
		),
	)
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
		randomBook := cd.Wishlist[randomIndex].Book.Name

		cd.SetCurrentBook(randomBook)
		cd.RemoveBookFromWishlist(randomBook)

		vb.setChatData(chatId, cd)

		vb.sendMessage(chatId, fmt.Sprintf("Тадааам! Вот ваша книга: \"%s\". Приятного чтения!", cd.Current.Book.Name))
	}()
}

func (vb *LitNightBot) handleAdd(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := HandleMultiArgs(strings.Split(message.CommandArguments(), "\n"))

	if len(booknames) == 0 {
		vb.sendMessage(chatId, "Эй, книжный искатель! 📚✨ Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде add, например:\n/add Моя первая книга")
		return
	}

	cd := vb.getChatData(chatId)

	if len(cd.Wishlist) >= 10 {
		vb.sendMessage(chatId,
			"Ой-ой! Похоже, ваш вишлист уже полон книг! 📚✨\nЧтобы добавить новую, давайте попрощаемся с одной из них.")
		return
	}

	if len(cd.Wishlist)+len(booknames) >= 10 {
		vb.sendMessage(chatId,
			"Ой-ой! Похоже, я не смогу запомнить столько книг! 📚✨\nЧтобы добавить новые, давайте попрощаемся с кем-то из старых.")
		return
	}

	cd.AddBooksToWishlist(booknames)

	vb.setChatData(chatId, cd)

	if len(booknames) == 1 {
		vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена.", booknames[0]))
	} else {
		vb.sendMessage(chatId, fmt.Sprintf("Книги \"%s\" добавлены.", strings.Join(booknames, "\", \"")))
	}
}

func (vb *LitNightBot) handleAddHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := HandleMultiArgs(strings.Split(message.CommandArguments(), "\n"))

	if len(booknames) == 0 {
		vb.sendMessage(chatId,
			"Эй, книжный искатель! 📚✨\n"+
				"Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде history-add, например:\n/history-add Моя первая книга",
		)
		return
	}

	cd := vb.getChatData(chatId)

	cd.AddBooksToHistory(booknames)

	vb.setChatData(chatId, cd)

	if len(booknames) == 1 {
		vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" добавлена в историю.", booknames[0]))
	} else {
		vb.sendMessage(chatId, fmt.Sprintf("Книги \"%s\" добавлены в историю.", strings.Join(booknames, "\", \"")))
	}
}

func (vb *LitNightBot) handleRemoveWishlist(message *tgbotapi.Message) {
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.handleEmptyRemove(message)
		return
	}

	chatId := message.Chat.ID
	vb.removeBookFromWishlist(chatId, bookname)
}

func (vb *LitNightBot) handleRemoveHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	bookname := message.CommandArguments()

	if bookname == "" {
		vb.sendMessage(chatId,
			"Чтобы удалить книгу из истории нужно сказать мне её название: /history_remove Название книги\n"+
				"Таков путь!",
		)
		return
	}

	vb.removeBookFromHistory(chatId, bookname)
}

func (vb *LitNightBot) removeBookFromHistory(chatId int64, bookname string) {
	cd := vb.getChatData(chatId)
	err := cd.RemoveBookFromHistory(bookname)
	vb.setChatData(chatId, cd)

	if err != nil {
		vb.sendMessage(chatId, err.Error())
		return
	}

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" удалена из списка", bookname))
}

func (vb *LitNightBot) removeBookFromWishlist(chatId int64, bookname string) {
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
			TruncateString("❌ "+bookname, 60),
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
		{
			Command:     "history",
			Description: "просмотр прочитанных",
		},
		{
			Command:     "add_history",
			Description: "добавить в прочитанные",
		},
		{
			Command:     "history_remove",
			Description: "удалить из прочитанных",
		},
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
		{
			Command:     "current_set",
			Description: "выбрать книгу вручную",
		},
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
		vb.handleWishlist(update.Message)
	case "add": // TODO сохранять автора
		vb.handleAdd(update.Message)
	case "current":
		vb.handleCurrent(update.Message)
	case "current_set":
		vb.handleCurrentSet(update.Message)
	case "current_random":
		vb.handleCurrentRandom(update.Message)
	case "current_complete":
		vb.handleCurrentComplete(update.Message)
	case "remove":
		vb.handleRemoveWishlist(update.Message)
	case "history":
		vb.handleHistoryList(update.Message)
	case "history_add":
		vb.handleAddHistory(update.Message)
	case "history_remove":
		vb.handleRemoveHistory(update.Message)
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
