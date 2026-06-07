package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const appName = "H3SaveWatcher"

// isSuspiciousPath returns true if the binary appears to be in a temp or build path.
func isSuspiciousPath(exePath string) bool {
	lower := strings.ToLower(exePath)
	suspects := []string{"fyne-cross", "tmp", "temp", os.TempDir()}
	for _, s := range suspects {
		if strings.Contains(lower, strings.ToLower(s)) {
			return true
		}
	}
	return false
}

func isStartupEnabled() bool {
	switch runtime.GOOS {
	case "darwin":
		return isStartupEnabledDarwin()
	case "windows":
		return isStartupEnabledWindows()
	default:
		return isStartupEnabledLinux()
	}
}

func enableStartup() error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("could not determine executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("could not resolve executable path: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return enableStartupDarwin(exe)
	case "windows":
		return enableStartupWindows(exe)
	default:
		return enableStartupLinux(exe)
	}
}

func disableStartup() error {
	switch runtime.GOOS {
	case "darwin":
		return disableStartupDarwin()
	case "windows":
		return disableStartupWindows()
	default:
		return disableStartupLinux()
	}
}

// --- macOS ---

func launchAgentPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.h3savewatcher.app.plist")
}

func isStartupEnabledDarwin() bool {
	_, err := os.Stat(launchAgentPath())
	return err == nil
}

func enableStartupDarwin(exe string) error {
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.h3savewatcher.app</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>--tray</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
`, exe)
	path := launchAgentPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(plist), 0644)
}

func disableStartupDarwin() error {
	return os.Remove(launchAgentPath())
}

// --- Linux ---

func autostartDesktopPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autostart", "h3savewatcher.desktop")
}

func isStartupEnabledLinux() bool {
	_, err := os.Stat(autostartDesktopPath())
	return err == nil
}

func enableStartupLinux(exe string) error {
	desktop := fmt.Sprintf("[Desktop Entry]\nType=Application\nName=%s\nExec=%s --tray\nX-GNOME-Autostart-enabled=true\n", appName, exe)
	path := autostartDesktopPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(desktop), 0644)
}

func disableStartupLinux() error {
	return os.Remove(autostartDesktopPath())
}
