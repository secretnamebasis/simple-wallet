package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/chzyer/readline"
	"github.com/creachadair/jrpc2/handler"
	"github.com/deroproject/derohe/blockchain"
	derodrpc "github.com/deroproject/derohe/cmd/derod/rpc"
	"github.com/deroproject/derohe/config"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/metrics"
	"github.com/deroproject/derohe/p2p"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
	"gopkg.in/natefinch/lumberjack.v2"
)

func configs() *fyne.Container {

	// here's what happens when we click on configs
	program.hyperlinks.configs.OnTapped = func() {
		updateHeader(program.hyperlinks.configs)
		program.buttons.connections.OnTapped = connections
		program.window.SetContent(program.containers.configs)
	}

	// here is the simulator
	program.buttons.simulator.OnTapped = simulator

	// let's make a simple way to connect via websocket
	program.buttons.ws_server.OnTapped = ws_server
	// and we'll hide it for now
	program.buttons.ws_server.Hide()

	// let's make a simple way to manage the rpc server
	program.buttons.rpc_server.OnTapped = rpc_server

	// let's start off by hiding it
	program.buttons.rpc_server.Hide()

	// let's make a simple way to update the password
	program.buttons.update_password.OnTapped = passwordUpdate

	// let's start off by hiding it
	program.buttons.update_password.Hide()

	// let's make a way to rescan balances manaually
	program.buttons.balance_rescan.OnTapped = balance_rescan
	// and let's hide it for now
	program.buttons.balance_rescan.Hide()

	// let's take care of those pesky notifications
	program.buttons.notifications.OnTapped = notifications

	// and here is the box it goes in
	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewAdaptiveGrid(3,
			layout.NewSpacer(),
			container.NewVBox(
				program.buttons.simulator,
				program.buttons.connections,
				program.buttons.ws_server,
				program.buttons.rpc_server,
				program.buttons.balance_rescan,
				program.buttons.update_password,
				program.buttons.notifications,
			),
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
		// container.NewVBox(program.buttons.tx_priority),
		// container.NewVBox(program.buttons.ringsize),
		program.containers.bottombar,
	)
}

