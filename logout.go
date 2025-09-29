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
	if program.wallet != nil {

		// and as long as there is an rpc server in memory
		if program.rpc_server != nil && globals.Arguments["--rpc-server"] != nil {
			// stop the rpc server
			program.rpc_server.RPCServer_Stop()

			// make it noticable
			program.labels.rpc_server.SetText("RPC: 🔴")

			// dump the creds
			program.entries.username.SetText("")
			program.entries.password.SetText("")

			// toggle the servers off
			program.toggles.rpc_server.SetSelected("off")

			// clear the rpc server from memory
			program.rpc_server = nil

			delete(globals.Arguments, "--rpc-server")
			delete(globals.Arguments, "--rpc-login")
			delete(globals.Arguments, "--rpc-bind")
		}
		if program.ws_server != nil {
			program.ws_server.Stop()
			program.toggles.ws_server.SetSelected("off")

		}

		// close out the wallet
		program.wallet.Close_Encrypted_Wallet()
		program.wallet = nil // completely remove the wallet

		// reset the balance
		program.labels.balance.SetText("BALANCE: 0")

		program.labels.loggedin.SetText("WALLET: 🔴")

		// clear the cache
		program.caches.assets = []asset{}
		program.caches.info = rpc.GetInfo_Result{}
		program.caches.pool = rpc.GetTxPool_Result{}

		// close windows if any
		if program.encryption != nil {
			program.encryption.Close()
		}
		if program.contracts != nil {
			program.contracts.Close()
		}
		if program.explorer != nil {
			program.explorer.Close()
		}

	}

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
	program.hyperlinks.generate.Hide()

	// hide labels
	program.labels.address.Hide()
	program.labels.balance.Hide()

	// hide buttons
	program.buttons.rpc_server.Hide()
	program.buttons.ws_server.Hide()
	program.buttons.update_password.Hide()
	program.buttons.assets.Hide()
	program.buttons.balance_rescan.Hide()

	// show labels
	program.labels.notice.Show()

	// show hyperlinks
	program.hyperlinks.login.Show()
	program.hyperlinks.login.Show()
	program.hyperlinks.create.Show()

	// update the header
	updateHeader(program.hyperlinks.home)

	program.window.SetTitle(program.name)

	// now we are logged out
	program.preferences.SetBool("loggedIn", false)

	// show home
	setContentAsHome()
}
