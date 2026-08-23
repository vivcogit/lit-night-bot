package chatdata

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

func readV1Fixture(t *testing.T) *ChatData {
	t.Helper()
	raw, err := os.ReadFile("../testdata/chat-data-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var data ChatData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatal(err)
	}
	return &data
}

func TestMigrateV1Fixture(t *testing.T) {
	old := readV1Fixture(t)
	migrated, result, err := MigrateV1(old, time.Date(2026, 8, 22, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if migrated.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("schema = %d", migrated.SchemaVersion)
	}
	if result.TotalBookCount != 47 || result.HistoryCount != 43 || result.WishlistCount != 3 || result.CurrentCount != 1 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(migrated.BooksWithStatus(StatusCompleted)) != 43 {
		t.Fatal("completed books were not preserved")
	}
	if len(migrated.BooksWithStatus(StatusWishlist)) != 3 {
		t.Fatal("wishlist books were not preserved")
	}
	if len(migrated.BooksWithStatus(StatusReading)) != 1 {
		t.Fatal("current book was not preserved")
	}
	current := migrated.FindBook("cc000001")
	if current == nil || current.Deadline == nil || current.Deadline.Format(time.RFC3339) != "2026-09-15T00:00:00+03:00" {
		t.Fatalf("current book or deadline was not preserved: %#v", current)
	}
	if migrated.FindBook("26000001") == nil || migrated.FindBook("aa000003") == nil {
		t.Fatal("legacy UUIDs were not preserved")
	}
	if result.NeedsReview != 35 {
		t.Fatalf("cards requiring review = %d, want 35", result.NeedsReview)
	}
	if migrated.MigrationComplete {
		t.Fatal("migration must remain locked while cards need review")
	}
	if migrated.Wishlist != nil || migrated.History != nil || migrated.Current != nil {
		t.Fatal("legacy collections leaked into v2")
	}
	if err := migrated.ValidateV2(); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(migrated)
	if err != nil {
		t.Fatal(err)
	}
	for _, legacyKey := range [][]byte{[]byte(`"wishlist":`), []byte(`"history":`), []byte(`"current_book":`)} {
		if bytes.Contains(encoded, legacyKey) {
			t.Fatalf("legacy key leaked into v2 JSON: %s", legacyKey)
		}
	}
	if _, _, err := MigrateV1(migrated, time.Now()); err == nil {
		t.Fatal("repeated migration must fail")
	}
}

func TestParseStructuredBook(t *testing.T) {
	title, authors := ParseStructuredBook("Книга | Автор 1; Автор 2")
	if title != "Книга" || len(authors) != 2 || authors[1] != "Автор 2" {
		t.Fatalf("unexpected parsed book: %q %#v", title, authors)
	}
}

func TestParseLegacyNameTreatsTextAroundQuotesConservatively(t *testing.T) {
	title, authors, needsReview := parseLegacyName("Карантин в «Гранд-отеле» - Енэ Рейте")
	if title != "Карантин в «Гранд-отеле» - Енэ Рейте" || len(authors) != 0 || !needsReview {
		t.Fatalf("ambiguous quoted name parsed as %q %#v review=%v", title, authors, needsReview)
	}

	title, authors, needsReview = parseLegacyName("✔ «Название» Настоящий автор")
	if title != "Название" || len(authors) != 1 || authors[0] != "Настоящий автор" || needsReview {
		t.Fatalf("marked quoted name parsed as %q %#v review=%v", title, authors, needsReview)
	}
}

func TestMigrateV1RepairsDuplicateUUIDsWithoutDroppingDistinctBooks(t *testing.T) {
	now := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	old := &ChatData{
		Wishlist: []WishlistItem{
			{Book: Book{UUID: "duplicate", Name: "«Первая» Автор"}},
			{Book: Book{UUID: "DUPLICATE", Name: "«Первая» Автор"}},
		},
		History: []HistoryItem{{Book: Book{UUID: "duplicate", Name: "«Вторая» Автор"}, Date: now.Add(-24 * time.Hour)}},
		Current: &CurrentBook{Book: Book{UUID: "duplicate", Name: "«Третья» Автор"}, Deadline: now.Add(14 * 24 * time.Hour)},
	}

	migrated, result, err := MigrateV1(old, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(migrated.Books) != 3 || result.TotalBookCount != 3 {
		t.Fatalf("migrated books = %d, result = %+v", len(migrated.Books), result)
	}
	if result.DuplicatesSkipped != 1 || result.IDsReassigned != 2 {
		t.Fatalf("unexpected duplicate repair report: %+v", result)
	}
	if result.NeedsReview != 2 {
		t.Fatalf("reassigned cards requiring review = %d, want 2", result.NeedsReview)
	}
	if err := migrated.ValidateV2(); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateV2PreservesBooksAndBlocksOldBinaryWrites(t *testing.T) {
	now := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	v2 := &ChatData{
		SchemaVersion: 2, MigrationComplete: true,
		Chat: &ChatMetadata{ID: -42, Type: "group", Title: "Клуб"},
		Books: []ClubBook{{
			ID: "book0001", Title: "Книга", Status: StatusCompleted, CompletedAt: &now,
			Ratings: []Rating{{UserID: 1, DisplayName: "Анна", Value: 9}},
			Reviews: []Review{{ID: "review01", UserID: 1, DisplayName: "Анна", Text: "Отзыв", CreatedAt: now}},
		}},
	}
	migrated, result, err := MigrateV2(v2)
	if err != nil {
		t.Fatal(err)
	}
	if migrated.SchemaVersion != CurrentSchemaVersion || migrated.MigrationComplete || result.TotalBookCount != 1 {
		t.Fatalf("unexpected v3 migration: result=%+v data=%#v", result, migrated)
	}
	if migrated.Chat == nil || migrated.Chat.Title != "Клуб" || len(migrated.Books) != 1 || len(migrated.Books[0].Ratings) != 1 || len(migrated.Books[0].Reviews) != 1 {
		t.Fatalf("v2 data was not preserved: %#v", migrated)
	}
	// This is the exact gate used by the previous v2 binary. It must remain
	// false so rollback cannot rewrite v3 JSON and strip delivery fields.
	oldV2WouldAllowWrite := migrated.SchemaVersion >= 2 && migrated.MigrationComplete
	if oldV2WouldAllowWrite {
		t.Fatal("a rolled-back v2 binary would accept and rewrite v3 data")
	}
	if err := migrated.ValidateV2(); err != nil {
		t.Fatal(err)
	}
}
