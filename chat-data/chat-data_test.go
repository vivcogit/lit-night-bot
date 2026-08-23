package chatdata

import (
	"strings"
	"testing"
	"time"
)

func TestBookDisplayAndProjection(t *testing.T) {
	tests := []struct {
		name    string
		book    ClubBook
		display string
	}{
		{name: "without author", book: ClubBook{ID: "a", Title: "Book"}, display: "Book"},
		{name: "one author", book: ClubBook{ID: "b", Title: "Book", Authors: []string{"Author"}}, display: "Book — Author"},
		{name: "several authors", book: ClubBook{ID: "c", Title: "Book", Authors: []string{"One", "Two"}}, display: "Book — One, Two"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.book.DisplayName(); got != test.display {
				t.Fatalf("DisplayName() = %q, want %q", got, test.display)
			}
			projected := test.book.GetBook()
			if projected.UUID != test.book.ID || projected.Name != test.display {
				t.Fatalf("GetBook() = %#v", projected)
			}
		})
	}
}

func TestChatTypeDetectionSupportsOldAndNewJSON(t *testing.T) {
	if !(&ChatData{}).IsPrivateChat(123) {
		t.Fatal("positive legacy chat ID must be treated as private")
	}
	if (&ChatData{}).IsPrivateChat(-123) {
		t.Fatal("negative legacy chat ID must be treated as a group")
	}
	private := &ChatData{Chat: &ChatMetadata{ID: -123, Type: "private"}}
	if !private.IsPrivateChat(-123) {
		t.Fatal("stored chat type must take precedence over ID fallback")
	}
	group := &ChatData{Chat: &ChatMetadata{ID: 123, Type: "supergroup"}}
	if group.IsPrivateChat(123) {
		t.Fatal("stored supergroup type was treated as private")
	}
}

func TestNewBookGeneratesShortUniqueID(t *testing.T) {
	first := NewBook("One")
	second := NewBook("Two")
	if len(first.UUID) != 8 || len(second.UUID) != 8 {
		t.Fatalf("unexpected UUID lengths: %q %q", first.UUID, second.UUID)
	}
	if first.UUID == second.UUID {
		t.Fatalf("generated duplicate UUID %q", first.UUID)
	}
}

func TestChatDataBookLifecycle(t *testing.T) {
	at := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	data := NewChatData()
	wishlist := data.AddBook("  Title  ", []string{" Author ", "", "Second"}, StatusWishlist, at)
	wishlistID := wishlist.ID
	completed := data.AddBook("Done", nil, StatusCompleted, at.Add(time.Hour))
	wishlist = data.FindBook(wishlistID)

	if wishlist.Title != "Title" || len(wishlist.Authors) != 2 || wishlist.Authors[0] != "Author" {
		t.Fatalf("AddBook did not normalize fields: %#v", wishlist)
	}
	if wishlist.Ratings == nil || wishlist.Reviews == nil {
		t.Fatal("new collections must be initialized")
	}
	if wishlist.CompletedAt != nil {
		t.Fatal("wishlist book must not be completed")
	}
	if completed.CompletedAt == nil || !completed.CompletedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("completed timestamp = %#v", completed.CompletedAt)
	}
	if got := data.FindBook(strings.ToUpper(wishlist.ID)); got == nil || got.Title != "Title" {
		t.Fatal("FindBook must be case-insensitive")
	}
	if data.CurrentBook() != nil {
		t.Fatal("unexpected current book")
	}
	wishlist.Status = StatusReading
	if got := data.CurrentBook(); got == nil || got.ID != wishlist.ID {
		t.Fatalf("CurrentBook() = %#v", got)
	}
	if got := data.BooksWithStatus(StatusReading, StatusCompleted); len(got) != 2 {
		t.Fatalf("BooksWithStatus() length = %d", len(got))
	}

	removed, err := data.RemoveBook(strings.ToUpper(completed.ID))
	if err != nil || removed.ID != completed.ID || len(data.Books) != 1 {
		t.Fatalf("RemoveBook() = %#v, %v; remaining=%d", removed, err, len(data.Books))
	}
	if _, err := data.RemoveBook("missing"); err == nil {
		t.Fatal("removing missing book must fail")
	}
}

