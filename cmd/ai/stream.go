package aicmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/rs-box/devctl/internal/ai"
	"github.com/rs-box/devctl/internal/spinner"
)

// streamGeneration sends prompt to client, echoing each response chunk to
// cmd's stdout as it arrives. The spinner is started lazily from onStart,
// once the client confirms the request is accepted and streaming is about to
// begin, so it never races any pre-flight logging emitted while building or
// sending the request. It returns the full accumulated response text.
func streamGeneration(cmd *cobra.Command, client ai.Client, prompt string) (string, error) {
	status := spinner.New(cmd.ErrOrStderr(), spinner.IsTerminal(cmd.ErrOrStderr()))
	defer status.Stop()

	started := false
	var response strings.Builder
	err := client.Generate(cmd.Context(), prompt,
		func() {
			status.Start()
		},
		func(chunk string) error {
			if !started {
				status.Stop()
				started = true
			}
			response.WriteString(chunk)
			_, err := io.WriteString(cmd.OutOrStdout(), chunk)
			return err
		},
	)

	status.Stop()
	if err != nil {
		if started {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
		return response.String(), err
	}

	if _, err := fmt.Fprintln(cmd.OutOrStdout()); err != nil {
		return response.String(), fmt.Errorf("write AI response: %w", err)
	}

	return response.String(), nil
}
