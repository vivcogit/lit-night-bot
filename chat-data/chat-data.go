package chatdata

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"

	"github.com/google/uuid"
)

const CurrentSchemaVersion = 3
const MaxReviewTextUTF16Units = 3500
const MaxUnfinishedReasonUTF16Units = 500

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

type UnfinishedReason struct {
	Code string `json:"code"`
	Text string `json:"text,omitempty"`
}

const (
	UnfinishedReasonNotEngaging  = "not_engaging"
	UnfinishedReasonTooDifficult = "too_difficult"
	UnfinishedReasonNotForClub   = "not_for_club"
	UnfinishedReasonNoTime       = "no_time"
	UnfinishedReasonOther        = "other"
)

func NewUnfinishedReason(code string, text string) (*UnfinishedReason, error) {
	reason := &UnfinishedReason{Code: code, Text: strings.TrimSpace(text)}
	if err := reason.Validate(); err != nil {
		return nil, err
	}
	return reason, nil
}

func (reason *UnfinishedReason) Validate() error {
	if reason == nil {
		return nil
	}
	switch reason.Code {
	case UnfinishedReasonNotEngaging, UnfinishedReasonTooDifficult, UnfinishedReasonNotForClub, UnfinishedReasonNoTime:
		if strings.TrimSpace(reason.Text) != "" {
			return errors.New("стандартная причина не должна содержать свой текст")
		}
	case UnfinishedReasonOther:
		if strings.TrimSpace(reason.Text) == "" {
			return errors.New("укажите причину")
		}
		if len(utf16.Encode([]rune(reason.Text))) > MaxUnfinishedReasonUTF16Units {
			return fmt.Errorf("причина слишком длинная — сократите её до %d символов", MaxUnfinishedReasonUTF16Units)
		}
	default:
		return errors.New("неизвестная причина")
	}
	return nil
}

func (reason *UnfinishedReason) DisplayText() string {
	if reason == nil {
		return ""
	}
	switch reason.Code {
	case UnfinishedReasonNotEngaging:
		return "Не увлекла"
	case UnfinishedReasonTooDifficult:
		return "Слишком тяжело читается"
	case UnfinishedReasonNotForClub:
		return "Не подошла для клуба"
	case UnfinishedReasonNoTime:
		return "Не успели и не хотим продолжать"
	case UnfinishedReasonOther:
		return strings.TrimSpace(reason.Text)
	default:
		return ""
	}
}

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

type ReviewReminder struct {
	UserID            int64      `json:"user_id"`
	DisplayName       string     `json:"display_name"`
	Username          string     `json:"username,omitempty"`
	DueAt             time.Time  `json:"due_at"`
	DeliveryClaimedAt *time.Time `json:"delivery_claimed_at,omitempty"`
}

func claimAvailable(claimedAt *time.Time, at time.Time, lease time.Duration) bool {
	return claimedAt == nil || !claimedAt.Add(lease).After(at)
}

func (reminder *ReviewReminder) ClaimDelivery(at time.Time, lease time.Duration) bool {
	if reminder == nil || reminder.DueAt.After(at) || !claimAvailable(reminder.DeliveryClaimedAt, at, lease) {
		return false
	}
	claimedAt := at
	reminder.DeliveryClaimedAt = &claimedAt
	return true
}