func TestBookRatingLifecycle(t *testing.T) {
	createdAt := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	book := ClubBook{ID: "book1", Title: "Book", Status: StatusCompleted, Ratings: []Rating{}}

	previous, err := book.SetRating(10, " Анна ", 7, createdAt)
	if err != nil || previous != nil || len(book.Ratings) != 1 {
		t.Fatalf("first SetRating() = %v, %v, %#v", previous, err, book.Ratings)
	}
	rating := book.RatingByUser(10)
	if rating == nil || rating.DisplayName != "Анна" || rating.Value != 7 || !rating.CreatedAt.Equal(createdAt) || !rating.UpdatedAt.Equal(createdAt) {
		t.Fatalf("unexpected first rating: %#v", rating)
	}

	previous, err = book.SetRating(10, "Анна Новая", 9, updatedAt)
	if err != nil || previous == nil || *previous != 7 || len(book.Ratings) != 1 {
		t.Fatalf("updated SetRating() = %v, %v, %#v", previous, err, book.Ratings)
	}
	rating = book.RatingByUser(10)
	if rating.Value != 9 || rating.DisplayName != "Анна Новая" || !rating.CreatedAt.Equal(createdAt) || !rating.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("unexpected updated rating: %#v", rating)
	}

	if _, err := book.SetRating(11, "Борис", 10, updatedAt); err != nil || len(book.Ratings) != 2 {
		t.Fatalf("second participant: %v, %#v", err, book.Ratings)
	}
	if !book.DeleteRating(10) || book.DeleteRating(10) || book.RatingByUser(10) != nil || len(book.Ratings) != 1 {
		t.Fatalf("DeleteRating failed: %#v", book.Ratings)
	}
}

func TestBookRatingRejectsInvalidInput(t *testing.T) {
	now := time.Now()
	reading := ClubBook{Status: StatusReading}
	if _, err := reading.SetRating(1, "Анна", 8, now); err == nil {
		t.Fatal("reading book accepted a rating")
	}
	completed := ClubBook{Status: StatusCompleted}
	for _, test := range []struct {
		userID int64
		value  int
	}{{userID: 0, value: 8}, {userID: 1, value: 0}, {userID: 1, value: 11}} {
		if _, err := completed.SetRating(test.userID, "Анна", test.value, now); err == nil {
			t.Fatalf("accepted user=%d value=%d", test.userID, test.value)
		}
	}
}

func TestFinishCurrentBookStatuses(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	for _, status := range []BookStatus{StatusCompleted, StatusUnfinished} {
		t.Run(string(status), func(t *testing.T) {
			data := NewChatData()
			book := data.AddBook("Книга", nil, StatusReading, now.Add(-24*time.Hour))
			deadline := now.Add(24 * time.Hour)
			book.Deadline = &deadline
			finished, err := data.FinishCurrentBook(book.ID, status, now)
			if err != nil || finished.Status != status || finished.Deadline != nil || data.CurrentBook() != nil {
				t.Fatalf("FinishCurrentBook() = %#v, %v", finished, err)
			}
			if status == StatusCompleted && (finished.CompletedAt == nil || !finished.CompletedAt.Equal(now) || finished.StoppedAt != nil) {
				t.Fatalf("completion dates = %#v", finished)
			}
			if status == StatusUnfinished && (finished.StoppedAt == nil || !finished.StoppedAt.Equal(now) || finished.CompletedAt != nil) {
				t.Fatalf("unfinished dates = %#v", finished)
			}
		})
	}
	data := NewChatData()
	book := data.AddBook("Книга", nil, StatusReading, now)
	if _, err := data.FinishCurrentBook(book.ID, StatusWishlist, now); err == nil {
		t.Fatal("unsupported final status was accepted")
	}
	if _, err := data.FinishCurrentBook("another", StatusCompleted, now); err == nil {
		t.Fatal("stale book ID was accepted")
	}
}

func TestBookRatingCloseAndReopen(t *testing.T) {
	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	book := ClubBook{ID: "book1", Title: "Book", Status: StatusCompleted}
	if err := book.CloseRatings(1, "Анна", now); err == nil {
		t.Fatal("empty ratings were closed")
	}
	if _, err := book.SetRating(1, "Анна", 8, now); err != nil {
		t.Fatal(err)
	}
	if err := book.CloseRatings(10, " Модератор ", now); err != nil {
		t.Fatal(err)
	}
	if book.RatingsClosedAt == nil || book.RatingsClosedBy != 10 || book.RatingsClosedByName != "Модератор" {
		t.Fatalf("unexpected close metadata: %#v", book)
	}
	if _, err := book.SetRating(2, "Борис", 9, now); err == nil {
		t.Fatal("closed ratings accepted a value")
	}
	if err := book.CloseRatings(10, "Модератор", now); err == nil {
		t.Fatal("ratings were closed twice")
	}
	if !book.ReopenRatings() || book.RatingsClosedAt != nil || book.RatingsClosedBy != 0 || book.RatingsClosedByName != "" {
		t.Fatalf("ratings were not reopened: %#v", book)
	}
	if book.ReopenRatings() {
		t.Fatal("already open ratings reported a change")
	}
	if _, err := book.SetRating(2, "Борис", 9, now); err != nil {
		t.Fatalf("reopened ratings rejected a value: %v", err)
	}
}

