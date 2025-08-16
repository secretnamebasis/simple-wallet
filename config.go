package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
	"go.etcd.io/bbolt"
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

	// let's hide it for a moment
	program.hyperlinks.rpc_server.Hide()

	// let's make a simple way for people to manage a pong server
	program.hyperlinks.pong_server.OnTapped = pong_server

	// and let's hide it for a moment
	program.hyperlinks.pong_server.Hide()

	return container.New(layout.NewVBoxLayout(),
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewCenter(program.hyperlinks.connections),
		container.NewCenter(program.hyperlinks.rpc_server),
		container.NewCenter(program.hyperlinks.pong_server),
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
		if !walletapi.Connected {

			// update the label and show 0s
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("🌐: 🟡")
				program.labels.height.SetText("⬡: 0000000")
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

		} else { // now that the wallet is connected

			// make sure this is green
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("🌐: 🟢")
			})
		}

		// then test the connection
		if err := walletapi.Connect(walletapi.Daemon_Endpoint); err != nil {

			//  if the number of tries is 1...
			if retries == 1 {
				fyne.DoAndWait(func() {

					// then notify the user
					dialog.ShowError(
						errors.New("auto connect is not working as expected, please set custom endpoint"),
						program.window,
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

			// retries is reset
			retries = 0

			// update the height and node label
			fyne.DoAndWait(func() {
				program.labels.connection.SetText("🌐: 🟢")

				height := walletapi.Get_Daemon_Height()
				if height > old_height {
					old_height = height

					// program.labels.height.TextStyle.Bold = true
					// program.labels.height.Refresh()
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
			dialog.ShowError(errors.New("cannot be empty"), program.window)
			return
		}

		// test the connection point
		if err := testConnection(form_entry.Text); err != nil {
			dialog.ShowError(err, program.window)
			return
		}

		// attempt to connect
		if err := walletapi.Connect(form_entry.Text); err != nil {
			dialog.ShowError(err, program.window)
			return
		} else {
			// tell the user how cool they are
			dialog.ShowInformation("Connection", "success", program.window)
		}
		// now the current node is the entry
		program.node.current = form_entry.Text

		// dump the entry
		form_entry.SetText("")

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
	connect := dialog.NewCustom("Custom Node Connection", "dismiss",
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
		"off",
		"on",
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
				dialog.ShowError(err, program.window)
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
	notice := widget.NewLabel(`
The RPC Server allows for external apps to connect with the wallet. Be conscientious with credentials, and only have ON when necessary.

RPC server runs at http://127.0.0.1:10103 
	`) // center it and wrap it
	notice.Alignment = fyne.TextAlignCenter
	notice.Wrapping = fyne.TextWrapWord

	// load up the widgets into a container
	content := container.NewVBox(
		layout.NewSpacer(),
		notice,
		program.entries.username, program.entries.password,
		container.NewCenter(program.toggles.server),
		layout.NewSpacer(),
	)

	// let's build a walkthru for the user, resize and show
	rpc := dialog.NewCustom("rpc server", "dismiss", content, program.window)
	rpc.Resize(program.size)
	rpc.Show()
}

func pong_server() {

	// let's start at the top
	add := widget.NewHyperlink("add new address", nil)
	add.Alignment = fyne.TextAlignCenter

	add.OnTapped = func() {

		// check if the pong server is true before adding anything to the db
		if !program.preferences.Bool("pong_server") {
			dialog.ShowError(errors.New("pong server is off"), program.window)
			return
		}

		callback := func(b bool) {
			// get the pass
			pass := program.entries.pass.Text

			// dump the pass
			program.entries.pass.SetText("")
			// in case they cancel
			if !b {
				return
			}

			// check the password
			if !program.wallet.Check_Password(pass) {
				dialog.ShowError(errors.New("wrong password"), program.window)
				return
			}

			// make a notice
			notice := widget.NewLabel("Add a new address to watch")
			notice.Alignment = fyne.TextAlignCenter
			notice.Wrapping = fyne.TextWrapWord

			// capture the name
			name := widget.NewEntry()
			name.SetPlaceHolder("Item Name")

			// ... description
			description := widget.NewEntry()
			description.SetPlaceHolder("Item Description")

			// and the address to be watched
			entry := widget.NewEntry()
			entry.SetPlaceHolder("address to be watched")

			// be sure to validate this entry
			entry.Validator = func(s string) error {
				if s == "" {
					return nil
				}

				// check whether the entry has been processed before, if yes skip it
				if err := program.db.View(func(tx *bbolt.Tx) error {
					if b := tx.Bucket([]byte("PING")); b != nil {
						if ok := b.Get([]byte(s)); ok != nil { // if existing in bucket
							// return error if in db
							return errors.New("already in db")
						}
					}
					return nil
				}); err != nil {
					return err
				}

				// if it isn't an address, don't use it
				addr, err := rpc.NewAddress(s)
				if err != nil {
					return err
				}

				// if it isn't integrated, don't use it
				if !addr.IsIntegratedAddress() {
					return errors.New("not an integrated address")
				}

				// determine if it has a replyback address flag
				hasReplyback := addr.Arguments.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataUint64)

				// if not...
				if !hasReplyback {
					return errors.New("does not have replyback request")
				}

				// determine if it has a destination port;
				// this port will serve as the unique identifier
				hasDST := addr.Arguments.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64)

				// if not...
				if !hasDST {
					return errors.New("does not have dst port")
				}

				// obtain the value of the destination port
				dstPort := addr.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64)

				// let's get a fresh list of addresses
				var addresses []string

				// and a fresh list of items we have in the db
				var items []listing

				// let get an updated list of ping addresses
				program.db.View(func(tx *bbolt.Tx) error {

					// go look in the ping bucket
					if b := tx.Bucket([]byte("PING")); b != nil {

						// make a cursor
						c := b.Cursor()

						// iterate of all the keys
						for k, v := c.First(); k != nil; k, v = c.Next() {

							// obviously this is slow at scale, but as this is a service...
							// we are are gonna have to check to see if things have changed

							// get all the addresses
							addresses = append(addresses, string(k))

							// and get all the items
							var i listing
							if err := json.Unmarshal(v, &i); err != nil {
								continue
							}
							items = append(items, i)
						}
					}
					return nil
				})

				// let's remind ourselves of all the ports we are watching
				var watchingPorts []uint64
				// now spin trough the addresses
				for _, address := range addresses {

					// build them out into address
					addr, err := rpc.NewAddress(address)

					// if there is an error show it
					if err != nil {
						dialog.ShowError(err, program.window)
						continue
					}

					// we are only going to use dst port addresses, for simplcity
					hasDST := addr.Arguments.Has(
						rpc.RPC_DESTINATION_PORT,
						rpc.DataUint64,
					)
					if !hasDST {
						continue
					}
					dst_port := addr.Arguments.Value(
						rpc.RPC_DESTINATION_PORT,
						rpc.DataUint64,
					).(uint64)
					watchingPorts = append(watchingPorts, dst_port)
				}
				// check for if the entry has one of the dst ports first
				if slices.Contains(watchingPorts, dstPort) {
					return errors.New("dst port already in use")
				}

				return nil
			}

			// let's make a way for the user to reply back to inbound pings
			replyback := widget.NewEntry()
			replyback.SetPlaceHolder("replyback msg can be uuid/url/secret, limit 100 char")
			replyback.Validator = func(s string) error {
				if s == "" {
					return nil
				}
				if len(s) > 100 {
					return errors.New("too long a reply")
				}
				return nil
			}

			var hashes []string
			for _, each := range program.caches.hashes {
				hashes = append(hashes, truncator(each.String()))
			}
			token := widget.NewSelect(hashes, func(s string) {})
			token.PlaceHolder = "Defaults to DERO, otherwise selected token"
			value := widget.NewEntry()
			value.SetPlaceHolder("how much to send back: eg 0.00002")
			supply := widget.NewEntry()
			supply.SetPlaceHolder("set supply, -1 is unlimited")

			items := []*widget.FormItem{
				widget.NewFormItem("", notice),
				widget.NewFormItem("", name),
				widget.NewFormItem("", description),
				widget.NewFormItem("", entry),
				widget.NewFormItem("", replyback),
				widget.NewFormItem("", token),
				widget.NewFormItem("", value),
				widget.NewFormItem("", supply),
			}
			callback := func(b bool) {
				// if they cancel
				if !b {
					return
				}

				// check whether the entry has been processed before, if yes skip it
				var have_already bool
				if err := program.db.View(func(tx *bbolt.Tx) error {
					if b := tx.Bucket([]byte("PING")); b != nil {
						if ok := b.Get([]byte(entry.Text)); ok != nil { // if existing in bucket
							have_already = true
							return errors.New("already in db")
						}
					}
					return nil
				}); err != nil {
					dialog.ShowError(err, program.window)
					return
				}

				if have_already {
					dialog.ShowError(errors.New("already in db"), program.window)
					return
				}

				var item = listing{}
				item.Address = entry.Text

				// build them out into address
				addr, err := rpc.NewAddress(item.Address)

				// if there is an error show it
				if err != nil {
					dialog.ShowError(err, program.window)
					return
				}

				// we are only going to use dst port addresses, for simplcity
				hasDST := addr.Arguments.Has(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64,
				)
				if !hasDST {
					return
				}
				dst_port := addr.Arguments.Value(
					rpc.RPC_DESTINATION_PORT,
					rpc.DataUint64,
				).(uint64)

				item.DST = dst_port

				if name.Text != "" {
					item.Name = name.Text
				}

				if description.Text != "" {
					item.Description = description.Text
				}

				if value.Text != "" {
					f, err := strconv.ParseFloat(value.Text, 64)
					if err != nil {
						dialog.ShowError(err, program.window)
						return
					}
					val := uint64(f * atomic_units)
					item.Sendback = val
				}
				if replyback.Text != "" {
					item.Replyback = replyback.Text
				}
				if token.SelectedIndex() != -1 {
					hash := program.caches.hashes[token.SelectedIndex()]
					if hash != crypto.ZEROHASH {
						item.Token = hash
					}
				} else {
					item.Token = crypto.ZEROHASH
				}

				if supply.Text != "" {
					in, err := strconv.Atoi(supply.Text)
					if err != nil {
						dialog.ShowError(err, program.window)
						return
					}
					item.Supply = in
				}

				// marshal the item to bytes
				bytes, err := json.Marshal(item)
				// show the error if any
				if err != nil {
					dialog.ShowError(err, program.window)
					return
				}

				// let's update the db
				program.db.Update(
					func(tx *bbolt.Tx) error {
						if b := tx.Bucket([]byte("PING")); b != nil {
							b.Put([]byte(item.Address), bytes)
						}
						return nil
					},
				)

				// make a message
				msg := truncator(item.Address) + " has been added to db"

				// show success
				dialog.ShowInformation("Pong Server", msg, program.window)
			}

			// let's walk them through it
			create := dialog.NewForm("Create a Service Address",
				"confirm", "dismiss", items, callback, program.window)
			create.Resize(program.size)
			create.Show()
		}
		dialog.ShowForm("Pong Server", "confirm", "dismiss", []*widget.FormItem{
			widget.NewFormItem("", program.entries.pass),
		}, callback, program.window)

	}

	if program.toggles.pong.Selected == "" {
		// now we have to turn on the pong server...
		program.toggles.pong.SetSelected("off")
	}

	// sideways is more fun
	program.toggles.pong.Horizontal = true

	// pretty simple, on or off
	program.toggles.pong.Options = []string{
		"off", "on",
	}

	// and here's what we'll do when they turn it off or on
	program.toggles.pong.OnChanged = func(s string) {
		if s == "off" {
			program.labels.pong.SetText("🏓: 🔴")
			program.preferences.SetBool("pong_server", false)
			program.toggles.pong.SetSelected("off")
		} else if s == "on" {
			program.labels.pong.SetText("🏓: 🟢")
			program.preferences.SetBool("pong_server", true)
			program.toggles.pong.SetSelected("on")
			// we are going to assume the error here

			go processTXQueues()

			var addresses []string

			// for as long as true, loop
			go func() {

				for program.preferences.Bool("pong_server") {
					// fmt.Println("starting over")
					// we'll take a break
					if !program.preferences.Bool("loggedIn") {
						fyne.DoAndWait(func() {
							program.toggles.pong.SetSelected("off")
						})
						return
					}

					// paranoooooia
					if !program.preferences.Bool("pong_server") {
						fyne.DoAndWait(func() {
							program.toggles.pong.SetSelected("off")
						})
						return
					}
					// let get an update list of ping addresses
					var items = []listing{}

					program.db.View(func(tx *bbolt.Tx) error {

						// go look in the ping bucket
						if b := tx.Bucket([]byte("PING")); b != nil {

							// make a cursor
							c := b.Cursor()

							// iterate of all the keys
							for k, v := c.First(); k != nil; k, v = c.Next() {

								// obviously this is slow at scale, but as this is a service...
								// we are are gonna have to check to see if things have changed

								// get all the addresses
								addresses = append(addresses, string(k))

								// and get all the items
								var i listing
								if err := json.Unmarshal(v, &i); err != nil {
									continue
								}
								items = append(items, i)
							}
						}
						return nil
					})

					watchingPorts := []uint64{}
					for _, i := range items {
						watchingPorts = append(watchingPorts, i.DST)
					}
					// range throught the all entries
					entries := allTransfers()
					for _, entry := range entries {

						// the cbor parser has been update
						// have to use older version or get a payload error
						entry.ProcessPayload()

						if entry.Coinbase || !entry.Incoming { // skip coinbase or outgoing, self generated transactions
							continue
						}

						// check whether the entry has been processed before, if yes skip it
						var already_processed bool
						program.db.View(func(tx *bbolt.Tx) error {
							if b := tx.Bucket([]byte("PONG")); b != nil {
								if ok := b.Get([]byte(entry.TXID)); ok != nil { // if existing in bucket
									already_processed = true
								}
							}
							return nil
						})

						if already_processed { // if already processed skip it
							continue
						}

						// check for if the entry has one of the dst ports first
						// check whether this service should handle the transfer
						if !entry.Payload_RPC.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) { // this service is expecting value to be specfic
							continue

						}
						dst := entry.Payload_RPC.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64)
						if !slices.Contains(watchingPorts, dst) {
							continue
						}

						// see if it has a replyback
						hasReplyback := entry.Payload_RPC.Has(
							rpc.RPC_REPLYBACK_ADDRESS,
							rpc.DataAddress,
						)

						// if it doesn't...
						if !hasReplyback { //|| entry.Sender == ""
							continue
						}

						// get the replyback address
						replyback_address := entry.Payload_RPC.Value(
							rpc.RPC_REPLYBACK_ADDRESS,
							rpc.DataAddress,
						).(rpc.Address)

						// if it isn't registered
						if !isRegistered(replyback_address.String()) {
							continue
						}
						var transfer rpc.Transfer
						transfer.Destination = replyback_address.String()

						// let's get the item they paid for
						var item listing
						for _, i := range items {

							// get the i.Address, as a new addr
							addr, err := rpc.NewAddress(i.Address)

							// if there is an err, just skip
							if err != nil {
								continue
							}

							// we are going to match the item's dst port against the entry dst port
							dst := addr.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64).(uint64)
							if dst != entry.DestinationPort {
								continue
							}

							// seeing as the match
							item = i
							break
						}

						// now we are going to load up some args
						var args rpc.Arguments
						if item.Supply != -1 {
							if item.Supply == 0 {
								// check if it is in the queue already
								hasAlready := false
								for _, this := range refunds {
									if entry.TXID == this.Entry.TXID {
										hasAlready = true
									}
								}

								// continue if the queue already has the item
								if hasAlready {
									continue
								}

								// inform the user
								msg := "exhausted supply for " + item.Name + ", sending refund"
								dialog.ShowInformation("Pong Server", msg, program.window)

								// build out the args
								transfer.Payload_RPC = rpc.Arguments{
									rpc.Argument{
										Name:     rpc.RPC_COMMENT,
										DataType: rpc.DataString,
										Value:    "refunded",
									},
									rpc.Argument{
										Name:     rpc.RPC_SOURCE_PORT,
										DataType: rpc.DataUint64,
										Value:    item.DST, // remind them what item
									},
								}

								// make sure to send back the suff they sent
								if entry.Amount != 0 {
									transfer.Amount = entry.Amount
								} else if entry.Payload_RPC.Has(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) {
									transfer.Amount = entry.Payload_RPC.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64).(uint64)
								} else {
									dialog.ShowError(errors.New("unkown amount, please debug"), program.window)
								}
								if entry.Payload_RPC.Has(rpc.RPC_ASSET, rpc.DataHash) {
									transfer.SCID = entry.Payload_RPC.Value(rpc.RPC_ASSET, rpc.DataHash).(crypto.Hash)
								}

								// set up the payload
								payload := []rpc.Transfer{
									transfer,
								}

								// transfer parameters
								ringsize := uint64(16)
								transfer_all := false
								scdata := rpc.Arguments{}
								gasstorage := uint64(0)
								dry_run := false

								// build a transfer
								tx, err := program.wallet.TransferPayload0(
									payload, ringsize, transfer_all,
									scdata, gasstorage, dry_run,
								)

								// if err, where ever the user is, notify them
								if err != nil {
									dialog.ShowError(err, program.window)
									continue
								}

								// append this refund to the queue
								refunds = append(refunds, queue{
									Item:  item,
									Entry: entry,
									Tx:    tx,
								})
								continue
							}

							// check for item details

							//add this arg if we are replying back
							if item.Replyback != "" {
								args = append(args, rpc.Argument{
									Name:     rpc.RPC_COMMENT,
									DataType: rpc.DataString,
									Value:    item.Replyback,
								})
							}

							// include this token if it isn't DERO
							if item.Token != crypto.ZEROHASH {
								transfer.SCID = item.Token
							}

							// specify how much we send back
							if item.Sendback != 0 {
								transfer.Amount = item.Sendback
							}

							// and load up the args
							transfer.Payload_RPC = args

							// build out the payload
							payload := []rpc.Transfer{
								transfer,
							}

							// set the transfer params
							ringsize := uint64(16)
							transfer_all := false
							scdata := rpc.Arguments{}
							gasstorage := uint64(0)
							dry_run := false

							// build a transfer
							tx, err := program.wallet.TransferPayload0(
								payload, ringsize, transfer_all,
								scdata, gasstorage, dry_run,
							)

							// where ever the user is, notify them
							if err != nil {
								dialog.ShowError(err, program.window)
							}

							// if it isn't in queue, add it
							pongs = append(pongs, queue{
								Item:  item,
								Entry: entry,
								Tx:    tx,
							})
						}
					}
				}
			}()

		} else { // if we aren't on or off, just be off
			program.toggles.pong.SetSelected("off")
		}
	}

	// let's make a simple way to review the addresses on file
	review := widget.NewHyperlink("review list", nil)

	// center it
	review.Alignment = fyne.TextAlignCenter

	// here is what we'll do when it is tapped
	review.OnTapped = func() {
		if !program.preferences.Bool("pong_server") {
			dialog.ShowError(errors.New("pong server is off"), program.window)
			return
		}
		var items []listing // we are going to need these

		// let get an updated list of ping addresses
		buildItemsList := func() {
			// empty the list
			items = []listing{}

			// peer into the db
			program.db.View(func(tx *bbolt.Tx) error {

				// go look in the ping bucket
				if b := tx.Bucket([]byte("PING")); b != nil {

					// make a cursor
					c := b.Cursor()

					// iterate of all the keys
					for k, v := c.First(); k != nil; k, v = c.Next() {

						// obviously this is slow at scale, but as this is a service...
						// we are are gonna have to check to see if things have changed

						// and get all the items
						var i listing
						if err := json.Unmarshal(v, &i); err != nil {
							continue
						}
						items = append(items, i)
					}
				}
				return nil
			})
		}

		buildItemsList()

		// let's make a new list
		list := new(widget.List)

		// let's use the length of the addresses slice
		list.Length = func() int { return len(items) }

		// return a container with the addr label, and a link 'x' to delete
		list.CreateItem = func() fyne.CanvasObject {

			return widget.NewLabel("")
		}

		// go through the container and set the text with a x to delete
		list.UpdateItem = func(id widget.ListItemID, item fyne.CanvasObject) {

			label := items[id].Name

			// go into the container and set the text of the the first object,
			// which "should" be a label
			item.(*widget.Label).SetText(label)
		}

		// when they select a pong address
		list.OnSelected = func(id widget.ListItemID) {
			list.Unselect(id)
			stats := new(dialog.CustomDialog)
			// let's make a title
			title := items[id].Name + " stats"

			// make a text block entry
			address := widget.NewEntry()
			address.Disable() // lock it down so they can't mess it up

			// now they can see the thing
			address.SetText(items[id].Address)

			// make a way to review the payload
			payload := widget.NewEntry()
			payload.MultiLine = true
			buildPayloadEntry := func() {
				p := "NAME: " + items[id].Name + "\n"
				p += "DESCRIPTION: " + items[id].Description + "\n"
				p += "ADDRESS: " + items[id].Address + "\n"
				p += "DST " + strconv.Itoa(int(items[id].DST)) + "\n"
				p += "REPLYBACK " + items[id].Replyback + "\n"
				p += "TOKEN " + items[id].Token.String() + "\n"
				p += "SUPPLY " + strconv.Itoa(items[id].Supply) + "\n"
				p += "SENDBACK " + rpc.FormatMoney(items[id].Sendback) + "\n"
				payload.SetText(p)
			}
			buildPayloadEntry()

			// make sure it is locked down
			payload.Disable()
			// empty the list
			pongs := []pong{}
			// let get an updated list of ping addresses
			buildTXIDList := func() {
				// empty the list
				pongs = []pong{}

				// peer into the db
				program.db.View(func(tx *bbolt.Tx) error {

					// go look in the ping bucket
					if b := tx.Bucket([]byte("PONG")); b != nil {

						// make a cursor
						c := b.Cursor()

						// iterate of all the keys
						for k, v := c.First(); k != nil; k, v = c.Next() {
							// obviously this is slow at scale, but as this is a service...
							// we are are gonna have to check to see if things have changed
							if !strings.Contains(string(v), items[id].Address) {
								continue
							}

							// let's get the pong
							var p pong

							// and let's unmarshal the bytes to the pong struct
							if err := json.Unmarshal(v, &p); err != nil {
								return err
							}

							// and then add it with the rest
							pongs = append(pongs, p)
						}
					}
					return nil
				})
			}

			buildTXIDList()
			// make a way to review the payload
			program.entries.pongs.SetText("")
			program.entries.pongs.MultiLine = true
			program.entries.pongs.Disable()
			sort.Slice(pongs, func(i, j int) bool {
				return pongs[i].Time.Before(pongs[j].Time)
			})
			buildTXIDSEntry := func() {
				t := ""
				for i, each := range pongs {
					t += each.Time.Local().Format("2006-01-02 15:04") + " " + each.Status + " " + each.Txid
					if i+1 != len(pongs) {
						t += "\n"
					}
				}
				program.entries.pongs.SetText(t)
			}
			buildTXIDSEntry()
			// easy way to just edit
			edit := widget.NewHyperlink("edit", nil)
			edit.Alignment = fyne.TextAlignCenter
			edit.OnTapped = func() {
				if !program.preferences.Bool("pong_server") {
					dialog.ShowError(errors.New("pong server is off"), program.window)
					return
				}
				notice := widget.NewLabel("Edit address on watch")
				notice.Alignment = fyne.TextAlignCenter
				notice.Wrapping = fyne.TextWrapWord
				name := widget.NewEntry()
				name.SetPlaceHolder("Item Name")
				description := widget.NewEntry()
				description.SetPlaceHolder("Item Description")
				replyback := widget.NewEntry()
				replyback.SetPlaceHolder("replyback msg can be uuid/url/secret, limit 100 char")
				replyback.Validator = func(s string) error {
					return nil
				}

				var hashes []string
				for _, each := range program.caches.hashes {
					hashes = append(hashes, truncator(each.String()))
				}
				token := widget.NewSelect(hashes, func(s string) {})
				token.PlaceHolder = "Defaults to DERO, otherwise selected token"
				value := widget.NewEntry()
				value.SetPlaceHolder("how much to send back: eg 0.00002")
				supply := widget.NewEntry()
				supply.SetPlaceHolder("set supply, -1 is unlimited")
				content := container.NewVBox(
					notice,
					name,
					description,
					replyback,
					token,
					value,
					supply,
				)
				callback := func(b bool) {
					// if they cancel
					if !b {
						return
					}

					var item = listing{}
					// these items cannot change, ever
					item.Address = items[id].Address
					item.DST = items[id].DST

					// these items can change
					if name.Text != "" {
						item.Name = name.Text
					} else {
						item.Name = items[id].Name
					}

					// check the description
					if description.Text != "" {
						item.Description = description.Text
					} else {
						item.Description = items[id].Description
					}

					// check the value
					if value.Text != "" {
						f, err := strconv.ParseFloat(value.Text, 64)
						if err != nil {
							dialog.ShowError(err, program.window)
							return
						}
						val := uint64(f * atomic_units)
						item.Sendback = val
					} else {
						item.Sendback = items[id].Sendback
					}

					// check the replyback message
					if replyback.Text != "" {
						item.Replyback = replyback.Text
					} else {
						item.Replyback = items[id].Replyback
					}

					// check the token
					if token.SelectedIndex() != -1 {
						hash := program.caches.hashes[token.SelectedIndex()]
						if hash != crypto.ZEROHASH {
							item.Token = hash
						}
					} else {
						item.Token = items[id].Token
					}

					// check the inventory
					if supply.Text != "" {
						in, err := strconv.Atoi(supply.Text)
						if err != nil {
							dialog.ShowError(err, program.window)
							return
						}
						item.Supply = in
					} else {
						item.Supply = items[id].Supply
					}

					// marshal the item to bytes
					bytes, err := json.Marshal(item)
					// show the error if any
					if err != nil {
						dialog.ShowError(err, program.window)
						return
					}

					// let's update the db
					program.db.Update(
						func(tx *bbolt.Tx) error {
							if b := tx.Bucket([]byte("PING")); b != nil {
								b.Put([]byte(item.Address), bytes)
							}
							return nil
						},
					)
					// reload the list
					buildItemsList()

					// and rebuild the last entry
					buildPayloadEntry()

					// make a message
					msg := truncator(item.Address) + " has been updated in db"

					// show success
					dialog.ShowInformation("Pong Server", msg, program.window)

					list.Refresh()
				}

				// let's walk them through it
				update := dialog.NewCustomConfirm("Update a Service Address",
					"confirm", "dismiss", content, callback, program.window)
				update.Resize(program.size)
				update.Show()
			}

			// easy way to just delete
			delete := widget.NewHyperlink("delete", nil)
			delete.Alignment = fyne.TextAlignCenter
			delete.OnTapped = func() {
				dialog.ShowCustomConfirm("Pong Server", "confirm", "dismiss", program.entries.pass, func(b bool) {
					// if the cancel
					if !b {
						return
					}
					// get the password
					pass := program.entries.pass.Text

					// dump password
					program.entries.pass.SetText("")

					if !program.wallet.Check_Password(pass) {
						dialog.ShowError(errors.New("wrong password"), program.window)
						return
					}

					// peer into the db
					if err := program.db.Update(func(tx *bbolt.Tx) error {
						// go look in the ping bucket
						if b := tx.Bucket([]byte("PING")); b != nil {
							if err := b.Delete([]byte(items[id].Address)); err != nil {
								return err
							}
						}
						return nil
					}); err != nil {
						dialog.ShowError(err, program.window)
						return
					}

					// rebuild the items list
					buildItemsList()

					// get a fresh list
					list.Refresh()

					// close out the screen
					stats.Dismiss()
				}, program.window)

			}

			// set them to a container
			content := container.NewVBox(
				address,
				widget.NewLabel("TXIDs"),
				program.entries.pongs,
				widget.NewLabel("Details"),
				payload,
				container.NewAdaptiveGrid(2,
					edit, delete,
				),
			)

			// load it up, resize and show it
			stats = dialog.NewCustom(title, "dismiss", content, program.window)
			stats.Resize(program.size)
			stats.Show()
		}

		// load it up into the dialog, resize and show
		review := dialog.NewCustom("Service Addresses",
			"dismiss", list, program.window)
		review.Resize(program.size)
		review.Show()
	}

	// let's make a nice message
	msg := "DERO service addresses make it very easy to reply to messages."
	msg += "This can be done automaticailly as a pong server. "
	msg += "When an address has been pinged, the server will pong back a payload"
	msg += "can be urls, tokens, license uuids, etc. "
	// add it to a notice
	notice := widget.NewLabel(msg)
	notice.Wrapping = fyne.TextWrapWord

	content := container.NewVBox(
		notice,
		container.NewCenter(program.toggles.pong), // radio group
		container.NewAdaptiveGrid(2,
			add,    // hyperlink
			review, // hyperlink
		),
	)
	pong := dialog.NewCustom("pong server", "dismiss", content, program.window)
	pong.Resize(program.size)
	pong.Show()
}
