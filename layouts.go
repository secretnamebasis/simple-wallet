package main

import "fyne.io/fyne/v2"

type twoThirds struct{}

func (t *twoThirds) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// this only works on 2 objects for the time being
	if len(objects) != 2 {
		return
	}

	// we assume the following size for the  left object
	left := size.Width * 2 / 3

	// and we assume the following size for the right object
	right := size.Width - left

	// let's resize and reposition the left object
	objects[0].Resize(fyne.NewSize(left, size.Height))
	objects[0].Move(fyne.NewPos(0, 0))

	// we are going to use just a tiny bit of padding
	padding := float32(2.0)

	// let's resize and reposition the left object
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

	// the height of the left side is fine
	var height float32 = left.Height

	// unless it is bigger than the right side
	if left.Height > right.Height {
		height = right.Height
	}

	// and return them
	return fyne.NewSize(width, height)
}

type responsiveGrid struct{}

func (r *responsiveGrid) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	cols := 1
	if size.Width >= 600 {
		cols = 2
	}

	count := len(objects)
	if count == 0 {
		return
	}

	// calculate tallest button
	cellHeight := float32(0)
	for _, obj := range objects {
		h := obj.MinSize().Height
		if h > cellHeight {
			cellHeight = h
		}
	}

	cellWidth := size.Width / float32(cols)
	padding := float32(3)

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
	return fyne.NewSize(300, 300)
}
