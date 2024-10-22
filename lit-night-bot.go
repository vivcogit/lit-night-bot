package main

import (
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

type UserAction string

const (
	UAStart           UserAction = "start"
	UAList            UserAction = "list"
	UAAdd             UserAction = "add"
	UACurrent         UserAction = "current"
	UACurrentSet      UserAction = "current_set"
	UACurrentRandom   UserAction = "current_random"
	UACurrentAbort    UserAction = "current_abort"
	UACurrentComplete UserAction = "current_complete"
	UACurrentDeadline UserAction = "current_deadline"
	UARemove          UserAction = "remove"
	UAHistory         UserAction = "history"
	UAHistoryAdd      UserAction = "history_add"
	UAHistoryRemove   UserAction = "history_remove"
)

type LitNightBot struct {
	bot      *tgbotapi.BotAPI
	dataPath string
}

func (vb *LitNightBot) sendProgressJokes(chatId int64) {
	rand.Seed(uint64((time.Now().UnixNano())))

	numMessages := rand.Intn(3) + 3

	rand.Shuffle(len(ProgressJokes), func(i, j int) {
		ProgressJokes[i], ProgressJokes[j] = ProgressJokes[j], ProgressJokes[i]
	})

	for i := 0; i < numMessages; i++ {
		vb.sendMessage(chatId, ProgressJokes[i])

		sleepDuration := time.Duration(rand.Intn(1000)+1000) * time.Millisecond
		time.Sleep(sleepDuration)
	}
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

func (vb *LitNightBot) editMessage(chatId int64, msgID int, text string, buttons [][]tgbotapi.InlineKeyboardButton) error {
	editMsg := tgbotapi.NewEditMessageText(chatId, msgID, text)
	var markup tgbotapi.InlineKeyboardMarkup
	if len(buttons) > 0 {
		markup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		editMsg.ReplyMarkup = &markup
	} else {
		editMsg.ReplyMarkup = nil
	}
	_, err := vb.bot.Send(editMsg)

	return err
}

func (vb *LitNightBot) sendMessage(chatId int64, text string) {
	vb.bot.Send(tgbotapi.NewMessage(chatId, text))
}

func (vb *LitNightBot) moveCurrentBook(chatId int64, messageID int, moveToHistory bool) {
	cd := vb.getChatData(chatId)
	currentBookName := cd.Current.Book.Name
	if moveToHistory {
		cd.AddBookToHistory(currentBookName)
	} else {
		cd.AddBookToWishlist(currentBookName)
	}
	cd.Current = CurrentBook{}
	vb.setChatData(chatId, cd)

	if moveToHistory {
		vb.editMessage(
			chatId,
			messageID,
			fmt.Sprintf(
				"📖 Книга \"%s\" теперь в истории!\nВремя выбрать новую приключенческую историю! 🚀",
				currentBookName,
			),
			nil,
		)
	} else {
		vb.editMessage(
			chatId,
			messageID,
			fmt.Sprintf(
				"📝 Книга \"%s\" вернулась в список ожидания!\nДавайте подберем для вас новую интересную историю! 📚✨",
				currentBookName,
			),
			nil,
		)
	}
}

func (vb *LitNightBot) handleCallbackQuery(update *tgbotapi.Update) {
	cbAction, cbParam, err := GetCallbackParam(update.CallbackQuery.Data)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	chatId := update.CallbackQuery.Message.Chat.ID
	messageId := update.CallbackQuery.Message.MessageID

	switch cbAction {
	case CBRemove:
		cd := vb.getChatData(chatId)
		_, err := cd.RemoveBookFromWishlist(cbParam)
		vb.setChatData(chatId, cd)

		if err != nil {
			vb.sendMessage(chatId, err.Error())
			return
		}

		callbackConfig := tgbotapi.NewCallback(
			update.CallbackQuery.ID,
			"🎉 Ура! Книга удалена из вашего списка желаемого! Теперь у вас больше времени для выбора новой! 📚",
		)
		vb.bot.Send(callbackConfig)

		text, buttons := getCleanWishlistMessage(cd)
		vb.editMessage(chatId, messageId, text, buttons)
		return
	case CBRemovePage:
		page, err := strconv.Atoi(cbParam)
		if err != nil {
			vb.sendMessage(chatId, "Ошибка обработки страницы")
			return
		}
		vb.showRemoveWishlistPage(chatId, messageId, page)

	case CBCancel:
		vb.editMessage(chatId, update.CallbackQuery.Message.MessageID, "🤭 Упс! Вы отменили действие! Не переживайте, в следующий раз все получится! 😉", nil)

	case CBCurrentToHistory:
		vb.moveCurrentBook(chatId, update.CallbackQuery.Message.MessageID, true)

	case CBCurrentToWishlist:
		vb.moveCurrentBook(chatId, update.CallbackQuery.Message.MessageID, false)

	default:
		log.Printf("Неизвестный callback: %s. Пожалуйста, позаботьтесь об этом, чтобы мы могли помочь вам выбрать следующую книгу! 📚😅", string(cbAction))
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

	vb.sendMessage(chatId, GetWishlistMessage(names))
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

	vb.sendMessage(chatId, "Вот ваши уже прочитанные книги:\n\n✔ "+strings.Join(names, "\n✔ ")+"\nОтличная работа! 👏📖")
}

func (vb *LitNightBot) handleCurrent(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	var msg string

	if cd.Current.Book.Name == "" {
		msg = "Похоже, у вас пока нет выбранной книги. Как насчёт выбрать что-нибудь интересное для чтения?"
	} else {
		msg = fmt.Sprintf(
			"В данный момент вы читаете книгу \"%s\" 📖\n"+
				"Как вам она? Делитесь впечатлениями! 😊\n"+
				"Кстати, у вас назначен дедлайн на %s.\n"+
				"Надеюсь, до этого времени вы не потеряетесь в мире страниц! 📅😈",
			cd.Current.Book.Name, cd.Current.Deadline.Format(DATE_LAYOUT))
	}

	vb.sendMessage(chatId, msg)
}

const DATE_LAYOUT = "02.01.2006"

func (vb *LitNightBot) handleCurrentDeadline(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	if cd.Current.Book.UUID == "" {
		vb.sendMessage(
			chatId,
			"Хей-хей! 🚀\n"+
				"Похоже, мы находимся в параллельной вселенной!\n"+
				"Устанавливать дедлайн без выбранной книги — это как пытаться запустить ракету без топлива. 🚀💨\n"+
				"Давайте сначала выберем книгу, а потом уже обсудим, когда будем её читать! Так мы точно не улетим в никуда! 📖✨",
		)
		return
	}

	dateStr := message.CommandArguments()

	date, err := time.Parse(DATE_LAYOUT, dateStr)

	if err != nil {
		vb.sendMessage(
			chatId,
			"Ой-ой, кажется, где-то закралась ошибка! 📅\n"+
				"Я не смог разобрать дату. Попробуй формат: дд.мм.гггг (например, 11.02.2024).\n"+
				"Давай ещё раз, я верю в тебя! 💪",
		)
		return
	}

	if date.Before(time.Now()) {
		vb.sendMessage(
			chatId,
			"Ой, похоже вы указали дату из прошлого! 😅\n"+
				"Мы, конечно, не Док и Марти, чтобы отправляться в прошлое на DeLorean.\n"+
				"Попробуйте выбрать что-то из будущего — ведь только вперёд, к новым приключениям! 🚀⏳",
		)
	}

	cd.SetDeadline(date)
	vb.setChatData(chatId, cd)

	vb.sendMessage(
		chatId,
		fmt.Sprintf(
			"🌟 Ура! Дедлайн установлен! 🌟\n\n"+
				"Вы выбрали дату %s для завершения чтения вашей книги. 🕒✨\n"+
				"Не забывайте, что мы всегда можем изменить его, если ваши планы изменятся!\n\n"+
				"Давайте сделаем это чтение увлекательным приключением, а не гонкой! 📚💨",
			date.Format(DATE_LAYOUT),
		),
	)
}

func (vb *LitNightBot) handleCurrentSet(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.sendMessage(chatId, "Извиняюсь, но функционал пока в разработке. Stay tuned как грится")
	// bookname := message.CommandArguments()

	// if bookname == "" {
	// 	vb.sendMessage(chatId, "/current_set <bookname>")
	// 	return
	// }

	// cd := vb.getChatData(chatId)

	// if cd.Current.Book.Name != "" {
	// 	vb.sendMessage(chatId,
	// 		fmt.Sprintf("О, кажется, вы уже читаете \"%s\"! 📖\n"+
	// 			"Может, сначала завершим эту книгу, прежде чем начать новое приключение? 😉",
	// 			cd.Current.Book.Name,
	// 		))
	// 	return
	// }

	// book, err := cd.RemoveBookFromWishlist(bookname)
	// cd.SetCurrentBook(bookname)
	// vb.setChatData(chatId, cd)

	// if err != nil && len(cd.Wishlist) > 0 {
	// 	vb.sendMessage(
	// 		chatId,
	// 		"Кажется, выбранная вами книга не из вашего вишлиста. 📚\n"+
	// 			"Может, в следующий раз стоит выбрать что-то из списка желаемого чтения? 😄",
	// 	)
	// 	return
	// }

	// vb.sendMessage(
	// 	chatId,
	// 	fmt.Sprintf(
	// 		"Отличный выбор! Теперь ваша новая книга для чтения — \"%s\". 📚✨\n"+
	// 			"Удачного чтения, и не забудьте вернуться для обсуждения! 😉",
	// 		bookname,
	// 	),
	// )
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
		randomBook := cd.Wishlist[randomIndex].Book
		cd.SetCurrentBook(randomBook)
		cd.RemoveBookFromWishlist(randomBook.UUID)

		vb.setChatData(chatId, cd)

		vb.sendMessage(chatId,
			fmt.Sprintf(
				"Тадааам! Вот ваша книга: \"%s\". Приятного чтения! 📚\n\n"+
					"И вот вам приятный бонус: я назначил автоматический дедлайн через 2 недели - %s!\n"+
					"Если хотите изменить его, просто используйте команду установки дедлайна.\n\n"+
					"Давайте сделаем так, чтобы время не ускользнуло, как в \"Докторе Кто\" — не забывайте о своих путешествиях во времени! 🕰️",
				randomBook.Name, cd.Current.Deadline.Format(DATE_LAYOUT),
			),
		)
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

func (vb *LitNightBot) removeBookFromHistory(chatId int64, uuid string) {
	cd := vb.getChatData(chatId)
	book, err := cd.RemoveBookFromHistory(uuid)
	vb.setChatData(chatId, cd)

	if err != nil {
		vb.sendMessage(chatId, err.Error())
		return
	}

	vb.sendMessage(chatId, fmt.Sprintf("Книга \"%s\" удалена из списка", book.Name))
}

func getButtonsForBooklist[T HasBook](
	booklist *[]T,
	prefix string,
	cbParamsGetter func(uuid string) string,
) [][]tgbotapi.InlineKeyboardButton {
	var buttons [][]tgbotapi.InlineKeyboardButton

	if len(*booklist) == 0 {
		return buttons
	}

	for _, item := range *booklist {
		button := tgbotapi.NewInlineKeyboardButtonData(
			prefix+" "+item.GetBook().Name,
			cbParamsGetter(item.GetBook().UUID),
		)

		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	button := tgbotapi.NewInlineKeyboardButtonData(
		"Отмена",
		GetCallbackParamStr(CBCancel, "_"),
	)

	inlineRow := tgbotapi.NewInlineKeyboardRow(button)

	return append(buttons, inlineRow)
}

func getCleanWishlistMessage(cd *ChatData) (string, [][]tgbotapi.InlineKeyboardButton) {
	buttons := getButtonsForBooklist(&cd.Wishlist, "❌", func(uuid string) string { return GetCallbackParamStr(CBRemove, uuid) })

	if len(cd.Wishlist) == 0 {
		return "Список книг пуст, удалять нечего", buttons
	}

	return "Вот ваши книги в списке:", buttons
}

const BooksPerPage = 5

func (vb *LitNightBot) handleRemoveFromWishlist(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.showRemoveWishlistPage(chatId, -1, 0)
}

func (vb *LitNightBot) showRemoveWishlistPage(chatId int64, messageID int, page int) {
	cd := vb.getChatData(chatId)
	books := cd.GetWishlistBooks()

	if len(books) == 0 {
		vb.sendMessage(chatId, "Ваш вишлист пуст, нечего удалять. Добавьте новые книги для удаления.")
		return
	}

	start := page * BooksPerPage
	end := start + BooksPerPage
	if end > len(books) {
		end = len(books)
	}

	booksOnPage := books[start:end]
	messageText := fmt.Sprintf("🗑️ Удаление книг (страница %d):\n\n", page+1)
	var buttons [][]tgbotapi.InlineKeyboardButton
	for i, book := range booksOnPage {
		button := tgbotapi.NewInlineKeyboardButtonData(
			book,
			GetCallbackParamStr(CBRemove, strconv.Itoa(start+i)),
		)
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(button))
	}

	var navButtons []tgbotapi.InlineKeyboardButton
	if start > 0 {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("⬅ Назад", GetCallbackParamStr(CBRemovePage, strconv.Itoa(page-1))))
	}
	if end < len(books) {
		navButtons = append(navButtons, tgbotapi.NewInlineKeyboardButtonData("Вперед ➡", GetCallbackParamStr(CBRemovePage, strconv.Itoa(page+1))))
	}

	if len(navButtons) > 0 {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardRow(navButtons...))
	}

	if messageID == -1 {
		msg := tgbotapi.NewMessage(chatId, messageText)
		if len(buttons) > 0 {
			msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
		}

		vb.bot.Send(msg)
	} else {
		vb.editMessage(chatId, messageID, messageText, buttons)
	}
}

func (vb *LitNightBot) handleCurrentAbort(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	currentBook := cd.Current.Book

	if currentBook.Name == "" {
		vb.sendMessage(
			chatId,
			"🚫 Ой-ой! Похоже, у вас нет текущей выбранной книги.\nКак насчет того, чтобы выбрать новую историю? 📚✨",
		)
		return
	}

	msg := tgbotapi.NewMessage(chatId, fmt.Sprintf("🤔 Что делать с отменяемой книгой \"%s\"?\nДавайте решим это вместе! 🎉", currentBook.Name))

	buttons := []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			"❌ Никогда",
			GetCallbackParamStr(CBCurrentToHistory, currentBook.UUID),
		),
		tgbotapi.NewInlineKeyboardButtonData(
			"🕑 Потом",
			GetCallbackParamStr(CBCurrentToWishlist, currentBook.UUID),
		),
		tgbotapi.NewInlineKeyboardButtonData(
			"Отмена",
			GetCallbackParamStr(CBCancel, "_"),
		),
	}

	inlineRow := tgbotapi.NewInlineKeyboardRow(buttons...)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineRow)
	msg.ReplyMarkup = keyboard

	vb.bot.Send(msg)
}

func (vb *LitNightBot) Init() {
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
			Command:     "current_deadline",
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
		vb.handleWishlist(update.Message)
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
		vb.sendMessage(update.Message.Chat.ID, "Unknown command")
	}
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
