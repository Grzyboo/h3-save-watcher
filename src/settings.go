package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// showSettings swaps the window content to the settings panel. The panel is
// rebuilt on every open so it always reflects current state (language,
// watch dir, autostart).
func showSettings(bus *Bus, s *State, ui *uiRefs, w fyne.Window) {
	ui.showingSettings = true
	w.SetContent(buildSettingsView(bus, s, ui, w))
}

func buildSettingsView(bus *Bus, s *State, ui *uiRefs, w fyne.Window) fyne.CanvasObject {
	closeBtn := widget.NewButton(s.T(KeyClose), func() {
		ui.showingSettings = false
		w.SetContent(ui.mainContent)
	})

	th := fyne.CurrentApp().Settings().Theme()
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	title := canvas.NewText(s.T(KeySettings), th.Color(theme.ColorNameForeground, variant))
	title.TextSize = th.Size(theme.SizeNameHeadingText)
	title.TextStyle = fyne.TextStyle{Bold: true}
	header := container.NewPadded(container.NewBorder(nil, nil, title, closeBtn))

	langLabel := widget.NewLabel(s.T(KeyLanguage))
	langLabel.Importance = widget.HighImportance
	flags := newFlagSelectorSized(func(lang Lang) { bus.Publish(LanguageChanged{Lang: lang}) }, 40)
	flags.Select(s.Lang())

	folderLabel := widget.NewLabel(s.T(KeyHotAFolder))
	folderLabel.Importance = widget.HighImportance
	ui.settingsDirLabel = widget.NewLabel(ui.dirLabel.Text)
	// Truncate instead of wrap: a wrapping label's MinSize height depends on
	// its current width, which makes the window grow on every SetContent.
	ui.settingsDirLabel.Truncation = fyne.TextTruncateEllipsis
	chooseBtn := widget.NewButtonWithIcon(s.T(KeyChooseDirectory), theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			bus.Publish(WatchDirChangeRequested{Dir: uri.Path()})
		}, w)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})
	dirRow := container.NewBorder(nil, nil, nil, chooseBtn, ui.settingsDirLabel)

	otherLabel := widget.NewLabel(s.T(KeyOther))
	otherLabel.Importance = widget.HighImportance
	ui.startupCheck = widget.NewCheck(s.T(KeyAddToAutostart), nil)
	ui.startupCheck.SetChecked(isStartupEnabled())
	ui.startupCheck.OnChanged = func(checked bool) {
		bus.Publish(StartupSetRequested{Enabled: checked})
	}
	startupDesc := widget.NewLabel(s.T(KeyAutostartDescription))
	startupDesc.Importance = widget.LowImportance

	body := container.NewPadded(container.NewVBox(
		langLabel,
		flags.Container,
		widget.NewSeparator(),
		folderLabel,
		dirRow,
		widget.NewSeparator(),
		otherLabel,
		ui.startupCheck,
		startupDesc,
	))
	return appBackground(container.NewBorder(header, nil, nil, nil, body))
}
