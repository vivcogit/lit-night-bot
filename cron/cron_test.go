package cron

import (
	"testing"
	"time"
)

func TestCronUsesApplicationLocation(t *testing.T) {
	moscow, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if got := newCron(moscow).Location(); got != moscow {
		t.Fatalf("cron location = %s, want %s", got, moscow)
	}
}
