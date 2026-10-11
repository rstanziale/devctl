package timing

import (
	"testing"
	"time"
)

func TestElapsedBetween(t *testing.T) {
	start := time.Date(2024, time.April, 27, 14, 30, 0, 0, time.UTC)
	end := start.Add(1250 * time.Millisecond)

	got := elapsedBetween(start, end)
	want := 1250 * time.Millisecond

	if got != want {
		t.Fatalf("unexpected elapsed\nwant: %s\ngot: %s", want, got)
	}
}

func TestElapsedBetweenRoundsMilliseconds(t *testing.T) {
	start := time.Date(2024, time.April, 27, 14, 30, 0, 0, time.UTC)
	end := start.Add(1250123 * time.Microsecond)

	got := elapsedBetween(start, end)
	want := 1250 * time.Millisecond

	if got != want {
		t.Fatalf("unexpected rounded elapsed\nwant: %s\ngot: %s", want, got)
	}
}
