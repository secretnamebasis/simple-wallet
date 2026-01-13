package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

func topbar() *fyne.Container {
	content := container.NewVBox(

		container.NewCenter(container.NewGridWithColumns(3,
			// we always want to know if we are connected
			container.NewCenter(program.labels.connection),
			// we also always want to know if we are logged in
			container.NewCenter(program.labels.loggedin),
			// gonna need to have access to an indexer
			container.NewCenter(program.labels.indexer),
		)),
	)

	// yeah... no sense in showing these settings right?
	if !fyne.CurrentDevice().IsMobile() {
		content.Add( // and have them fall horizontally
			container.NewCenter(container.NewGridWithColumns(2,

				// as well as if the web socket
				program.labels.ws_server,

				// also want to see if the rpc server is on
				program.labels.rpc_server,
			)))
	}

	// again, let's center this container
	return container.NewCenter(content)
}
