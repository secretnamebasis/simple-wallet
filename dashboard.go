package main

import (
	"bytes"
	"errors"
	"image/jpeg"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
)

func dashboard() *fyne.Container {
	// simple way to find your keys
	program.buttons.keys.OnTapped = keys

	// simple way to find all dero transfers
	program.buttons.transactions.OnTapped = txList

	// simple way to review assets and their transfer histories :)
	program.buttons.assets.OnTapped = assetsList

	// we'll return all this stuff into the home as a dashboard
	return container.NewAdaptiveGrid(3,
		program.buttons.transactions,
		program.buttons.assets,
		program.buttons.keys,
	)
}

func keys() {
	// simple block of text
	program.entries.seed.MultiLine = true
	program.entries.seed.Wrapping = fyne.TextWrapWord

	// make sure these are disabled, like for real
	program.entries.seed.Disable()
	program.entries.secret.Disable()
	program.entries.public.Disable()

	// simple way to see keys
	var k *dialog.FormDialog

	// if they press enter, it is the same as clicking confirm
	program.entries.pass.OnSubmitted = func(s string) {
		k.Dismiss()
		k.Submit()
	}
	callback := func(b bool) {
		if b {
			// check the password for all sensitive actions
			if !program.wallet.Check_Password(program.entries.pass.Text) {
				// if they get is wrong, tell them
				showError(errors.New("wrong password"))
				return
			} else { // if they get it right

				// here is a scroll window
				scrollwindow := container.NewScroll(
					container.NewVBox(
						// here is the seed phrase
						program.labels.seed, program.entries.seed,

						// here is the public key
						program.labels.public, program.entries.public,

						// here is the secret key
						program.labels.secret, program.entries.secret,
					),
				)
				// let's make a dialog window with all the keys included
				keys := dialog.NewCustom("Keys", dismiss, scrollwindow, program.window)

				keys.Resize(program.size)
				keys.Show()
				return
			}
		}
	}

	// create a simple form content
	content := []*widget.FormItem{widget.NewFormItem("", program.entries.pass)}

	// set the content and callback
	k = dialog.NewForm("Display Keys?", confirm, dismiss, content, callback, program.window)

	k.Show()

	// dump password when done
	program.entries.pass.SetText("")
}
func txList() {
	// here are all the sent entries
	s_entries := getSentTransfers()

	sort.Slice(s_entries, func(i, j int) bool {
		return s_entries[i].Height > s_entries[j].Height
	})

	// let's make a list of transactions
	sent := new(widget.List)

	// we'll use the length of s_entries for the count of widget's to return
	sent.Length = func() int { return len(s_entries) }

	// here is the widget that we are going to use for each item of the list
	sent.CreateItem = createLabel
	// then let's update the item to contain the content
	var s_table *widget.Table
	updateSent := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(s_entries) {
			return
		}

		// let's make sure the entry is bodied
		s_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		time_stamp := s_entries[lii].Time.Local().Format("2006-01-02 15:04")
		txid := truncator(s_entries[lii].TXID)
		amount := s_entries[lii].Amount
		container := co.(*fyne.Container)
		container.Objects[0].(*widget.Label).SetText(time_stamp)
		container.Objects[1].(*widget.Label).SetText(txid)
		container.Objects[2].(*widget.Label).SetText(rpc.FormatMoney(amount))

	}
	// set the update item
	sent.UpdateItem = updateSent

	// then when we select one of them, let' open it up!
	onSelected := func(id widget.ListItemID) {
		sent.Unselect(id)

		// body the s_entries
		e := s_entries[id]

		lines := strings.Split(e.String(), "\n")
		keys := []string{}
		values := []string{}
		for _, line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			value := pair[1]
			keys = append(keys, key)
			values = append(values, value)
		}

		s_table = widget.NewTable(
			func() (rows int, cols int) { return len(lines), 2 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(tci widget.TableCellID, co fyne.CanvasObject) {
				label := co.(*widget.Label)
				if len(keys) <= tci.Row {
					label.SetText("")
					return
				}

				switch tci.Col {
				case 0:
					label.SetText(keys[tci.Row])
				case 1:
					label.SetText(values[tci.Row])
				}
			},
		)
		s_table.SetColumnWidth(0, largestMinSize(keys).Width)
		s_table.SetColumnWidth(1, largestMinSize(values).Width)
		s_table.OnSelected = func(id widget.TableCellID) {
			s_table.UnselectAll()
			var data string
			if id.Col == 1 && id.Row < len(values) {
				data = values[id.Row]
				program.application.Clipboard().SetContent(data)
				showInfoFast("", data+" copied to clipboard", program.window)
			}
		}

		// load up dialog with the container
		txs := dialog.NewCustom("Transaction Detail", dismiss, s_table, program.window)

		txs.Resize(program.size)
		txs.Show()
	}
	sent.OnSelected = onSelected
	// here are all the entries
	r_entries := getReceivedTransfers()

	sort.Slice(r_entries, func(i, j int) bool {
		return r_entries[i].Height > r_entries[j].Height
	})

	// let's make a list of received transactions
	received := new(widget.List)

	// we'll use the length of r_entries for the count of widget's to return
	received.Length = func() int { return len(r_entries) }

	// here is the widget that we are going to use for each item of the list
	received.CreateItem = createLabel

	var r_table *widget.Table
	// then let's update the item to contain the content
	updateReceived := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(r_entries) {
			return
		}
		// let's make sure the entry is bodied
		r_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		time_stamp := r_entries[lii].Time.Local().Format("2006-01-02 15:04")
		txid := truncator(r_entries[lii].TXID)
		amount := r_entries[lii].Amount
		container := co.(*fyne.Container)
		container.Objects[0].(*widget.Label).SetText(time_stamp)
		container.Objects[1].(*widget.Label).SetText(txid)
		container.Objects[2].(*widget.Label).SetText(rpc.FormatMoney(amount))

	}

	// set the update item field
	received.UpdateItem = updateReceived

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		received.Unselect(id)

		// body the r_entries
		e := r_entries[id]

		lines := strings.Split(e.String(), "\n")
		keys := []string{}
		values := []string{}
		for _, line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			value := pair[1]
			keys = append(keys, key)
			values = append(values, value)
		}

		r_table = widget.NewTable(
			func() (rows int, cols int) { return len(lines), 2 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(tci widget.TableCellID, co fyne.CanvasObject) {
				label := co.(*widget.Label)
				if len(keys) <= tci.Row {
					label.SetText("")
					return
				}

				switch tci.Col {
				case 0:
					label.SetText(keys[tci.Row])
				case 1:
					label.SetText(values[tci.Row])
				}
			},
		)
		r_table.SetColumnWidth(0, largestMinSize(keys).Width)
		r_table.SetColumnWidth(1, largestMinSize(values).Width)
		r_table.OnSelected = func(id widget.TableCellID) {
			r_table.UnselectAll()
			var data string
			if id.Col == 1 && id.Row < len(values) {
				data = values[id.Row]
				program.application.Clipboard().SetContent(data)
				showInfoFast("", data+" copied to clipboard", program.window)
			}
		}

		// load up dialog with the container
		txs := dialog.NewCustom("Transaction Detail", dismiss, r_table, program.window)

		txs.Resize(program.size)
		txs.Show()
	}

	// set the on selected field
	received.OnSelected = onSelected

	// here are all the coinbase entries
	c_entries := getCoinbaseTransfers()

	sort.Slice(c_entries, func(i, j int) bool {
		return c_entries[i].Height > c_entries[j].Height
	})

	// let's make a list of transactions
	coinbase := new(widget.List)

	// we'll use the length of c_entries for the count of widget's to return
	coinbase.Length = func() int { return len(c_entries) }

	// here is the widget that we are going to use for each item of the list
	coinbase.CreateItem = createLabel

	// then let's update the item to contain the content
	var c_table *widget.Table
	updateCoins := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(c_entries) {
			return
		}
		// let's make sure the entry is bodied
		c_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		time_stamp := c_entries[lii].Time.Local().Format("2006-01-02 15:04")
		txid := truncator(c_entries[lii].BlockHash)
		amount := c_entries[lii].Amount
		container := co.(*fyne.Container)
		container.Objects[0].(*widget.Label).SetText(time_stamp)
		container.Objects[1].(*widget.Label).SetText(txid)
		container.Objects[2].(*widget.Label).SetText(rpc.FormatMoney(amount))

	}
	// set the update item
	coinbase.UpdateItem = updateCoins

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		coinbase.Unselect(id)

		// body the c_entries
		e := c_entries[id]

		lines := strings.Split(e.String(), "\n")
		keys := []string{}
		values := []string{}
		for _, line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			value := pair[1]
			keys = append(keys, key)
			values = append(values, value)
		}

		c_table = widget.NewTable(
			func() (rows int, cols int) { return len(lines), 2 },
			func() fyne.CanvasObject { return widget.NewLabel("") },
			func(tci widget.TableCellID, co fyne.CanvasObject) {
				label := co.(*widget.Label)
				if len(keys) <= tci.Row {
					label.SetText("")
					return
				}

				switch tci.Col {
				case 0:
					label.SetText(keys[tci.Row])
				case 1:
					label.SetText(values[tci.Row])
				}
			},
		)
		c_table.SetColumnWidth(0, largestMinSize(keys).Width)
		c_table.SetColumnWidth(1, largestMinSize(values).Width)

		c_table.OnSelected = func(id widget.TableCellID) {
			c_table.UnselectAll()
			var data string
			if id.Col == 1 && id.Row < len(values) {
				data = values[id.Row]
				program.application.Clipboard().SetContent(data)
				showInfoFast("", data+" copied to clipboard", program.window)
			}
		}

		// load up dialog with the container
		txs := dialog.NewCustom("Transaction Detail", dismiss, c_table, program.window)

		txs.Resize(program.size)
		txs.Show()
	}
	// set the field
	coinbase.OnSelected = onSelected

	search_entry := widget.NewEntry()
	searchBtn := widget.NewButtonWithIcon("Filter", theme.SearchIcon(), func() {})
	search_entry.ActionItem = searchBtn

	var tabs *container.AppTabs
	filterer := func() {
		search := strings.ToLower(search_entry.Text)

		switch tabs.Selected().Text {
		case "Sent":
			s := getSentTransfers()
			sort.Slice(s, func(i, j int) bool {
				return s[i].Height > s[j].Height
			})
			if search == "" {
				s_entries = s
			} else {
				s_entries = []rpc.Entry{}
				for _, each := range s {
					if strings.Contains(strings.ToLower(each.String()), search) {
						s_entries = append(s_entries, each)
					}
				}
			}
			sent.Refresh()
		case "Received":
			r := getReceivedTransfers()
			sort.Slice(r, func(i, j int) bool {
				return r[i].Height > r[j].Height
			})
			if search == "" {
				r_entries = r
			} else {
				r_entries = []rpc.Entry{}
				for _, each := range r {
					if strings.Contains(strings.ToLower(each.String()), search) {
						r_entries = append(r_entries, each)
					}
				}
			}
			received.Refresh()
		case "Coinbase":
			c := getCoinbaseTransfers()
			sort.Slice(c, func(i, j int) bool {
				return c[i].Height > c[j].Height
			})
			if search == "" {
				c_entries = c

			} else {
				c_entries = []rpc.Entry{}
				for _, each := range c {
					if strings.Contains(strings.ToLower(each.String()), search) {
						c_entries = append(c_entries, each)
					}
				}
			}
			coinbase.Refresh()
		}

	}
	search_entry.OnChanged = func(s string) {
		filterer()
	}

	tabs = container.NewAppTabs(
		container.NewTabItem("Sent", container.NewBorder(search_entry, nil, nil, nil, sent)),
		container.NewTabItem("Received", container.NewBorder(search_entry, nil, nil, nil, received)),
		container.NewTabItem("Coinbase", container.NewBorder(search_entry, nil, nil, nil, coinbase)),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	tabs.OnSelected = func(ti *container.TabItem) {
		search_entry.SetText("")
		filterer()
	}
	txs := dialog.NewCustom("transactions", dismiss, tabs, program.window)
	txs.Resize(program.size)
	txs.Show()
}

