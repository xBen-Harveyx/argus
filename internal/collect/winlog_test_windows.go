//go:build windows

package collect

import (
	"testing"
	"time"
)

func TestFormatEventTime(t *testing.T) {
	ts := time.Date(2026, 4, 19, 14, 30, 45, 123000000, time.FixedZone("EDT", -4*3600))
	got := formatEventTime(ts)
	want := "2026-04-19T18:30:45.123Z"
	if got != want {
		t.Fatalf("formatEventTime() = %q, want %q", got, want)
	}
}
