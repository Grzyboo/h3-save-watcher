package main

import (
	"fmt"
	"image/color"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func buildUI(bus *Bus, s *State, ui *uiRefs, logs *logStore, w fyne.Window) {
	ui.dirLabel = widget.NewLabel(s.T(KeyNoDirectorySelected))
	ui.dirLabel.Wrapping = fyne.TextWrapBreak

	ui.browseBtn = widget.NewButtonWithIcon(s.T(KeyChooseDirectory), theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			bus.Publish(WatchDirChangeRequested{Dir: uri.Path()})
		}, w)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})

	logPanel := buildLogPanel(logs, s.T)

	ui.selectedLabel = widget.NewLabel(s.T(KeySelectedFolder))
	ui.selectedLabel.Importance = widget.HighImportance

	dirBg := canvas.NewRectangle(color.NRGBA{R: 50, G: 50, B: 50, A: 255})
	dirRow := container.NewBorder(nil, nil, ui.selectedLabel, nil, ui.dirLabel)
	dirPanel := container.NewStack(dirBg, container.NewPadded(dirRow))

	btnBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	btnPanel := container.NewStack(btnBg, container.NewPadded(ui.browseBtn))

	versionLabel := widget.NewLabel(appVersion)
	versionLabel.Importance = widget.LowImportance

	flagBar := container.NewHBox(
		versionLabel,
		makeFlagButton(flagEN, LangEN, bus),
		makeFlagButton(flagPL, LangPL, bus),
		makeFlagButton(flagUA, LangUA, bus),
		makeFlagButton(flagRU, LangRU, bus),
	)

	startupBtnLabel := s.T(KeyStartupEnable)
	if isStartupEnabled() {
		startupBtnLabel = s.T(KeyStartupDisable)
	}
	ui.startupBtn = widget.NewButton(startupBtnLabel, func() {
		bus.Publish(StartupToggleRequested{})
	})
	ui.startupBtn.Importance = widget.LowImportance

	topBar := container.NewPadded(container.NewBorder(nil, nil, nil, btnPanel, dirPanel))
	bottomBar := container.NewBorder(nil, nil, ui.startupBtn, flagBar)
	content := container.NewBorder(topBar, bottomBar, nil, nil, logPanel)
	bg := canvas.NewRectangle(color.NRGBA{R: 23, G: 23, B: 23, A: 255})
	w.SetContent(container.NewStack(bg, content))
	log.Println("window content set")
}

func refreshStartupBtn(s *State, ui *uiRefs) {
	if ui.startupBtn == nil {
		return
	}
	if isStartupEnabled() {
		ui.startupBtn.SetText(s.T(KeyStartupDisable))
	} else {
		ui.startupBtn.SetText(s.T(KeyStartupEnable))
	}
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

func makeFlagButton(imgData []byte, lang Lang, bus *Bus) *widget.Button {
	res := fyne.NewStaticResource("flag", imgData)
	btn := widget.NewButtonWithIcon("", res, func() {
		bus.Publish(LanguageChanged{Lang: lang})
	})
	btn.Importance = widget.LowImportance
	return btn
}

func runAutoDiscovery(bus *Bus, s *State, w fyne.Window) {
	go func() {
		found := findInstallation()
		if found != "" {
			dialog.ShowConfirm(
				s.T(KeyInstallationFound),
				fmt.Sprintf(s.T(KeyInstallationFoundMsg), found),
				func(ok bool) {
					if ok {
						bus.Publish(WatchDirChangeRequested{Dir: found})
					}
				},
				w,
			)
		} else {
			dialog.ShowInformation(
				s.T(KeyInstallationNotFound),
				s.T(KeyInstallationNotFoundMsg),
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
