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

			// we also always want to know if we are logged in
			program.labels.loggedin,

			// as well as if the web socket
			program.labels.ws_server,

			// also want to see if the rpc server is on
			program.labels.rpc_server,
		),
	)
}
