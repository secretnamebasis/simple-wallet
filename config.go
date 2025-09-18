package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

func configs() *fyne.Container {

	// here's what happens when we click on configs
	program.hyperlinks.configs.OnTapped = func() {
		updateHeader(program.hyperlinks.configs)
		program.buttons.connections.OnTapped = connections
		program.window.SetContent(program.containers.configs)
	}

	// let's make a simple way to manage the rpc server
	program.buttons.rpc_server.OnTapped = rpc_server

	// let's start off by hiding it
	program.buttons.rpc_server.Hide()

	// let's make a simple way to manage the rpc server
	program.buttons.update_password.OnTapped = passwordUpdate

	// let's start off by hiding it
	program.buttons.update_password.Hide()

	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewVBox(program.buttons.connections),
		container.NewVBox(program.buttons.rpc_server),
		container.NewVBox(program.buttons.update_password),
		// container.NewVBox(program.buttons.tx_priority),
		// container.NewVBox(program.buttons.ringsize	),
		layout.NewSpacer(),
		program.containers.bottombar,
	)
}

// simple way to maintain connection to one of the nodes hardcoded
func maintain_connection() {

	// we will track retries
	var retries int
	// track the height
	var height int64
	// the purpose of this function is to obtain topo height every second
	ticker := time.NewTicker(time.Second * 2)
	// this is an infinite loop
	for range ticker.C {
		// assuming the localhost connection works
		walletapi.Daemon_Endpoint = program.node.current

		// get the height directly from the daemon
		if getDaemonInfo().TopoHeight == 0 || walletapi.Connect(walletapi.Daemon_Endpoint) != nil {

			// update the label and show 0s
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("NODE: 🟡") // signalling unstable
				program.labels.height.SetText("BLOCK: 0000000")
				if program.buttons.register.Visible() {
					program.buttons.register.Disable()
				} else if program.activities.registration.Visible() {
					program.activities.registration.Stop()
					program.activities.registration.Hide()
				}
			})

			if program.preferences.Bool("mainnet") {
				// now we need to range and connect
				var fastest int64 = 10000 // we assume 10 second

				// now range through the nodes in the list
				for _, node := range program.node.list {

					// make a start time for determining how fast this goes
					start := time.Now()
					// here is a helper
					if err := testConnection(node.ip); err != nil {
						continue
					}
					// now that the connection has been tested, get the time
					result := time.Now().UnixMilli() - start.UnixMilli()

					// if the result is faster than fastest
					if result < fastest {

						// it is now the new fastest
						fastest = result

						// and it is now the current node
						program.node.current = node.ip
					}
				}
				// assuming the fastest connection works
				walletapi.Daemon_Endpoint = program.node.current
			}

			// re-test the connection
			if err := walletapi.Connect(walletapi.Daemon_Endpoint); err != nil {
				// show why?
				// showError(err)

				// be sure to drop the rpc server
				if program.rpc_server != nil {
					program.rpc_server = nil
				}
				//  if the number of tries is 1...
				if retries == 1 {
					fyne.DoAndWait(func() {

						// then notify the user
						showError(err)
						// update the label
						program.labels.connection.SetText("NODE: 🔴")
						program.labels.height.SetText("BLOCK: 0000000")
						program.buttons.send.Disable()
						program.entries.recipient.Disable()
						program.buttons.token_add.Disable()
						program.buttons.balance_rescan.Disable()
						program.buttons.contract_installer.Disable()
						program.buttons.contract_interactor.Disable()
					})

					// and if we are logged in, update to offline mode
					if program.preferences.Bool("loggedIn") {
						program.wallet.SetOfflineMode()
					}
					// don't break the loop
				}
				// increment the retries
				retries++
				fmt.Println("connection retry attempt", retries)
			}
		} else {

			// the connection should be working
			height = getDaemonInfo().TopoHeight
			// now if they are able to connect...
			// update the height and node label
			fyne.DoAndWait(func() {

				program.labels.connection.SetText("NODE: 🟢")

				// obviously registration is different
				if !program.buttons.register.Visible() &&
					!program.activities.registration.Visible() &&
					!program.wallet.IsRegistered() {

					// and let them register
					program.buttons.register.Show()
					program.buttons.register.Enable()
				}
			})

			// retries is reset
			retries = 0

			// simple way to see if height has changed
			if height > 0 {
				fyne.DoAndWait(func() {
					program.labels.height.SetText(
						"BLOCK: " + strconv.Itoa(int(walletapi.Get_Daemon_Height())),
					)
					program.labels.height.Refresh()
					program.buttons.send.Enable()
					program.entries.recipient.Enable()
					program.buttons.token_add.Enable()
					program.buttons.balance_rescan.Enable()
					program.buttons.contract_installer.Enable()
					program.buttons.contract_interactor.Enable()
				})
				// the above love to bark on signal interrupts
			}
			// also set the wallet to online mode
			if program.preferences.Bool("loggedIn") {
				program.wallet.SetOnlineMode()
			}
		}
	}
}