func (reminder *ReviewReminder) ReleaseDeliveryClaim() {
	if reminder != nil {
		reminder.DeliveryClaimedAt = nil
	}
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
	ID                     string             `json:"id"`
	Title                  string             `json:"title"`
	Authors                []string           `json:"authors"`
	LegacyName             string             `json:"legacy_name,omitempty"`
	NeedsReview            bool               `json:"needs_review,omitempty"`
	Status                 BookStatus         `json:"status"`
	AddedAt                time.Time          `json:"added_at"`
	StartedAt              *time.Time         `json:"started_at,omitempty"`
	CompletedAt            *time.Time         `json:"completed_at,omitempty"`
	StoppedAt              *time.Time         `json:"stopped_at,omitempty"`
	UnfinishedReason       *UnfinishedReason  `json:"unfinished_reason,omitempty"`
	Deadline               *time.Time         `json:"deadline,omitempty"`
	Ratings                []Rating           `json:"ratings"`
	RatingsClosedAt        *time.Time         `json:"ratings_closed_at,omitempty"`
	RatingsClosedBy        int64              `json:"ratings_closed_by,omitempty"`
	RatingsClosedByName    string             `json:"ratings_closed_by_name,omitempty"`
	Reviews                []Review           `json:"reviews"`
	ReviewRequestDueAt     *time.Time         `json:"review_request_due_at,omitempty"`
	ReviewRequestRetryAt   *time.Time         `json:"review_request_retry_not_before,omitempty"`
	ReviewRequestClaimedAt *time.Time         `json:"review_request_claimed_at,omitempty"`
	ReviewRequestSentAt    *time.Time         `json:"review_request_sent_at,omitempty"`
	ReviewRequestMsgID     int                `json:"review_request_message_id,omitempty"`
	ReviewReminders        []ReviewReminder   `json:"review_reminders,omitempty"`
	DiscussionSummary      *DiscussionSummary `json:"discussion_summary,omitempty"`
}

func (book *ClubBook) ReviewByUser(userID int64) *Review {
	for index := range book.Reviews {
		if book.Reviews[index].UserID == userID {
			return &book.Reviews[index]
		}
	}
	return nil
}

func (book *ClubBook) SetReview(userID int64, displayName string, username string, text string, at time.Time) (bool, error) {
	if book.Status != StatusCompleted {
		return false, errors.New("оставить отзыв можно только на обсуждённую книгу")
	}
	if userID <= 0 {
		return false, errors.New("не удалось определить участника")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return false, errors.New("отзыв не может быть пустым")
	}
	if len(utf16.Encode([]rune(text))) > MaxReviewTextUTF16Units {
		return false, fmt.Errorf("отзыв слишком длинный — сократите его до %d символов", MaxReviewTextUTF16Units)
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "Участник"
	}
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	if review := book.ReviewByUser(userID); review != nil {
		review.DisplayName = displayName
		review.Username = username
		review.Text = text
		review.UpdatedAt = at
		book.CancelReviewReminder(userID)
		return true, nil
	}
	book.Reviews = append(book.Reviews, Review{
		ID:          getUUID(),
		UserID:      userID,
		DisplayName: displayName,
		Username:    username,
		Text:        text,
		CreatedAt:   at,
		UpdatedAt:   at,
	})
	book.CancelReviewReminder(userID)
	return false, nil
}

func (book *ClubBook) DeleteReview(userID int64) bool {
	for index := range book.Reviews {
		if book.Reviews[index].UserID != userID {
			continue
		}
		book.Reviews = append(book.Reviews[:index], book.Reviews[index+1:]...)
		return true
	}
	return false
}

func (book *ClubBook) ScheduleReviewRequest(dueAt time.Time) bool {
	if book.Status != StatusCompleted || book.RatingsClosedAt == nil || book.ReviewRequestSentAt != nil {
		return false
	}
	book.ReviewRequestDueAt = &dueAt
	book.ReviewRequestRetryAt = nil
	book.ReviewRequestClaimedAt = nil
	return true
}

func (book *ClubBook) CancelPendingReviewRequest() bool {
	if book.ReviewRequestSentAt != nil || book.ReviewRequestDueAt == nil {
		return false
	}
	book.ReviewRequestDueAt = nil
	book.ReviewRequestRetryAt = nil
	book.ReviewRequestClaimedAt = nil
	book.ReviewReminders = nil
	return true
}

func (book *ClubBook) ClaimReviewRequest(at time.Time, lease time.Duration) bool {
	if book.ReviewRequestDueAt == nil || book.ReviewRequestSentAt != nil || (book.ReviewRequestRetryAt != nil && book.ReviewRequestRetryAt.After(at)) || !claimAvailable(book.ReviewRequestClaimedAt, at, lease) {
		return false
	}
	book.ReviewRequestClaimedAt = &at
	return true
}

func (book *ClubBook) DeferReviewRequest(retryAt time.Time) {
	book.ReviewRequestClaimedAt = nil
	book.ReviewRequestRetryAt = &retryAt
}

func (book *ClubBook) ReleaseReviewRequestClaim() {
	book.ReviewRequestClaimedAt = nil
}

