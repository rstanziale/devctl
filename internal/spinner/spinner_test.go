package spinner

import (
	"bytes"
	"testing"
	"time"
)

func TestStartAndStopRenderStatusAndClearLine(t *testing.T) {
	var output bytes.Buffer
	spinner := newSpinner(&output, true, time.Hour)

	spinner.Start()
	spinner.Stop()

	if got, want := output.String(), "\rGenerating...\r\033[2K"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestDisabledSpinnerWritesNothing(t *testing.T) {
	var output bytes.Buffer
	spinner := New(&output, false)

	spinner.Start()
	spinner.Stop()

	if got := output.String(); got != "" {
		t.Fatalf("output = %q, want no output", got)
	}
}

func TestIsTerminalRejectsNonFileWriter(t *testing.T) {
	if IsTerminal(&bytes.Buffer{}) {
		t.Fatal("expected buffer not to be a terminal")
	}
}
