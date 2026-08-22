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
