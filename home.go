package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
)

func home() *fyne.Container {

	// when ever we go home, let's do this
	program.hyperlinks.home.OnTapped = func() {
		updateHeader(program.hyperlinks.home)

		// sometimes things fall through the cracks in the login screen
		if program.entries.wallet.entry.Text != "" || program.entries.pass.Text != "" {
			program.entries.wallet.entry.SetText("")
			program.entries.pass.SetText("")
		}

		// set container
		setContentAsHome()
	}

	// let's be clear about the software
	program.labels.notice = makeCenteredWrappedLabel(`
THIS SOFTWARE IS ALPHA STAGE SOFTWARE
USE ONLY FOR TESTING & EVALUATION PURPOSES 
`)

	return container.New(layout.NewVBoxLayout(),
		program.containers.topbar,
		layout.NewSpacer(),
		program.labels.notice,
		container.NewAdaptiveGrid(2,
			container.NewCenter(
				container.NewHBox(
					program.labels.address,
					program.hyperlinks.address,
				)),
			container.NewCenter(
				program.labels.balance,
			),
		),
		program.containers.send,
		program.containers.register,
		program.containers.dashboard,
		layout.NewSpacer(),
		program.containers.bottombar,
	)
}
