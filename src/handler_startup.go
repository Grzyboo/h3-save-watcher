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
func registerStartupHandlers(bus *Bus, a *App) {
	Subscribe(bus, func(e StartupToggleRequested) { a.startupToggleFlow(bus) })
	Subscribe(bus, func(e StartupEnabled) {
		fyne.Do(func() {
			a.refreshStartupBtn()
			dialog.ShowInformation(a.T(KeyStartupEnableTitle), a.T(KeyStartupSuccess), a.window)
		})
	})
	Subscribe(bus, func(e StartupDisabled) {
		fyne.Do(func() {
			a.refreshStartupBtn()
			dialog.ShowInformation(a.T(KeyStartupDisableTitle), a.T(KeyStartupRemoved), a.window)
		})
	})
}

// startupToggleFlow runs the enable/disable dialog flow. It runs on the bus
// goroutine; all UI touches are wrapped in fyne.Do.
func (a *App) startupToggleFlow(bus *Bus) {
	if !isStartupEnabled() {
		// Check for suspicious path before enabling.
		exe, err := os.Executable()
		if err == nil {
			exe, _ = filepath.EvalSymlinks(exe)
		}
		if isSuspiciousPath(exe) {
			fyne.Do(func() {
				dialog.ShowConfirm(
					a.T(KeyStartupWarnTitle),
					fmt.Sprintf(a.T(KeyStartupWarnMsg), exe),
					func(proceed bool) {
						if proceed {
							a.confirmEnableStartup(bus)
						}
					},
					a.window,
				)
			})
			return
		}
		a.confirmEnableStartup(bus)
	} else {
		fyne.Do(func() {
			dialog.ShowConfirm(
				a.T(KeyStartupDisableTitle),
				a.T(KeyStartupDisableConfirm),
				func(ok bool) {
					if !ok {
						return
					}
					if err := disableStartup(); err != nil {
						dialog.ShowError(fmt.Errorf(a.T(KeyStartupError), err), a.window)
						return
					}
					bus.Publish(StartupDisabled{})
				},
				a.window,
			)
		})
	}
}

func (a *App) confirmEnableStartup(bus *Bus) {
	fyne.Do(func() {
		dialog.ShowConfirm(
			a.T(KeyStartupEnableTitle),
			a.T(KeyStartupEnableConfirm),
			func(ok bool) {
				if !ok {
					return
				}
				if err := enableStartup(); err != nil {
					dialog.ShowError(fmt.Errorf(a.T(KeyStartupError), err), a.window)
					return
				}
				bus.Publish(StartupEnabled{})
			},
			a.window,
		)
	})
}
