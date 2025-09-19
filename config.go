package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/blockchain"
	derodrpc "github.com/deroproject/derohe/cmd/derod/rpc"
	"github.com/deroproject/derohe/config"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/p2p"
	"github.com/deroproject/derohe/transaction"
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

	program.buttons.simulator.OnTapped = simulator

	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewVBox(program.buttons.simulator),
		container.NewVBox(program.buttons.connections),
		container.NewVBox(program.buttons.rpc_server),
		container.NewVBox(program.buttons.update_password),
		// container.NewVBox(program.buttons.tx_priority),
		// container.NewVBox(program.buttons.ringsize),
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

	// so before we get started, let's assume that localhost is "first"
	program.node.current = program.node.list[1].ip

	// now if you have stated a preference...
	if program.node.list[0].ip != "" {
		program.node.current = program.node.list[0].ip
	}

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
			fmt.Println("connecting to", program.node.current)
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

	var msg string = "Auto-connects to "
	// post up the current node
	current_node := makeCenteredWrappedLabel("")
	current_node.SetText("Current Node: " + program.node.current)

	// make a way for them to enter and address
	form_entry := widget.NewEntry()

	// build a pleasing and simple list
	label := widget.NewRichTextFromMarkdown("") // wrap it
	label.Wrapping = fyne.TextWrapWord
	table := widget.NewTable(
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
		table.SetColumnWidth(i, 200)
	}
	table.HideSeparators = true
	table.OnSelected = func(id widget.TableCellID) {
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
			table.UnselectAll()
			table.Refresh()
			showInfoFast("Copied", data, program.window)
		}
	}

	set_label := func() {
		var opts string
		// let's show off a list
		switch program.toggles.network.Selected {
		case "mainnet":
			opts = "preferred if set, then localhost, then the fastest public node:\n\n"
		case "testnet":
			opts = program.node.current
		case "simulator":
			opts = program.node.current
		}

		label.ParseMarkdown(msg + opts)
	}

	set_label()

	changed := func(s string) {
		switch s {
		case "mainnet":
			program.toggles.network.SetSelected("mainnet")
			table.Show()
			globals.Arguments["--testnet"] = false
			globals.Arguments["--simulator"] = false
			program.preferences.SetBool("mainnet", true)
			program.node.current = "127.0.0.1:10102"
			form_entry.PlaceHolder = "127.0.0.1:10102"
			current_node.SetText("Current Node: " + program.node.current)
			form_entry.Refresh()
			globals.InitNetwork()
			set_label()
		case "testnet":
			program.toggles.network.SetSelected("testnet")
			table.Hide()
			globals.Arguments["--testnet"] = true
			globals.Arguments["--simulator"] = false
			program.preferences.SetBool("mainnet", false)
			program.node.current = "127.0.0.1:40402"
			form_entry.PlaceHolder = program.node.current
			current_node.SetText("Current Node: " + program.node.current)
			label.ParseMarkdown(msg)
			form_entry.Refresh()
			globals.InitNetwork()
			set_label()
		case "simulator":
			program.toggles.network.SetSelected("simulator")
			table.Hide()
			globals.Arguments["--testnet"] = true
			globals.Arguments["--simulator"] = true
			program.preferences.SetBool("mainnet", false)
			program.node.current = "127.0.0.1:20000"
			form_entry.PlaceHolder = program.node.current
			current_node.SetText("Current Node: " + program.node.current)
			form_entry.Refresh()
			label.ParseMarkdown(msg)
			globals.InitNetwork()
			set_label()
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

	save := widget.NewHyperlink("save connection", nil)

	save.Alignment = fyne.TextAlignCenter

	save.OnTapped = func() {
		// obviously...
		if form_entry.Text == "" {
			showError(errors.New("cannot be empty"))
			return
		}
		endpoint := form_entry.Text
		// form_entry.SetText("")

		// test the connection point
		if err := testConnection(endpoint); err != nil {
			showError(err)
			return
		}
		program.node.list[0] = struct {
			ip   string
			name string
		}{ip: endpoint, name: "preferred"}

		file, err := os.Create("preferred")
		if err != nil {
			showError(err)
			return
		}
		if _, err := io.WriteString(file, endpoint); err != nil {
			showError(err)
			return
		}
		path, _ := filepath.Abs(file.Name())

		showInfo("Saved Preferred", "node endpoint saved to "+path)
		table.Refresh()
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

	desiredSize := fyne.NewSize(450, 200)

	fixedSizeTable := container.NewGridWrap(desiredSize, table)
	centered := container.NewCenter(fixedSizeTable)

	// walk the user through the process
	content := container.NewBorder(
		container.NewVBox(
			container.NewCenter(program.toggles.network),
			form_entry,
			container.NewAdaptiveGrid(2, save, set),
			current_node,
			label,
		),
		nil,
		nil,
		nil,
		centered,
	)

	connect := dialog.NewCustom("Custom Node Connection", dismiss, content, program.window)

	// resize and show
	size := fyne.NewSize(program.size.Width/3*2, program.size.Height/3*2)
	connect.Resize(size)
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
				showError(err)
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

func simulator() {
	if program.toggles.simulator.Selected == "" {
		program.toggles.simulator.SetSelected("off")
	}
	program.toggles.simulator.Horizontal = true
	program.toggles.simulator.Options = []string{
		"off",
		"on",
	}
	program.toggles.simulator.Required = true
	program.toggles.simulator.OnChanged = func(s string) {
		if s == "" {
			program.toggles.simulator.SetSelected("off")
		}
		switch s {
		case "on":
			program.toggles.simulator.SetSelected("on")
			program.preferences.SetBool("mainnet", false)
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
				wallet.SetNetwork(program.preferences.Bool("mainnet"))

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
				"--rpc-bind":  daemon_endpoint,
				"--testnet":   !program.preferences.Bool("mainnet"), // F*** IT, WE'LL DO IT LIVE!
				"--simulator": true,                                 // obviously
				"--p2p-bind":  ":0",
			}

			// now, we'll init the network
			globals.InitNetwork()

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
			program.caches.simulator_chain, err = blockchain.Blockchain_Start(simulation) //start chain in simulator mode
			if err != nil {
				panic(err)
			}

			simulation["chain"] = program.caches.simulator_chain

			p2p.P2P_Init(simulation)
			// we should probably consider the "toggle" very seriously
			simulator_server, err := derodrpc.RPCServer_Start(simulation)
			if err != nil {
				panic(err)
			}
			fmt.Println(simulator_server)
			// and let's simulate a bunch of users
			program.caches.simulator_wallets = []*walletapi.Wallet_Disk{}
			program.caches.simulator_rpcservers = []*rpcserver.RPCServer{}
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
			program.toggles.server.Disable()
			program.entries.username.Disable()
			program.entries.password.Disable()
			program.buttons.update_password.Disable()
			var wg sync.WaitGroup
			wg.Add(len(simulation_seeds))
			for i, seed := range simulation_seeds {
				go func() {
					defer wg.Done()
					n := "simulation_wallet_" + strconv.Itoa(i) + ".db"
					wallet := create_wallet(n, seed)
					if err := program.caches.simulator_chain.Add_TX_To_Pool(wallet.GetRegistrationTX()); err != nil {
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
						program.caches.simulator_rpcservers = append(program.caches.simulator_rpcservers, r)
					}
					program.caches.simulator_wallets = append(program.caches.simulator_wallets, wallet)
					time.Sleep(time.Millisecond * 20) // little breathing room
				}()
			}
			wg.Wait()

			// now let's go mine a block
			single_block := func() {
				// every block has a id, or a hash
				var blid crypto.Hash

				for {
					// using the genesis wallet, get a block and miniblock template
					bl, mbl, _, _, err := program.caches.simulator_chain.Create_new_block_template_mining(genesis_wallet.GetAddress())
					if err != nil {
						panic(err)
					}
					// now let's get the timestamp of the block
					ts := bl.Timestamp
					// and let's serialize the miniblock we used
					serial := mbl.Serialize()

					// and let's just accept it as is
					if _, blid, _, err = program.caches.simulator_chain.Accept_new_block(ts, serial); err != nil {
						panic(err)
					} else if !blid.IsZero() {
						// assuming that the hash is not zero, break the loop
						break
					}
				}
			}
			single_block() // mined genesis
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
					bl, _, _, _, err := program.caches.simulator_chain.Create_new_block_template_mining(genesis_wallet.GetAddress())
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
		case "off":
			program.toggles.simulator.SetSelected("off")
			if program.simulator_server != nil {
				program.simulator_server.RPCServer_Stop()
				p2p.P2P_Shutdown()
				program.caches.simulator_chain.Shutdown()
				for _, r := range program.caches.simulator_rpcservers {
					go r.RPCServer_Stop()
				}
			}
			program.toggles.server.Enable()
			program.entries.username.Enable()
			program.entries.password.Enable()
			program.buttons.update_password.Enable()
		default:
		}
	}

	notice := makeCenteredWrappedLabel(`
	The simulator provides a convenient place to simulate the DERO blockchain for testing and evaluation purposes.
	
	The simulator rpc runs on 127.0.0.1:20000 and you will need to change connections to 'simulator' in order to access the simulated network.
	
	21 simulator wallets can be found in the ` + globals.GetDataDirectory() + ` folder and have no password. 
	
	The wallets are started with rpc servers on with no username and password and can be found starting on 127.0.0.1:30000 and up, eg 30000 is wallet 0, 30001 is wallet 1, etc`)
	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		notice,
		container.NewCenter(program.toggles.simulator),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	simulate := dialog.NewCustom("simulator server", dismiss, content, program.window)
	simulate.Resize(program.size)
	simulate.Show()
}
