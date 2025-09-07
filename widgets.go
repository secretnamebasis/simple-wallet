package main

import (
	"fyne.io/fyne/v2"
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
func (t *tappableIcon) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.icon)
}
