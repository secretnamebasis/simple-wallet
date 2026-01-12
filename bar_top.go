package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func topbar() *fyne.Container {

	content := container.NewVBox(
		// we are going to want to watch the current height, always
		container.NewCenter(
			// we always want to know if we are connected
			program.labels.height,
		),
		container.NewGridWithColumns(2,
			// we always want to know if we are connected
			program.labels.connection,
			// we also always want to know if we are logged in
			program.labels.loggedin,
		),
		// and have them fall horizontally
		container.NewHBox(

			// as well as if the web socket
			program.labels.ws_server,

			// also want to see if the rpc server is on
			program.labels.rpc_server,

			// gonna need to have access to an indexer
			program.labels.indexer,
		),
	)
	// again, let's center this container
	return container.NewCenter(content)
}