func TestBookReviewLifecycleAndPerUserReminder(t *testing.T) {
	now := time.Date(2026, time.August, 23, 18, 0, 0, 0, time.UTC)
	closedAt := now
	book := ClubBook{ID: "book1", Title: "Book", Status: StatusCompleted, RatingsClosedAt: &closedAt, RatingsClosedBy: 1}
	dueAt := now.Add(15 * time.Minute)
	if !book.ScheduleReviewRequest(dueAt) || book.ReviewRequestDueAt == nil || !book.ReviewRequestDueAt.Equal(dueAt) {
		t.Fatalf("review request was not scheduled: %#v", book)
	}
	book.MarkReviewRequestSent(dueAt, 42)
	if book.ReviewRequestDueAt != nil || book.ReviewRequestSentAt == nil || book.ReviewRequestMsgID != 42 {
		t.Fatalf("review request was not marked sent: %#v", book)
	}
	reminderAt := dueAt.Add(24 * time.Hour)
	if err := book.SetReviewReminder(2, " Борис ", "boris", reminderAt); err != nil {
		t.Fatal(err)
	}
	if err := book.SetReviewReminder(2, "Борис Новый", "new_boris", reminderAt.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if len(book.ReviewReminders) != 1 || book.ReviewReminders[0].DisplayName != "Борис Новый" {
		t.Fatalf("reminder was duplicated instead of updated: %#v", book.ReviewReminders)
	}
	if err := book.SetReviewReminder(3, "Вера", "vera", reminderAt); err != nil {
		t.Fatal(err)
	}
	updated, err := book.SetReview(2, "Борис", "boris", " Первая мысль. ", reminderAt)
	if err != nil || updated || len(book.Reviews) != 1 || len(book.ReviewReminders) != 1 || book.ReviewReminders[0].UserID != 3 {
		t.Fatalf("unexpected first review: updated=%v err=%v book=%#v", updated, err, book)
	}
	updated, err = book.SetReview(2, "Борис", "boris", "Новый текст", reminderAt.Add(time.Hour))
	if err != nil || !updated || book.Reviews[0].Text != "Новый текст" {
		t.Fatalf("review was not updated: updated=%v err=%v review=%#v", updated, err, book.Reviews[0])
	}
	if !book.DeleteReview(2) || book.DeleteReview(2) {
		t.Fatal("review deletion must succeed exactly once")
	}
}

func TestPendingReviewRequestCanBeCancelledBeforeSending(t *testing.T) {
	now := time.Now()
	book := ClubBook{Status: StatusCompleted, RatingsClosedAt: &now}
	if !book.ScheduleReviewRequest(now.Add(15*time.Minute)) || !book.CancelPendingReviewRequest() || book.ReviewRequestDueAt != nil {
		t.Fatalf("pending request was not cancelled: %#v", book)
	}
	book.ScheduleReviewRequest(now.Add(15 * time.Minute))
	book.MarkReviewRequestSent(now, 1)
	if book.CancelPendingReviewRequest() {
		t.Fatal("sent request must not be cancelled")
	}
}

func TestParseStructuredBookCases(t *testing.T) {
	tests := []struct {
		input       string
		title       string
		authorCount int
	}{
		{input: "  Title  ", title: "Title"},
		{input: "Title |", title: "Title"},
		{input: "Title | One; ; Two ", title: "Title", authorCount: 2},
		{input: "Title | Author | Suffix", title: "Title", authorCount: 1},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			title, authors := ParseStructuredBook(test.input)
			if title != test.title || len(authors) != test.authorCount {
				t.Fatalf("ParseStructuredBook() = %q, %#v", title, authors)
			}
		})
	}
}

