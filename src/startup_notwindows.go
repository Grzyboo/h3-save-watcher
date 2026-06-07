//go:build !windows

package main

func isStartupEnabledWindows() bool  { return false }
func enableStartupWindows(_ string) error { return nil }
func disableStartupWindows() error        { return nil }
