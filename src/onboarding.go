package main

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// onboarding is the first-run setup wizard: language selection, game
// directory selection and misc settings. Nothing is persisted until the
// wizard finishes (OnboardingFinishRequested), so an app restart (e.g. after
// a self-update) starts the wizard over from the first panel.
//
// All methods run on Fyne's goroutine (button callbacks, fyne.Do), except
// detect which publishes its results via fyne.Do.
type onboarding struct {
	bus *Bus
	s   *State
	win fyne.Window

	panel         int
	scanID        int
	lang          Lang
	dir           string
	installations []string
	autoDetected  bool
	autostart     bool

	// Panel 2 widget refs — valid only while panel == 2.
	installList *widget.List
	infoLabel   *widget.Label
}

func newOnboarding(bus *Bus, s *State, w fyne.Window) *onboarding {
	return &onboarding{bus: bus, s: s, win: w, autostart: true}
}

// Root returns the first panel; later panels are swapped in via SetContent.
func (o *onboarding) Root() fyne.CanvasObject {
	return o.panel1()
}

func (o *onboarding) show(c fyne.CanvasObject) {
	o.win.SetContent(c)
}

// --- panel 1: language selection ---

func (o *onboarding) panel1() fyne.CanvasObject {
	o.panel = 1

	title := widget.NewLabel(o.s.T(KeySelectLanguage))
	title.Alignment = fyne.TextAlignCenter
	title.Importance = widget.HighImportance

	next := widget.NewButton(o.s.T(KeyNext), func() { o.show(o.panel2()) })
	next.Importance = widget.HighImportance
	if o.lang == "" {
		next.Disable()
	}

	flags := newFlagSelectorSized(func(lang Lang) {
		o.lang = lang
		// In-memory only — persisted when the wizard finishes, so a restart
		// mid-onboarding starts over with no language selected.
		o.s.setLang(lang)
		title.SetText(o.s.T(KeySelectLanguage))
		next.SetText(o.s.T(KeyNext))
		next.Enable()
	}, 96)
	if o.lang != "" {
		flags.Select(o.lang)
	}

	body := container.NewCenter(container.NewVBox(title, container.NewCenter(flags.Container)))
	bottom := container.NewPadded(container.NewBorder(nil, nil, nil, next))
	return appBackground(container.NewBorder(nil, bottom, nil, nil, body))
}

// --- panel 2: game directory selection ---

func (o *onboarding) panel2() fyne.CanvasObject {
	o.panel = 2

	title := widget.NewLabel(o.s.T(KeyChooseInstallation))
	title.Alignment = fyne.TextAlignCenter
	title.Importance = widget.HighImportance

	o.infoLabel = widget.NewLabel("")
	// Truncate instead of wrap: a wrapping label's MinSize height depends on
	// its current width, which makes the window grow on every SetContent.
	o.infoLabel.Truncation = fyne.TextTruncateEllipsis
	o.infoLabel.Hide()

	var next *widget.Button
	o.installList = widget.NewList(
		func() int { return len(o.installations) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(o.installations) {
				obj.(*widget.Label).SetText(o.installations[id])
			}
		},
	)
	o.installList.OnSelected = func(id widget.ListItemID) {
		if id < len(o.installations) {
			o.dir = o.installations[id]
			next.Enable()
		}
	}

	addBtn := widget.NewButtonWithIcon(o.s.T(KeyAddInstallation), theme.FolderOpenIcon(), func() {
		d := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			o.addInstallation(uri.Path())
		}, o.win)
		d.Resize(fyne.NewSize(800, 600))
		d.Show()
	})

	prev := widget.NewButton(o.s.T(KeyPrevious), func() {
		// Leaving the panel backwards forgets everything done here.
		o.installations = nil
		o.autoDetected = false
		o.dir = ""
		o.show(o.panel1())
	})
	next = widget.NewButton(o.s.T(KeyNext), func() { o.show(o.panel3()) })
	next.Importance = widget.HighImportance
	if o.dir == "" {
		next.Disable()
	}

	top := container.NewPadded(container.NewVBox(title, o.infoLabel))
	bottom := container.NewPadded(container.NewVBox(
		container.NewHBox(addBtn),
		container.NewBorder(nil, nil, prev, next),
	))
	content := container.NewBorder(top, bottom, nil, nil, container.NewPadded(o.installList))

	if o.installations == nil {
		// First visit — scan for installations in the background.
		o.scanID++
		go o.detect(o.scanID)
	} else {
		// Re-entering from panel 3 — restore the previous scan + selection.
		o.refreshInfoLabel()
		for i, p := range o.installations {
			if p == o.dir {
				o.installList.Select(i)
				break
			}
		}
	}

	return appBackground(content)
}

// refreshInfoLabel shows the detection info line matching the current state:
// "Automatically detected:" when the scan found installations, the not-found
// message when it found nothing, and hides the line when only manually added
// folders are present.
func (o *onboarding) refreshInfoLabel() {
	switch {
	case o.autoDetected:
		o.infoLabel.SetText(o.s.T(KeyAutoDetected))
		o.infoLabel.Show()
	case len(o.installations) == 0:
		o.infoLabel.SetText(o.s.T(KeyNoInstallationsFound))
		o.infoLabel.Show()
	default:
		o.infoLabel.Hide()
	}
}

// detect scans the filesystem for H3 installations and merges the results
// into the panel 2 list (keeping any folder the user added manually).
func (o *onboarding) detect(id int) {
	found := findAllInstallations()
	fyne.Do(func() {
		if o.panel != 2 || id != o.scanID {
			return
		}
		for _, p := range found {
			if !slices.Contains(o.installations, p) {
				o.installations = append(o.installations, p)
			}
		}
		o.autoDetected = len(found) > 0
		o.refreshInfoLabel()
		o.installList.Refresh()
	})
}

// addInstallation validates a user-picked folder and adds it to the list,
// selecting it automatically.
func (o *onboarding) addInstallation(path string) {
	root, err := resolveH3Root(path)
	if err != nil {
		dialog.ShowError(err, o.win)
		return
	}
	for i, p := range o.installations {
		if p == root {
			o.installList.Select(i)
			return
		}
	}
	o.installations = append(o.installations, root)
	o.refreshInfoLabel()
	o.installList.Refresh()
	o.installList.Select(len(o.installations) - 1)
}

// --- panel 3: misc settings ---

func (o *onboarding) panel3() fyne.CanvasObject {
	o.panel = 3

	check := widget.NewCheck(o.s.T(KeyAddToAutostart), func(checked bool) { o.autostart = checked })
	check.SetChecked(o.autostart)

	desc := widget.NewLabel(o.s.T(KeyAutostartDescription))
	desc.Importance = widget.LowImportance

	prev := widget.NewButton(o.s.T(KeyPrevious), func() {
		// Leaving the panel backwards forgets everything done here.
		o.autostart = true
		o.show(o.panel2())
	})
	next := widget.NewButton(o.s.T(KeyNext), func() {
		o.bus.Publish(OnboardingFinishRequested{Dir: o.dir, Lang: o.lang, Autostart: o.autostart})
	})
	next.Importance = widget.HighImportance

	body := container.NewCenter(container.NewVBox(check, desc))
	bottom := container.NewPadded(container.NewBorder(nil, nil, prev, next))
	return appBackground(container.NewBorder(nil, bottom, nil, nil, body))
}
