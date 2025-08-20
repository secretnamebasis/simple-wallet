package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func bottombar() *fyne.Container {
	// let's center this container
	return container.NewCenter(
		// and let's make them all fall horizontally
		container.NewHBox(
			// we have a home link
			program.hyperlinks.home,
			// we have some tools that are accessible at login
			program.hyperlinks.tools,
			// we have some basic configs, more after login
			program.hyperlinks.configs,
			// here is a simple login/logout pattern
			program.hyperlinks.login, program.hyperlinks.logout,

			// // make a simple way to lock the screen after login
			program.hyperlinks.lockscreen,
		),
	)
}
