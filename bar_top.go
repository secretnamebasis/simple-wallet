package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func topbar() *fyne.Container {

	// again, let's center this container
	return container.NewCenter(
		// and have them fall horizontally
		container.NewHBox(

			// we are going to want to watch the current height, always
			program.labels.height,

			// we always want to know if we are connected
			program.labels.connection,

			// and we also always want to know if we are logged in
			program.labels.loggedin,

			// lastly, we want to see if this server is on
			program.labels.rpc_server,
		),
	)
}