// simple way to maintain connection to one of the nodes hardcoded
func maintain_connection() {

	// we will track retries
	var retries, height int64 // track the height

	// the purpose of this function is to obtain topo height every 2 seconds
	ticker := time.NewTicker(time.Second * 2)

	// so before we get started, let's assume that localhost is "first"
	program.node.current = program.node.list[1].ip

	// now if you have stated a preference...
	if program.node.list[0].ip != "" {
		program.node.current = program.node.list[0].ip
	}
	var isDancing bool

	// this is an 2 second loop
	for range ticker.C {
		// assuming the localhost connection works, if not preference
		walletapi.Daemon_Endpoint = program.node.current
		// the connection should be working
		height = walletapi.Get_Daemon_TopoHeight()
		// get the height directly from the daemon
		if height == 0 || // if it is not zero, it will advance
			// otherwise, try to connect to the walletapi
			walletapi.Connect(walletapi.Daemon_Endpoint) != nil { // if it fails...

			// update the label and show dancing bit
			dance := func() {
				var (
					stop   = func() { isDancing = false }
					update = func(msg string) {
						text := "BLOCK: " + msg
						fyne.DoAndWait(func() { program.labels.height.SetText(text) })
						time.Sleep(100 * time.Millisecond)
					}

					bit  uint16 = 1
					bits uint16 = bit // start with a bit

					max uint8 = 13 // zero index

					isConnected bool = strings.Contains(program.labels.connection.Text, "✅")
				)

				for !isConnected {
					isDancing = true
					defer stop()

					for range max {
						isConnected = strings.Contains(program.labels.connection.Text, "✅")

						if isConnected {
							return
						}

						update(fmt.Sprintf("%014b", bits))
						bits <<= bit // shift left, fill rightmost bitwith 1
					}
					for range max {
						isConnected = strings.Contains(program.labels.connection.Text, "✅")

						if isConnected {
							return
						}

						update(fmt.Sprintf("%014b", bits))
						bits >>= bit // shift right, drop rightmost bit
					}
				}
			}
			if !isDancing {
				go dance()
			}

			fyne.DoAndWait(func() {
				program.labels.connection.SetText("NODE: 🟡") // signalling unstable
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
				logger.Info("Ranging node list")
				// now range through the nodes in the list
				for _, node := range program.node.list {

					// make a start time for determining how fast this goes
					start := time.Now()
					// here is a helper
					if err := testConnection(node.ip); err != nil {
						continue
					}
					if node.name == "preferred" {
						logger.Info("Connecting to preferred node")
						break
					}
					// now that the connection has been tested, get the time
					result := time.Now().UnixMilli() - start.UnixMilli()

					// if the result is faster than fastest
					if result < fastest {

						// it is now the new fastest
						fastest = result

						// and it is now the current node
						program.node.current = node.ip

						logger.Info("Fastest node", node.name, node.ip, "ping", result)
					}
				}
				// assuming the fastest connection works
				walletapi.Daemon_Endpoint = program.node.current
			}
			logger.Info(fmt.Sprintf("Connecting to: %s", program.node.current))
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
						if strings.Contains(err.Error(), "Mainnet/TestNet") {
							showError(errors.New("please visit connections page and select appropriate network"), program.window)
						} else {
							showError(err, program.window)

						}
						// update the label
						program.labels.connection.SetText("NODE: 🔴")
						program.labels.height.SetText("BLOCK: 00000000000000")
						program.buttons.send.Disable()
						program.entries.recipient.Disable()
						program.buttons.token_add.Disable()
						program.buttons.balance_rescan.Disable()
					})

					// and if we are logged in, update to offline mode
					if program.preferences.Bool("loggedIn") {
						program.wallet.SetOfflineMode()
					}
					// don't break the loop
				}
				// increment the retries
				retries++
				logger.Error(errors.New("connection"), "connection retry attempt", retries)
			}
		} else {

			// now if they are able to connect...
			// update the height and node label
			fyne.DoAndWait(func() {

				program.labels.connection.SetText("NODE: ✅")
				program.labels.current_node.SetText("Current Node: " + program.node.current)
				program.labels.current_node.Refresh()
				// obviously registration is different
				if program.preferences.Bool("loggedIn") && program.wallet != nil {
					if !program.wallet.IsRegistered() {
						program.buttons.register.Enable()
					}
					if !program.activities.registration.Visible() {
						// and let them register
						program.buttons.register.Show()
					}
				}
			})

			// retries is reset
			retries = 0

			// simple way to see if height has changed
			if height >= walletapi.Get_Daemon_Height() {
				fyne.DoAndWait(func() {
					program.labels.height.SetText(
						fmt.Sprintf(("BLOCK: %0" + strconv.Itoa(len(max_height)) + "d"), height),
					)
					program.labels.height.Refresh()
					program.buttons.send.Enable()
					program.entries.recipient.Enable()
					program.buttons.token_add.Enable()
					program.buttons.balance_rescan.Enable()
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

	program.sliders.network.Step = 0.0001
	program.sliders.network.Orientation = widget.Horizontal

	// post up the current node
	program.labels.current_node = makeCenteredWrappedLabel("")
	program.labels.current_node.SetText("Current Node: " + program.node.current)

	// build a pleasing and simple list
	program.tables.connections = widget.NewTable(
		func() (rows int, cols int) { return len(program.node.list), 2 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			switch tci.Col {
			case 0:
				label.SetText(program.node.list[tci.Row].name)
			case 1:
				label.SetText("not saved")
				if program.node.list[tci.Row].ip != "" {
					label.SetText(program.node.list[tci.Row].ip)
				}
			}
		},
	)
	for i := range 2 {
		program.tables.connections.SetColumnWidth(i, 200)
	}
	program.tables.connections.HideSeparators = true
	program.tables.connections.OnSelected = func(id widget.TableCellID) {
		var data string
		if id.Col == 0 && id.Row > 1 {
			data = program.node.list[id.Row].name

		} else if id.Col == 1 {
			if program.node.list[id.Row].ip != "" {
				data = program.node.list[id.Row].ip
			}
		}
		if data != "" {
			program.application.Clipboard().SetContent(data)
			program.tables.connections.UnselectAll()
			program.tables.connections.Refresh()
			showInfoFast("Copied", data, program.window)
		}
	}

	if program.preferences.Bool("loggedIn") {
		program.sliders.network.Disable()
	} else {
		program.sliders.network.Enable()
	}

	save := widget.NewHyperlink("save connection", nil)

	save.Alignment = fyne.TextAlignCenter

	save.OnTapped = func() {
		// obviously...
		if program.entries.node.Text == "" {
			showError(errors.New("cannot be empty"), program.window)
			return
		}
		endpoint := program.entries.node.Text
		// program.entries.node.SetText("")

		// test the connection point
		if err := testConnection(endpoint); err != nil {
			showError(err, program.window)
			return
		}
		program.node.list[0] = struct {
			ip   string
			name string
		}{ip: endpoint, name: "preferred"}

		file, err := os.Create("preferred.conf")
		if err != nil {
			showError(err, program.window)
			return
		}
		if _, err := io.WriteString(file, endpoint); err != nil {
			showError(err, program.window)
			return
		}
		path, _ := filepath.Abs(file.Name())

		showInfo("Saved Preferred", "node endpoint saved to "+path, program.window)
		program.tables.connections.Refresh()
	}
	// make a way for them to set the node endpoint
	set := widget.NewHyperlink("set connection", nil)
	set.Alignment = fyne.TextAlignCenter

	// so when they tap on it
	onTapped := func() {

		// obviously...
		if program.entries.node.Text == "" {
			showError(errors.New("cannot be empty"), program.window)
			return
		}

		// copy the program.entries.node
		endpoint := program.entries.node.Text

		// test the connection point
		if err := testConnection(endpoint); err != nil {
			showError(err, program.window)
			return
		}

		// attempt to connect
		if err := walletapi.Connect(endpoint); err != nil {
			showError(err, program.window)
			return
		} else {
			// tell the user how cool they are
			showInfoFast("Connection", "success", program.window)
		}
		// now the current node is the entry
		program.node.current = endpoint

		// change the label
		program.labels.current_node.SetText("Current Node: " + program.node.current)

		// set the walletapi endpoint for the maintain_connection function
		walletapi.Daemon_Endpoint = program.node.current
	}

	// set the on tapped function
	set.OnTapped = onTapped

	program.entries.node.OnSubmitted = func(s string) {
		onTapped()
	}

	desiredSize := fyne.NewSize(450, 200)

	fixedSizeTable := container.NewGridWrap(desiredSize, program.tables.connections)
	centered := container.NewCenter(fixedSizeTable)

	// walk the user through the process
	content := container.NewBorder(
		container.NewVBox(
			container.NewAdaptiveGrid(3, program.labels.mainnet, program.labels.testnet, program.labels.simulator),
			program.sliders.network,
			program.entries.node,
			container.NewAdaptiveGrid(2, save, set),
			program.labels.current_node,
			program.labels.notice,
		),
		nil,
		nil,
		nil,
		centered,
	)
	if program.sliders.network.Value < 0.5 {
		program.sliders.network.SetValue(0.1337)
	}
	program.sliders.network.Refresh()
	connect := dialog.NewCustom("Custom Node Connection", dismiss, content, program.window)

	// resize and show
	size := fyne.NewSize(program.size.Width/3*2, program.size.Height/3*2)
	connect.Resize(size)
	connect.Show()
}

func ws_server() {

	if !program.entries.port.Disabled() {
		program.entries.port.SetText("44326")
	}

	notice := makeCenteredWrappedLabel("WS Server runs at ws://127.0.0.1:" + program.entries.port.Text + "/xswd")

	// let's position toggle horizontally
	program.toggles.ws_server.Horizontal = true

	// simple options to choose from
	program.toggles.ws_server.Options = []string{
		"off", "on",
	}
	onChanged := func(s string) {
		switch s {
		case "on":

			var p int
			var err error
			port := program.entries.port.Text
			if port != "" {
				p, err = strconv.Atoi(port)
				if err != nil {
					showError(err, program.window)
					program.toggles.ws_server.SetSelected("off")
					return
				} else if p < 10000 {
					showError(errors.New("port must be above 10000"), program.window)
					program.toggles.ws_server.SetSelected("off")
					return
				}
			}
			notice.SetText("WS Server runs at ws://127.0.0.1:" + program.entries.port.Text + "/xswd")
			program.entries.port.Disable()

			program.ws_server = xswdServer(p)

			// let's set some custom methods
			for _, each := range []struct {
				method      string
				handlerfunc handler.Func
			}{
				{
					method:      "GetAssets",
					handlerfunc: handler.New(getAssets),
				},
				{
					method:      "GetAssetBalance",
					handlerfunc: handler.New(getAssetBalance),
				},
				{
					method:      "AttemptEPOCHWithAddr",
					handlerfunc: handler.New(attemptEPOCHWithAddr),
				},
			} {
				program.ws_server.SetCustomMethod(each.method, each.handlerfunc)
			}

			if program.ws_server != nil && program.ws_server.IsRunning() {
				// assuming there are no errors here...
				program.toggles.ws_server.SetSelected("on")
				program.labels.ws_server.SetText("WS: ✅")
			}
		case "off":
			program.toggles.ws_server.SetSelected("off")
			program.labels.ws_server.SetText("WS: 🔴")
			if program.ws_server != nil {
				program.ws_server.Stop()
			}
			program.entries.port.Enable()
			// default:
		}
	}

	program.toggles.ws_server.OnChanged = onChanged

	// if there isn't anything toggled, set to off
	if program.toggles.ws_server.Selected == "" {
		program.toggles.ws_server.SetSelected("off")
	}

	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		makeCenteredWrappedLabel(`
The WS Server allows for external apps to connect with the wallet. 

Application requests will arrive as pop-ups for confirmation or dismissal.

Only have ON when necessary.
		`),
		program.entries.port,
		notice,
		container.NewCenter(program.toggles.ws_server),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	ws := dialog.NewCustom("ws server", dismiss, content, program.window)
	ws.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/2))
	ws.Show()
	ws.SetOnClosed(func() {
		// let's be clear about the software
		program.labels.notice = makeCenteredWrappedLabel(`
THIS SOFTWARE IS ALPHA STAGE SOFTWARE
USE ONLY FOR TESTING & EVALUATION PURPOSES 
`)
	})
}

func rpc_server() {

	// so here are some creds
	program.entries.username.SetPlaceHolder("username")
	if program.entries.password.Text == "" {
		program.entries.password.SetText(randomWords(4, "-"))
	}
	program.entries.password.SetPlaceHolder("p@55w0rd")

	// obviously, passwords are passwords
	program.entries.password.Password = true

	// let's position toggle horizontally
	program.toggles.rpc_server.Horizontal = true

	// simple options to choose from
	program.toggles.rpc_server.Options = []string{
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
			program.toggles.rpc_server.SetSelected("on")

			// let's gather the creds from the text entries

			username := program.entries.username.Text

			password := program.entries.password.Text

			creds := username + ":" + password

			if username == "" && password == "" {
				creds = "" // just in case
			} else {
				// now let's load up the global args with some creds
				globals.Arguments["--rpc-login"] = creds
			}

			// turn on the rpc server
			globals.Arguments["--rpc-server"] = true

			// and then bind it to a ip; we are going to use localhost
			globals.Arguments["--rpc-bind"] = "127.0.0.1:10103"

			// now we'll label this server with the application's Unique ID
			program.rpc_server, err = rpcserver.RPCServer_Start(program.wallet, program.application.UniqueID())
			if err != nil {
				showError(err, program.window)
				return
			}

			// and change the label
			program.labels.rpc_server.SetText("RPC: ✅")

		case "off": // but if the rpc server toggle is off

			// make sure it is off
			program.toggles.rpc_server.SetSelected("off")

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
	program.toggles.rpc_server.OnChanged = onChanged

	// if there isn't anything toggled, set to off
	if program.toggles.rpc_server.Selected == "" {
		program.toggles.rpc_server.SetSelected("off")
	}

	// make a notice
	notice := makeCenteredWrappedLabel(`
The RPC Server allows for external apps to connect with the wallet. Be conscientious with credentials, and only have ON when necessary.

RPC server runs at http://127.0.0.1:10103/json_rpc 
	`)

	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		notice,
		program.entries.username, program.entries.password,
		container.NewCenter(program.toggles.rpc_server),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	rpc := dialog.NewCustom("rpc server", dismiss, content, program.window)
	rpc.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/2))
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
			showError(errors.New("wrong password"), program.window)
			return
		}

		// now run the update dialog
		var update *dialog.CustomDialog

		// make sure it is a password entry widget
		program.entries.pass.Password = true

		// here's what we are going to do...
		change_password := func() {

			// get the new password
			new_pass := program.entries.pass.Text

			// dump the entry
			program.entries.pass.SetText("")

			// now check for an err on the set password
			err := program.wallet.Set_Encrypted_Wallet_Password(new_pass)

			// if there is an error
			if err != nil {
				showError(err, program.window)
				return
			} else { // otherwise
				update.Dismiss() // close the password dialog
				// and notify the user of update
				showInfoFast("UPDATE PASSWORD", "Password has been successfully updated", program.window)
			}
		}

		// allow press ENTER/RETURN passthrough
		program.entries.pass.OnSubmitted = func(s string) {
			change_password()
		}

		// in case they want to just click the button instead
		program.hyperlinks.save.OnTapped = change_password

		// set the content
		content := container.NewVBox(
			layout.NewSpacer(),
			program.entries.pass,
			container.NewCenter(program.hyperlinks.save),
			layout.NewSpacer(),
		)

		// fill the dialog
		update = dialog.NewCustom("update password", dismiss, content, program.window)

		// resize and show it
		update.Resize(password_size)
		update.Show()

	}

	// create a simple form
	content := widget.NewForm(widget.NewFormItem("", program.entries.pass))

	// set the dialog with a pass entry field and the callback
	pass_confirm = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, content, callback, program.window)

	// and show it
	pass_confirm.Show()
}

