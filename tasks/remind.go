package tasks

import (
	"lit-night-bot/bot"
	"lit-night-bot/io"
	"math/rand"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var OneDayReminderJokes = []string{
	"Внимание! Завтра дедлайн — до окончания осталось 1 день! Книга уже греет своё место на вашем столе.",
	"Остался всего 1 день до дедлайна. Кажется, время вспомнить, как переворачивать страницы!",
	"Дедлайн завтра, осталось 1 день. Это ваш шанс стать чемпионом чтения!",
	"Один день до дедлайна — вы на финишной прямой, осталось всего 24 часа.",
	"Завтра дедлайн! До окончания остался 1 день. Книга надеется на вас!",
	"Время тик-так: осталось 1 день до дедлайна. Книга уже танцует от нетерпения!",
	"Остался 1 день! Завтра дедлайн, а вы ещё можете всё исправить.",
	"Завтра дедлайн! До окончания остался всего 1 день. Вы в игре!",
	"Книга напоминает: дедлайн завтра, остался 1 день. Поднимайте настрой и начинайте!",
	"1 день до дедлайна. Это ваш последний шанс стать другом этой книги!",
}

var OneWeekReminderJokes = []string{
	"До дедлайна осталась неделя — 7 дней. Это достаточно, чтобы дочитать... если начать прямо сейчас!",
	"Книга сообщает: осталось 7 дней до дедлайна. Не теряйте времени зря!",
	"Семь дней до дедлайна! Книга шепчет: «Я всё ещё тут, на полке!»",
	"Неделя до дедлайна — осталось 7 дней! Начните сегодня, и вы победите.",
	"До окончания осталась ровно неделя — 7 дней! Книга надеется, что вы её не забыли.",
	"Неделя до дедлайна. Осталось 7 дней — отличный срок, чтобы прочитать пару глав.",
	"7 дней до дедлайна! Это целая вечность или момент — решать вам.",
	"Осталась неделя до дедлайна. Кажется, книга грустит, если вы ещё не начали.",
	"Семь дней до дедлайна! До окончания осталось всего 7 дней — ещё есть шанс!",
	"До дедлайна осталось 7 дней. Время достать книгу, пока неделя не пролетела!",
}

func Remind(spec string, texts []string, days int) *Task {
	return &Task{
		CB: func(logger *logrus.Entry, iocd *io.IoChatData, lnb *bot.LitNightBot) {
			files, err := iocd.GetDatasList()
			if err != nil {
				logger.WithError(err).Error("Failed to list files")
				return
			}

			deadlineTarget := time.Now().AddDate(0, 0, days).Truncate(24 * time.Hour)

			for _, file := range files {
				chatId, err := strconv.ParseInt(file, 10, 64)
				if err != nil {
					logger.WithField("file", file).WithError(err).Warn("Failed to parse file name as chat ID")
					continue
				}

				chatData := iocd.GetChatData(chatId)
				if chatData == nil {
					logger.WithField("file", file).Warn("Failed to get chat data")
					continue
				}

				if chatData.Current.Deadline.Truncate(24 * time.Hour).Equal(deadlineTarget) {
					randomIndex := rand.Intn(len(texts))
					randomMessage := texts[randomIndex]

					logger.WithField("file", file).Infof(
						"Remind to chat %s about deadline in %d days with book \"%s\"",
						file, days, chatData.Current.Book.Name,
					)
					lnb.SendPlainMessage(chatId, randomMessage)
				}
			}
		},
		Spec: spec,
	}
}
