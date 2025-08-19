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
		program.hyperlinks.connections.OnTapped = connections
		program.window.SetContent(program.containers.configs)
	}

	// let's make a simple way to manage the rpc server
	program.hyperlinks.rpc_server.OnTapped = rpc_server

	return container.New(layout.NewVBoxLayout(),
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewCenter(program.hyperlinks.connections),
		container.NewCenter(program.hyperlinks.rpc_server),
		layout.NewSpacer(),
		program.containers.bottombar,
	)
}

// simple way to maintain connection to one of the nodes hardcoded
func maintain_connection() {

	// we will track retries
	var retries int
	old_height := walletapi.Get_Daemon_Height()
	// this is an infinite loop
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// now if the wallet isn't connected...
		if !walletapi.Connected ||
			!walletapi.IsDaemonOnline() ||
			testConnection(program.node.current) != nil {

			// update the label and show 0s
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("🌐: 🟡")
				program.labels.height.SetText("⬡: 0000000")
				if program.buttons.register.Visible() {
					program.buttons.register.Disable()
				} else if program.activities.registration.Visible() {
					program.activities.registration.Stop()
					program.activities.registration.Hide()
				}
			})

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

			// now that the nodes have been tested, make the wallet endpoint the current node
			walletapi.Daemon_Endpoint = program.node.current

		}
		// then test the connection
		// this is hella slow because it is a dial with no context...
		if err := walletapi.Connect(walletapi.Daemon_Endpoint); err != nil {
			//  if the number of tries is 1...
			if retries == 1 {
				fyne.DoAndWait(func() {

					// then notify the user
					showError(
						errors.New("auto connect is not working as expected, please set custom endpoint"),
					)
					// update the label
					program.labels.connection.SetText("🌐: 🔴")
				})

				// and if we are logged in, update to offline mode
				if program.preferences.Bool("loggedIn") {
					program.wallet.SetOfflineMode()
				}
				// don't break the loop
			}
			// increment the retries
			retries++
		} else {
			// now if they are able to connect...
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("🌐: 🟢")
				if !program.buttons.register.Visible() &&
					!program.activities.registration.Visible() &&
					!program.wallet.IsRegistered() {
					program.buttons.register.Show()
				}
				program.buttons.register.Enable()
			})

			// retries is reset
			retries = 0

			// update the height and node label
			fyne.DoAndWait(func() {
				height := walletapi.Get_Daemon_Height()
				if height > old_height {
					old_height = height
					program.labels.height.SetText(
						"⬡: " + strconv.Itoa(int(walletapi.Get_Daemon_Height())),
					)
					program.labels.height.Refresh()
				}
				// also set the wallet to online mode
				if program.preferences.Bool("loggedIn") {
					program.wallet.SetOnlineMode()
				}
			})
		}
	}
}

func connections() {

	// post up the current node
	current_node := widget.NewLabel("Current Node: " + program.node.current)

	// make a way for them to enter and address
	form_entry := widget.NewEntry()
	form_entry.PlaceHolder = "127.0.0.1:10102"

	// make a way for them to set the node endpoint
	set := widget.NewHyperlink("set connection", nil)
	set.Alignment = fyne.TextAlignCenter

	// so when they tap on it
	set.OnTapped = func() {

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
			showInfo("Connection", "success")
		}
		// now the current node is the entry
		program.node.current = endpoint

		// change the label
		current_node.SetText("Current Node: " + program.node.current)

		// set the walletapi endpoint for the maintain_connection function
		walletapi.Daemon_Endpoint = program.node.current
	}

	// let's show off a list
	var list string

	// here are the hard coded nodes
	for _, node := range program.node.list {
		list += "- " + node.ip + " " + node.name + "\n\n"
	}

	// build a pleasing and simple list
	msg := widget.NewRichTextFromMarkdown(
		fmt.Sprintf(
			"%s\n\n%s", "Auto-connects to localhost first, or fastest public node:", list,
		),
	) // wrap it
	msg.Wrapping = fyne.TextWrapWord

	// walk the user through the process
	connect := dialog.NewCustom("Custom Node Connection", dismiss,
		container.New(layout.NewVBoxLayout(),
			layout.NewSpacer(),
			form_entry,
			set,
			current_node,
			msg,
			layout.NewSpacer(),
		), program.window,
	)

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
	program.toggles.server.OnChanged = func(s string) {
		// if on...
		if s == "on" {
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
			program.labels.rpc_server.SetText("📡: 🟢")

		} else if s == "off" { // but if the rpc server toggle is off

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
					program.labels.rpc_server.SetText("📡: 🔴")
				}
			}
		}
	}

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
