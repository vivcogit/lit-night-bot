package bot

import (
	"fmt"
	"html"
	chatdata "lit-night-bot/chat-data"
	"math"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type readingStatistics struct {
	CompletedCount       int
	UnfinishedCount      int
	BooksPerMonth        float64
	AverageReadingDays   float64
	ReadingDurationCount int
}

type ratedBookStatistic struct {
	Book        chatdata.ClubBook
	Average     float64
	RatingCount int
	Spread      int
}

type clubRatingStatistics struct {
	Average       float64
	RatingCount   int
	Highest       *ratedBookStatistic
	Lowest        *ratedBookStatistic
	Controversial *ratedBookStatistic
}

type personalRatingStatistics struct {
	Highest *ratedBookStatistic
	Lowest  *ratedBookStatistic
}

func monthsInclusive(first time.Time, last time.Time) int {
	if first.IsZero() || last.IsZero() {
		return 0
	}
	months := (last.Year()-first.Year())*12 + int(last.Month()-first.Month()) + 1
	if months < 1 {
		return 1
	}
	return months
}

func calculateReadingStatistics(data *chatdata.ChatData, now time.Time) readingStatistics {
	result := readingStatistics{}
	var firstCompletion time.Time
	totalReadingDays := 0.0
	for _, book := range data.Books {
		switch book.Status {
		case chatdata.StatusCompleted:
			result.CompletedCount++
			completedAt := book.AddedAt
			if book.CompletedAt != nil {
				completedAt = *book.CompletedAt
			}
			if !completedAt.IsZero() && (firstCompletion.IsZero() || completedAt.Before(firstCompletion)) {
				firstCompletion = completedAt
			}
			if book.StartedAt != nil && book.CompletedAt != nil && !book.CompletedAt.Before(*book.StartedAt) {
				totalReadingDays += book.CompletedAt.Sub(*book.StartedAt).Hours() / 24
				result.ReadingDurationCount++
			}
		case chatdata.StatusUnfinished:
			result.UnfinishedCount++
		}
	}
	if months := monthsInclusive(firstCompletion, now); months > 0 {
		result.BooksPerMonth = float64(result.CompletedCount) / float64(months)
	}
	if result.ReadingDurationCount > 0 {
		result.AverageReadingDays = totalReadingDays / float64(result.ReadingDurationCount)
	}
	return result
}

func ratedBook(book chatdata.ClubBook) *ratedBookStatistic {
	if book.Status != chatdata.StatusCompleted || len(book.Ratings) == 0 {
		return nil
	}
	minimum, maximum, total := 10, 1, 0
	for _, rating := range book.Ratings {
		total += rating.Value
		if rating.Value < minimum {
			minimum = rating.Value
		}
		if rating.Value > maximum {
			maximum = rating.Value
		}
	}
	return &ratedBookStatistic{
		Book:        book,
		Average:     float64(total) / float64(len(book.Ratings)),
		RatingCount: len(book.Ratings),
		Spread:      maximum - minimum,
	}
}

func preferRatedBook(candidate *ratedBookStatistic, current *ratedBookStatistic, higher bool) bool {
	if current == nil {
		return true
	}
	if math.Abs(candidate.Average-current.Average) > 0.000001 {
		if higher {
			return candidate.Average > current.Average
		}
		return candidate.Average < current.Average
	}
	if candidate.RatingCount != current.RatingCount {
		return candidate.RatingCount > current.RatingCount
	}
	return strings.ToLower(candidate.Book.DisplayName()) < strings.ToLower(current.Book.DisplayName())
}

func calculateClubRatingStatistics(data *chatdata.ChatData) clubRatingStatistics {
	result := clubRatingStatistics{}
	total := 0
	for _, book := range data.Books {
		candidate := ratedBook(book)
		if candidate == nil {
			continue
		}
		for _, rating := range book.Ratings {
			total += rating.Value
			result.RatingCount++
		}
		if preferRatedBook(candidate, result.Highest, true) {
			copy := *candidate
			result.Highest = &copy
		}
		if preferRatedBook(candidate, result.Lowest, false) {
			copy := *candidate
			result.Lowest = &copy
		}
		if candidate.RatingCount >= 2 && (result.Controversial == nil || candidate.Spread > result.Controversial.Spread ||
			(candidate.Spread == result.Controversial.Spread && (candidate.RatingCount > result.Controversial.RatingCount ||
				(candidate.RatingCount == result.Controversial.RatingCount && strings.ToLower(candidate.Book.DisplayName()) < strings.ToLower(result.Controversial.Book.DisplayName()))))) {
			copy := *candidate
			result.Controversial = &copy
		}
	}
	if result.RatingCount > 0 {
		result.Average = float64(total) / float64(result.RatingCount)
	}
	return result
}

func calculatePersonalRatingStatistics(data *chatdata.ChatData, userID int64) personalRatingStatistics {
	result := personalRatingStatistics{}
	for _, book := range data.Books {
		if book.Status != chatdata.StatusCompleted {
			continue
		}
		rating := book.RatingByUser(userID)
		if rating == nil {
			continue
		}
		candidate := &ratedBookStatistic{Book: book, Average: float64(rating.Value), RatingCount: 1}
		if preferRatedBook(candidate, result.Highest, true) {
			copy := *candidate
			result.Highest = &copy
		}
		if preferRatedBook(candidate, result.Lowest, false) {
			copy := *candidate
			result.Lowest = &copy
		}
	}
	return result
}

func formatStatisticNumber(value float64) string {
	return strings.Replace(fmt.Sprintf("%.1f", value), ".", ",", 1)
}

func formatReadingStatistics(stats readingStatistics) string {
	var text strings.Builder
	text.WriteString(fmt.Sprintf("✅ Прочитано: <b>%d</b>\n", stats.CompletedCount))
	text.WriteString(fmt.Sprintf("🚫 Не дочитано: <b>%d</b>\n", stats.UnfinishedCount))
	text.WriteString(fmt.Sprintf("📅 В среднем: <b>%s книги в месяц</b>\n", formatStatisticNumber(stats.BooksPerMonth)))
	if stats.ReadingDurationCount == 0 {
		text.WriteString("⏱ Среднее время чтения: <i>нет данных</i>")
	} else {
		text.WriteString(fmt.Sprintf("⏱ Среднее время чтения: <b>%s дня</b>", formatStatisticNumber(stats.AverageReadingDays)))
	}
	return text.String()
}

func formatRatedBookStat(stat *ratedBookStatistic) string {
	if stat == nil {
		return "<i>нет данных</i>"
	}
	return fmt.Sprintf("«%s» — <b>%s</b>", html.EscapeString(stat.Book.DisplayName()), formatStatisticNumber(stat.Average))
}

func renderClubStatistics(data *chatdata.ChatData, now time.Time) string {
	reading := calculateReadingStatistics(data, now)
	ratings := calculateClubRatingStatistics(data)
	var text strings.Builder
	text.WriteString("📊 <b>СТАТИСТИКА КНИЖНОГО КЛУБА</b>\n\n")
	text.WriteString("📚 <b>Чтение</b>\n")
	text.WriteString(formatReadingStatistics(reading))
	text.WriteString("\n\n⭐ <b>Оценки</b>\n")
	if ratings.RatingCount == 0 {
		text.WriteString("Средняя оценка клуба: <i>нет данных</i>\n")
	} else {
		text.WriteString(fmt.Sprintf("Средняя оценка клуба: <b>%s из 10</b>\n", formatStatisticNumber(ratings.Average)))
	}
	text.WriteString("🏆 Самая высокая: " + formatRatedBookStat(ratings.Highest) + "\n")
	text.WriteString("📉 Самая низкая: " + formatRatedBookStat(ratings.Lowest) + "\n")
	if ratings.Controversial == nil {
		text.WriteString("🎭 Самая спорная: <i>нет данных</i>\n")
	} else {
		text.WriteString(fmt.Sprintf("🎭 Самая спорная: «%s» — разброс <b>%d</b>\n", html.EscapeString(ratings.Controversial.Book.DisplayName()), ratings.Controversial.Spread))
	}
	text.WriteString(fmt.Sprintf("🗳 Поставлено оценок: <b>%d</b>", ratings.RatingCount))
	return text.String()
}

func renderPersonalStatistics(data *chatdata.ChatData, userID int64, now time.Time) string {
	reading := calculateReadingStatistics(data, now)
	ratings := calculatePersonalRatingStatistics(data, userID)
	var text strings.Builder
	text.WriteString("📊 <b>МОЯ ЧИТАТЕЛЬСКАЯ СТАТИСТИКА</b>\n\n")
	text.WriteString("📚 <b>Чтение</b>\n")
	text.WriteString(formatReadingStatistics(reading))
	text.WriteString("\n\n⭐ <b>Мои оценки</b>\n")
	text.WriteString("🏆 Самая высокая: " + formatRatedBookStat(ratings.Highest) + "\n")
	text.WriteString("📉 Самая низкая: " + formatRatedBookStat(ratings.Lowest))
	return text.String()
}

func (lnb *LitNightBot) handleStatistics(message *tgbotapi.Message, logger *logrus.Entry) {
	data := lnb.iocd.GetOrCreateChatData(message.Chat.ID)
	text := renderClubStatistics(data, time.Now())
	if data.IsPrivateChat(message.Chat.ID) {
		text = renderPersonalStatistics(data, message.Chat.ID, time.Now())
	}
	buttons := [][]tgbotapi.InlineKeyboardButton{tgbotapi.NewInlineKeyboardRow(tgbotapi.NewInlineKeyboardButtonData("Закрыть", GetCallbackParamStr(CBCancel)))}
	if _, err := lnb.SendHTMLMessage(message.Chat.ID, text, buttons); err != nil {
		logger.WithError(err).Error("Failed to send statistics")
	}
}
