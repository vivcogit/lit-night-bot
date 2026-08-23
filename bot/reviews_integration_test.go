package bot

import (
	"errors"
	"fmt"
	"io"
	chatdata "lit-night-bot/chat-data"
	chatio "lit-night-bot/io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type telegramRecorder struct {
	mu        sync.Mutex
	texts     []string
	failSends int
}

func (recorder *telegramRecorder) Do(request *http.Request) (*http.Response, error) {
	result := `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"test_bot"}}`
	if strings.HasSuffix(request.URL.Path, "/getMe") {
		return telegramHTTPResponse(request, result), nil
	}
	_ = request.ParseForm()
	text := request.FormValue("text")
	recorder.mu.Lock()
	if recorder.failSends > 0 {
		recorder.failSends--
		recorder.mu.Unlock()
		return nil, errors.New("telegram unavailable")
	}
	recorder.texts = append(recorder.texts, text)
	messageID := len(recorder.texts)
	recorder.mu.Unlock()
	result = fmt.Sprintf(`{"ok":true,"result":{"message_id":%d,"date":1,"chat":{"id":-42,"type":"group"},"text":%q}}`, messageID, text)
	return telegramHTTPResponse(request, result), nil
}

func telegramHTTPResponse(request *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    request,
	}
}

func (recorder *telegramRecorder) snapshot() []string {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	return append([]string(nil), recorder.texts...)
}

func newReviewIntegrationBot(t *testing.T) (*LitNightBot, *chatio.IoChatData, *telegramRecorder) {
	t.Helper()
	recorder := &telegramRecorder{}
	api, err := tgbotapi.NewBotAPIWithClient("test", "https://telegram.invalid/bot%s/%s", recorder)
	if err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	entry := logrus.NewEntry(logger)
	storage := chatio.NewIOChatData(entry, t.TempDir())
	return &LitNightBot{bot: api, iocd: storage, logger: entry, location: time.UTC}, storage, recorder
}

func scheduledReviewData(now time.Time) *chatdata.ChatData {
	data := chatdata.NewChatData()
	data.Chat = &chatdata.ChatMetadata{ID: -42, Type: "group", Title: "Клуб"}
	closedAt := now
	dueAt := now.Add(reviewRequestDelay)
	data.Books = append(data.Books, chatdata.ClubBook{
		ID: "done0001", Title: "Обсуждённая", Status: chatdata.StatusCompleted,
		RatingsClosedAt: &closedAt, RatingsClosedBy: 1, ReviewRequestDueAt: &dueAt,
	})
	return data
}

func TestDueReviewRequestIsSentOnceAfterFifteenMinutes(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(14*time.Minute), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("request sent too early: %d", got)
	}
	lnb.ProcessDueReviews(now.Add(15*time.Minute), lnb.logger)
	lnb.ProcessDueReviews(now.Add(16*time.Minute), lnb.logger)
	texts := recorder.snapshot()
	if len(texts) != 1 || !strings.Contains(texts[0], "Короткое послесловие") {
		t.Fatalf("request must be sent exactly once: %#v", texts)
	}
	saved := storage.GetChatData(-42).FindBook("done0001")
	if saved.ReviewRequestSentAt == nil || saved.ReviewRequestDueAt != nil || saved.ReviewRequestClaimedAt != nil {
		t.Fatalf("delivery state was not finalized: %#v", saved)
	}
}

func TestChoosingNextBookFlushesPendingReviewImmediately(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	data.Books = append(data.Books, chatdata.ClubBook{ID: "wish0001", Title: "Следующая", Status: chatdata.StatusWishlist})
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	if !lnb.setCurrentBook(-42, data, "wish0001", lnb.logger) {
		t.Fatal("next book was not selected")
	}
	texts := recorder.snapshot()
	if len(texts) != 2 || !strings.Contains(texts[0], "Текущая книга") || !strings.Contains(texts[1], "Короткое послесловие") {
		t.Fatalf("unexpected immediate flow: %#v", texts)
	}
	lnb.ProcessDueReviews(now.Add(24*time.Hour), lnb.logger)
	if got := len(recorder.snapshot()); got != 2 {
		t.Fatalf("immediate request was duplicated by cron: %d messages", got)
	}
}

