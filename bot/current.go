package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"lit-night-bot/utils"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/rand"
)

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

	vb.sendMessage(chatId, msg, nil)
}

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
			nil,
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
			nil,
		)
		return
	}

	if date.Before(time.Now()) {
		vb.sendMessage(
			chatId,
			"Ой, похоже вы указали дату из прошлого! 😅\n"+
				"Мы, конечно, не Док и Марти, чтобы отправляться в прошлое на DeLorean.\n"+
				"Попробуйте выбрать что-то из будущего — ведь только вперёд, к новым приключениям! 🚀⏳",
			nil,
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
		nil,
	)
}

func (vb *LitNightBot) handleCurrentSet(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	vb.sendMessage(chatId, "Извиняюсь, но функционал пока в разработке. Stay tuned как грится", nil)
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
			nil,
		)
		return
	}

	cd.AddBookToHistory(currentBook)
	cd.Current = chatdata.CurrentBook{}

	vb.setChatData(chatId, cd)

	vb.sendMessage(
		chatId,
		fmt.Sprintf(
			"Ура! Книга \"%s\" прочитана! 🎉\n"+
				"Надеюсь, она оставила вам море впечатлений.\n"+
				"Готовы к следующему литературному приключению?",
			currentBook,
		),
		nil,
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
			nil,
		)
		return
	}

	if len(cd.Wishlist) == 0 {
		vb.sendMessage(chatId, "Ваш вишлист пуст! Добавьте книги, чтобы я мог выбрать одну для вас.", nil)
		return
	}

	go func() {
		vb.sendProgressJokes(chatId)

		randomIndex := rand.Intn(len(cd.Wishlist))
		randomBook := cd.Wishlist[randomIndex].Book
		cd.SetCurrentBook(randomBook)
		cd.RemoveBookFromWishlist(randomBook.UUID)

		vb.setChatData(chatId, cd)

		vb.sendMessage(
			chatId,
			fmt.Sprintf(
				"Тадааам! Вот ваша книга: \"%s\". Приятного чтения! 📚\n\n"+
					"И вот вам приятный бонус: я назначил автоматический дедлайн через 2 недели - %s!\n"+
					"Если хотите изменить его, просто используйте команду установки дедлайна.\n\n"+
					"Давайте сделаем так, чтобы время не ускользнуло, как в \"Докторе Кто\" — не забывайте о своих путешествиях во времени! 🕰️",
				randomBook.Name, cd.Current.Deadline.Format(DATE_LAYOUT),
			),
			nil,
		)
	}()
}

func (vb *LitNightBot) handleCurrentAbort(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	currentBook := cd.Current.Book

	if currentBook.Name == "" {
		vb.sendMessage(
			chatId,
			"🚫 Ой-ой! Похоже, у вас нет текущей выбранной книги.\nКак насчет того, чтобы выбрать новую историю? 📚✨",
			nil,
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
			GetCallbackParamStr(CBCancel),
		),
	}

	inlineRow := tgbotapi.NewInlineKeyboardRow(buttons...)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineRow)
	msg.ReplyMarkup = keyboard

	vb.bot.Send(msg)
}

func (vb *LitNightBot) moveCurrentBook(chatId int64, messageID int, moveToHistory bool) {
	cd := vb.getChatData(chatId)
	currentBookName := cd.Current.Book.Name
	if moveToHistory {
		cd.AddBookToHistory(currentBookName)
	} else {
		cd.AddBookToWishlist(currentBookName)
	}
	cd.Current = chatdata.CurrentBook{}
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

func (vb *LitNightBot) handleAdd(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.CommandArguments(), "\n"))

	if len(booknames) == 0 {
		vb.sendMessage(
			chatId,
			"Эй, книжный искатель! "+
				"📚✨ Чтобы добавить новую книгу в ваш вишлист, просто укажите её название в команде add, например:\n/add Моя первая книга",
			nil,
		)
		return
	}

	cd := vb.getChatData(chatId)

	cd.AddBooksToWishlist(booknames)

	vb.setChatData(chatId, cd)

	var textMessage string
	if len(booknames) == 1 {
		textMessage = fmt.Sprintf("Книга \"%s\" добавлена.", booknames[0])
	} else {
		textMessage = fmt.Sprintf("Книги \"%s\" добавлены.", strings.Join(booknames, "\", \""))
	}

	vb.sendMessage(chatId, textMessage, nil)
}
