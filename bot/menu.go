package bot

import (
	chatdata "lit-night-bot/chat-data"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getMenuButton(text string, action CallbackAction) []tgbotapi.InlineKeyboardButton {
	return []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(
			text,
			GetCallbackParamStr(action),
		),
	}
}

func getCurrentBookMenu(cd *chatdata.ChatData) [][]tgbotapi.InlineKeyboardButton {
	if cd.Current.Book.UUID != "" {
		return []([]tgbotapi.InlineKeyboardButton){
			getMenuButton("📘 Текущая книга", CBCurrentShow),
			getMenuButton("📅 Изменить дедлайн", CBCurrentChangeDeadlineRequest),
			getMenuButton("✅ Завершить книгу", CBCurrentComplete),
			getMenuButton("❌ Отменить книгу", CBCurrentAbort),
		}
	}

	return []([]tgbotapi.InlineKeyboardButton){
		getMenuButton("🎲 Случайная книга", CBCurrentRandom),
	}
}

func getWishlistMenu() [][]tgbotapi.InlineKeyboardButton {
	return []([]tgbotapi.InlineKeyboardButton){
		getMenuButton("📚 Показать вишлист", CBWishlistShow),
		getMenuButton("➕ Добавить в вишлист", CBWishlistAddBookRequest),
		getMenuButton("🧹 Чистка вишлиста", CBWishlistClean),
	}
}

func getHistoryMenu() [][]tgbotapi.InlineKeyboardButton {
	return []([]tgbotapi.InlineKeyboardButton){
		getMenuButton("🕰️ Показать историю", CBHistoryShow),
		getMenuButton("🧹 Чистка истории", CBHistoryShow),
	}
}

func (vb *LitNightBot) handleMenu(message *tgbotapi.Message) {
	chatId := message.Chat.ID
	cd := vb.getChatData(chatId)

	var buttons [][]tgbotapi.InlineKeyboardButton

	buttons = append(buttons, getCurrentBookMenu(cd)...)
	buttons = append(buttons, getWishlistMenu()...)
	buttons = append(buttons, getHistoryMenu()...)
	buttons = append(buttons, getMenuButton("❎ Закрыть меню", CBMenuClose))

	vb.sendMessage(chatId, menuText, buttons)
}
