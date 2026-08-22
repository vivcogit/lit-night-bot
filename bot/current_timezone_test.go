package bot

import (
	"testing"
	"time"
)

func TestDeadlineCalendarUsesApplicationLocationWhenRuntimeIsUTC(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = originalLocal })

	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 14, 4, 0, 0, 0, time.UTC)
	deadline, err := parseDeadlineDate("15.09.2026", now, moscow)
	if err != nil {
		t.Fatal(err)
	}
	if got := deadline.Format(time.RFC3339); got != "2026-09-15T00:00:00+03:00" {
		t.Fatalf("deadline = %s", got)
	}
	if got := automaticDeadline(now, moscow).Format(time.RFC3339); got != "2026-09-28T07:00:00+03:00" {
		t.Fatalf("automatic deadline = %s", got)
	}
}

func TestParseDeadlineRejectsPastMoscowDateAtUTCDateBoundary(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.September, 14, 21, 30, 0, 0, time.UTC)
	if _, err := parseDeadlineDate("14.09.2026", now, moscow); err != errDeadlineInPast {
		t.Fatalf("error = %v, want errDeadlineInPast", err)
	}
}
