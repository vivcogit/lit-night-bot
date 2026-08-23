package bot

import (
	chatdata "lit-night-bot/chat-data"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func callbackData(t *testing.T, value *string) string {
	t.Helper()
	if value == nil {
		t.Fatal("button has no callback data")
	}
	return *value
}

func testBooks(count int) []chatdata.ClubBook {
	books := make([]chatdata.ClubBook, count)
	for index := range books {
		books[index] = chatdata.ClubBook{ID: string(rune('a' + index)), Title: "Book", Status: chatdata.StatusWishlist}
	}
	return books
}

func TestCallbackParametersRoundTrip(t *testing.T) {
	encoded := GetCallbackParamStr(CBHistoryYear, "2026", "2")
	if encoded != "h_year:2026:2" {
		t.Fatalf("encoded callback = %q", encoded)
	}
	action, params, err := GetCallbackParam(encoded)
	if err != nil || action != CBHistoryYear || len(params) != 2 || params[1] != "2" {
		t.Fatalf("decoded callback = %q, %#v, %v", action, params, err)
	}
	if _, _, err := GetCallbackParam("  "); err == nil {
		t.Fatal("empty callback must fail")
	}
}

func TestReviewCallbackKeepsMenuMessageForEditing(t *testing.T) {
	if shouldRemoveMenuMessage(menuText, CBBooksReview) {
		t.Fatal("review callback must keep the menu message because it edits it in place")
	}
	if !shouldRemoveMenuMessage(menuText, CBWishlistShow) {
		t.Fatal("regular menu callbacks must keep the existing removal behavior")
	}
	if shouldRemoveMenuMessage("another message", CBBooksReview) {
		t.Fatal("non-menu messages must not be removed by the menu cleanup")
	}
}

func TestGetBooklistPageBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		page     int
		wantPage int
		wantLen  int
		last     bool
	}{
		{name: "empty", count: 0, page: 0, wantPage: 0, wantLen: 0, last: true},
		{name: "negative", count: 12, page: -1, wantPage: 0, wantLen: 5, last: false},
		{name: "first", count: 12, page: 0, wantPage: 0, wantLen: 5, last: false},
		{name: "middle", count: 12, page: 1, wantPage: 1, wantLen: 5, last: false},
		{name: "last", count: 12, page: 2, wantPage: 2, wantLen: 2, last: true},
		{name: "too large", count: 12, page: 99, wantPage: 2, wantLen: 2, last: true},
		{name: "full page", count: 5, page: 0, wantPage: 0, wantLen: 5, last: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			books := testBooks(test.count)
			got, page, last := GetBooklistPage(&books, test.page)
			if page != test.wantPage || len(got) != test.wantLen || last != test.last {
				t.Fatalf("page=%d len=%d last=%v", page, len(got), last)
			}
		})
	}
}

func TestBooklistFormattingAndButtons(t *testing.T) {
	books := []chatdata.ClubBook{
		{ID: "one", Title: "First", Authors: []string{"Author"}},
		{ID: "two", Title: "Second"},
	}
	if got := GetBooklistString(&books); got != "1. First — Author\n2. Second\n" {
		t.Fatalf("GetBooklistString() = %q", got)
	}
	buttons := GetButtonsForBooklist(&books, "☑", CBCurrentChooseBook, 3)
	if len(buttons) != 2 || buttons[0][0].Text != "☑ First — Author" {
		t.Fatalf("unexpected buttons: %#v", buttons)
	}
	if got := callbackData(t, buttons[0][0].CallbackData); got != "c_manual:one:3" {
		t.Fatalf("callback = %q", got)
	}

	logger := logrus.NewEntry(logrus.New())
	empty := []chatdata.ClubBook{}
	text, emptyButtons := GetBooklistPageMessage(1, 0, logger, &empty, "Nothing", "-", CBBookShow, CBWishlistChangePage, "Title")
	if text != "Nothing" || emptyButtons != nil {
		t.Fatalf("empty page = %q, %#v", text, emptyButtons)
	}

	six := testBooks(6)
	text, pageButtons := GetBooklistPageMessage(1, 0, logger, &six, "Nothing", "-", CBBookShow, CBWishlistChangePage, "Title")
	if text != "Title (страница 1):\n\n" || len(pageButtons) != 6 {
		t.Fatalf("page message = %q, buttons=%d", text, len(pageButtons))
	}
}