func connections() {
	var msg string = "Auto-connects to localhost"
	// post up the current node
	current_node := widget.NewLabel("")
	current_node.SetText("Current Node: " + program.node.current)

	// make a way for them to enter and address
	form_entry := widget.NewEntry()

	// build a pleasing and simple list
	label := widget.NewRichTextFromMarkdown("") // wrap it
	label.Wrapping = fyne.TextWrapWord
	set_label := func() {
		var opts string
		if program.toggles.network.Selected == "mainnet" {
			form_entry.PlaceHolder = "127.0.0.1:10102"
			// let's show off a list
			var list string

			// here are the hard coded nodes
			for _, node := range program.node.list {
				list += "- " + node.ip + " " + node.name + "\n\n"
			}
			opts = ", or fastest public node:\n\n" + list
		}

		label.ParseMarkdown(msg + opts)
	}
	set_label()
	changed := func(s string) {
		switch s {
		case "mainnet":
			program.toggles.network.SetSelected("mainnet")
			globals.Arguments["--testnet"] = false
			globals.Arguments["--simulator"] = false
			program.preferences.SetBool("mainnet", true)
			program.node.current = "127.0.0.1:10102"
			form_entry.PlaceHolder = "127.0.0.1:10102"
			current_node.SetText("Current Node: " + program.node.current)
			set_label()
			form_entry.Refresh()
			globals.InitNetwork()
		case "testnet":
			program.toggles.network.SetSelected("testnet")
			globals.Arguments["--testnet"] = true
			globals.Arguments["--simulator"] = false
			program.preferences.SetBool("mainnet", false)
			program.node.current = "127.0.0.1:40402"
			form_entry.PlaceHolder = program.node.current
			current_node.SetText("Current Node: " + program.node.current)
			label.ParseMarkdown(msg)
			form_entry.Refresh()
			globals.InitNetwork()
		case "simulator":
			program.toggles.network.SetSelected("simulator")
			globals.Arguments["--testnet"] = true
			globals.Arguments["--simulator"] = true
			program.preferences.SetBool("mainnet", false)
			program.node.current = "127.0.0.1:20000"
			form_entry.PlaceHolder = program.node.current
			current_node.SetText("Current Node: " + program.node.current)
			form_entry.Refresh()
			label.ParseMarkdown(msg)
			globals.InitNetwork()
		}
	}
	options := []string{"mainnet", "testnet", "simulator"}
	program.toggles.network.Options = options
	program.toggles.network.OnChanged = changed
	if program.toggles.network.Selected == "" {
		program.toggles.network.SetSelected("mainnet")
	}
	program.toggles.network.Horizontal = true

	if program.preferences.Bool("loggedIn") {
		program.toggles.network.Disable()
	} else {
		program.toggles.network.Enable()
	}

	// make a way for them to set the node endpoint
	set := widget.NewHyperlink("set connection", nil)
	set.Alignment = fyne.TextAlignCenter

	// so when they tap on it
	onTapped := func() {

		// obviously...
		if form_entry.Text == "" {
			showError(errors.New("cannot be empty"))
			return
		}

		// copy the form_entry
		endpoint := form_entry.Text

		// dump the entry
		form_entry.SetText("")

		// test the connection point
		if err := testConnection(endpoint); err != nil {
			showError(err)
			return
		}

		// attempt to connect
		if err := walletapi.Connect(endpoint); err != nil {
			showError(err)
			return
		} else {
			// tell the user how cool they are
			showInfoFast("Connection", "success", program.window)
		}
		// now the current node is the entry
		program.node.current = endpoint

		// change the label
		current_node.SetText("Current Node: " + program.node.current)

		// set the walletapi endpoint for the maintain_connection function
		walletapi.Daemon_Endpoint = program.node.current
	}

	// set the on tapped function
	set.OnTapped = onTapped

	form_entry.OnSubmitted = func(s string) {
		onTapped()
	}

	// walk the user through the process
	content := container.New(layout.NewVBoxLayout(),
		layout.NewSpacer(),
		container.NewCenter(program.toggles.network),
		form_entry,
		set,
		current_node,
		label,
		layout.NewSpacer(),
	)

	connect := dialog.NewCustom("Custom Node Connection", dismiss, content, program.window)

	// resize and show
	connect.Resize(program.size)
	connect.Show()
}

