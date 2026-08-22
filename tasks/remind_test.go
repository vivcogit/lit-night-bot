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
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 23, 7, 0, 0, 0, moscow)
	deadlineUTC := time.Date(2026, time.August, 23, 21, 0, 0, 0, time.UTC)
	if !reminderDue(now, deadlineUTC, 1, moscow) {
		t.Fatal("deadline instant must be compared as a Moscow calendar date")
	}
}
