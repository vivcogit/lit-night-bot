package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"
	"testing"
	"time"
)

func statisticTime(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 12, 0, 0, 0, time.UTC)
}

func statisticsFixture() (*chatdata.ChatData, time.Time) {
	data := chatdata.NewChatData()
	firstStart := statisticTime(2026, time.January, 1)
	firstEnd := statisticTime(2026, time.January, 15)
	secondStart := statisticTime(2026, time.March, 5)
	secondEnd := statisticTime(2026, time.March, 15)
	thirdEnd := statisticTime(2026, time.May, 1)
	stopped := statisticTime(2026, time.June, 1)
	data.Books = []chatdata.ClubBook{
		{
			ID: "highest", Title: "Высокая <оценка>", Authors: []string{"Автор & Co"}, Status: chatdata.StatusCompleted,
			AddedAt: firstEnd, StartedAt: &firstStart, CompletedAt: &firstEnd,
			Ratings: []chatdata.Rating{{UserID: 1, DisplayName: "Анна", Value: 10}, {UserID: 2, DisplayName: "Борис", Value: 6}},
		},
		{
			ID: "lowest", Title: "Низкая", Status: chatdata.StatusCompleted,
			AddedAt: secondEnd, StartedAt: &secondStart, CompletedAt: &secondEnd,
			Ratings: []chatdata.Rating{{UserID: 1, DisplayName: "Анна", Value: 4}, {UserID: 2, DisplayName: "Борис", Value: 5}, {UserID: 3, DisplayName: "Вера", Value: 6}},
		},
		{ID: "unrated", Title: "Без оценки", Status: chatdata.StatusCompleted, AddedAt: thirdEnd, CompletedAt: &thirdEnd},
		{ID: "unfinished", Title: "Брошена", Status: chatdata.StatusUnfinished, AddedAt: stopped, StoppedAt: &stopped},
		{ID: "wishlist", Title: "Вишлист", Status: chatdata.StatusWishlist, AddedAt: stopped},
	}
	return data, statisticTime(2026, time.August, 22)
}

func TestMonthsInclusive(t *testing.T) {
	for _, test := range []struct {
		first time.Time
		last  time.Time
		want  int
	}{
		{first: statisticTime(2026, time.January, 1), last: statisticTime(2026, time.January, 31), want: 1},
		{first: statisticTime(2025, time.December, 1), last: statisticTime(2026, time.February, 1), want: 3},
		{first: statisticTime(2027, time.January, 1), last: statisticTime(2026, time.January, 1), want: 1},
		{want: 0},
	} {
		if got := monthsInclusive(test.first, test.last); got != test.want {
			t.Errorf("monthsInclusive(%v, %v) = %d, want %d", test.first, test.last, got, test.want)
		}
	}
}

func TestCalculateReadingStatistics(t *testing.T) {
	data, now := statisticsFixture()
	stats := calculateReadingStatistics(data, now)
	if stats.CompletedCount != 3 || stats.UnfinishedCount != 1 {
		t.Fatalf("unexpected counts: %#v", stats)
	}
	if diff := stats.BooksPerMonth - 0.375; diff < -0.000001 || diff > 0.000001 {
		t.Fatalf("books per month = %f", stats.BooksPerMonth)
	}
	if stats.ReadingDurationCount != 2 || stats.AverageReadingDays != 12 {
		t.Fatalf("reading duration = %#v", stats)
	}
}

func TestCalculateClubRatingStatistics(t *testing.T) {
	data, _ := statisticsFixture()
	stats := calculateClubRatingStatistics(data)
	if stats.RatingCount != 5 || stats.Average != 6.2 {
		t.Fatalf("club totals = %#v", stats)
	}
	if stats.Highest == nil || stats.Highest.Book.ID != "highest" || stats.Highest.Average != 8 {
		t.Fatalf("highest = %#v", stats.Highest)
	}
	if stats.Lowest == nil || stats.Lowest.Book.ID != "lowest" || stats.Lowest.Average != 5 {
		t.Fatalf("lowest = %#v", stats.Lowest)
	}
	if stats.Controversial == nil || stats.Controversial.Book.ID != "highest" || stats.Controversial.Spread != 4 {
		t.Fatalf("controversial = %#v", stats.Controversial)
	}
}

func TestClubStatisticsRendering(t *testing.T) {
	data, now := statisticsFixture()
	text := renderClubStatistics(data, now)
	for _, fragment := range []string{
		"СТАТИСТИКА КНИЖНОГО КЛУБА", "Прочитано: <b>3</b>", "Не дочитано: <b>1</b>",
		"0,4 книги в месяц", "12,0 дня", "6,2 из 10", "Поставлено оценок: <b>5</b>",
		"Высокая &lt;оценка&gt;", "Автор &amp; Co", "разброс <b>4</b>",
	} {
		if !strings.Contains(text, fragment) {
			t.Errorf("club statistics does not contain %q: %s", fragment, text)
		}
	}
	if strings.Contains(text, "Высокая <оценка>") {
		t.Fatalf("unsafe HTML: %s", text)
	}
}

func TestPersonalStatisticsUsesOnlyOwnersRatings(t *testing.T) {
	data, now := statisticsFixture()
	stats := calculatePersonalRatingStatistics(data, 1)
	if stats.Highest == nil || stats.Highest.Book.ID != "highest" || stats.Highest.Average != 10 {
		t.Fatalf("personal highest = %#v", stats.Highest)
	}
	if stats.Lowest == nil || stats.Lowest.Book.ID != "lowest" || stats.Lowest.Average != 4 {
		t.Fatalf("personal lowest = %#v", stats.Lowest)
	}
	text := renderPersonalStatistics(data, 1, now)
	for _, forbidden := range []string{"Средняя оценка клуба", "Самая спорная", "Поставлено оценок", "6,2 из 10"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("personal statistics leaked club data %q: %s", forbidden, text)
		}
	}
	for _, fragment := range []string{"МОЯ ЧИТАТЕЛЬСКАЯ СТАТИСТИКА", "Высокая &lt;оценка&gt;", "— <b>10,0</b>", "«Низкая» — <b>4,0</b>"} {
		if !strings.Contains(text, fragment) {
			t.Errorf("personal statistics does not contain %q: %s", fragment, text)
		}
	}
}

func TestStatisticsWithoutBooksOrRatings(t *testing.T) {
	data := chatdata.NewChatData()
	text := renderClubStatistics(data, statisticTime(2026, time.August, 22))
	if !strings.Contains(text, "Прочитано: <b>0</b>") || !strings.Contains(text, "0,0 книги в месяц") || strings.Count(text, "нет данных") != 5 {
		t.Fatalf("unexpected empty statistics: %s", text)
	}
}