func TestParseLegacyNameCases(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		author      string
		needsReview bool
	}{
		{name: "", title: "Без названия", needsReview: true},
		{name: "«Лес» Светлана Тюльбашева", title: "Лес", author: "Светлана Тюльбашева"},
		{name: "Рут Озеки «Моя рыба будет жить»", title: "Моя рыба будет жить", author: "Рут Озеки"},
		{name: "Title — Author", title: "Title", author: "Author", needsReview: true},
		{name: "Title - Author", title: "Title", author: "Author", needsReview: true},
		{name: "Title, Author", title: "Title", author: "Author", needsReview: true},
		{name: "Only title", title: "Only title", needsReview: true},
		{name: "«Only title»", title: "Only title", needsReview: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			title, authors, needsReview := parseLegacyName(test.name)
			if title != test.title || needsReview != test.needsReview {
				t.Fatalf("parseLegacyName() = %q, %#v, %v", title, authors, needsReview)
			}
			if test.author == "" && len(authors) != 0 {
				t.Fatalf("unexpected authors: %#v", authors)
			}
			if test.author != "" && (len(authors) != 1 || authors[0] != test.author) {
				t.Fatalf("authors = %#v, want %q", authors, test.author)
			}
		})
	}
}

func TestValidateV2Failures(t *testing.T) {
	validBook := ClubBook{ID: "abc12345", Title: "Book", Status: StatusWishlist}
	tests := []struct {
		name string
		data *ChatData
	}{
		{name: "nil", data: nil},
		{name: "wrong schema", data: &ChatData{SchemaVersion: 1}},
		{name: "empty id", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{Title: "Book", Status: StatusWishlist}}}},
		{name: "duplicate id ignoring case", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{validBook, {ID: "ABC12345", Title: "Other", Status: StatusWishlist}}}},
		{name: "empty title", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "id", Title: "  ", Status: StatusWishlist}}}},
		{name: "unknown status", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "id", Title: "Book", Status: "unknown"}}}},
		{name: "several current books", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusReading}, {ID: "2", Title: "Two", Status: StatusReading}}}},
		{name: "rating on unread book", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusWishlist, Ratings: []Rating{{UserID: 1, Value: 8}}}}}},
		{name: "rating without user", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusCompleted, Ratings: []Rating{{Value: 8}}}}}},
		{name: "rating outside scale", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusCompleted, Ratings: []Rating{{UserID: 1, Value: 11}}}}}},
		{name: "duplicate participant rating", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusCompleted, Ratings: []Rating{{UserID: 1, Value: 8}, {UserID: 1, Value: 9}}}}}},
		{name: "closed without participant", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusCompleted, RatingsClosedAt: func() *time.Time { value := time.Now(); return &value }()}}}},
		{name: "close metadata without date", data: &ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{{ID: "1", Title: "One", Status: StatusCompleted, RatingsClosedBy: 1}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.data.ValidateV2(); err == nil {
				t.Fatal("ValidateV2() must fail")
			}
		})
	}
	if err := (&ChatData{SchemaVersion: CurrentSchemaVersion, Books: []ClubBook{validBook}}).ValidateV2(); err != nil {
		t.Fatalf("valid data rejected: %v", err)
	}
}

func TestMigrateV1ErrorsAndFallbacks(t *testing.T) {
	if _, _, err := MigrateV1(nil, time.Now()); err == nil {
		t.Fatal("nil migration must fail")
	}
	if _, _, err := MigrateV1(NewChatData(), time.Now()); err == nil {
		t.Fatal("v2 migration must fail")
	}

	now := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	legacy := &ChatData{
		Wishlist: []WishlistItem{{Book: Book{Name: "First", UUID: "ABC"}}},
		History:  []HistoryItem{{Book: Book{Name: "Second", UUID: "abc"}}},
	}
	repaired, result, err := MigrateV1(legacy, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(repaired.Books) != 2 || result.IDsReassigned != 1 || !repaired.Books[1].NeedsReview || strings.EqualFold(repaired.Books[0].ID, repaired.Books[1].ID) {
		t.Fatalf("case-insensitive duplicate UUID was not repaired: books=%#v result=%+v", repaired.Books, result)
	}

	legacy = &ChatData{History: []HistoryItem{{Book: Book{Name: "Title", UUID: "history1"}}}}
	migrated, _, err := MigrateV1(legacy, now)
	if err != nil {
		t.Fatal(err)
	}
	book := migrated.FindBook("history1")
	if book == nil || book.CompletedAt == nil || !book.CompletedAt.Equal(now) || !book.AddedAt.Equal(now) {
		t.Fatalf("zero legacy date fallback failed: %#v", book)
	}
}
