package main

import (
	"encoding/json"
	chatdata "lit-night-bot/chat-data"
	"os"
	"testing"
	"time"
)

func TestV1FixtureMatchesCurrentSchema(t *testing.T) {
	raw, err := os.ReadFile("testdata/chat-data-v1.json")
	if err != nil {
		t.Fatalf("read v1 fixture: %v", err)
	}

	var data chatdata.ChatData
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("unmarshal v1 fixture: %v", err)
	}

	if got, want := len(data.History), 43; got != want {
		t.Fatalf("history length = %d, want %d", got, want)
	}
	if got, want := len(data.Wishlist), 3; got != want {
		t.Fatalf("wishlist length = %d, want %d", got, want)
	}
	if data.Current == nil || data.Current.Book.UUID != "cc000001" {
		t.Fatalf("current book = %#v, want UUID cc000001", data.Current)
	}
	if got, want := data.Current.Deadline.Format(time.RFC3339), "2026-09-15T00:00:00+03:00"; got != want {
		t.Fatalf("current deadline = %q, want %q", got, want)
	}

	wantYears := map[int]int{2024: 18, 2025: 15, 2026: 10}
	gotYears := make(map[int]int)
	seenUUIDs := make(map[string]struct{}, len(data.History))

	for _, item := range data.History {
		if item.Book.Name == "" {
			t.Fatal("history contains a book with an empty name")
		}
		if item.Book.UUID == "" {
			t.Fatalf("book %q has an empty UUID", item.Book.Name)
		}
		if _, exists := seenUUIDs[item.Book.UUID]; exists {
			t.Fatalf("duplicate UUID %q", item.Book.UUID)
		}
		seenUUIDs[item.Book.UUID] = struct{}{}
		gotYears[item.Date.Year()]++
	}
	for _, item := range data.Wishlist {
		if item.Book.Name == "" {
			t.Fatal("wishlist contains a book with an empty name")
		}
		if item.Book.UUID == "" {
			t.Fatalf("wishlist book %q has an empty UUID", item.Book.Name)
		}
		if _, exists := seenUUIDs[item.Book.UUID]; exists {
			t.Fatalf("duplicate UUID %q", item.Book.UUID)
		}
		seenUUIDs[item.Book.UUID] = struct{}{}
	}
	if _, exists := seenUUIDs[data.Current.Book.UUID]; exists {
		t.Fatalf("duplicate UUID %q", data.Current.Book.UUID)
	}

	for year, want := range wantYears {
		if got := gotYears[year]; got != want {
			t.Errorf("history count for %d = %d, want %d", year, got, want)
		}
	}
}
