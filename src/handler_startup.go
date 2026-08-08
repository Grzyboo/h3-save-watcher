package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// registerStartupHandlers wires the autostart checkbox: StartupSetRequested
// applies the change right away (keeping the suspicious-path warning when
// enabling); the StartupEnabled/Disabled facts sync the checkbox state.
// Failures show an error dialog and revert the checkbox.
func registerStartupHandlers(bus *Bus, s *State, ui *uiRefs) {
	Subscribe(bus, func(e StartupSetRequested) {
		if !e.Enabled {
			applyStartup(bus, s, ui, false)
			return
		}
		// Check for suspicious path before enabling.
		exe, err := os.Executable()
		if err == nil {
			exe, _ = filepath.EvalSymlinks(exe)
		}
		if err == nil && isSuspiciousPath(exe) {
			fyne.Do(func() {
				dialog.ShowConfirm(
					s.T(KeyStartupWarnTitle),
					fmt.Sprintf(s.T(KeyStartupWarnMsg), exe),
					func(proceed bool) {
						if proceed {
							applyStartup(bus, s, ui, true)
						} else {
							setStartupCheckSilently(ui, false)
						}
					},
					ui.window,
				)
			})
			return
		}
		applyStartup(bus, s, ui, true)
	})
	Subscribe(bus, func(e StartupEnabled) {
		fyne.Do(func() { setStartupCheckSilently(ui, true) })
	})
	Subscribe(bus, func(e StartupDisabled) {
		fyne.Do(func() { setStartupCheckSilently(ui, false) })
	})
}

// applyStartup enables or disables the autostart entry. It runs on the bus
// goroutine or in a dialog callback; all UI touches are wrapped in fyne.Do.
func applyStartup(bus *Bus, s *State, ui *uiRefs, enable bool) {
	var err error
	if enable {
		err = enableStartup()
	} else {
		err = disableStartup()
	}
	if err != nil {
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf(s.T(KeyStartupError), err), ui.window)
			setStartupCheckSilently(ui, !enable)
		})
		return
	}
	if enable {
		bus.Publish(StartupEnabled{})
	} else {
		bus.Publish(StartupDisabled{})
	}
}

// setStartupCheckSilently syncs the settings autostart checkbox without
// triggering its OnChanged (which would republish StartupSetRequested).
// Must be called on Fyne's goroutine.
func setStartupCheckSilently(ui *uiRefs, checked bool) {
	if ui.startupCheck == nil {
		return
	}
	handler := ui.startupCheck.OnChanged
	ui.startupCheck.OnChanged = nil
	ui.startupCheck.SetChecked(checked)
	ui.startupCheck.OnChanged = handler
}
