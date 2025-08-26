package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// https://docs.fyne.io/extend/custom-widget
// every custom widget requires 2 things:
//  the widget and the renderer

// first we have to make a thing to tap, eg the widget
type tappableIcon struct {
	widget.BaseWidget              // extend base functionality
	icon              *widget.Icon // the icon
	onTapped          func()       // what happens when tapped
}

// now implement a widget builder
func newTappableIcon(re fyne.Resource, tapped func()) *tappableIcon {
	// let's build the object
	ti := &tappableIcon{
		icon:     widget.NewIcon(re), // we'll accept any resource
		onTapped: tapped,             // and function to perform once tapped
	}
	ti.ExtendBaseWidget(ti) // now extend the base widget to the object

	// return the pointer
	return ti
}

// and then implement the renderer
func (t *tappableIcon) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.icon)
}

// here is what happens when the thing is tapped
func (t *tappableIcon) Tapped(p *fyne.PointEvent) {
	// if on tapped function is defined
	if t.onTapped != nil {
		// run the function
		t.onTapped()
	}
}

// TappedSecondary is required for the Tappable interface
func (t *tappableIcon) TappedSecondary(p *fyne.PointEvent) {
	// do nothing when the right click
}

// now let's create the entry widget that contains the tappable icon widget

// entryWithIcon is a custom Entry widget with a tappable icon as an action area
type entryWithIcon struct {
	widget.BaseWidget               // extend base behaviors
	entry             *widget.Entry // include a entry field
	actionArea        *tappableIcon // as well as the action area
	window            fyne.Window   // and the window that it will reside in
}

// newEntryWithIcon creates a new entry with an an icon and sets up its behavior
func newEntryWithIcon(pw fyne.Window, icon fyne.Resource, action func()) *entryWithIcon {
	// build the object into memory
	ewi := &entryWithIcon{
		entry:  widget.NewEntry(),
		window: pw,
	}
	// extend the base behaviors
	ewi.ExtendBaseWidget(ewi)

	// load the tappable icon into the actionArea with the desired icon and action
	ewi.actionArea = newTappableIcon(icon, action)

	return ewi
}

// CreateRenderer defines how the entryWithIcon should be rendered
func (f *entryWithIcon) CreateRenderer() fyne.WidgetRenderer {
	return &entryWithIconRenderer{
		entryWithIcon: f, // include the entryWithIcon
		objects: []fyne.CanvasObject{ // explicitly state the objects
			f.entry,      // text entry
			f.actionArea, // embedded icon
		},
	}
}

// custom renderers require these methods for the interface to work:
// Layout
// MinSize
// Refresh
// Destroy
// Objects

// entryWithIconRenderer handles layout and drawing for the entryWithIcon
type entryWithIconRenderer struct {
	entryWithIcon *entryWithIcon      // the actual widget
	objects       []fyne.CanvasObject // and the objects it contains
}

func (r *entryWithIconRenderer) Layout(size fyne.Size) {
	padding := theme.InnerPadding()

	// capture the icon's minsize
	iconSize := r.entryWithIcon.actionArea.icon.MinSize()

	// resize the entry area to be the size of the widget
	r.entryWithIcon.entry.Resize(size)

	// position the icon on the right inside the entry
	x := size.Width - iconSize.Width - padding

	// place the icon half way from the top and bottom
	vertical_center := 2

	// position the icon in the middle of the entry
	y := (size.Height - iconSize.Height) / float32(vertical_center)

	// move the icon into place and resize it
	r.entryWithIcon.actionArea.Move(fyne.NewPos(x, y))
	r.entryWithIcon.actionArea.Resize(iconSize)

	// refresh for good measure
	r.entryWithIcon.entry.Refresh()
}

// MinSize returns the minimum size of the widget including the icon
func (r *entryWithIconRenderer) MinSize() fyne.Size {
	// capture the minsize of the entry and actionArea
	entryMin := r.entryWithIcon.entry.MinSize()
	iconMin := r.entryWithIcon.actionArea.MinSize()

	// include entry , icon & padding as width
	width := entryMin.Width + iconMin.Width + theme.InnerPadding()
	// height is the greater of the two
	height := fyne.Max(entryMin.Height, iconMin.Height)

	// and return the size
	return fyne.NewSize(width, height)
}

// required: redraw items
func (r *entryWithIconRenderer) Refresh() {
	r.entryWithIcon.entry.Refresh()
	r.entryWithIcon.actionArea.Refresh()
}

// required for interface
func (r *entryWithIconRenderer) Destroy() {
	// TODO for now
}

// required: must return the objects of the renderer
func (r *entryWithIconRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
