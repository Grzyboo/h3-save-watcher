package main

import (
	"bytes"
	"image/color"
	"image/png"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	flagFrameStrokeWidth  = 2
	flagFrameCornerRadius = 6
)

// flagOrder is the display order of language flags in the onboarding wizard
// and the settings panel.
var flagOrder = []struct {
	name string
	img  []byte
	lang Lang
}{
	{"pl.svg", flagPL, LangPL},
	{"gb.svg", flagEN, LangEN},
	{"ua.svg", flagUA, LangUA},
	{"ru.svg", flagRU, LangRU},
}

// flagSelector is a row of flag images where at most one flag is selected;
// the selected flag is highlighted with a rounded frame in the theme's
// primary color. Selection changes are reported via onSelect (called on
// Fyne's goroutine).
type flagSelector struct {
	Container *fyne.Container
	frames    map[Lang]*canvas.Rectangle
	selected  Lang
	onSelect  func(Lang)
}

// newFlagSelector builds the flag row with flags rendered at their native
// size.
func newFlagSelector(onSelect func(Lang)) *flagSelector {
	return newFlagSelectorSized(onSelect, 0)
}

// newFlagSelectorSized builds the flag row with flags scaled to the given
// width in pixels, preserving aspect ratio.
func newFlagSelectorSized(onSelect func(Lang), flagWidth float32) *flagSelector {
	f := &flagSelector{
		Container: container.NewHBox(),
		frames:    make(map[Lang]*canvas.Rectangle),
		onSelect:  onSelect,
	}
	for _, fl := range flagOrder {
		lang := fl.lang
		w, h := flagDimensions(fl.img)
		if flagWidth > 0 && w > 0 {
			h = h * flagWidth / w
			w = flagWidth
		}
		img := canvas.NewImageFromResource(fyne.NewStaticResource(fl.name, fl.img))
		img.FillMode = canvas.ImageFillContain
		img.ScaleMode = canvas.ImageScaleSmooth
		img.SetMinSize(fyne.NewSize(w, h))

		tap := newTappableImage(img, func() {
			f.Select(lang)
			if f.onSelect != nil {
				f.onSelect(lang)
			}
		})

		// One stroked rounded rectangle — a single path, so there are no
		// corner seams. Transparent stroke = hidden, so the layout does not
		// shift when the selection changes.
		frame := canvas.NewRectangle(color.Transparent)
		frame.StrokeColor = color.Transparent
		frame.StrokeWidth = flagFrameStrokeWidth
		frame.CornerRadius = flagFrameCornerRadius
		f.frames[lang] = frame
		f.Container.Add(container.NewStack(frame, container.NewPadded(tap)))
	}
	return f
}

// Select marks lang as selected (framed) and clears the others.
// An empty lang clears the selection.
func (f *flagSelector) Select(lang Lang) {
	f.selected = lang
	accent := accentColor()
	for l, frame := range f.frames {
		frame.StrokeColor = color.Transparent
		if l == lang {
			frame.StrokeColor = accent
		}
		frame.Refresh()
	}
}

// Selected returns the currently selected language ("" if none).
func (f *flagSelector) Selected() Lang {
	return f.selected
}

// accentColor returns the current theme's primary color.
func accentColor() color.Color {
	settings := fyne.CurrentApp().Settings()
	return settings.Theme().Color(theme.ColorNamePrimary, settings.ThemeVariant())
}

func flagDimensions(data []byte) (float32, float32) {
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err == nil {
		return float32(cfg.Width), float32(cfg.Height)
	}
	// The bundled SVG flags all use a 640x480 viewBox.
	if bytes.Contains(data, []byte("<svg")) {
		return 640, 480
	}
	return 40, 25
}

// tappableImage is a canvas image that reports taps; buttons cannot be used
// here because they cap icons at the inline icon size.
type tappableImage struct {
	widget.BaseWidget
	img   *canvas.Image
	onTap func()
}

func newTappableImage(img *canvas.Image, onTap func()) *tappableImage {
	t := &tappableImage{img: img, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tappableImage) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tappableImage) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.img)
}