func TestPaginationNavigation(t *testing.T) {
	first := *GetPaginationNavButtons(0, false, CBWishlistChangePage)
	if len(first) != 2 || first[0].Text != "Завершить" || first[1].Text != "➡" {
		t.Fatalf("first navigation = %#v", first)
	}
	if got := callbackData(t, first[1].CallbackData); got != "wl_clean_page:1" {
		t.Fatalf("next callback = %q", got)
	}
	middle := *GetPaginationNavButtons(2, false, CBWishlistChangePage)
	if len(middle) != 3 || middle[0].Text != "⬅" || middle[2].Text != "➡" {
		t.Fatalf("middle navigation = %#v", middle)
	}
	last := *GetPaginationNavButtons(2, true, CBWishlistChangePage)
	if len(last) != 2 || last[0].Text != "⬅" || last[1].Text != "Завершить" {
		t.Fatalf("last navigation = %#v", last)
	}
}

func TestStatusLabels(t *testing.T) {
	tests := map[chatdata.BookStatus]string{
		chatdata.StatusWishlist:   "📚 В вишлисте",
		chatdata.StatusReading:    "📖 Читаем",
		chatdata.StatusCompleted:  "✅ Прочитана",
		chatdata.StatusPostponed:  "⏸ Отложена",
		chatdata.StatusUnfinished: "🚫 Не дочитана",
		chatdata.StatusExcluded:   "🗑 Исключена",
		"custom":                  "custom",
	}
	for status, want := range tests {
		if got := statusLabel(status); got != want {
			t.Errorf("statusLabel(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestRenderDetailedBookCardAndButtons(t *testing.T) {
	completed := time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC)
	deadline := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	book := &chatdata.ClubBook{
		ID:          "book1",
		Title:       "Title <tag>",
		Authors:     []string{"One & Two"},
		LegacyName:  "Old <name>",
		NeedsReview: true,
		Status:      chatdata.StatusCompleted,
		CompletedAt: &completed,
		Deadline:    &deadline,
		Ratings:     []chatdata.Rating{{Value: 4}, {Value: 5}},
		Reviews:     []chatdata.Review{{ID: "r1"}},
	}
	text := renderBookCard(book)
	for _, fragment := range []string{"Title &lt;tag&gt;", "One &amp; Two", "⭐ 4,5 из 10 · 2 оценки", "Old &lt;name&gt;", "20.08.2026", "01.09.2026"} {
		if !strings.Contains(text, fragment) {
			t.Errorf("card does not contain %q: %s", fragment, text)
		}
	}
	buttons := bookCardButtons(book)
	if len(buttons) != 6 {
		t.Fatalf("button rows = %d, want 6", len(buttons))
	}
	book.Authors = nil
	book.NeedsReview = false
	buttons = bookCardButtons(book)
	if len(buttons) != 4 {
		t.Fatalf("minimal button rows = %d, want 4", len(buttons))
	}
}

func TestHistoryHelpers(t *testing.T) {
	first := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	second := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	books := []chatdata.ClubBook{
		{ID: "old", Title: "Old", Status: chatdata.StatusCompleted, AddedAt: first, CompletedAt: &first},
		{ID: "new", Title: "New", Status: chatdata.StatusCompleted, AddedAt: second, CompletedAt: &second},
		{ID: "fallback", Title: "Fallback", Status: chatdata.StatusCompleted, AddedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
	}
	if got := completedYear(books[2]); got != 2024 {
		t.Fatalf("completedYear fallback = %d", got)
	}
	selected := historyBooksForYear(books, 2025)
	if len(selected) != 2 || selected[0].ID != "new" || selected[1].ID != "old" {
		t.Fatalf("history order = %#v", selected)
	}
	years, counts := historyYears(books)
	if len(years) != 2 || years[0] != 2025 || years[1] != 2024 || counts[2025] != 2 {
		t.Fatalf("years=%#v counts=%#v", years, counts)
	}
	if averageRating(nil) != 0 || averageRating([]chatdata.Rating{{Value: 2}, {Value: 5}}) != 3.5 {
		t.Fatal("averageRating returned wrong value")
	}
	emptyText := renderHistoryYear(2026, nil, 0, 2026, false)
	if !strings.Contains(emptyText, "В этом году") || strings.Contains(emptyText, "<b>Архив:</b>") {
		t.Fatalf("unexpected empty history: %s", emptyText)
	}
}

func TestHistoryYearButtons(t *testing.T) {
	lnb := &LitNightBot{}
	books := []chatdata.ClubBook{{ID: "book", Title: "Book"}}
	years := []int{2026, 2025, 2024}
	counts := map[int]int{2026: 1, 2025: 2, 2024: 3}
	current := lnb.historyYearButtons(2026, books, years, counts, 2026)
	if len(current) != 4 || current[0][0].Text != "2025 · 2 книг" || current[2][0].Text != "📖 Открыть карточку" {
		t.Fatalf("current year buttons = %#v", current)
	}
	archive := lnb.historyYearButtons(2025, books, years, counts, 2026)
	if archive[1][0].Text != "← Вернуться к 2026" {
		t.Fatalf("archive buttons = %#v", archive)
	}
}

func TestCurrentBookSelectionChecks(t *testing.T) {
	lnb := &LitNightBot{}
	data := chatdata.NewChatData()
	if got := lnb.checkCanChooseBook(data); !strings.Contains(got, "Вишлист пуст") {
		t.Fatalf("empty warning = %q", got)
	}
	data.AddBook("Wish", nil, chatdata.StatusWishlist, time.Now())
	if got := lnb.checkCanChooseBook(data); got != "" {
		t.Fatalf("unexpected warning = %q", got)
	}
	data.AddBook("Current", nil, chatdata.StatusReading, time.Now())
	if got := lnb.checkCanChooseBook(data); !strings.Contains(got, "Current") {
		t.Fatalf("current warning = %q", got)
	}
}

func TestCurrentCompletionStatusButtons(t *testing.T) {
	buttons := currentCompletionButtons("book0001")
	if len(buttons) != 3 || buttons[0][0].Text != "✅ Обсудили" || buttons[1][0].Text != "🚫 Не дочитали / бросили" {
		t.Fatalf("unexpected completion buttons: %#v", buttons)
	}
	for index, want := range []CallbackAction{CBCurrentMarkCompleted, CBCurrentMarkUnfinished} {
		action, params, err := GetCallbackParam(*buttons[index][0].CallbackData)
		if err != nil || action != want || len(params) != 1 || params[0] != "book0001" {
			t.Fatalf("button %d callback = %q, %#v, %v", index, action, params, err)
		}
	}
}

func TestUnfinishedCardHasStatusButNoRatingControls(t *testing.T) {
	ended := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	book := &chatdata.ClubBook{ID: "book0001", Title: "Брошенная книга", Status: chatdata.StatusUnfinished, StoppedAt: &ended}
	text := renderBookCard(book)
	if !strings.Contains(text, "🚫 Не дочитана") || !strings.Contains(text, "Завершили чтение: 22.08.2026") || strings.Contains(text, "Оцен") {
		t.Fatalf("unexpected unfinished card: %s", text)
	}
	for _, row := range bookCardButtons(book) {
		for _, button := range row {
			if strings.Contains(button.Text, "Оцен") {
				t.Fatalf("rating button on unfinished book: %q", button.Text)
			}
		}
	}
}

func TestTruncateButtonUsesRunes(t *testing.T) {
	short := "Коротко"
	if got := truncateButton(short); got != short {
		t.Fatalf("short value changed: %q", got)
	}
	long := strings.Repeat("📚", 50)
	got := truncateButton(long)
	if len([]rune(got)) != 42 || !strings.HasSuffix(got, "…") {
		t.Fatalf("truncated value length=%d suffix=%q", len([]rune(got)), got)
	}
}