func rpc_server() {

	// so here are some creds
	program.entries.username.SetPlaceHolder("username")
	program.entries.password.SetPlaceHolder("p@55w0rd")
	// obviously, passwords are passwords
	program.entries.password.Password = true

	// let's position toggle horizontally
	program.toggles.server.Horizontal = true

	// simple options to choose from
	program.toggles.server.Options = []string{
		"off", "on",
	}

	// when they toggle the options
	onChanged := func(s string) {
		// if on...
		switch s {
		case "on":
			// let's assume an error
			var err error

			// and let's set the server as "on"
			program.toggles.server.SetSelected("on")

			// let's gather the creds from the text entries
			creds := program.entries.username.Text + ":" + program.entries.password.Text

			// now let's load up the global args with some creds
			globals.Arguments["--rpc-login"] = creds

			// turn on the rpc server
			globals.Arguments["--rpc-server"] = true

			// and then bind it to a ip; we are going to use localhost
			globals.Arguments["--rpc-bind"] = "127.0.0.1:10103"

			// now we'll label this server with the application's Unique ID
			program.rpc_server, err = rpcserver.RPCServer_Start(program.wallet, program.application.UniqueID())
			if err != nil {
				showError(err)
				return
			}

			// and change the label
			program.labels.rpc_server.SetText("RPC: 🟢")

		case "off": // but if the rpc server toggle is off

			// make sure it is off
			program.toggles.server.SetSelected("off")

			// and make sure to check if the server argument is in memory
			if globals.Arguments["--rpc-server"] != nil &&
				globals.Arguments["--rpc-server"].(bool) { // and check if it is turned on

				// and if the server is in memory
				if program.rpc_server != nil &&
					globals.Arguments["--rpc-server"] != nil { // along with its global argument

					// stop the server
					program.rpc_server.RPCServer_Stop()

					// and delete the following from the global arguments
					for _, arg := range []string{"--rpc-login", "--rpc-server", "--rpc-bind"} {
						delete(globals.Arguments, arg)
					}

					// the reset the label
					program.labels.rpc_server.SetText("RPC: 🔴")
				}
			}
		}
	}
	// set the on changed function
	program.toggles.server.OnChanged = onChanged

	// if there isn't anything toggled, set to off
	if program.toggles.server.Selected == "" {
		program.toggles.server.SetSelected("off")
	}

	// make a notice
	notice := makeCenteredWrappedLabel(`
The RPC Server allows for external apps to connect with the wallet. Be conscientious with credentials, and only have ON when necessary.

RPC server runs at http://127.0.0.1:10103 
	`)

	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		notice,
		program.entries.username, program.entries.password,
		container.NewCenter(program.toggles.server),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	rpc := dialog.NewCustom("rpc server", dismiss, content, program.window)
	rpc.Resize(program.size)
	rpc.Show()
}

func passwordUpdate() {
	// let's get some confirmation
	var pass_confirm *dialog.ConfirmDialog

	// allow press ENTER/RETURN passthrough
	program.entries.pass.OnSubmitted = func(s string) {
		pass_confirm.Confirm()
	}

	// get password callback function
	callback := func(b bool) {
		if !b { // in case of cancellation
			return
		}

		// copy the password
		password := program.entries.pass.Text

		// dump the entry
		program.entries.pass.SetText("")

		// check the password
		if ok := program.wallet.Check_Password(password); !ok {
			showError(errors.New("wrong password"))
			return
		}

		// now run the update dialog
		var update *dialog.CustomDialog

		// show them a new placeholder
		program.entries.password.SetPlaceHolder("n3w-w41137-p@55w0rd")

		// make sure it is a password entry widget
		program.entries.password.Password = true

		// here's what we are going to do...
		change_password := func() {

			// get the new password
			new_pass := program.entries.password.Text

			// dump the entry
			program.entries.password.SetText("")

			// now check for an err on the set password
			err := program.wallet.Set_Encrypted_Wallet_Password(new_pass)

			// if there is an error
			if err != nil {
				showError(err)
				return
			} else { // otherwise
				update.Dismiss() // close the password dialog
				// and notify the user of update
				showInfoFast("UPDATE PASSWORD", "Password has been successfully updated", program.window)
			}
		}

		// allow press ENTER/RETURN passthrough
		program.entries.password.OnSubmitted = func(s string) {
			change_password()
		}

		// in case they want to just click the button instead
		program.hyperlinks.save.OnTapped = change_password

		// set the content
		content := container.NewVBox(
			layout.NewSpacer(),
			program.entries.password,
			container.NewCenter(program.hyperlinks.save),
			layout.NewSpacer(),
		)

		// fill the dialog
		update = dialog.NewCustom("update password", dismiss, content, program.window)

		// resize and show it
		update.Resize(program.size)
		update.Show()

	}

	// create a simple form
	content := widget.NewForm(widget.NewFormItem("", program.entries.pass))

	// set the dialog with a pass entry field and the callback
	pass_confirm = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, content, callback, program.window)

	// and show it
	pass_confirm.Show()
}