func simulator() {

	program.buttons.simulation.OnTapped = func() {
		if strings.Contains(program.buttons.simulation.Text, "ON") {

			go func() {
				fyne.DoAndWait(func() {
					program.buttons.simulation.SetText("TURN SIMULATOR OFF")
					program.buttons.simulation.Refresh()
					program.sliders.network.Step = 0.0001
					program.sliders.network.SetValue(0.85)
				})
			}()

			// program.preferences.SetBool("mainnet", false)
			// let's turn on a simulation of the blockchain
			// before we get started, let's clear something up
			globals.Arguments = nil // now that we have taken care of that...
			// let's go pretend we are the captain
			genesis_seed := "0206a2fca2d2da068dfa8f792ef190a352d656910895f6c541d54877fca95a77"

			create_wallet := func(n, s string) *walletapi.Wallet_Disk {

				b, err := hex.DecodeString(s)
				if err != nil {
					panic(err) // someting is wong, doc
				}

				// let's find the old one
				filename := filepath.Join(globals.GetDataDirectory(), n)
				// and get rid of it
				os.Remove(filename)

				// create the wallet
				wallet, err := walletapi.Create_Encrypted_Wallet(filename, "", new(crypto.BNRed).SetBytes(b))
				if err != nil {
					panic(err) // not good
				}

				// set the network
				wallet.SetNetwork(false)

				// save
				wallet.Save_Wallet()
				return wallet
			}
			filename := filepath.Join(globals.GetDataDirectory(), "genesis")
			genesis_wallet := create_wallet(filename, genesis_seed)

			genesis_tx := transaction.Transaction{
				Transaction_Prefix: transaction.Transaction_Prefix{
					Version: 1,
					Value:   112345,
				},
			}
			// every block has a miner..
			// the genesis_tx will use genesis wallet public key
			copy(genesis_tx.MinerAddress[:], genesis_wallet.GetAddress().PublicKey.EncodeCompressed())

			// now serialize the transaction into bytes
			b := genesis_tx.Serialize()

			// and stringify the bytes
			tx := fmt.Sprintf("%x", b)

			// // now config the testnet
			config.Testnet.Genesis_Tx = tx // mainnet uses the same tx
			config.Mainnet.Genesis_Tx = config.Testnet.Genesis_Tx

			// // generate the genesis block
			genesis_block := blockchain.Generate_Genesis_Block()

			// // get the hash of the block
			genesis_hash := genesis_block.GetHash()

			// // now config the testnet
			config.Testnet.Genesis_Block_Hash = genesis_hash // mainnet uses the same hash
			config.Mainnet.Genesis_Block_Hash = config.Testnet.Genesis_Block_Hash

			// chose where the endpoint location
			daemon_endpoint := "127.0.0.1:20000"

			// here is a list of arguments
			globals.Arguments = map[string]interface{}{
				"--rpc-bind":     daemon_endpoint,
				"--testnet":      true,
				"--debug":        true, // to get more info
				"--simulator":    true, // obviously
				"--p2p-bind":     ":0",
				"--getwork-bind": "127.0.0.1:10100",
			}

			l, lerr := readline.NewEx(&readline.Config{
				//Prompt:          "\033[92mDERO:\033[32m»\033[0m",
				Prompt:      "\033[92mDEROSIM:\033[32m>>>\033[0m ",
				HistoryFile: filepath.Join(os.TempDir(), "derosim_readline.tmp"),
				// AutoComplete:    completer,
				InterruptPrompt: "^C",
				EOFPrompt:       "exit",

				HistorySearchFold: true,
				// FuncFilterInputRune: filterInput,
			})
			if lerr != nil {
				fmt.Printf("Error starting readline err: %s\n", lerr)
				return
			}
			defer l.Close()

			// now, we'll init the network
			globals.InitNetwork()

			// parse arguments and setup logging , print basic information
			globals.InitializeLog(l.Stdout(), &lumberjack.Logger{
				Filename:   filepath.Join(globals.GetDataDirectory(), "simulator"+".log"),
				MaxSize:    100, // megabytes
				MaxBackups: 2,
			})

			// let's clean up anything that was here before
			os.RemoveAll(globals.GetDataDirectory())

			// lets create data directories
			if err := os.MkdirAll(globals.GetDataDirectory(), 0750); err != nil {
				panic(err)
			}

			simulation := map[string]interface{}{
				"--simulator": true,
			}
			var err error
			program.node.simulator_chain, err = blockchain.Blockchain_Start(simulation) //start chain in simulator mode
			if err != nil {
				panic(err)
			}

			simulation["chain"] = program.node.simulator_chain

			p2p.P2P_Init(simulation)

			go derodrpc.Getwork_server()
			// we should probably consider the "toggle" very seriously
			program.simulator_server, err = derodrpc.RPCServer_Start(simulation)
			if err != nil {
				panic(err)
			}
			// and let's simulate a bunch of users
			program.node.simulator_wallets = []*walletapi.Wallet_Disk{}
			program.node.simulator_rpcservers = []*rpcserver.RPCServer{}
			simulation_seeds := []string{
				"171eeaa899e360bf1a8ada7627aaea9fdad7992463581d935a8838f16b1ff51a",
				"193faf64d79e9feca5fce8b992b4bb59b86c50f491e2dc475522764ca6666b6b",
				"2e49383ac5c938c268921666bccfcb5f0c4d43cd3ed125c6c9e72fc5620bc79b",
				"1c8ee58431e21d1ef022ccf1f53fec36f5e5851d662a3dd96ced3fc155445120",
				"19182604625563f3ff913bb8fb53b0ade2e0271ca71926edb98c8e39f057d557",
				"2a3beb8a57baa096512e85902bb5f1833f1f37e79f75227bbf57c4687bfbb002",
				"055e43ebff20efff612ba6f8128caf990f2bf89aeea91584e63179b9d43cd3ab",
				"2ccb7fc12e867796dd96e246aceff3fea1fdf78a28253c583017350034c31c81",
				"279533d87cc4c637bf853e630480da4ee9d4390a282270d340eac52a391fd83d",
				"03bae8b71519fe8ac3137a3c77d2b6a164672c8691f67bd97548cb6c6f868c67",
				"2b9022d0c5ee922439b0d67864faeced65ebce5f35d26e0ee0746554d395eb88",
				"1a63d5cf9955e8f3d6cecde4c9ecbd538089e608741019397824dc6a2e0bfcc1",
				"10900d25e7dc0cec35fcca9161831a02cb7ed513800368529ba8944eeca6e949",
				"2af6630905d73ee40864bd48339f297908a0731a6c4c6fa0a27ea574ac4e4733",
				"2ac9a8984c988fcb54b261d15bc90b5961d673bffa5ff41c8250c7e262cbd606",
				"040572cec23e6df4f686192b776c197a50591836a3dd02ba2e4a7b7474382ccd",
				"2b2b029cfbc5d08b5d661e6fa444102d387780bec088f4dd41a4a537bf9762af",
				"1812298da90ded6457b2a20fd52d09f639584fb470c715617db13959927be7f8",
				"1eee334e1f533aa1ac018124cf3d5efa20e52f54b05e475f6f2cff3476b4a92f",
				"2c34e7978ce249aebed33e14cdd5177921ecd78fbe58d33bbec21f22b80af7a5",
				"083e7fe96e8415ea119ec6c4d0ebe233e86b53bd4e2f7598505317efc23ae34b",
				"0fd7f8db0ed6cbe3bf300258619d8d4a2ff8132ef3c896f6e3fa65a6c92bdf9a",
			}
			// the rpc servers are going to be turned on automatically
			// program.toggles.ws_server.Disable() // do we disable here? I was pretty sure the simulator doesn't auto turn on...
			program.toggles.rpc_server.Disable()
			program.entries.username.Disable()
			program.entries.password.Disable()
			for i, seed := range simulation_seeds {
				n := "simulation_wallet_" + strconv.Itoa(i) + ".db"
				wallet := create_wallet(n, seed)
				if err := program.node.simulator_chain.Add_TX_To_Pool(wallet.GetRegistrationTX()); err != nil {
					panic(err)
				}
				// point the wallet at the daemon
				wallet.SetDaemonAddress(daemon_endpoint)
				wallet.SetOnlineMode() // turn it on

				// choose where the wallet will serve from
				wallet_endpoint := "127.0.0.1:" + strconv.Itoa(30000+i)
				globals.Arguments["--rpc-bind"] = wallet_endpoint

				// now start the server endpoint
				if r, err := rpcserver.RPCServer_Start(wallet, n); err != nil {
					panic(err)
				} else {
					program.node.simulator_rpcservers = append(program.node.simulator_rpcservers, r)
				}
				program.node.simulator_wallets = append(program.node.simulator_wallets, wallet)
				time.Sleep(time.Millisecond * 20) // little breathing room
			}

			// now let's go mine a block
			single_block := func() error {
				// every block has a id, or a hash
				var blid crypto.Hash

				for {
					// using the genesis wallet, get a block and miniblock template
					bl, mbl, _, _, err := program.node.simulator_chain.Create_new_block_template_mining(genesis_wallet.GetAddress())
					if err != nil {
						return err
					}
					// now let's get the timestamp of the block
					ts := bl.Timestamp
					// and let's serialize the miniblock we used
					serial := mbl.Serialize()

					// and let's just accept it as is
					if _, blid, _, err = program.node.simulator_chain.Accept_new_block(ts, serial); err != nil {
						msg := "please completely restart wallet software to create new simulation"
						return errors.New(msg)
					} else if !blid.IsZero() {
						// assuming that the hash is not zero, break the loop
						break
					}
				}
				return nil
			}
			if err := single_block(); err != nil {
				showError(err, program.window)
				fyne.DoAndWait(func() {

					program.sliders.network.Step = 0.0001
					program.sliders.network.SetValue(0.15)
					program.sliders.network.Refresh()
				})
				return
			} // mined genesis
			single_block() // let's advance the blocks
			single_block() // registrations get loaded into the pool
			single_block() // need them to all get processed
			single_block() // and this is a great place to start

			// we have a different connective function
			// go walletapi.Keep_Connectivity()
			// we already have an in-wallet explorer

			// let's automatically mine blocks
			go func() {
				last := time.Now()
				for {
					if program.node.simulator_chain == nil {
						return
					}
					bl, _, _, _, err := program.node.simulator_chain.Create_new_block_template_mining(genesis_wallet.GetAddress())
					if err != nil {
						continue
					}
					// we aren't going to be using the noautomine feature right now
					blocktime := time.Duration((config.BLOCK_TIME * uint64(time.Second)))
					if time.Since(last) > blocktime || len(bl.Tx_hashes) >= 1 {
						single_block()    // we are just panicing...
						last = time.Now() // like last time? XD
					}
					time.Sleep(time.Second)
				}
			}()

			// we aren't logging so... not sure why we would start a cron...
			// let's see if it works?.. lol
		}

		if strings.Contains(program.buttons.simulation.Text, "OFF") {
			go func() {
				fyne.DoAndWait(func() {
					program.buttons.simulation.SetText("RESTART WALLET TO LAUNCH AGAIN")
					program.buttons.simulation.Disable()
					program.sliders.network.Step = 0.0001
					program.sliders.network.SetValue(0.15)
					program.sliders.network.Refresh()
				})
			}()
			metrics.Set.UnregisterAllMetrics()
			if program.simulator_server != nil {
				program.simulator_server.RPCServer_Stop()
				p2p.P2P_Shutdown()
				program.node.simulator_chain.Shutdown()
				for _, r := range program.node.simulator_rpcservers {
					go r.RPCServer_Stop()
				}
				program.node.simulator_chain = nil
				program.simulator_server = nil
			}
			program.toggles.rpc_server.Enable()
			program.entries.username.Enable()
			program.entries.password.Enable()
			if program.preferences.Bool("loggedIn") {
				logout()
			}
		}
	}

	notice := widget.NewLabel(`
The simulator provides a convenient place to simulate the DERO blockchain for testing and evaluation purposes.

You will need to completely shut down the wallet to create a new simulator. This prevents duplicate block histories.
	
The simulator RPC runs on 127.0.0.1:20000 and the wallet will connect automatically. There is a mining getwork server running on 127.0.0.1:10100.
	
There are 21 registered, passwordless simulator wallets found in folder: ./testnet_simulator/ 
	
These wallets are started with RPC servers ON without username or password. Endpoints can be found starting on 127.0.0.1:30000 and up, eg 30000 is wallet 0, 30001 is wallet 1, etc`)
	notice.Wrapping = fyne.TextWrapWord
	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		notice,
		container.NewCenter(program.buttons.simulation),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	simulate := dialog.NewCustom("simulator server", dismiss, content, program.window)
	simulate.Resize(program.size)
	simulate.Show()
}