func (book *ClubBook) MarkReviewRequestSent(at time.Time, messageID int) {
	book.ReviewRequestSentAt = &at
	book.ReviewRequestDueAt = nil
	book.ReviewRequestRetryAt = nil
	book.ReviewRequestClaimedAt = nil
	book.ReviewRequestMsgID = messageID
}

func (book *ClubBook) ReviewCollectionOpen() bool {
	return book != nil && book.Status == StatusCompleted && (book.ReviewRequestDueAt != nil || book.ReviewRequestClaimedAt != nil || book.ReviewRequestSentAt != nil)
}

func (book *ClubBook) SetReviewReminder(userID int64, displayName string, username string, dueAt time.Time) error {
	if !book.ReviewCollectionOpen() {
		return errors.New("сбор отзывов ещё не начался")
	}
	if userID <= 0 {
		return errors.New("не удалось определить участника")
	}
	if book.ReviewByUser(userID) != nil {
		return errors.New("ваш отзыв уже сохранён")
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = "Участник"
	}
	username = strings.TrimPrefix(strings.TrimSpace(username), "@")
	for index := range book.ReviewReminders {
		if book.ReviewReminders[index].UserID == userID {
			book.ReviewReminders[index] = ReviewReminder{UserID: userID, DisplayName: displayName, Username: username, DueAt: dueAt}
			return nil
		}
	}
	book.ReviewReminders = append(book.ReviewReminders, ReviewReminder{UserID: userID, DisplayName: displayName, Username: username, DueAt: dueAt})
	return nil
}

func (book *ClubBook) CancelReviewReminder(userID int64) bool {
	for index := range book.ReviewReminders {
		if book.ReviewReminders[index].UserID != userID {
			continue
		}
		book.ReviewReminders = append(book.ReviewReminders[:index], book.ReviewReminders[index+1:]...)
		return true
	}
	return false
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
	SchemaVersion int `json:"schema_version,omitempty"`
	// Deprecated: in schema v3 this field is always false. It is kept only as
	// a poison pill so a rolled-back v2 binary refuses to rewrite v3 data.
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
	// MigrationComplete stays false in v3 so a rolled-back v2 binary refuses to
	// rewrite the file and discard fields it does not understand.
	return &ChatData{SchemaVersion: CurrentSchemaVersion, MigrationComplete: false, Books: []ClubBook{}}
}

func (cd *ChatData) IsLegacy() bool {
	return cd != nil && cd.SchemaVersion < CurrentSchemaVersion
}

func (cd *ChatData) IsFutureSchema() bool {
	return cd != nil && cd.SchemaVersion > CurrentSchemaVersion
}

func (cd *ChatData) HasBooksNeedingReview() bool {
	if cd == nil {
		return false
	}
	for _, book := range cd.Books {
		if book.NeedsReview {
			return true
		}
	}
	return false
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
	return cd.FinishCurrentBookWithReason(expectedID, status, nil, at)
}

