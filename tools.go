// this is the largest part of the repo, and the most fun.

package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

func tools() *fyne.Container {

	// let's do this when we click on tool
	program.hyperlinks.tools.OnTapped = func() {
		updateHeader(program.hyperlinks.tools)
		program.window.SetContent(program.containers.tools)
	}

	// let's set the functions for each of these links
	program.buttons.explorer.OnTapped = explorer
	program.buttons.encryption.OnTapped = encryption
	program.buttons.contracts.OnTapped = contracts

	// and then set them
	program.containers.toolbox = container.NewVBox(
		program.buttons.encryption,
		program.buttons.explorer,
		program.buttons.contracts,
	)

	// and now, let's hide them
	program.containers.toolbox.Hide()

	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewAdaptiveGrid(3,
			layout.NewSpacer(),
			program.containers.toolbox,
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
		program.containers.bottombar,
	)

}
