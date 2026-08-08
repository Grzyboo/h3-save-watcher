package main

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type installation struct {
	path   string
	manual bool
}

type installationListLayout struct {
	length func() int
}

func (l *installationListLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
}

func (l *installationListLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) == 0 {
		return fyne.Size{}
	}

	minSize := objects[0].MinSize()
	count := l.length()
	if count == 0 {
		minSize.Height = 0
		return minSize
	}

	minSize.Height = minSize.Height*float32(count) + theme.Padding()*float32(count-1)
	return minSize
}

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
	installations []installation
	autostart     bool

	// Panel 2 widget refs — valid only while panel == 2.
	installList    *widget.List
	installSection *fyne.Container
	infoLabel      *widget.Label
	selectionHint  *widget.Label
	nextButton     *widget.Button
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

func newPanelCounter(text string) *widget.Label {
	counter := widget.NewLabel(text)
	counter.Alignment = fyne.TextAlignTrailing
	counter.Importance = widget.HighImportance
	return counter
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

	top := container.NewPadded(container.NewStack(title, newPanelCounter("1/3")))
	body := container.NewCenter(flags.Container)
	bottom := container.NewPadded(container.NewBorder(nil, nil, nil, next))
	return appBackground(container.NewBorder(top, bottom, nil, nil, body))
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

	o.selectionHint = widget.NewLabel("")
	o.selectionHint.Alignment = fyne.TextAlignCenter
	o.selectionHint.Importance = widget.DangerImportance
	o.selectionHint.Hide()

	o.nextButton = widget.NewButton(o.s.T(KeyNext), func() { o.show(o.panel3()) })
	o.nextButton.Importance = widget.HighImportance
	if o.dir == "" {
		o.nextButton.Disable()
	}

	o.installList = widget.NewList(
		func() int { return len(o.installations) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(o.installations) {
				installation := o.installations[id]
				prefix := o.s.T(KeyAutoDetected)
				if installation.manual {
					prefix = o.s.T(KeyManuallyAdded)
				}
				obj.(*widget.Label).SetText(prefix + installation.path)
			}
		},
	)
	o.installList.OnSelected = func(id widget.ListItemID) {
		if id < len(o.installations) {
			o.dir = o.installations[id].path
			o.nextButton.Enable()
			o.refreshSelectionHint()
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
	listContainer := container.New(
		&installationListLayout{length: func() int { return len(o.installations) }},
		o.installList,
	)
	o.installSection = container.NewVBox(
		listContainer,
		container.NewCenter(addBtn),
		container.NewCenter(o.selectionHint),
	)

	prev := widget.NewButton(o.s.T(KeyPrevious), func() {
		// Leaving the panel backwards forgets everything done here.
		o.installations = nil
		o.dir = ""
		o.show(o.panel1())
	})
	heading := container.NewVBox(title, o.infoLabel)
	top := container.NewPadded(container.NewStack(heading, newPanelCounter("2/3")))
	bottom := container.NewPadded(container.NewBorder(nil, nil, prev, o.nextButton))
	content := container.NewBorder(top, bottom, nil, nil, container.NewPadded(o.installSection))

	if o.installations == nil {
		// First visit — scan for installations in the background.
		o.scanID++
		go o.detect(o.scanID)
	} else {
		// Re-entering from panel 3 — restore the previous scan + selection.
		o.refreshInfoLabel()
		for i, installation := range o.installations {
			if installation.path == o.dir {
				o.installList.Select(i)
				break
			}
		}
	}
	o.refreshSelectionHint()

	return appBackground(content)
}

func (o *onboarding) refreshInstallList() {
	o.installList.Refresh()
	o.installSection.Refresh()
	o.refreshSelectionHint()
}

func (o *onboarding) refreshSelectionHint() {
	if o.nextButton == nil || o.selectionHint == nil {
		return
	}
	if !o.nextButton.Disabled() {
		o.selectionHint.Hide()
		return
	}

	key := KeyInstallHintMany
	switch len(o.installations) {
	case 0:
		key = KeyInstallHintEmpty
	case 1:
		key = KeyInstallHintOne
	}
	o.selectionHint.SetText(o.s.T(key))
	o.selectionHint.Show()
}

// refreshInfoLabel shows the not-found message when the list is empty and
// hides it once an installation is available.
func (o *onboarding) refreshInfoLabel() {
	if len(o.installations) == 0 {
		o.infoLabel.SetText(o.s.T(KeyNoInstallationsFound))
		o.infoLabel.Show()
		return
	}
	o.infoLabel.Hide()
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
			if !slices.ContainsFunc(o.installations, func(installation installation) bool {
				return installation.path == p
			}) {
				o.installations = append(o.installations, installation{path: p})
			}
		}
		o.refreshInfoLabel()
		o.refreshInstallList()
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
	for i, installation := range o.installations {
		if installation.path == root {
			o.installList.Select(i)
			return
		}
	}
	o.installations = append([]installation{{path: root, manual: true}}, o.installations...)
	o.refreshInfoLabel()
	o.refreshInstallList()
	o.installList.Select(0)
}

// --- panel 3: misc settings ---

func (o *onboarding) panel3() fyne.CanvasObject {
	o.panel = 3

	title := widget.NewLabel(o.s.T(KeyOtherSettings))
	title.Alignment = fyne.TextAlignCenter
	title.Importance = widget.HighImportance

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

	top := container.NewPadded(container.NewStack(title, newPanelCounter("3/3")))
	body := container.NewCenter(container.NewVBox(check, desc))
	bottom := container.NewPadded(container.NewBorder(nil, nil, prev, next))
	return appBackground(container.NewBorder(top, bottom, nil, nil, body))
}