func balance_rescan() {
	// nice big notice
	big_notice :=
		"This action is a non-cancelable action that will clear all transfer history and balances.\n" +
			"Balances are nearly instant in resync; however...\n" +
			"Tx history depends on node status, eg pruned/full...\n" +
			"Some txs may not be available at your node connection.\n" +
			"For full history, and privacy, run a full node starting at block 0 upto current topoheight.\n" +
			"This operation could take a long time with many token assets and transfers."

	// create a callback function
	callback := func(b bool) {
		// if they cancel
		if !b {
			return
		}

		// start a sync activity widget
		// syncing := widget.NewActivity()
		// syncing.Start()
		notice := makeCenteredWrappedLabel("Beginning Scan")
		prog := widget.NewProgressBar()
		content := container.NewVBox(
			layout.NewSpacer(),
			prog,
			// syncing,
			notice,
			layout.NewSpacer(),
		)
		// set it to a splash screen
		syncro := dialog.NewCustomWithoutButtons("syncing", content, program.window)

		// resize and show
		syncro.Resize(program.size)
		syncro.Show()

		// rebuild the hash list
		buildAssetHashList()

		prog.Min = 0
		prog.Max = 1

		// as a go routine
		go func() {

			// clean the wallet
			program.wallet.Clean()

			// it will be helpful to know when we are done
			var done bool

			// as a go routine...
			go func() {

				// keep track of the start
				var start int

				// now we are going to have this spin every second
				ticker := time.NewTicker(time.Millisecond * 300)
				for range ticker.C {
					h := getDaemonInfo().Height

					// if we are done, break this loop
					if done {
						break
					}

					// get transfers
					transfers := getTransfersByHeight(
						uint64(start), uint64(h),
						crypto.ZEROHASH,
						true, true, true,
					)

					// measure them
					current_len := len(transfers)

					if current_len == 0 {
						continue
					}

					// set the start higher up the chain
					end_of_index := current_len - 1
					start = int(transfers[end_of_index].Height)
					ratio := float64(start) / float64(h)
					fyne.DoAndWait(func() {
						prog.SetValue(ratio)
					})

					// now spin through the transfers at the point of difference
					for _, each := range transfers {

						// update the notice
						fyne.DoAndWait(func() {
							notice.SetText("Blockheight: " + strconv.Itoa(int(each.Height)) + " Timestamp: " + each.Time.String())
						})

						// take a small breather between updates
						time.Sleep(time.Millisecond)
					}

				}
			}()
			// set notice to a default
			fyne.DoAndWait(func() {
				notice.SetText("Syncing tokens")
			})
			// then sync the wallet for DERO
			if err := program.wallet.Sync_Wallet_Memory_With_Daemon(); err != nil {
				// if there is an error, notify the user
				showError(err, program.window)
				syncro.Dismiss()
				return
			} else {
				// now range through each token in the cache one at a time
				desired := 1
				capacity_channel := make(chan struct{}, desired)
				var wg sync.WaitGroup
				wg.Add(len(program.caches.assets))
				for _, asset := range program.caches.assets {
					go func() {
						new_job := struct{}{}

						capacity_channel <- new_job
						defer wg.Done()
						// assume there could be an error
						var err error

						// then add each scid back to the map
						hash := crypto.HashHexToHash(asset.hash)
						if err = program.wallet.TokenAdd(hash); err != nil {
							// if err, show it
							showError(err, program.window)
							// but don't stop, just continue the loop
							return
						}
						// set notice to a default
						fyne.DoAndWait(func() {
							notice.SetText("Syncing " + asset.hash)
						})
						// and then sync scid internally with the daemon
						if err = program.wallet.Sync_Wallet_Memory_With_Daemon_internal(hash); err != nil {
							// if err, show it
							showError(err, program.window)
							syncro.Dismiss()
							// but don't stop, just continue the loop
							return
						}

						<-capacity_channel
					}()
				}
				wg.Wait()
			}
			// when done, shut down the sync status in the go routine
			fyne.DoAndWait(func() {
				done = true
				// syncing.Stop()
				syncro.Dismiss()
			})
		}()

	}

	// here is the rescan dialog
	rescan := dialog.NewConfirm("Balance Rescan", big_notice, callback, program.window) // set to the main window

	// resize and show
	rescan.Resize(program.size)
	rescan.Show()
}

