package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const binaryPrefix = "devctl-"

// Discover scans dir for plugin binaries and returns the discovered plugins.
// On Linux/macOS it requires the executable bit to be set.
// On Windows it requires the .exe extension.
// Returns nil, nil if the directory does not exist.
func Discover(dir string) ([]Plugin, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var plugins []Plugin
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name, ok := pluginName(entry.Name())
		if !ok {
			continue
		}

		if runtime.GOOS != "windows" {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			if info.Mode()&0o111 == 0 {
				continue // not executable
			}
		}

		plugins = append(plugins, Plugin{
			Name: name,
			Path: filepath.Join(dir, entry.Name()),
		})
	}

	return plugins, nil
}

// pluginName extracts the command name from a plugin binary filename.
// On Windows, requires and strips the .exe suffix.
// Returns ("", false) if the filename does not match the plugin naming convention.
func pluginName(filename string) (string, bool) {
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(filename, ".exe") {
			return "", false
		}
		filename = strings.TrimSuffix(filename, ".exe")
	}

	if !strings.HasPrefix(filename, binaryPrefix) {
		return "", false
	}

	name := strings.TrimPrefix(filename, binaryPrefix)
	if name == "" {
		return "", false
	}

	return name, true
}
