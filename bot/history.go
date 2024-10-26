package bot

import (
	"fmt"
	"lit-night-bot/utils"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (vb *LitNightBot) handleAddHistory(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	booknames := utils.CleanStrSlice(strings.Split(message.CommandArguments(), "\n"))

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
