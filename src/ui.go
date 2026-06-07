package main

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"log"
)

func buildUI(state *App, w fyne.Window, fyneApp fyne.App) {
	state.dirLabel = widget.NewLabel(state.T(KeyNoDirectorySelected))
	state.dirLabel.Wrapping = fyne.TextWrapBreak

	state.browseBtn = widget.NewButtonWithIcon(state.T(KeyChooseDirectory), theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			state.setWatchDir(uri.Path())
		}, w)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})

	logPanel := buildLogPanel(state)

	state.selectedLabel = widget.NewLabel(state.T(KeySelectedFolder))
	state.selectedLabel.Importance = widget.HighImportance

	dirBg := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	dirRow := container.NewBorder(nil, nil, state.selectedLabel, nil, state.dirLabel)
	dirPanel := container.NewStack(dirBg, container.NewPadded(dirRow))

	btnBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	btnPanel := container.NewStack(btnBg, container.NewPadded(state.browseBtn))

	versionLabel := widget.NewLabel(appVersion)
	versionLabel.Importance = widget.LowImportance

	flagBar := container.NewHBox(
		versionLabel,
		makeFlagButton(flagEN, LangEN, state),
		makeFlagButton(flagPL, LangPL, state),
		makeFlagButton(flagUA, LangUA, state),
		makeFlagButton(flagRU, LangRU, state),
	)

	startupBtnLabel := state.T(KeyStartupEnable)
	if isStartupEnabled() {
		startupBtnLabel = state.T(KeyStartupDisable)
	}
	state.startupBtn = widget.NewButton(startupBtnLabel, func() {
		state.handleStartupToggle()
	})
	state.startupBtn.Importance = widget.LowImportance

	topBar := container.NewPadded(container.NewBorder(nil, nil, nil, btnPanel, dirPanel))
	bottomBar := container.NewBorder(nil, nil, state.startupBtn, flagBar)
	content := container.NewBorder(topBar, bottomBar, nil, nil, logPanel)
	bg := canvas.NewRectangle(color.NRGBA{R: 23, G: 23, B: 23, A: 255})
	w.SetContent(container.NewStack(bg, content))
	log.Println("window content set")
}

func setupTray(state *App, w fyne.Window, fyneApp fyne.App) {
	if desk, ok := fyneApp.(desktop.App); ok {
		trayIcon := fyne.NewStaticResource("tray", appIcon)
		desk.SetSystemTrayIcon(trayIcon)
		desk.SetSystemTrayMenu(fyne.NewMenu("H3SaveWatcher",
			fyne.NewMenuItem(state.T(KeyTrayShow), func() {
				log.Println("tray show clicked")
				fyne.Do(func() {
					log.Println("fyne.Do: calling w.Show()")
					w.Show()
					w.RequestFocus()
					log.Println("fyne.Do: w.Show() done")
				})
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem(state.T(KeyTrayQuit), func() {
				fyne.Do(func() {
					fyneApp.Quit()
				})
			}),
		))
	}
}

func makeFlagButton(imgData []byte, lang Lang, state *App) *widget.Button {
	res := fyne.NewStaticResource("flag", imgData)
	btn := widget.NewButtonWithIcon("", res, func() {
		state.setLang(lang)
	})
	btn.Importance = widget.LowImportance
	return btn
}

func runAutoDiscovery(state *App, w fyne.Window) {
	go func() {
		found := findInstallation()
		if found != "" {
			dialog.ShowConfirm(
				state.T(KeyInstallationFound),
				fmt.Sprintf(state.T(KeyInstallationFoundMsg), found),
				func(ok bool) {
					if ok {
						state.setWatchDir(found)
					}
				},
				w,
			)
		} else {
			dialog.ShowInformation(
				state.T(KeyInstallationNotFound),
				state.T(KeyInstallationNotFoundMsg),
				w,
			)
		}
	}()
}

func newFyneApp(title string) (fyne.App, fyne.Window) {
	fyneApp := app.New()
	w := fyneApp.NewWindow(title)
	w.Resize(fyne.NewSize(700, 450))
	return fyneApp, w
}
