package chatdata

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const CurrentSchemaVersion = 2

type Book struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

func NewBook(name string) Book { return Book{Name: name, UUID: getUUID()} }

type HistoryItem struct {
	Book Book      `json:"book"`
	Date time.Time `json:"date"`
}

type WishlistItem struct {
	Book Book `json:"book"`
}

type CurrentBook struct {
	Book     Book      `json:"book"`
	Deadline time.Time `json:"deadline"`
}

type BookStatus string

const (
	StatusWishlist   BookStatus = "wishlist"
	StatusReading    BookStatus = "reading"
	StatusCompleted  BookStatus = "completed"
	StatusPostponed  BookStatus = "postponed"
	StatusUnfinished BookStatus = "unfinished"
	StatusExcluded   BookStatus = "excluded"
)

type Rating struct {
	UserID      int64     `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Value       int       `json:"value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

func (book *ClubBook) RatingByUser(userID int64) *Rating {
	for index := range book.Ratings {
		if book.Ratings[index].UserID == userID {
			return &book.Ratings[index]
		}
	}
	return nil
}

// SetRating creates a user's rating or updates their existing rating.
// The returned pointer contains the previous value when a rating was changed.
func (book *ClubBook) SetRating(userID int64, displayName string, value int, at time.Time) (*int, error) {
	if book.Status != StatusCompleted {
		return nil, errors.New("оценить можно только прочитанную книгу")
	}
	if book.RatingsClosedAt != nil {
		return nil, errors.New("сбор оценок завершён — при необходимости его можно возобновить")
	}
	if userID <= 0 {
		return nil, errors.New("не удалось определить участника")
	}
	if value < 1 || value > 10 {
		return nil, errors.New("оценка должна быть от 1 до 10")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "Участник"
	}
	if rating := book.RatingByUser(userID); rating != nil {
		previous := rating.Value
		rating.Value = value
		rating.DisplayName = displayName
		rating.UpdatedAt = at
		return &previous, nil
	}
	book.Ratings = append(book.Ratings, Rating{
		UserID:      userID,
		DisplayName: displayName,
		Value:       value,
		CreatedAt:   at,
		UpdatedAt:   at,
	})
	return nil, nil
}

func (book *ClubBook) DeleteRating(userID int64) bool {
	for index := range book.Ratings {
		if book.Ratings[index].UserID != userID {
			continue
		}
		book.Ratings = append(book.Ratings[:index], book.Ratings[index+1:]...)
		return true
	}
	return false
}

func (book *ClubBook) CloseRatings(userID int64, displayName string, at time.Time) error {
	if book.Status != StatusCompleted {
		return errors.New("завершить сбор оценок можно только для прочитанной книги")
	}
	if book.RatingsClosedAt != nil {
		return errors.New("сбор оценок уже завершён")
	}
	if len(book.Ratings) == 0 {
		return errors.New("пока нет оценок")
	}
	if userID <= 0 {
		return errors.New("не удалось определить участника")
	}
	closedAt := at
	book.RatingsClosedAt = &closedAt
	book.RatingsClosedBy = userID
	book.RatingsClosedByName = strings.TrimSpace(displayName)
	if book.RatingsClosedByName == "" {
		book.RatingsClosedByName = "Участник"
	}
	return nil
}

func (book *ClubBook) ReopenRatings() bool {
	if book.RatingsClosedAt == nil {
		return false
	}
	book.RatingsClosedAt = nil
	book.RatingsClosedBy = 0
	book.RatingsClosedByName = ""
	return true
}

type Review struct {
	ID          string    `json:"id"`
	UserID      int64     `json:"user_id"`
	DisplayName string    `json:"display_name"`
	Username    string    `json:"username,omitempty"`
	Text        string    `json:"text"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

type DiscussionSummary struct {
	Text       string    `json:"text"`
	EditorID   int64     `json:"editor_user_id"`
	EditorName string    `json:"editor_name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

type ChatMetadata struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type ClubBook struct {
	ID                  string             `json:"id"`
	Title               string             `json:"title"`
	Authors             []string           `json:"authors"`
	LegacyName          string             `json:"legacy_name,omitempty"`
	NeedsReview         bool               `json:"needs_review,omitempty"`
	Status              BookStatus         `json:"status"`
	AddedAt             time.Time          `json:"added_at"`
	StartedAt           *time.Time         `json:"started_at,omitempty"`
	CompletedAt         *time.Time         `json:"completed_at,omitempty"`
	StoppedAt           *time.Time         `json:"stopped_at,omitempty"`
	Deadline            *time.Time         `json:"deadline,omitempty"`
	Ratings             []Rating           `json:"ratings"`
	RatingsClosedAt     *time.Time         `json:"ratings_closed_at,omitempty"`
	RatingsClosedBy     int64              `json:"ratings_closed_by,omitempty"`
	RatingsClosedByName string             `json:"ratings_closed_by_name,omitempty"`
	Reviews             []Review           `json:"reviews"`
	DiscussionSummary   *DiscussionSummary `json:"discussion_summary,omitempty"`
}

func (book ClubBook) DisplayName() string {
	if len(book.Authors) == 0 {
		return book.Title
	}
	return fmt.Sprintf("%s — %s", book.Title, strings.Join(book.Authors, ", "))
}

func (book ClubBook) GetBook() Book {
	return Book{Name: book.DisplayName(), UUID: book.ID}
}

type ChatData struct {
	SchemaVersion     int           `json:"schema_version,omitempty"`
	MigrationComplete bool          `json:"migration_complete,omitempty"`
	Chat              *ChatMetadata `json:"chat,omitempty"`
	Books             []ClubBook    `json:"books,omitempty"`

	// Legacy v1 fields are read only by the migration flow.
	Wishlist []WishlistItem `json:"wishlist,omitempty"`
	History  []HistoryItem  `json:"history,omitempty"`
	Current  *CurrentBook   `json:"current_book,omitempty"`
}

func (cd *ChatData) IsPrivateChat(chatID int64) bool {
	if cd != nil && cd.Chat != nil && cd.Chat.Type != "" {
		return cd.Chat.Type == "private"
	}
	// Legacy v2 JSON files do not contain chat metadata. Telegram uses positive
	// IDs for private chats and negative IDs for groups and supergroups.
	return chatID > 0
}

type MigrationResult struct {
	WishlistCount     int
	HistoryCount      int
	CurrentCount      int
	NeedsReview       int
	TotalBookCount    int
	DuplicatesSkipped int
	IDsReassigned     int
}

type HasBook interface {
	GetBook() Book
}

func (wi WishlistItem) GetBook() Book { return wi.Book }
func (hi HistoryItem) GetBook() Book  { return hi.Book }

func getUUID() string { return uuid.New().String()[:8] }

func NewChatData() *ChatData {
	return &ChatData{SchemaVersion: CurrentSchemaVersion, MigrationComplete: true, Books: []ClubBook{}}
}

func (cd *ChatData) IsLegacy() bool {
	return cd != nil && cd.SchemaVersion < CurrentSchemaVersion
}

func (cd *ChatData) BooksWithStatus(statuses ...BookStatus) []ClubBook {
	wanted := make(map[BookStatus]struct{}, len(statuses))
	for _, status := range statuses {
		wanted[status] = struct{}{}
	}
	books := make([]ClubBook, 0)
	for _, book := range cd.Books {
		if _, ok := wanted[book.Status]; ok {
			books = append(books, book)
		}
	}
	return books
}

func (cd *ChatData) FindBook(id string) *ClubBook {
	for i := range cd.Books {
		if strings.EqualFold(cd.Books[i].ID, id) {
			return &cd.Books[i]
		}
	}
	return nil
}

func (cd *ChatData) CurrentBook() *ClubBook {
	for i := range cd.Books {
		if cd.Books[i].Status == StatusReading {
			return &cd.Books[i]
		}
	}
	return nil
}

// FinishCurrentBook closes the active reading session and records either the
// completion date or the date when the club stopped reading the book.
func (cd *ChatData) FinishCurrentBook(expectedID string, status BookStatus, at time.Time) (*ClubBook, error) {
	if status != StatusCompleted && status != StatusUnfinished {
		return nil, errors.New("нужно выбрать: прочитали или не дочитали")
	}
	book := cd.CurrentBook()
	if book == nil || !strings.EqualFold(book.ID, expectedID) {
		return nil, errors.New("текущая книга уже изменилась")
	}
	book.Status = status
	book.CompletedAt = nil
	book.StoppedAt = nil
	if status == StatusCompleted {
		book.CompletedAt = &at
	} else {
		book.StoppedAt = &at
	}
	book.Deadline = nil
	return book, nil
}

func (cd *ChatData) RemoveBook(id string) (*ClubBook, error) {
	for i := range cd.Books {
		if strings.EqualFold(cd.Books[i].ID, id) {
			book := cd.Books[i]
			cd.Books = append(cd.Books[:i], cd.Books[i+1:]...)
			return &book, nil
		}
	}
	return nil, errors.New("книга не найдена")
}

func (cd *ChatData) AddBook(title string, authors []string, status BookStatus, at time.Time) *ClubBook {
	cleanAuthors := make([]string, 0, len(authors))
	for _, author := range authors {
		if author = strings.TrimSpace(author); author != "" {
			cleanAuthors = append(cleanAuthors, author)
		}
	}
	book := ClubBook{
		ID:      getUUID(),
		Title:   strings.TrimSpace(title),
		Authors: cleanAuthors,
		Status:  status,
		AddedAt: at,
		Ratings: []Rating{},
		Reviews: []Review{},
	}
	if status == StatusCompleted {
		completedAt := at
		book.CompletedAt = &completedAt
	}
	cd.Books = append(cd.Books, book)
	return &cd.Books[len(cd.Books)-1]
}

func ParseStructuredBook(input string) (string, []string) {
	parts := strings.SplitN(input, "|", 2)
	title := strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return title, nil
	}
	authors := strings.Split(parts[1], ";")
	cleanAuthors := make([]string, 0, len(authors))
	for _, author := range authors {
		if author = strings.TrimSpace(author); author != "" {
			cleanAuthors = append(cleanAuthors, author)
		}
	}
	return title, cleanAuthors
}

func parseLegacyName(name string) (title string, authors []string, needsReview bool) {
	name = strings.TrimSpace(strings.TrimLeft(name, "✔✅☑•· "))
	if name == "" {
		return "Без названия", nil, true
	}
	if strings.HasPrefix(name, "«") {
		if end := strings.Index(name, "»"); end > 1 {
			title = strings.TrimSpace(name[len("«"):end])
			author := strings.TrimFunc(name[end+len("»"):], func(r rune) bool {
				return unicode.IsSpace(r) || r == ',' || r == '-' || r == '—'
			})
			if author == "" {
				return title, nil, true
			}
			return title, []string{author}, false
		}
	}
	if start := strings.Index(name, "«"); start > 0 {
		if relativeEnd := strings.Index(name[start+len("«"):], "»"); relativeEnd >= 0 {
			end := relativeEnd + start + len("«")
			author := strings.TrimSpace(name[:start])
			title = strings.TrimSpace(name[start+len("«") : end])
			suffix := strings.TrimSpace(name[end+len("»"):])
			if suffix != "" {
				return name, nil, true
			}
			if author != "" && title != "" {
				return title, []string{author}, false
			}
		}
	}
	for _, separator := range []string{" — ", " - "} {
		if parts := strings.SplitN(name, separator, 2); len(parts) == 2 {
			return strings.TrimSpace(parts[0]), []string{strings.TrimSpace(parts[1])}, true
		}
	}
	if parts := strings.SplitN(name, ",", 2); len(parts) == 2 {
		return strings.TrimSpace(parts[0]), []string{strings.TrimSpace(parts[1])}, true
	}
	return name, nil, true
}

func legacyBook(book Book, status BookStatus, at time.Time) ClubBook {
	title, authors, needsReview := parseLegacyName(book.Name)
	record := ClubBook{
		ID:          book.UUID,
		Title:       title,
		Authors:     authors,
		LegacyName:  book.Name,
		NeedsReview: needsReview,
		Status:      status,
		AddedAt:     at,
		Ratings:     []Rating{},
		Reviews:     []Review{},
	}
	if record.ID == "" {
		record.ID = getUUID()
		record.NeedsReview = true
	}
	return record
}

func (book ClubBook) sameMigrationRecord(other ClubBook) bool {
	book.ID = strings.ToLower(book.ID)
	other.ID = strings.ToLower(other.ID)
	return reflect.DeepEqual(book, other)
}

func MigrateV1(old *ChatData, now time.Time) (*ChatData, MigrationResult, error) {
	if old == nil {
		return nil, MigrationResult{}, errors.New("нет данных для миграции")
	}
	if !old.IsLegacy() {
		return nil, MigrationResult{}, errors.New("данные уже мигрированы")
	}
	migrated := NewChatData()
	result := MigrationResult{}
	seen := make(map[string]int)
	appendBook := func(book ClubBook) error {
		key := strings.ToLower(book.ID)
		if index, exists := seen[key]; exists {
			if migrated.Books[index].sameMigrationRecord(book) {
				result.DuplicatesSkipped++
				return nil
			}
			for {
				book.ID = getUUID()
				key = strings.ToLower(book.ID)
				if _, collision := seen[key]; !collision {
					break
				}
			}
			book.NeedsReview = true
			result.IDsReassigned++
		}
		seen[key] = len(migrated.Books)
		if book.NeedsReview {
			result.NeedsReview++
		}
		migrated.Books = append(migrated.Books, book)
		return nil
	}
	for _, item := range old.Wishlist {
		if err := appendBook(legacyBook(item.Book, StatusWishlist, now)); err != nil {
			return nil, result, err
		}
		result.WishlistCount++
	}
	for _, item := range old.History {
		at := item.Date
		if at.IsZero() {
			at = now
		}
		book := legacyBook(item.Book, StatusCompleted, at)
		book.CompletedAt = &at
		if err := appendBook(book); err != nil {
			return nil, result, err
		}
		result.HistoryCount++
	}
	if old.Current != nil && old.Current.Book.UUID != "" {
		book := legacyBook(old.Current.Book, StatusReading, now)
		startedAt := now
		book.StartedAt = &startedAt
		if !old.Current.Deadline.IsZero() {
			deadline := old.Current.Deadline
			book.Deadline = &deadline
		}
		if err := appendBook(book); err != nil {
			return nil, result, err
		}
		result.CurrentCount = 1
	}
	result.TotalBookCount = len(migrated.Books)
	migrated.MigrationComplete = result.NeedsReview == 0
	if err := migrated.ValidateV2(); err != nil {
		return nil, result, err
	}
	return migrated, result, nil
}

func (cd *ChatData) ValidateV2() error {
	if cd == nil || cd.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("неверная версия схемы")
	}
	seen := make(map[string]struct{}, len(cd.Books))
	reading := 0
	validStatuses := map[BookStatus]struct{}{
		StatusWishlist: {}, StatusReading: {}, StatusCompleted: {},
		StatusPostponed: {}, StatusUnfinished: {}, StatusExcluded: {},
	}
	for _, book := range cd.Books {
		if strings.TrimSpace(book.ID) == "" {
			return errors.New("книга без ID")
		}
		key := strings.ToLower(book.ID)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("дублирующийся UUID %q", book.ID)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(book.Title) == "" {
			return fmt.Errorf("книга %q без названия", book.ID)
		}
		if _, ok := validStatuses[book.Status]; !ok {
			return fmt.Errorf("книга %q с неизвестным статусом %q", book.ID, book.Status)
		}
		if book.Status == StatusReading {
			reading++
		}
		ratingUsers := make(map[int64]struct{}, len(book.Ratings))
		for _, rating := range book.Ratings {
			if book.Status != StatusCompleted {
				return fmt.Errorf("книга %q с оценками не прочитана", book.ID)
			}
			if rating.UserID <= 0 {
				return fmt.Errorf("книга %q содержит оценку без участника", book.ID)
			}
			if rating.Value < 1 || rating.Value > 10 {
				return fmt.Errorf("книга %q содержит оценку вне диапазона 1–10", book.ID)
			}
			if _, exists := ratingUsers[rating.UserID]; exists {
				return fmt.Errorf("книга %q содержит повторную оценку участника %d", book.ID, rating.UserID)
			}
			ratingUsers[rating.UserID] = struct{}{}
		}
		if book.RatingsClosedAt != nil && book.RatingsClosedBy <= 0 {
			return fmt.Errorf("книга %q содержит завершённый сбор оценок без участника", book.ID)
		}
		if book.RatingsClosedAt == nil && (book.RatingsClosedBy != 0 || book.RatingsClosedByName != "") {
			return fmt.Errorf("книга %q содержит данные завершения без даты", book.ID)
		}
	}
	if reading > 1 {
		return fmt.Errorf("найдено текущих книг: %d", reading)
	}
	return nil
}
