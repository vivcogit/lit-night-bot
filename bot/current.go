package bot

import (
	"fmt"
	chatdata "lit-night-bot/chat-data"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/rand"
)

func (lnb *LitNightBot) handleCurrent(update *tgbotapi.Update, logger *logrus.Entry) {
	logger.Info("Handling current book display")
	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	var msg string

	if cd.Current.Book.Name == "" {
		msg = "Похоже, у вас пока нет выбранной книги. Как насчёт выбрать что-нибудь интересное для чтения?"
		logger.Info("No current book selected")
	} else {
		msg = fmt.Sprintf(
			"В данный момент вы читаете книгу \"%s\" 📖\n"+
				"Как вам она? Делитесь впечатлениями! 😊\n"+
				"Кстати, у вас назначен дедлайн на %s.\n"+
				"Надеюсь, до этого времени вы не потеряетесь в мире страниц! 📅😈",
			cd.Current.Book.Name, cd.Current.Deadline.Format(DATE_LAYOUT))
		logger.WithField("current_book", cd.Current.Book.Name).Info("Current book displayed")
	}

	lnb.sendPlainMessage(chatID, msg)
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

func (lnb *LitNightBot) handleCurrentDeadlineRequest(update *tgbotapi.Update, logger *logrus.Entry) {
	logger.Info("Requesting deadline change for current book")
	chatID := getUpdateChatID(update)

	cd := lnb.getChatData(chatID)
	if cd.Current.Book.UUID == "" {
		lnb.handleCurrentDeadlineNoBook(chatID)
		return
	}

	lnb.sendPlainMessage(chatID, setDeadlineRequestMessage)
	logger.WithField("current_book", cd.Current.Book.Name).Info("Deadline request for current book")
}

func (lnb *LitNightBot) handleCurrentDeadline(update *tgbotapi.Update, logger *logrus.Entry) {
	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	if cd.Current.Book.UUID == "" {
		lnb.handleCurrentDeadlineNoBook(chatID)
		return
	}

	date, err := time.Parse(DATE_LAYOUT, update.Message.Text)

	if err != nil {
		lnb.sendPlainMessage(
			chatID,
			"Ой-ой, кажется, где-то закралась ошибка! 📅\n"+
				"Я не смог разобрать дату. Попробуй формат: дд.мм.гггг (например, 11.02.2024).\n"+
				"Давай ещё раз, я верю в тебя! 💪",
		)
		logger.WithField("input_date", update.Message.Text).Error("Failed to parse date for deadline")
		return
	}

	if date.Before(time.Now()) {
		lnb.sendPlainMessage(
			chatID,
			"Ой, похоже вы указали дату из прошлого! 😅\n"+
				"Мы, конечно, не Док и Марти, чтобы отправляться в прошлое на DeLorean.\n"+
				"Попробуйте выбрать что-то из будущего — ведь только вперёд, к новым приключениям! 🚀⏳",
		)
		logger.WithField("input_date", date).Warn("Date set in the past")
		return
	}

	cd.SetDeadline(date)
	lnb.setChatData(chatID, cd)

	lnb.sendPlainMessage(
		chatID,
		fmt.Sprintf(
			"🌟 Ура! Дедлайн установлен! 🌟\n\n"+
				"Вы выбрали дату %s для завершения чтения вашей книги. 🕒✨\n"+
				"Не забывайте, что мы всегда можем изменить его, если ваши планы изменятся!\n\n"+
				"Давайте сделаем это чтение увлекательным приключением, а не гонкой! 📚💨",
			date.Format(DATE_LAYOUT),
		),
	)
	logger.WithField("deadline_date", date).Info("Deadline set for current book")
}

func (lnb *LitNightBot) handleCurrentComplete(update *tgbotapi.Update, logger *logrus.Entry) {
	logger.Info("Marking current book as complete")
	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	currentBook := cd.Current.Book.Name
	if currentBook == "" {
		lnb.sendPlainMessage(
			chatID,
			"Хмм... Похоже, у вас ещё нет книги в процессе чтения.\n"+
				"Давайте выберем что-нибудь интересное и погрузимся в новые страницы! 📚✨",
		)
		logger.Info("No current book to complete")
		return
	}

	cd.AddBookToHistory(currentBook)
	cd.Current = chatdata.CurrentBook{}

	lnb.setChatData(chatID, cd)

	lnb.sendPlainMessage(
		chatID,
		fmt.Sprintf(
			"Ура! Книга \"%s\" прочитана! 🎉\n"+
				"Надеюсь, она оставила вам море впечатлений.\n"+
				"Готовы к следующему литературному приключению?",
			currentBook,
		),
	)
	logger.WithField("completed_book", currentBook).Info("Current book marked as completed")
}

func (lnb *LitNightBot) handleCurrentRandom(update *tgbotapi.Update, logger *logrus.Entry) {
	logger.Info("Handling random book selection")
	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	msg := lnb.checkCanChooseBook(cd)

	if msg != "" {
		lnb.sendPlainMessage(chatID, msg)
		return
	}

	lnb.sendProgressJokes(chatID)

	randomIndex := rand.Intn(len(cd.Wishlist))
	randomBook := cd.Wishlist[randomIndex].Book

	lnb.handleCurrentSet(update, cd, randomBook)

	logger.WithField("random_book", randomBook.Name).Info("Random book selected from wishlist")
}

func (lnb *LitNightBot) handleCurrentChoose(update *tgbotapi.Update, uuid string, logger *logrus.Entry) {
	logger.Info("Handling manual book chosen")

	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	book := FindBookByUUID(&cd.Wishlist, uuid)

	if book == nil {
		lnb.sendPlainMessage(chatID, "Что-то пошло не так и я не могу найти книгу")
		return
	}

	lnb.handleCurrentSet(update, cd, book.GetBook())
	lnb.removeMessage(chatID, update.CallbackQuery.Message.MessageID)
}

func (lnb *LitNightBot) checkCanChooseBook(cd *chatdata.ChatData) string {
	if cd.Current.Book.Name != "" {
		return fmt.Sprintf("Вы уже читаете \"%s\"\n"+
			"Эта книга не заслуживает такого обращения!\n"+
			"Но если вы хотите новую, давайте найдем ее вместе!\n"+
			"Но сначала скажите ей об отмене",
			cd.Current.Book.Name,
		)
	}

	if len(cd.Wishlist) == 0 {
		return "Ваш вишлист пуст! Добавьте книги, чтобы я мог выбрать одну для вас."
	}

	return ""
}

func (lnb *LitNightBot) handleCurrentSet(update *tgbotapi.Update, cd *chatdata.ChatData, book chatdata.Book) {
	chatID := getUpdateChatID(update)

	cd.SetCurrentBook(book)
	cd.RemoveBookFromWishlist(book.UUID)

	lnb.setChatData(chatID, cd)

	lnb.sendPlainMessage(
		chatID,
		fmt.Sprintf(
			"Тадааам! Вот ваша книга: \"%s\". Приятного чтения! 📚\n\n"+
				"И вот вам приятный бонус: я назначил автоматический дедлайн через 2 недели - %s!\n"+
				"Если хотите изменить его, просто воспользуйтесь кнопкой в меню.\n\n"+
				"Давайте сделаем так, чтобы время не ускользнуло, как в \"Докторе Кто\" — не забывайте о своих путешествиях во времени! 🕰️",
			book.Name, cd.Current.Deadline.Format(DATE_LAYOUT),
		),
	)
}

func (lnb *LitNightBot) handleCurrentAbort(update *tgbotapi.Update, logger *logrus.Entry) {
	logger.Info("Aborting current book")
	chatID := getUpdateChatID(update)
	cd := lnb.getChatData(chatID)

	currentBook := cd.Current.Book

	if currentBook.Name == "" {
		lnb.sendPlainMessage(
			chatID,
			"🚫 Ой-ой! Похоже, у вас нет текущей выбранной книги.\nКак насчет того, чтобы выбрать новую историю? 📚✨",
		)
		logger.Info("No current book to abort")
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("🤔 Что делать с отменяемой книгой \"%s\"?\nДавайте решим это вместе! 🎉", currentBook.Name))

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
	logger.WithField("current_book", currentBook.Name).Info("Abort request for current book")
}

func (lnb *LitNightBot) moveCurrentBook(chatId int64, messageID int, moveToHistory bool, logger *logrus.Entry) {
	logger.Info("Moving current book to history")
	cd := lnb.getChatData(chatId)
	currentBookName := cd.Current.Book.Name
	if moveToHistory {
		cd.AddBookToHistory(currentBookName)
		logger.WithField("current_book", currentBookName).Info("Book moved to history")
	} else {
		cd.AddBookToWishlist(currentBookName)
		logger.WithField("current_book", currentBookName).Info("Book moved to wishlist")
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