func TestAddingWishlistBookAloneDoesNotFlushReview(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	data.Books = append(data.Books, chatdata.ClubBook{ID: "wish0001", Title: "Добавленная", Status: chatdata.StatusWishlist})
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(10*time.Minute), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("adding a wishlist book sent the review request: %d", got)
	}
}

func TestConcurrentReviewProcessorsSerializePerChat(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	var wait sync.WaitGroup
	wait.Add(2)
	for range 2 {
		go func() {
			defer wait.Done()
			lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
		}()
	}
	wait.Wait()
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("concurrent processors sent %d requests, want 1", got)
	}
}

func TestClaimedDeliveryIsNotRetriedAfterRestart(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	book := data.FindBook("done0001")
	if !book.ClaimReviewRequest(now.Add(reviewRequestDelay)) {
		t.Fatal("request was not claimed")
	}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(24*time.Hour), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("uncertain claimed delivery was duplicated: %d", got)
	}
}

func TestTelegramFailureReleasesClaimForRetry(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failSends = 1
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	afterFailure := storage.GetChatData(-42).FindBook("done0001")
	if afterFailure.ReviewRequestClaimedAt != nil || afterFailure.ReviewRequestDueAt == nil || afterFailure.ReviewRequestSentAt != nil {
		t.Fatalf("failed delivery was not released for retry: %#v", afterFailure)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay+time.Minute), lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("released delivery was not retried exactly once: %d", got)
	}
}

func TestParticipantReminderDoesNotAffectAnotherReview(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	book := data.FindBook("done0001")
	book.MarkReviewRequestSent(now, 10)
	book.Reviews = []chatdata.Review{{ID: "review01", UserID: 1, DisplayName: "Анна", Text: "Готово", CreatedAt: now}}
	book.ReviewReminders = []chatdata.ReviewReminder{{UserID: 2, DisplayName: "Борис", DueAt: now}}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now, lnb.logger)
	texts := recorder.snapshot()
	if len(texts) != 1 || !strings.Contains(texts[0], `tg://user?id=2`) || strings.Contains(texts[0], `tg://user?id=1`) {
		t.Fatalf("reminder was not addressed only to Boris: %#v", texts)
	}
	saved := storage.GetChatData(-42).FindBook("done0001")
	if len(saved.Reviews) != 1 || saved.Reviews[0].UserID != 1 || len(saved.ReviewReminders) != 0 {
		t.Fatalf("independent participant state was corrupted: %#v", saved)
	}
}

func TestCurrentSchemaWithRollbackGuardIsAcceptedAndFutureSchemaRejected(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	current := chatdata.NewChatData()
	if current.MigrationComplete {
		t.Fatal("rollback guard must remain false")
	}
	if err := storage.SaveChatData(-42, current); err != nil {
		t.Fatal(err)
	}
	update := &tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: -42, Type: "group", Title: "Клуб"}}}
	if !lnb.allowUpdate(update, lnb.logger) {
		t.Fatal("current v3 data was blocked by the legacy rollback guard")
	}

	future := chatdata.NewChatData()
	future.SchemaVersion = chatdata.CurrentSchemaVersion + 1
	if err := storage.SaveChatData(-42, future); err != nil {
		t.Fatal(err)
	}
	if lnb.allowUpdate(update, lnb.logger) {
		t.Fatal("future schema was accepted")
	}
	texts := recorder.snapshot()
	if len(texts) != 1 || !strings.Contains(texts[0], "более новой версией") {
		t.Fatalf("future-schema explanation was not sent: %#v", texts)
	}
}
