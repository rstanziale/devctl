package doctor

import (
	"context"
	"testing"

	"github.com/rs-box/devctl/internal/config"
)

func TestToolCheckName(t *testing.T) {
	tool := config.Tool{Name: "go", Command: "go"}
	check := NewToolCheckCommand(tool)

	expected := "tool go"
	if check.Name() != expected {
		t.Errorf("expected %q, got %q", expected, check.Name())
	}
}

func TestToolCheckToolExists(t *testing.T) {
	tool := config.Tool{Name: "go", Command: "go"}
	check := NewToolCheckCommand(tool)

	err := check.Run(context.Background())
	if err != nil {
		t.Errorf("expected no error for existing tool 'go', got %v", err)
	}
}

func TestToolCheckToolNotExists(t *testing.T) {
	tool := config.Tool{Name: "nonexistenttool", Command: "nonexistenttool"}
	check := NewToolCheckCommand(tool)

	err := check.Run(context.Background())
	if err == nil {
		t.Error("expected error for non-existing tool, got nil")
	}
}
