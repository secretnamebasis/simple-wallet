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
		if program.entries.wallet.Text != "" || program.entries.pass.Text != "" {
			program.entries.wallet.SetText("")
			program.entries.pass.SetText("")
		}

		// set container
		setContentAsHome()
	}

	// front and center
	program.labels.notice.Alignment = fyne.TextAlignCenter

	// let's be clear about the software
	program.labels.notice.SetText(`
THIS SOFTWARE IS ALPHA STAGE SOFTWARE
USE ONLY FOR TESTING & EVALUATION PURPOSES 
		`)

	return container.New(layout.NewVBoxLayout(),
		program.containers.topbar,
		layout.NewSpacer(),
		program.containers.dashboard,
		container.NewCenter(program.labels.notice),
		program.containers.send,
		program.containers.register,
		layout.NewSpacer(),
		program.containers.bottombar,
	)
}
