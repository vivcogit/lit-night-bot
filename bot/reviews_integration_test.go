package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	chatdata "lit-night-bot/chat-data"
	chatio "lit-night-bot/io"
	"lit-night-bot/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

type telegramRecorder struct {
	mu          sync.Mutex
	texts       []string
	failSends   int
	failAPI     int
	failCode    int
	retryAfter  int
	deletions   int
	onFirstSend func()
}

func (recorder *telegramRecorder) Do(request *http.Request) (*http.Response, error) {
	result := `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test","username":"test_bot"}}`
	if strings.HasSuffix(request.URL.Path, "/getMe") {
		return telegramHTTPResponse(request, result), nil
	}
	_ = request.ParseForm()
	if strings.HasSuffix(request.URL.Path, "/deleteMessage") {
		recorder.mu.Lock()
		recorder.deletions++
		recorder.mu.Unlock()
		return telegramHTTPResponse(request, `{"ok":true,"result":true}`), nil
	}
	text := request.FormValue("text")
	recorder.mu.Lock()
	if recorder.failSends > 0 {
		recorder.failSends--
		recorder.mu.Unlock()
		return nil, errors.New("telegram unavailable")
	}
	if recorder.failAPI > 0 {
		recorder.failAPI--
		code := recorder.failCode
		if code == 0 {
			code = 500
		}
		retryAfter := recorder.retryAfter
		recorder.mu.Unlock()
		result := fmt.Sprintf(`{"ok":false,"error_code":%d,"description":"Telegram error","parameters":{"retry_after":%d}}`, code, retryAfter)
		return telegramHTTPResponse(request, result), nil
	}
	recorder.texts = append(recorder.texts, text)
	messageID := len(recorder.texts)
	hook := recorder.onFirstSend
	recorder.onFirstSend = nil
	recorder.mu.Unlock()
	if hook != nil {
		hook()
	}
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

func countTextsContaining(texts []string, fragment string) int {
	count := 0
	for _, text := range texts {
		if strings.Contains(text, fragment) {
			count++
		}
	}
	return count
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

func TestDeleteMessageAcceptsBooleanTelegramResponse(t *testing.T) {
	lnb, _, recorder := newReviewIntegrationBot(t)
	if err := lnb.removeMessage(-42, 10); err != nil {
		t.Fatalf("successful deleteMessage returned an error: %v", err)
	}
	if recorder.deletions != 1 {
		t.Fatalf("delete requests = %d, want 1", recorder.deletions)
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

func TestClaimedDeliveryIsRecoveredAfterLease(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	book := data.FindBook("done0001")
	claimedAt := now.Add(reviewRequestDelay)
	if !book.ClaimReviewRequest(claimedAt, reviewDeliveryClaimLease) {
		t.Fatal("request was not claimed")
	}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(claimedAt.Add(reviewDeliveryClaimLease-time.Second), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("fresh claim was retried before its lease expired: %d", got)
	}
	lnb.ProcessDueReviews(claimedAt.Add(reviewDeliveryClaimLease), lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("expired claim was not recovered: %d", got)
	}
	if saved := storage.GetChatData(-42).FindBook("done0001"); saved.ReviewRequestSentAt == nil || saved.ReviewRequestClaimedAt != nil {
		t.Fatalf("recovered request was not finalized: %#v", saved)
	}
}

func TestTelegramTransportFailureWaitsForLeaseBeforeRetry(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failSends = 1
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	afterFailure := storage.GetChatData(-42).FindBook("done0001")
	if afterFailure.ReviewRequestClaimedAt == nil || afterFailure.ReviewRequestDueAt == nil || afterFailure.ReviewRequestSentAt != nil {
		t.Fatalf("ambiguous delivery did not retain its claim: %#v", afterFailure)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay+reviewDeliveryRetryBackoff), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("ambiguous delivery was retried before lease expiry: %d", got)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay+reviewDeliveryClaimLease), lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("ambiguous delivery was not recovered after lease: %d", got)
	}
}

func TestTelegramAPIRejectionReleasesClaimForRetry(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failAPI = 1
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	if err := data.FindBook("done0001").SetReviewReminder(2, "Борис", "", now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	afterFailure := storage.GetChatData(-42).FindBook("done0001")
	if afterFailure.ReviewRequestClaimedAt != nil || afterFailure.ReviewRequestSentAt != nil {
		t.Fatalf("definitively rejected delivery kept its claim: %#v", afterFailure)
	}
	if len(afterFailure.ReviewReminders) != 1 {
		t.Fatalf("retryable failure discarded an existing reminder: %#v", afterFailure.ReviewReminders)
	}
	if err := storage.GetChatData(-42).ValidateV2(); err != nil {
		t.Fatalf("retryable transition produced invalid data: %v", err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay+reviewDeliveryRetryBackoff), lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("definitively rejected delivery was not retried: %d", got)
	}
}

func TestTelegramRateLimitDefersRetry(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failAPI = 1
	recorder.failCode = 429
	recorder.retryAfter = 600
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	failedAt := now.Add(reviewRequestDelay)
	lnb.ProcessDueReviews(failedAt, lnb.logger)
	deferred := storage.GetChatData(-42).FindBook("done0001")
	if deferred.ReviewRequestClaimedAt != nil || deferred.ReviewRequestDueAt == nil || deferred.ReviewRequestRetryAt == nil || !deferred.ReviewRequestRetryAt.Equal(failedAt.Add(10*time.Minute)) {
		t.Fatalf("rate-limited request was not deferred: %#v", deferred)
	}
	if _, usable := lnb.sendPendingReviewRequests(-42, storage.GetChatData(-42), failedAt.Add(time.Minute), false, lnb.logger); !usable {
		t.Fatal("immediate flush invalidated the snapshot")
	}
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("immediate flush bypassed RetryAfter: %d", got)
	}
	lnb.ProcessDueReviews(failedAt.Add(9*time.Minute), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("rate-limited request retried too early: %d", got)
	}
	lnb.ProcessDueReviews(failedAt.Add(10*time.Minute), lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("rate-limited request was not retried: %d", got)
	}
}

func TestRateLimitStartsWhenTelegramResponds(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failAPI = 1
	recorder.failCode = 429
	recorder.retryAfter = 600
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	failedAt := now.Add(reviewRequestDelay)
	responseAt := failedAt.Add(2 * time.Minute)
	lnb.reviewNow = func() time.Time { return responseAt }
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(failedAt, lnb.logger)
	deferred := storage.GetChatData(-42).FindBook("done0001")
	if deferred.ReviewRequestRetryAt == nil || !deferred.ReviewRequestRetryAt.Equal(responseAt.Add(10*time.Minute)) {
		t.Fatalf("RetryAfter starts at %#v, want response time %s", deferred.ReviewRequestRetryAt, responseAt)
	}
}

func TestRateLimitFallbackIsIsolatedByDelivery(t *testing.T) {
	lnb, _, _ := newReviewIntegrationBot(t)
	at := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	requestA := reviewDeliveryKey{chatID: -42, bookID: "book-a"}
	requestB := reviewDeliveryKey{chatID: -43, bookID: "book-b"}
	reminderA := reviewDeliveryKey{chatID: -42, bookID: "book-a", userID: 7}
	lnb.rememberAutomatedReviewRetry(requestA, at.Add(time.Hour))
	if lnb.automatedReviewDeliveryAllowed(requestA, at) {
		t.Fatal("rate-limited request was allowed")
	}
	if !lnb.automatedReviewDeliveryAllowed(requestB, at) {
		t.Fatal("independent chat was throttled")
	}
	if !lnb.automatedReviewDeliveryAllowed(reminderA, at) {
		t.Fatal("independent participant reminder was throttled")
	}
}

func TestRateLimitFallbackSurvivesRetryStateSaveFailure(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failAPI = 1
	recorder.failCode = 429
	recorder.retryAfter = 3600
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	if err := storage.SaveChatData(-42, scheduledReviewData(now)); err != nil {
		t.Fatal(err)
	}
	saveCalls := 0
	lnb.reviewStateSaver = func(chatID int64, data *chatdata.ChatData) error {
		saveCalls++
		if saveCalls >= 2 && saveCalls <= 4 {
			return errors.New("storage temporarily unavailable")
		}
		return storage.SaveChatData(chatID, data)
	}
	failedAt := now.Add(reviewRequestDelay)
	lnb.ProcessDueReviews(failedAt, lnb.logger)
	persisted := storage.GetChatData(-42).FindBook("done0001")
	if persisted.ReviewRequestClaimedAt == nil || persisted.ReviewRequestRetryAt != nil {
		t.Fatalf("disk must retain only the pre-send claim: %#v", persisted)
	}
	lnb.ProcessDueReviews(failedAt.Add(reviewDeliveryClaimLease), lnb.logger)
	lnb.ProcessDueReviews(failedAt.Add(59*time.Minute), lnb.logger)
	if got := countTextsContaining(recorder.snapshot(), "Короткое послесловие"); got != 0 {
		t.Fatalf("request retried before RetryAfter despite failed persistence: %d", got)
	}
	lnb.ProcessDueReviews(failedAt.Add(time.Hour), lnb.logger)
	if got := countTextsContaining(recorder.snapshot(), "Короткое послесловие"); got != 1 {
		t.Fatalf("request was not retried at RetryAfter: %d", got)
	}
}

func TestPermanentTelegramRejectionCancelsDelivery(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	recorder.failAPI = 1
	recorder.failCode = 403
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	if err := data.FindBook("done0001").SetReviewReminder(2, "Борис", "", now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	cancelled := storage.GetChatData(-42).FindBook("done0001")
	if cancelled.ReviewRequestClaimedAt != nil || cancelled.ReviewRequestDueAt != nil || cancelled.ReviewRequestSentAt != nil {
		t.Fatalf("permanently rejected delivery remained pending: %#v", cancelled)
	}
	if len(cancelled.ReviewReminders) != 0 {
		t.Fatalf("permanently rejected delivery kept orphan reminders: %#v", cancelled.ReviewReminders)
	}
	lnb.ProcessDueReviews(now.Add(24*time.Hour), lnb.logger)
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("permanently rejected delivery retried: %d", got)
	}
}

func TestExpiredReminderClaimIsRecovered(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	book := data.FindBook("done0001")
	book.MarkReviewRequestSent(now, 10)
	claimedAt := now.Add(-reviewDeliveryClaimLease)
	book.ReviewReminders = []chatdata.ReviewReminder{{UserID: 2, DisplayName: "Борис", DueAt: now.Add(-time.Hour), DeliveryClaimedAt: &claimedAt}}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now, lnb.logger)
	if got := len(recorder.snapshot()); got != 1 {
		t.Fatalf("expired reminder claim was not recovered: %d", got)
	}
	if saved := storage.GetChatData(-42).FindBook("done0001"); len(saved.ReviewReminders) != 0 {
		t.Fatalf("recovered reminder was not finalized: %#v", saved.ReviewReminders)
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

func TestReviewProcessorDoesNotRewriteFutureSchema(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	raw, err := json.Marshal(scheduledReviewData(now))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	document["schema_version"] = chatdata.CurrentSchemaVersion + 1
	document["future_only"] = map[string]any{"must_survive": true}
	raw, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	path := storage.GetChatDataFilePath(-42)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, raw) {
		t.Fatalf("future schema was rewritten:\nbefore=%s\nafter=%s", raw, after)
	}
	if got := len(recorder.snapshot()); got != 0 {
		t.Fatalf("future schema triggered %d Telegram messages", got)
	}
}

func TestFinalPersistenceFailureAbortsChatSnapshot(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	second := *data.FindBook("done0001")
	second.ID = "done0002"
	second.Title = "Вторая"
	data.Books = append(data.Books, second)
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	dataDir := filepath.Dir(storage.GetChatDataFilePath(-42))
	movedDir := dataDir + "-offline"
	recorder.onFirstSend = func() {
		if err := os.Rename(dataDir, movedDir); err != nil {
			t.Errorf("move storage: %v", err)
			return
		}
		if err := os.WriteFile(dataDir, []byte("blocked"), 0o600); err != nil {
			t.Errorf("block storage: %v", err)
		}
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	if err := os.Remove(dataDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(movedDir, dataDir); err != nil {
		t.Fatal(err)
	}
	saved := storage.GetChatData(-42)
	first := saved.FindBook("done0001")
	secondSaved := saved.FindBook("done0002")
	if first.ReviewRequestSentAt != nil || first.ReviewRequestClaimedAt == nil {
		t.Fatalf("failed delivery poisoned persisted state: %#v", first)
	}
	if secondSaved.ReviewRequestSentAt != nil || secondSaved.ReviewRequestClaimedAt != nil {
		t.Fatalf("processor continued with stale snapshot: %#v", secondSaved)
	}
	if recorder.deletions != 1 {
		t.Fatalf("compensation deletions = %d, want 1", recorder.deletions)
	}
}

func TestPostCommitDurabilityFailureDoesNotDeleteDeliveredRequest(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	now := time.Date(2026, 8, 23, 18, 0, 0, 0, time.UTC)
	data := scheduledReviewData(now)
	second := *data.FindBook("done0001")
	second.ID = "done0002"
	second.Title = "Вторая"
	data.Books = append(data.Books, second)
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	saveCalls := 0
	lnb.reviewStateSaver = func(chatID int64, state *chatdata.ChatData) error {
		saveCalls++
		if err := storage.SaveChatData(chatID, state); err != nil {
			return err
		}
		if saveCalls >= 2 {
			return &utils.PostCommitDurabilityError{Err: errors.New("directory sync failed")}
		}
		return nil
	}
	lnb.ProcessDueReviews(now.Add(reviewRequestDelay), lnb.logger)
	saved := storage.GetChatData(-42)
	first := saved.FindBook("done0001")
	secondSaved := saved.FindBook("done0002")
	if first.ReviewRequestSentAt == nil {
		t.Fatalf("committed request lost sent marker: %#v", first)
	}
	if secondSaved.ReviewRequestSentAt != nil || secondSaved.ReviewRequestClaimedAt != nil {
		t.Fatalf("processor continued after uncertain durability: %#v", secondSaved)
	}
	if recorder.deletions != 0 {
		t.Fatalf("committed Telegram request was compensated with %d deletions", recorder.deletions)
	}
	if got := countTextsContaining(recorder.snapshot(), "Короткое послесловие"); got != 1 {
		t.Fatalf("unexpected review request count: %d", got)
	}
}

func TestPendingCardReviewBlocksOrdinaryActionsButAllowsReviewFlow(t *testing.T) {
	lnb, storage, recorder := newReviewIntegrationBot(t)
	data := chatdata.NewChatData()
	data.Chat = &chatdata.ChatMetadata{ID: -42, Type: "group", Title: "Клуб"}
	data.Books = []chatdata.ClubBook{
		{ID: "review01", Title: "Проверить", Status: chatdata.StatusWishlist, NeedsReview: true},
		{ID: "normal01", Title: "Обычная", Status: chatdata.StatusWishlist},
	}
	if err := storage.SaveChatData(-42, data); err != nil {
		t.Fatal(err)
	}
	chat := &tgbotapi.Chat{ID: -42, Type: "group", Title: "Клуб"}
	blocked := &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "blocked", Data: GetCallbackParamStr(CBRatingOpen, "review01"), Message: &tgbotapi.Message{Chat: chat}}}
	if lnb.allowUpdate(blocked, lnb.logger) {
		t.Fatal("ordinary action was allowed before migrated cards were reviewed")
	}
	texts := recorder.snapshot()
	if len(texts) == 0 || !strings.Contains(texts[len(texts)-1], "проверьте карточки") {
		t.Fatalf("missing card-review explanation: %#v", texts)
	}
	allowed := &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "allowed", Data: GetCallbackParamStr(CBBooksReview, "0"), Message: &tgbotapi.Message{Chat: chat}}}
	if !lnb.allowUpdate(allowed, lnb.logger) {
		t.Fatal("card-review action was blocked by the migration gate")
	}
	staleCallback := &tgbotapi.Update{CallbackQuery: &tgbotapi.CallbackQuery{ID: "stale", Data: GetCallbackParamStr(CBBookEditTitle, "normal01"), Message: &tgbotapi.Message{Chat: chat}}}
	if lnb.allowUpdate(staleCallback, lnb.logger) {
		t.Fatal("stale callback edited a book that does not need migration review")
	}
	staleReply := &tgbotapi.Update{Message: &tgbotapi.Message{Chat: chat, Text: "Новое", ReplyToMessage: &tgbotapi.Message{Text: "Введите название.\n\nbook_title:normal01:10"}}}
	if lnb.allowUpdate(staleReply, lnb.logger) {
		t.Fatal("stale reply edited a book that does not need migration review")
	}
}
