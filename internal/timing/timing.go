package timing

import (
	"time"

	"github.com/rs-box/devctl/internal/logger"
)

const iso8601MillisLayout string = "2006-01-02T15:04:05.000"

var log = logger.GetLogger(logger.WithComponent("Timing"))

func LogCommandTiming(commandName string) func() {
	start := time.Now()
	log.Debug("timing start", "command", commandName, "start", start.Format(iso8601MillisLayout))

	return func() {
		end := time.Now()
		elapsed := elapsedBetween(start, end)
		log.Debug("timing end", "command", commandName, "end", end.Format(iso8601MillisLayout), "elapsed", elapsed)
	}
}

func elapsedBetween(start, end time.Time) time.Duration {
	return end.Sub(start).Round(time.Millisecond)
}