func (cd *ChatData) FinishCurrentBookWithReason(expectedID string, status BookStatus, reason *UnfinishedReason, at time.Time) (*ClubBook, error) {
	if status != StatusCompleted && status != StatusUnfinished {
		return nil, errors.New("нужно выбрать: прочитали или не дочитали")
	}
	book := cd.CurrentBook()
	if book == nil || !strings.EqualFold(book.ID, expectedID) {
		return nil, errors.New("текущая книга уже изменилась")
	}
	if status == StatusUnfinished {
		if err := reason.Validate(); err != nil {
			return nil, err
		}
	}
	book.Status = status
	book.CompletedAt = nil
	book.StoppedAt = nil
	book.UnfinishedReason = nil
	if status == StatusCompleted {
		book.CompletedAt = &at
	} else {
		book.StoppedAt = &at
		book.UnfinishedReason = reason
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
	if old.SchemaVersion >= 2 {
		return nil, MigrationResult{}, errors.New("ожидалась схема v1")
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
	migrated.MigrationComplete = false
	if err := migrated.ValidateV2(); err != nil {
		return nil, result, err
	}
	return migrated, result, nil
}

func MigrateV2(old *ChatData) (*ChatData, MigrationResult, error) {
	if old == nil || old.SchemaVersion != 2 {
		return nil, MigrationResult{}, errors.New("ожидалась схема v2")
	}
	migrated := *old
	migrated.SchemaVersion = CurrentSchemaVersion
	// This is a compatibility guard: a rolled-back v2 binary sees schema v3 as
	// non-legacy, but refuses to write it while MigrationComplete is false.
	migrated.MigrationComplete = false
	result := MigrationResult{TotalBookCount: len(migrated.Books)}
	for _, book := range migrated.Books {
		switch book.Status {
		case StatusWishlist:
			result.WishlistCount++
		case StatusReading:
			result.CurrentCount++
		case StatusCompleted, StatusUnfinished:
			result.HistoryCount++
		}
		if book.NeedsReview {
			result.NeedsReview++
		}
	}
	if err := migrated.ValidateV2(); err != nil {
		return nil, result, err
	}
	return &migrated, result, nil
}

func (cd *ChatData) ValidateV2() error {
	if cd == nil || cd.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("неверная версия схемы")
	}
	if cd.MigrationComplete {
		return fmt.Errorf("migration_complete должен оставаться false в схеме v%d", CurrentSchemaVersion)
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
		if book.UnfinishedReason != nil {
			if book.Status != StatusUnfinished {
				return fmt.Errorf("книга %q содержит причину без статуса «не дочитали»", book.ID)
			}
			if err := book.UnfinishedReason.Validate(); err != nil {
				return fmt.Errorf("книга %q содержит некорректную причину: %w", book.ID, err)
			}
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
		if book.ReviewRequestDueAt != nil && (book.Status != StatusCompleted || book.RatingsClosedAt == nil || book.ReviewRequestSentAt != nil) {
			return fmt.Errorf("книга %q содержит некорректное ожидание запроса отзывов", book.ID)
		}
		if book.ReviewRequestClaimedAt != nil && (book.ReviewRequestDueAt == nil || book.ReviewRequestSentAt != nil) {
			return fmt.Errorf("книга %q содержит некорректный захват доставки запроса отзывов", book.ID)
		}
		if book.ReviewRequestRetryAt != nil && (book.ReviewRequestDueAt == nil || book.ReviewRequestSentAt != nil || book.Status != StatusCompleted || book.RatingsClosedAt == nil) {
			return fmt.Errorf("книга %q содержит некорректный retry запроса отзывов", book.ID)
		}
		if book.ReviewRequestSentAt != nil && book.Status != StatusCompleted {
			return fmt.Errorf("книга %q содержит запрос отзывов без завершённого чтения", book.ID)
		}
		reviewUsers := make(map[int64]struct{}, len(book.Reviews))
		for _, review := range book.Reviews {
			if book.Status != StatusCompleted || review.UserID <= 0 || strings.TrimSpace(review.Text) == "" {
				return fmt.Errorf("книга %q содержит некорректный отзыв", book.ID)
			}
			if _, exists := reviewUsers[review.UserID]; exists {
				return fmt.Errorf("книга %q содержит повторный отзыв участника %d", book.ID, review.UserID)
			}
			reviewUsers[review.UserID] = struct{}{}
		}
		reminderUsers := make(map[int64]struct{}, len(book.ReviewReminders))
		for _, reminder := range book.ReviewReminders {
			if !book.ReviewCollectionOpen() || reminder.UserID <= 0 || reminder.DueAt.IsZero() {
				return fmt.Errorf("книга %q содержит некорректное напоминание об отзыве", book.ID)
			}
			if _, exists := reviewUsers[reminder.UserID]; exists {
				return fmt.Errorf("книга %q содержит напоминание участнику с отзывом %d", book.ID, reminder.UserID)
			}
			if _, exists := reminderUsers[reminder.UserID]; exists {
				return fmt.Errorf("книга %q содержит повторное напоминание участнику %d", book.ID, reminder.UserID)
			}
			if reminder.DeliveryClaimedAt != nil && reminder.DueAt.IsZero() {
				return fmt.Errorf("книга %q содержит некорректный захват доставки напоминания", book.ID)
			}
			reminderUsers[reminder.UserID] = struct{}{}
		}
	}
	if reading > 1 {
		return fmt.Errorf("найдено текущих книг: %d", reading)
	}
	return nil
}
