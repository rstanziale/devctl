package doctor

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/rs-box/devctl/internal/config"
)

type ToolCheckCommand struct {
	tool config.Tool
}

func NewToolCheckCommand(tool config.Tool) *ToolCheckCommand {
	return &ToolCheckCommand{tool: tool}
}

func (t *ToolCheckCommand) Name() string {
	return fmt.Sprintf("tool %s", t.tool.Name)
}

func (t *ToolCheckCommand) Run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if _, err := exec.LookPath(t.tool.Command); err != nil {
		return fmt.Errorf("tool %q not found in PATH: %w", t.tool.Command, err)
	}
	return nil
}
