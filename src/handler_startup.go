package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

// registerStartupHandlers wires the startup toggle: StartupToggleRequested
// runs the dialog flow; the StartupEnabled/StartupDisabled facts relabel the
// button and show the info dialog.
func registerStartupHandlers(bus *Bus, s *State, ui *uiRefs) {
	Subscribe(bus, func(e StartupToggleRequested) { startupToggleFlow(bus, s, ui) })
	Subscribe(bus, func(e StartupEnabled) {
		fyne.Do(func() {
			refreshStartupBtn(s, ui)
			dialog.ShowInformation(s.T(KeyStartupEnableTitle), s.T(KeyStartupSuccess), ui.window)
		})
	})
	Subscribe(bus, func(e StartupDisabled) {
		fyne.Do(func() {
			refreshStartupBtn(s, ui)
			dialog.ShowInformation(s.T(KeyStartupDisableTitle), s.T(KeyStartupRemoved), ui.window)
		})
	})
}

// startupToggleFlow runs the enable/disable dialog flow. It runs on the bus
// goroutine; all UI touches are wrapped in fyne.Do.
func startupToggleFlow(bus *Bus, s *State, ui *uiRefs) {
	if !isStartupEnabled() {
		// Check for suspicious path before enabling.
		exe, err := os.Executable()
		if err == nil {
			exe, _ = filepath.EvalSymlinks(exe)
		}
		if isSuspiciousPath(exe) {
			fyne.Do(func() {
				dialog.ShowConfirm(
					s.T(KeyStartupWarnTitle),
					fmt.Sprintf(s.T(KeyStartupWarnMsg), exe),
					func(proceed bool) {
						if proceed {
							confirmEnableStartup(bus, s, ui)
						}
					},
					ui.window,
				)
			})
			return
		}
		confirmEnableStartup(bus, s, ui)
	} else {
		fyne.Do(func() {
			dialog.ShowConfirm(
				s.T(KeyStartupDisableTitle),
				s.T(KeyStartupDisableConfirm),
				func(ok bool) {
					if !ok {
						return
					}
					if err := disableStartup(); err != nil {
						dialog.ShowError(fmt.Errorf(s.T(KeyStartupError), err), ui.window)
						return
					}
					bus.Publish(StartupDisabled{})
				},
				ui.window,
			)
		})
	}
}

func confirmEnableStartup(bus *Bus, s *State, ui *uiRefs) {
	fyne.Do(func() {
		dialog.ShowConfirm(
			s.T(KeyStartupEnableTitle),
			s.T(KeyStartupEnableConfirm),
			func(ok bool) {
				if !ok {
					return
				}
				if err := enableStartup(); err != nil {
					dialog.ShowError(fmt.Errorf(s.T(KeyStartupError), err), ui.window)
					return
				}
				bus.Publish(StartupEnabled{})
			},
			ui.window,
		)
	})
}
