package tasks

import (
	"testing"
	"time"
)

func TestReminderDueUsesCalendarDatesInConfiguredLocation(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 23, 7, 0, 0, 0, moscow)

	if !reminderDue(now, time.Date(2026, time.August, 24, 0, 0, 0, 0, moscow), 1, moscow) {
		t.Fatal("tomorrow's local midnight must trigger the one-day reminder")
	}
	if reminderDue(now, time.Date(2026, time.August, 25, 0, 0, 0, 0, moscow), 1, moscow) {
		t.Fatal("a deadline two calendar days away must not trigger the one-day reminder")
	}
	if !reminderDue(now, time.Date(2026, time.August, 30, 23, 30, 0, 0, moscow), 7, moscow) {
		t.Fatal("time of day must not affect a seven-day calendar reminder")
	}
}

func TestReminderDueNormalizesDeadlineToConfiguredLocation(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = originalLocal })

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	deadline, err := time.Parse(time.RFC3339, "2026-09-15T00:00:00+03:00")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 14, 4, 0, 0, 0, time.UTC)
	if !reminderDue(now, deadline, 1, moscow) {
		t.Fatal("a migrated +03:00 deadline must trigger one day earlier in the Moscow calendar")
	}
	if reminderDue(now.AddDate(0, 0, -1), deadline, 1, moscow) {
		t.Fatal("the reminder must not trigger two Moscow calendar days before the deadline")
	}
}
