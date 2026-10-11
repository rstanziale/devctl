package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestInfoGoesToStdoutWithComponent(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	SetOutput(&out, &errOut)
	t.Cleanup(func() {
		SetOutput(nil, nil)
	})

	Init(0)
	log := GetLogger(WithComponent("Runner"))
	log.Info("command completed", "name", "doctor")

	stdout := out.String()
	if !strings.Contains(stdout, "[INFO]") {
		t.Fatalf("expected INFO level, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[Runner] command completed") {
		t.Fatalf("expected component and message, got: %s", stdout)
	}
	if !strings.Contains(stdout, "name=doctor") {
		t.Fatalf("expected structured field, got: %s", stdout)
	}
	if errOut.Len() != 0 {
		t.Fatalf("expected no stderr output, got: %s", errOut.String())
	}
}

func TestErrorGoesToStderr(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	SetOutput(&out, &errOut)
	t.Cleanup(func() {
		SetOutput(nil, nil)
	})

	Init(0)
	log := GetLogger(WithComponent("Runner"))
	log.Error("command failed", "name", "bootstrap")

	if out.Len() != 0 {
		t.Fatalf("expected no stdout output, got: %s", out.String())
	}

	stderr := errOut.String()
	if !strings.Contains(stderr, "[ERROR]") {
		t.Fatalf("expected ERROR level, got: %s", stderr)
	}
	if !strings.Contains(stderr, "[Runner] command failed") {
		t.Fatalf("expected component and message, got: %s", stderr)
	}
}

func TestVerboseLevelGatesDebugAndTrace(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer
	SetOutput(&out, &errOut)
	t.Cleanup(func() {
		SetOutput(nil, nil)
	})

	log := GetLogger(WithComponent("DoctorCommand"))

	Init(0)
	log.Debug("hidden debug")
	log.Trace("hidden trace")
	if out.Len() != 0 {
		t.Fatalf("expected no stdout output at info level, got: %s", out.String())
	}

	Init(1)
	log.Debug("visible debug")
	stdoutAtDebug := out.String()
	if !strings.Contains(stdoutAtDebug, "[DEBUG]") || !strings.Contains(stdoutAtDebug, "visible debug") {
		t.Fatalf("expected visible debug log, got: %s", stdoutAtDebug)
	}
	if strings.Contains(stdoutAtDebug, "hidden trace") {
		t.Fatalf("did not expect trace at -v verbosity, got: %s", stdoutAtDebug)
	}

	Init(2)
	log.Trace("visible trace")
	stdoutAtTrace := out.String()
	if !strings.Contains(stdoutAtTrace, "[TRACE]") || !strings.Contains(stdoutAtTrace, "visible trace") {
		t.Fatalf("expected visible trace log, got: %s", stdoutAtTrace)
	}
}
