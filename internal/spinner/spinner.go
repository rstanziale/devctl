// Package spinner provides minimal terminal progress feedback for long-running operations.
package spinner

import (
	"io"
	"os"
	"sync"
	"time"
)

const defaultInterval = 120 * time.Millisecond

var frames = []string{"|", "/", "-", "\\"}

// Spinner displays progress until Stop is called.
type Spinner struct {
	writer   io.Writer
	enabled  bool
	interval time.Duration
	done     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

// New creates a spinner that writes to writer when enabled is true.
func New(writer io.Writer, enabled bool) *Spinner {
	return newSpinner(writer, enabled, defaultInterval)
}

// IsTerminal reports whether writer is an interactive character device.
func IsTerminal(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func newSpinner(writer io.Writer, enabled bool, interval time.Duration) *Spinner {
	return &Spinner{
		writer:   writer,
		enabled:  enabled,
		interval: interval,
		done:     make(chan struct{}),
	}
}

// Start begins rendering progress in the background.
func (s *Spinner) Start() {
	if !s.enabled {
		return
	}
	_, _ = io.WriteString(s.writer, "\rGenerating...")
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for index := 0; ; index++ {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				_, _ = io.WriteString(s.writer, "\rGenerating... "+frames[index%len(frames)])
			}
		}
	}()
}

// Stop clears the status line. It is safe to call more than once.
func (s *Spinner) Stop() {
	if !s.enabled {
		return
	}
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
		_, _ = io.WriteString(s.writer, "\r\033[2K")
	})
}
