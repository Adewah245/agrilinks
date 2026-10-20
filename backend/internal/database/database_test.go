package database

import (
	"testing"
	"time"
)

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback int
		want     int
	}{
		{name: "unset", fallback: 20, want: 20},
		{name: "valid", value: "35", fallback: 20, want: 35},
		{name: "invalid", value: "many", fallback: 20, want: 20},
		{name: "below minimum", value: "0", fallback: 20, want: 20},
		{name: "above maximum", value: "500", fallback: 20, want: 200},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("AGRO_SHIELD_TEST_INT", test.value)

			if got := envInt("AGRO_SHIELD_TEST_INT", test.fallback, 1, 200); got != test.want {
				t.Fatalf("envInt() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		fallback time.Duration
		want     time.Duration
	}{
		{name: "unset", fallback: 30 * time.Minute, want: 30 * time.Minute},
		{name: "valid", value: "45m", fallback: 30 * time.Minute, want: 45 * time.Minute},
		{name: "invalid", value: "soon", fallback: 30 * time.Minute, want: 30 * time.Minute},
		{name: "not positive", value: "0s", fallback: 30 * time.Minute, want: 30 * time.Minute},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("AGRO_SHIELD_TEST_DURATION", test.value)

			if got := envDuration("AGRO_SHIELD_TEST_DURATION", test.fallback); got != test.want {
				t.Fatalf("envDuration() = %s, want %s", got, test.want)
			}
		})
	}
}

func TestConnectDBRejectsEmptyURL(t *testing.T) {
	if _, err := ConnectDB("   "); err == nil {
		t.Fatal("ConnectDB() accepted an empty database URL")
	}
}
