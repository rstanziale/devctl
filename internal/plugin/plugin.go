package plugin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// Plugin represents an external binary registered as a devctl subcommand.
type Plugin struct {
	Name string
	Path string
}

// ExitError wraps a plugin's non-zero exit code so the caller can propagate it
// without terminating the process itself, keeping deferred cleanup (e.g. the
// signal-aware root context in main.go) intact.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	return e.Err
}

// ToCobraCommand wraps the plugin binary in a cobra.Command that passes all
// arguments to the binary and forwards stdin/stdout/stderr.
// It overrides PersistentPreRunE to bypass devctl's config validation, so
// plugins do not require a config file to be present.
func (p Plugin) ToCobraCommand() *cobra.Command {
	return &cobra.Command{
		Use:                p.Name,
		Short:              fmt.Sprintf("External plugin: %s", p.Name),
		DisableFlagParsing: true,
		SilenceErrors:      true,
		SilenceUsage:       true,
		// Override the root PersistentPreRunE so plugins do not require a config file.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use the command's context so the plugin process is cancelled when
			// devctl receives an interrupt/termination signal.
			c := exec.CommandContext(cmd.Context(), p.Path, args...) //nolint:gosec
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr

			if err := c.Run(); err != nil {
				if ctxErr := cmd.Context().Err(); ctxErr != nil {
					return ctxErr
				}

				var exitErr *exec.ExitError
				if errors.As(err, &exitErr) {
					return &ExitError{Code: exitErr.ExitCode(), Err: err}
				}
				return err
			}

			return nil
		},
	}
}
