package main

import (
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
)

// here is the simple way we log out
func logout() {
	// update the header
	updateHeader(program.hyperlinks.logout)

	// if there is a wallet in memory and it saves
	if program.wallet != nil && program.wallet.Save_Wallet() == nil {

		// dump the keys
		program.entries.seed.SetText("")
		program.entries.seed.Refresh()
		program.entries.public.SetText("")
		program.entries.public.Refresh()
		program.entries.secret.SetText("")
		program.entries.secret.Refresh()

		// and as long as we are logged in...
		if program.preferences.Bool("loggedIn") {

			// and as long as there is an rpc server in memory
			if program.rpc_server != nil && globals.Arguments["--rpc-server"] != nil {
				// stop the rpc server
				program.rpc_server.RPCServer_Stop()

				// make it noticable
				program.labels.rpc_server.SetText("RPC: 🔴")

				// dump the creds
				program.entries.username.SetText("")
				program.entries.password.SetText("")

				// toggle the server
				program.toggles.server.SetSelected("off") // just in case

				// clear the rpc server from memory
				program.rpc_server = nil

				delete(globals.Arguments, "--rpc-server")
				delete(globals.Arguments, "--rpc-login")
				delete(globals.Arguments, "--rpc-bind")
			}

			// close out the wallet
			program.wallet.Close_Encrypted_Wallet()

			// dump it from memory
			program.wallet = nil

			// reset the balance
			program.labels.balance.SetText("BALANCE: 0")

			program.labels.loggedin.SetText("WALLET: 🔴")

			program.viewer_window.Close()

			// clear the cache
			program.caches.assets = []asset{}
			program.caches.info = rpc.GetInfo_Result{}
			program.caches.pool = rpc.GetTxPool_Result{}

		}
	}

	// the wallet is no longer logged in
	program.preferences.SetBool("loggedIn", false)

	// there is the off chance that they log out before they are registered
	if program.containers.register.Visible() {

		// stop and hide the registration activity
		program.activities.registration.Stop()
		program.activities.registration.Hide()

		// hide the label
		program.labels.counter.Hide()

		// hide the container
		program.containers.register.Hide()

		// hide the buttons
		program.buttons.register.Show()
	}

	// hide containers
	program.containers.dashboard.Hide()
	program.containers.send.Hide()
	program.containers.toolbox.Hide()

	// hide hyperlinks
	program.hyperlinks.logout.Hide()
	program.hyperlinks.tools.Hide()
	program.hyperlinks.lockscreen.Hide()
	program.hyperlinks.address.Hide()

	// hide labels
	program.labels.address.Hide()
	program.labels.balance.Hide()

	// hide buttons
	program.buttons.rpc_server.Hide()
	program.buttons.update_password.Hide()
	program.buttons.assets.Hide()

	// show labels
	program.labels.notice.Show()

	// show hyperlinks
	program.hyperlinks.login.Show()
	program.hyperlinks.login.Show()
	program.hyperlinks.create.Show()

	// update the header
	updateHeader(program.hyperlinks.home)

	program.window.SetTitle(program.name)

	// show home
	setContentAsHome()
}
