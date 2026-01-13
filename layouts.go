package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type twoThirds struct {
	Orientation fyne.TextAlign
}

func (t *twoThirds) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// this only works on 2 objects for the time being
	if len(objects) != 2 {
		return
	}

	var left, right float32
	padding := theme.Padding() // theme padding for consistency

	switch t.Orientation {
	case fyne.TextAlignLeading:
		left = size.Width * 2 / 3
		right = size.Width - left
	case fyne.TextAlignTrailing:
		right = size.Width * 2 / 3
		left = size.Width - right
	default:
		// unknown orientation.... an error case
		left = size.Width / 2
		right = size.Width / 2
	}

	// let's resize and reposition the left object
	objects[0].Resize(fyne.NewSize(left, size.Height))
	objects[0].Move(fyne.NewPos(padding, 0))

	// resize and position the right object
	objects[1].Resize(fyne.NewSize(right-padding, size.Height))
	objects[1].Move(fyne.NewPos(left+padding, 0))
}

func (t *twoThirds) MinSize(objects []fyne.CanvasObject) fyne.Size {
	// again, only works with two objects for the moment
	if len(objects) != 2 {
		return fyne.NewSize(0, 0)
	}
	// we have a left side
	left := objects[0].MinSize()

	// and we have a right side
	right := objects[1].MinSize()

	// add those up
	width := left.Width + right.Width

	// the height of the largest object
	height := fyne.Max(left.Height, right.Height)

	// and return them
	return fyne.NewSize(width, height)
}

type responsiveGrid struct{}

func (r *responsiveGrid) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	cols := 1
	if size.Width >= ((program.size.Width / 3) - (theme.Padding() * 2)) {
		cols = 2
	}

	count := len(objects)
	if count == 0 {
		return
	}

	// calculate tallest button
	cellHeight := float32(0)

	// range through the objects
	for _, obj := range objects {

		// get its min
		h := obj.MinSize().Height

		// and compare
		cellHeight = fyne.Max(h, cellHeight)
	}

	cellWidth := size.Width / float32(cols)
	padding := theme.Padding()

	for i, obj := range objects {
		row := i / cols
		col := i % cols
		x := float32(col) * cellWidth
		y := float32(row) * (cellHeight + padding)
		obj.Move(fyne.NewPos(x, y))
		obj.Resize(fyne.NewSize(cellWidth-padding, cellHeight))
	}
}

func (r *responsiveGrid) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(program.size.Width/4, program.size.Height/2)
}

type graph struct {
	widget.BaseWidget
	hd_map map[int]int
}

type graphRenderer struct {
	graph   *graph
	objects []fyne.CanvasObject
}

func (g *graph) CreateRenderer() fyne.WidgetRenderer {
	objects := []fyne.CanvasObject{}

	// Dummy object to force re-rendering
	bg := canvas.NewRectangle(color.Transparent)
	objects = append(objects, bg)

	// return your renderer
	return &graphRenderer{
		graph:   g,
		objects: objects,
	}
}

func (r *graphRenderer) Layout(size fyne.Size) {
	r.objects = []fyne.CanvasObject{makeGraph(r.graph.hd_map, size.Width, size.Height)}
}

// Minimum size for the graph
func (r *graphRenderer) MinSize() fyne.Size {
	if fyne.CurrentDevice().IsMobile() {
		return fyne.NewSize(300, 400)
	}
	return fyne.NewSize(600, 400)
}

func (r *graphRenderer) Refresh() {
	r.Layout(r.graph.Size())
	canvas.Refresh(r.graph)
}

func (r *graphRenderer) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *graphRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *graphRenderer) Destroy() {}