func assetsList() {

	// let's just refresh the hash cache
	// buildAssetHashList()

	var list *fyne.Container

	if hashesLength() == 0 {
		list = container.NewVBox(
			layout.NewSpacer(),
			makeCenteredWrappedLabel("Please visit tools, and Add SCID to your collection"),
			layout.NewSpacer(),
		)
	} else {
		assets := program.caches.assets

		program.lists.asset_list.HideSeparators = true

		// let's use the length of the hash cache for the number of objects to make
		program.lists.asset_list.Length = func() int { return len(assets) }

		// and here is the widget we'll use for each item in the list
		program.lists.asset_list.CreateItem = createImageLabel

		updateItem := func(lii widget.ListItemID, co fyne.CanvasObject) {
			if lii >= len(assets) {
				co.(*fyne.Container).RemoveAll()
			} else {
				// here is the asset details
				asset := assets[lii]

				// here is the container
				contain := co.(*fyne.Container)
				// here is the container holding the image
				padded := contain.Objects[0].(*fyne.Container)

				// here is the image object
				img := padded.Objects[0].(*canvas.Image)
				img.Resource = theme.BrokenImageIcon()

				// we'll use the asset image when not nil
				if asset.image != nil {
					buf := new(bytes.Buffer)
					h, w := float32(25), float32(25)
					i := setSCIDThumbnail(asset.image, h, w)
					err := jpeg.Encode(buf, i, nil)
					if err != nil {
						showError(err)
					}
					b := buf.Bytes()
					img.Resource = fyne.NewStaticResource("", b)
				}
				img.Refresh()

				text := asset.name
				label := contain.Objects[1].(*widget.Label)
				label.Alignment = fyne.TextAlignCenter
				label.SetText(text)
				label.Refresh()

				text = truncator(asset.hash)
				label = contain.Objects[2].(*widget.Label)
				label.Alignment = fyne.TextAlignCenter
				label.SetText(text)
				label.Refresh()
				// now we'll check the balance of the asset against the address
				bal, _ := program.wallet.Get_Balance_scid(
					crypto.HashHexToHash(asset.hash),
				)
				text = rpc.FormatMoney(bal)
				label = contain.Objects[3].(*widget.Label)
				label.Alignment = fyne.TextAlignCenter
				label.SetText(text)
				label.Refresh()

			}
		}

		// and now here is how we want each item updated
		program.lists.asset_list.UpdateItem = updateItem

		//
		onSelected := func(id widget.ListItemID) {

			program.lists.asset_list.Unselect(id)

			// here is the asset hash
			asset := assets[id]

			// let's get the entries
			entries := program.wallet.GetAccount().EntriesNative[crypto.HashHexToHash(asset.hash)]

			sort.Slice(entries, func(i, j int) bool {
				return entries[i].Time.After(entries[j].Time)
			})

			// now let's get the scid as a string
			scid := asset.hash

			// again, let's make a list
			entries_list := new(widget.List)

			// set the length based on entries
			entries_list.Length = func() int {
				length := len(entries)
				if length == 0 {
					length = 1 // we are just going to make this up
				}
				return length
			}

			// we'll use a label for the list item
			entries_list.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }

			updateItem := func(lii widget.ListItemID, co fyne.CanvasObject) {
				label := co.(*widget.Label)

				if len(entries) == 0 {
					bal, _ := program.wallet.Get_Balance_scid(
						crypto.HashHexToHash(asset.hash),
					)
					if bal != 0 {
						label.SetText("entries are syncing in the background, pls come back later")
					} else {
						label.SetText("ERROR")
					}
					return
				}
				// let's make sure the entry is bodied
				entries[lii].ProcessPayload()

				// let's split the entry up
				parts := strings.Split(entries[lii].String(), "\n")

				// get the tx type from the header
				tx_type := parts[0]

				// make a timestamp string in local format
				time_stamp := entries[lii].Time.Local().Format("2006-01-02 15:04")

				hash := truncator(entries[lii].TXID)

				// here's the simple label
				text := time_stamp + " " + hash + " " + tx_type

				// and let's set the text for it
				label.SetText(text)
			}
			// here is how we'll update the item to look
			entries_list.UpdateItem = updateItem

			// so now, when we select an entry on the list we'll show the transfer details
			onSelected := func(id widget.ListItemID) {
				entries_list.Unselect(id)

				lines := strings.Split(entries[id].String(), "\n")
				keys := []string{}
				values := []string{}
				for _, line := range lines {
					if line == "" {
						continue
					}
					pair := strings.Split(line, ": ")
					key := pair[0]
					value := pair[1]
					keys = append(keys, key)
					values = append(values, value)
				}
				table := widget.NewTable(
					func() (rows int, cols int) { return len(lines), 2 },
					func() fyne.CanvasObject { return widget.NewLabel("") },
					func(tci widget.TableCellID, co fyne.CanvasObject) {
						label := co.(*widget.Label)
						if len(keys) <= tci.Row {
							return
						}
						switch tci.Col {
						case 0:
							label.SetText(keys[tci.Row])
						case 1:
							label.SetText(values[tci.Row])
						}
					},
				)
				table.SetColumnWidth(0, largestMinSize(keys).Width)
				table.SetColumnWidth(1, largestMinSize(values).Width)
				table.Refresh()
				table.OnSelected = func(id widget.TableCellID) {

					if id.Col > 0 {
						data := values[id.Row]
						program.application.Clipboard().SetContent(values[id.Row])
						showInfo("", data+" copied to clipboard")
					}
				}

				// we'll truncate the scid for the title
				title := truncator(scid)

				// now set it, resize it and show it
				entry := dialog.NewCustom(title, dismiss, container.NewStack(table), program.window)
				entry.Resize(program.size)
				entry.Show()
			}
			entries_list.OnSelected = onSelected

			var smart_contract_details *dialog.CustomDialog
			scid_hyperlink := widget.NewHyperlink(scid, nil)
			scid_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(scid)
				showInfo("", scid+" copied to clipboard")
			}
			img := canvas.NewImageFromImage(setSCIDThumbnail(asset.image, float32(250), float32(250)))
			img.FillMode = canvas.ImageFillOriginal
			contain := container.NewPadded(img)
			hash := crypto.HashHexToHash(asset.hash)
			bal, _ := program.wallet.Get_Balance_scid(hash)

			label_balance := widget.NewLabel("Balance: " + rpc.FormatMoney(bal))

			// this simple go routine will update the balance every second
			var updating bool = true
			go func() {
				for range time.NewTicker(time.Second * 2).C {
					if updating {
						updated, _ := program.wallet.Get_Balance_scid(hash)
						fyne.DoAndWait(func() { label_balance.SetText("Balance: " + rpc.FormatMoney(updated)) })
					} else {
						return
					}
				}
			}()

			address := widget.NewEntry()
			address.Validator = func(s string) error {
				if s == "" {
					return nil
				}

				addr, err := rpc.NewAddress(s)
				if err != nil {
					if a_string, err := program.wallet.NameToAddress(s); err != nil {
						return err
					} else {
						addr, _ = rpc.NewAddress(a_string)
					}
				}
				if addr.IsIntegratedAddress() {
					return errors.New("TODO: integrated addresses")
				}
				if !isRegistered(addr.String()) {
					return errors.New("unregistered address")
				}
				return nil
			}
			balance := widget.NewEntry()
			balance.SetPlaceHolder("1.337")
			callback := func() {
				fl, err := strconv.ParseFloat(balance.Text, 64)
				if err != nil {
					showError(err)
					return
				}

				if fl < 0 {
					showError(errors.New("cannot send less than zero"))
					return
				}

				amount := uint64(fl * atomic_units)
				if amount == 0 {
					showError(errors.New("cannot send zero"))
					return
				}

				// obviously, we can't send to no one,
				// especially not a non-validated no one
				if address.Text == "" {
					showError(errors.New("cannot send to empty address"))
					return
				}

				// get entry
				send_recipient := address.Text

				// dump the entry text
				address.SetText("")

				// let's validate on the send action

				// if less than 4 char...
				if len(send_recipient) < 4 {
					showError(errors.New("cannot be less than 5 char"))
					return
				}

				// first check to see if it is an address
				addr, err := rpc.NewAddress(send_recipient)
				// if it is not an address...
				if err != nil {
					// check to see if it is a name
					a, err := program.wallet.NameToAddress(send_recipient)
					if err != nil {
						// she barks
						// fmt.Println(err)
					}
					// if a valid , they are the receiver
					if a != "" {
						if strings.EqualFold(
							a, program.wallet.GetAddress().String(),
						) {
							showError(errors.New("cannot send to self"))
							return
						} else {
							program.receiver = a
						}
					}
				}
				// now if the address is an integrated address...
				program.receiver = addr.BaseAddress().String()
				// at this point, we should be fairly confident
				if program.receiver == "" {
					showError(errors.New("error obtaining receiver"))
					return
				}
				// also, would make sense to make sure that it is not self
				if strings.EqualFold(program.receiver, program.wallet.GetAddress().String()) {
					showError(errors.New("cannot send to self"))
					return
				}

				// but just to be extra sure...
				// let's see if the receiver is not registered
				if !isRegistered(program.receiver) {
					showError(errors.New("unregistered address"))
					return
				}
				// obtain their DERO balance
				bal := program.wallet.GetAccount().Balance_Mature
				// and check
				if bal < 80 {
					showError(errors.New("balance is too low, please refill wallet"))
					return
				}

				// obtain their balance
				bal = program.wallet.GetAccount().Balance[hash]
				// and check
				if bal < amount {
					showError(errors.New("balance is too low, please refill"))
					return
				}
				payload := []rpc.Transfer{
					{
						SCID:        hash,
						Destination: program.receiver,
						Amount:      amount,
						// Payload_RPC: args, // when is this necessary?
					},
				}

				// drop the receiver address
				program.receiver = ""

				label := makeCenteredWrappedLabel("sending asset... should see txid soon")
				syncro := widget.NewActivity()
				syncro.Start()
				content := container.NewVBox(label, container.NewCenter(syncro))
				transact := dialog.NewCustom("Transaction Dispatched", dismiss, content, program.window)
				transact.Resize(program.size)
				transact.Show()
				var retries int
				// we are going to get the current default ringsize
				rings := uint64(program.wallet.GetRingSize())

				// we are NOT sending all, which is not implemented
				send_all := false

				// we are not doing a sc call
				scdata := rpc.Arguments{}

				// there is no gasstorage in this call
				gasstorage := uint64(0)

				// and we are not going to dry run to get data from the transfer
				dry_run := false

				go func() {
				try_again:
					// and let's build a transaction
					tx, err := program.wallet.TransferPayload0(
						payload, rings, send_all,
						scdata, gasstorage, dry_run,
					)
					// if this explodes...
					if err != nil {

						// show the error
						showError(err)
						// now let's make sure each of these are re-enabled

						return
					}
					if !program.preferences.Bool("loggedIn") {
						return
					}

					// let's send this to the node for processing
					if err := program.wallet.SendTransaction(tx); err != nil {

						if retries < 4 {
							retries++
							goto try_again
						} else {
							fyne.DoAndWait(func() {
								// if it errors out, show the err
								showError(err)
								syncro.Stop()
								transact.Dismiss()
							})
						}
						// and return

					} else { // if we have success

						// let's make a link
						link := truncator(tx.GetHash().String())

						// set it into a new widget -> consider launching your own explorer
						txid := widget.NewHyperlink(link, nil)

						// align it center
						txid.Alignment = fyne.TextAlignCenter

						// when tapped, copy to clipboard
						txid.OnTapped = func() {
							program.application.Clipboard().SetContent(tx.GetHash().String())
							showInfo("", "txid copied to clipboard")
						}
						fyne.DoAndWait(func() {
							syncro.Stop()
							transact.Dismiss()
							// set it to a new dialog screen and show
							dialog.ShowCustom(
								"Transaction Dispatched", "dismissed",
								container.NewVBox(txid), program.window,
							)
						})
					}
				}()
			}
			address.OnSubmitted = func(s string) {
				callback()
			}
			address.ActionItem = widget.NewButtonWithIcon("Send", theme.MailSendIcon(), callback)
			address.SetPlaceHolder("receiver address: dero...")
			lay := &twoThirds{}
			lay.Orientation = fyne.TextAlignTrailing
			send := container.New(lay, balance, address)
			content := container.NewVBox(
				container.NewCenter(scid_hyperlink),
				contain,
				container.NewCenter(widget.NewLabel(asset.name)),
				container.NewCenter(label_balance),
				send,
			)

			confirm := widget.NewHyperlink("Are You Sure?", nil)

			onTapped := func() {

				// delete the item from the EntriesNative Map
				delete(program.wallet.GetAccount().EntriesNative, crypto.HashHexToHash(scid))

				// rebuild the asset cache
				buildAssetHashList()

				assets = program.caches.assets

				// for good measure, we'll refresh the list
				program.lists.asset_list.Refresh()
				smart_contract_details.Dismiss()
			}
			confirm.OnTapped = onTapped

			result := getSCValues(scid)
			// fmt.Println(result)
			// let's make some tabs
			tabs := container.NewAppTabs(
				container.NewTabItem("Details",
					content,
				),
				container.NewTabItem("Code",
					container.NewScroll(
						widget.NewRichTextWithText(getSCCode(scid).Code),
					),
				),
				container.NewTabItem("Balances",
					container.NewScroll(
						getSCIDBalancesContainer(result.Balances),
					),
				),
				container.NewTabItem("String Variables",
					container.NewScroll(
						getSCIDStringVarsContainer(result.VariableStringKeys),
					),
				),
				container.NewTabItem("Uint64 Variables",
					container.NewScroll(
						getSCIDUint64VarsContainer(result.VariableUint64Keys),
					),
				),
				container.NewTabItem("Entries",
					entries_list,
				),
				container.NewTabItem("Remove",
					container.NewCenter(confirm),
				),
			)

			// kind of looks nice on the side
			tabs.SetTabLocation(container.TabLocationLeading)
			// we'll use the truncated scid as the header for the transfers
			title := truncator(scid)
			// set the entries in the dialog, resize and show
			smart_contract_details = dialog.NewCustom(title, dismiss, tabs, program.window)
			// this will stop the simple balance update tool
			smart_contract_details.SetOnClosed(func() {
				updating = false
			})
			smart_contract_details.Resize(program.size)
			smart_contract_details.Show()
		}

		program.lists.asset_list.OnSelected = onSelected

		filter := widget.NewEntry()
		filterer := func() {
			s := strings.ToLower(filter.Text)
			if s != "" {
				assets = []asset{}
				for _, each := range program.caches.assets {

					if strings.Contains(strings.ToLower(each.hash), s) || strings.Contains(strings.ToLower(each.name), s) {
						assets = append(assets, asset{
							name:  each.name,
							hash:  each.hash,
							image: each.image,
						})
					}
				}
			} else {
				assets = program.caches.assets
			}

			program.lists.asset_list.Refresh()
		}
		filter.OnChanged = func(s string) {
			filterer()
		}
		filter.ActionItem = widget.NewButtonWithIcon("Filter", theme.SearchIcon(), filterer)

		// let's set the asset list into a new list
		list = container.NewBorder(
			filter,
			nil,
			nil,
			nil,
			program.lists.asset_list)

		list.Refresh()

	}

	// and we'll set the scroll into a new dialog, resize and show
	collection := dialog.NewCustom("Collectibles", dismiss, list, program.window)
	collection.Resize(program.size)
	collection.Show()
}
