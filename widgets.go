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

// CreateRenderer defines how the tappableIcon should be rendered
func (f *tappableIcon) CreateRenderer() fyne.WidgetRenderer {
	return &tappableIconRenderer{
		tappableIcon: f,
		objects: []fyne.CanvasObject{ // explicitly state the objects
			f.icon,
		},
	}
}

// custom renderers require these methods for the interface to work:
// Layout
// MinSize
// Refresh
// Destroy
// Objects

// tappableIconRenderer handles layout and drawing for the tappableIcon
type tappableIconRenderer struct {
	*tappableIcon                     // the actual widget
	objects       []fyne.CanvasObject // and the objects it contains
}

func (r *tappableIconRenderer) Layout(size fyne.Size) {
	padding := theme.InnerPadding()

	// capture the icon's minsize
	iconSize := r.icon.MinSize()

	// resize the icon area to be the size of the widget
	r.icon.Resize(size)

	// position the icon on the right inside the entry
	x := size.Width - iconSize.Width - padding

	// place the icon half way from the top and bottom
	vertical_center := 2

	// position the icon in the middle of the entry
	y := (size.Height - iconSize.Height) / float32(vertical_center)

	// move the icon into place and resize it
	r.icon.Move(fyne.NewPos(x, y))
	r.icon.Resize(iconSize)

	// refresh for good measure
	r.icon.Refresh()
}

// MinSize returns the minimum size of the widget including the icon
func (r *tappableIconRenderer) MinSize() fyne.Size {
	// capture the minsize of the entry and actionArea
	iconMin := r.icon.MinSize()

	// and return the size
	return fyne.NewSize(iconMin.Width, iconMin.Height)
}

// required: redraw items
func (r *tappableIconRenderer) Refresh() {
	r.icon.Refresh()
}

// required for interface
func (r *tappableIconRenderer) Destroy() {
	// TODO for now
}

// required: must return the objects of the renderer
func (r *tappableIconRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}
