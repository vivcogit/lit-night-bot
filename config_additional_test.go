package main

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetConfig(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DATA_PATH", "/tmp/lit-night-test")
	t.Setenv("DEBUG", "1")
	t.Setenv("TIMEZONE", "")
	config := GetConfig()
	if config.token != "test-token" || config.dataPath != "/tmp/lit-night-test" || !config.isDebug || config.location.String() != defaultTimezone {
		t.Fatalf("config = %#v", config)
	}
}

func TestGetConfigTimezoneOverride(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DATA_PATH", "/tmp/lit-night-test")
	t.Setenv("TIMEZONE", "Asia/Yekaterinburg")
	config := GetConfig()
	if config.location.String() != "Asia/Yekaterinburg" {
		t.Fatalf("location = %s", config.location)
	}
}

func TestGetConfigPanicsForInvalidTimezone(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	t.Setenv("DATA_PATH", "/tmp/lit-night-test")
	t.Setenv("TIMEZONE", "Invalid/Timezone")
	defer func() {
		if recover() == nil {
			t.Fatal("GetConfig() must panic for an invalid timezone")
		}
	}()
	GetConfig()
}

func TestGetConfigPanicsForMissingRequiredValues(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		dataPath string
	}{
		{name: "token", dataPath: "/tmp/data"},
		{name: "data path", token: "token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("TELEGRAM_BOT_TOKEN", test.token)
			t.Setenv("DATA_PATH", test.dataPath)
			defer func() {
				if recover() == nil {
					t.Fatal("GetConfig() must panic")
				}
			}()
			GetConfig()
		})
	}
}

func TestGetLoggerModes(t *testing.T) {
	debug := getLogger(true)
	if debug.Logger.Level != logrus.DebugLevel {
		t.Fatalf("debug level = %v", debug.Logger.Level)
	}
	production := getLogger(false)
	if production.Logger.Level != logrus.InfoLevel {
		t.Fatalf("production level = %v", production.Logger.Level)
	}
}
