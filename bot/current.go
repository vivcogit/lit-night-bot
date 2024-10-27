package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"golang.org/x/exp/rand"
)

func (lnb *LitNightBot) handleCurrent(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

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

	lnb.sendPlainMessage(chatId, msg)
}

func (lnb *LitNightBot) handleCurrentDeadlineNoBook(chatId int64) {
	lnb.sendPlainMessage(
		chatId,
		"Хей-хей! 🚀\n"+
			"Похоже, мы находимся в параллельной вселенной!\n"+
			"Устанавливать дедлайн без выбранной книги — это как пытаться запустить ракету без топлива. 🚀💨\n"+
			"Давайте сначала выберем книгу, а потом уже обсудим, когда будем её читать! Так мы точно не улетим в никуда! 📖✨",
	)
}

func (lnb *LitNightBot) handleCurrentDeadlineRequest(message *tgbotapi.Message) {
	chatId := message.Chat.ID

	cd := lnb.getChatData(chatId)
	if cd.Current.Book.UUID == "" {
		lnb.handleCurrentDeadlineNoBook(chatId)
		return
	}

	lnb.sendPlainMessage(chatId, setDeadlineRequestMessage)
}

func (lnb *LitNightBot) handleCurrentDeadline(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	if cd.Current.Book.UUID == "" {
		lnb.handleCurrentDeadlineNoBook(chatId)
		return
	}

	date, err := time.Parse(DATE_LAYOUT, message.Text)

	if err != nil {
		lnb.sendPlainMessage(
			chatId,
			"Ой-ой, кажется, где-то закралась ошибка! 📅\n"+
				"Я не смог разобрать дату. Попробуй формат: дд.мм.гггг (например, 11.02.2024).\n"+
				"Давай ещё раз, я верю в тебя! 💪",
		)
		return
	}

	if date.Before(time.Now()) {
		lnb.sendPlainMessage(
			chatId,
			"Ой, похоже вы указали дату из прошлого! 😅\n"+
				"Мы, конечно, не Док и Марти, чтобы отправляться в прошлое на DeLorean.\n"+
				"Попробуйте выбрать что-то из будущего — ведь только вперёд, к новым приключениям! 🚀⏳",
		)
	}

	cd.SetDeadline(date)
	lnb.setChatData(chatId, cd)

	lnb.sendPlainMessage(
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

func (lnb *LitNightBot) handleCurrentComplete(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	currentBook := cd.Current.Book.Name
	if currentBook == "" {
		lnb.sendPlainMessage(
			chatId,
			"Хмм... Похоже, у вас ещё нет книги в процессе чтения.\n"+
				"Давайте выберем что-нибудь интересное и погрузимся в новые страницы! 📚✨",
		)
		return
	}

	cd.AddBookToHistory(currentBook)
	cd.Current = chatdata.CurrentBook{}

	lnb.setChatData(chatId, cd)

	lnb.sendPlainMessage(
		chatId,
		fmt.Sprintf(
			"Ура! Книга \"%s\" прочитана! 🎉\n"+
				"Надеюсь, она оставила вам море впечатлений.\n"+
				"Готовы к следующему литературному приключению?",
			currentBook,
		),
	)
}

func (lnb *LitNightBot) handleCurrentRandom(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	if cd.Current.Book.Name != "" {
		lnb.sendPlainMessage(
			chatId,
			fmt.Sprintf("Вы уже читаете \"%s\"\n"+
				"Эта книга не заслуживает такого обращения!\n"+
				"Но если вы хотите новую, давайте найдем ее вместе!\n"+
				"Но сначала скажите ей об отмене",
				cd.Current.Book.Name,
			),
		)
		return
	}

	if len(cd.Wishlist) == 0 {
		lnb.sendPlainMessage(
			chatId,
			"Ваш вишлист пуст! Добавьте книги, чтобы я мог выбрать одну для вас.",
		)
		return
	}

	go func() {
		lnb.sendProgressJokes(chatId)

		randomIndex := rand.Intn(len(cd.Wishlist))
		randomBook := cd.Wishlist[randomIndex].Book
		cd.SetCurrentBook(randomBook)
		cd.RemoveBookFromWishlist(randomBook.UUID)

		lnb.setChatData(chatId, cd)

		lnb.sendPlainMessage(
			chatId,
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

func (lnb *LitNightBot) handleCurrentAbort(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := lnb.getChatData(chatId)

	currentBook := cd.Current.Book

	if currentBook.Name == "" {
		lnb.sendPlainMessage(
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
			GetCallbackParamStr(CBCancel),
		),
	}

	inlineRow := tgbotapi.NewInlineKeyboardRow(buttons...)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(inlineRow)
	msg.ReplyMarkup = keyboard

	lnb.bot.Send(msg)
}

func (lnb *LitNightBot) moveCurrentBook(chatId int64, messageID int, moveToHistory bool) {
	cd := lnb.getChatData(chatId)
	currentBookName := cd.Current.Book.Name
	if moveToHistory {
		cd.AddBookToHistory(currentBookName)
	} else {
		cd.AddBookToWishlist(currentBookName)
	}
	cd.Current = chatdata.CurrentBook{}
	lnb.setChatData(chatId, cd)

	if moveToHistory {
		lnb.editMessage(
			chatId,
			messageID,
			fmt.Sprintf(
				"📖 Книга \"%s\" теперь в истории!\nВремя выбрать новую приключенческую историю! 🚀",
				currentBookName,
			),
			nil,
		)
	} else {
		lnb.editMessage(
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
