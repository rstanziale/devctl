package config

// ParseConcurrencyMode converts an input string to a known mode, defaulting to parallel.
func ParseConcurrencyMode(mode string) ConcurrencyMode {
	switch ConcurrencyMode(mode) {
	case ModeSequential:
		return ModeSequential
	case ModeSemaphore:
		return ModeSemaphore
	default:
		return ModeParallel
	}
}

// IsValidConcurrencyMode reports whether mode is one of the supported values.
func IsValidConcurrencyMode(mode ConcurrencyMode) bool {
	switch mode {
	case ModeSequential, ModeParallel, ModeSemaphore:
		return true
	default:
		return false
	}
}

// ParseTaskMode converts an input string to a known task mode, defaulting to parallel.
func ParseTaskMode(mode string) TaskMode {
	switch TaskMode(mode) {
	case TaskModeParallel:
		return TaskModeParallel
	default:
		return TaskModeSequential
	}
}

// IsValidTaskMode reports whether mode is one of the supported task execution modes.
func IsValidTaskMode(mode TaskMode) bool {
	switch mode {
	case TaskModeSequential, TaskModeParallel:
		return true
	default:
		return false
	}
}

// IsValidAIEngine reports whether engine is one of the supported AI engines.
func IsValidAIEngine(engine AIEngine) bool {
	switch engine {
	case AIEngineOllama:
		return true
	default:
		return false
	}
}

// IsValidHealthcheckType reports whether t is one of the supported healthcheck types.
func IsValidHealthcheckType(t HealthcheckType) bool {
	switch t {
	case HealthcheckHTTP, HealthcheckTCP:
		return true
	default:
		return false
	}
}
