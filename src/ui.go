package main

import (
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// appBackground wraps content in the application's dark background.
func appBackground(c fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(color.NRGBA{R: 23, G: 23, B: 23, A: 255})
	return container.NewStack(bg, c)
}

// buildUI builds the main view and returns it without showing it — main
// decides whether the window shows the main view or the onboarding wizard.
func buildUI(bus *Bus, s *State, ui *uiRefs, logs *logStore, w fyne.Window) fyne.CanvasObject {
	ui.dirLabel = widget.NewLabel(s.T(KeyNoDirectorySelected))
	// Truncate instead of wrap: a wrapping label's MinSize height depends on
	// its current width, which makes the window grow on every SetContent.
	ui.dirLabel.Truncation = fyne.TextTruncateEllipsis

	logPanel := buildLogPanel(logs, s.T)

	ui.selectedLabel = widget.NewLabel(s.T(KeySelectedFolder))
	ui.selectedLabel.Importance = widget.HighImportance

	dirBg := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	dirRow := container.NewBorder(nil, nil, ui.selectedLabel, nil, ui.dirLabel)
	dirPanel := container.NewStack(dirBg, container.NewPadded(dirRow))

	settingsBtn := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		showSettings(bus, s, ui, w)
	})
	settingsBtn.Importance = widget.LowImportance

	versionLabel := widget.NewLabel(appVersion)
	versionLabel.Importance = widget.LowImportance

	topBar := container.NewPadded(container.NewBorder(nil, nil, nil, settingsBtn, dirPanel))
	bottomBar := container.NewBorder(nil, nil, nil, versionLabel)
	content := container.NewBorder(topBar, bottomBar, nil, nil, logPanel)
	ui.mainContent = appBackground(content)
	log.Println("main view built")
	return ui.mainContent
}

func setupTray(s *State, w fyne.Window, fyneApp fyne.App) {
	if desk, ok := fyneApp.(desktop.App); ok {
		trayIcon := fyne.NewStaticResource("tray", appIcon)
		desk.SetSystemTrayIcon(trayIcon)
		desk.SetSystemTrayMenu(fyne.NewMenu("Ajit",
			fyne.NewMenuItem(s.T(KeyTrayShow), func() {
				log.Println("tray show clicked")
				fyne.Do(func() {
					log.Println("fyne.Do: calling w.Show()")
					w.Show()
					w.RequestFocus()
					log.Println("fyne.Do: w.Show() done")
				})
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(s.T(KeyTrayQuit), func() {
				fyne.Do(func() {
					fyneApp.Quit()
				})
			}),
		))
	}
}

func newFyneApp(title string) (fyne.App, fyne.Window) {
	fyneApp := app.New()
	w := fyneApp.NewWindow(title)
	w.Resize(fyne.NewSize(700, 450))
	return fyneApp, w
}