func notifications() {
	notice := `
There is a entry notification system for monitoring inbound/outbound transfers.

By default, this system is off; this is to preserve the sanity of the devs. 

You are welcome to turn it on and off as you would like. 
	`
	program.sliders.notifications.Step = 0.0001
	on, off := widget.NewLabel("ON"), widget.NewLabel("OFF")
	if program.sliders.notifications.Value < 0.50 {
		go func() {
			fyne.DoAndWait(func() {
				program.sliders.notifications.SetValue(0.235)
				program.sliders.notifications.Refresh()
				on.TextStyle.Bold = false
				off.TextStyle.Bold = true
				on.Refresh()
				off.Refresh()
			})
		}()
	}
	program.sliders.notifications.OnChanged = func(f float64) {
		switch {
		case f < 0.50:
			program.preferences.SetBool("notifications", false)
		case f > 0.50:
			program.preferences.SetBool("notifications", true)
		}
	}
	program.sliders.notifications.OnChangeEnded = func(f float64) {
		switch {
		case f < 0.50:
			program.sliders.notifications.SetValue(0.235)
			go func() {
				fyne.DoAndWait(func() {
					on.TextStyle.Bold = false
					off.TextStyle.Bold = true
					on.Refresh()
					off.Refresh()
				})
			}()
		case f > 0.50:
			program.sliders.notifications.SetValue(0.765)

			go func() {
				fyne.DoAndWait(func() {
					on.TextStyle.Bold = true
					off.TextStyle.Bold = false
					on.Refresh()
					off.Refresh()
				})
			}()
		}
	}
	program.sliders.notifications.Orientation = widget.Horizontal
	content := container.NewAdaptiveGrid(1,
		widget.NewLabel(notice),
		container.NewVBox(
			program.sliders.notifications,
			container.NewAdaptiveGrid(2,
				container.NewCenter(off),
				container.NewCenter(on),
			),
		),
	)

	d := dialog.NewCustom("Notifications", dismiss, content, program.window)

	d.Resize(password_size)
	d.Show()
}
