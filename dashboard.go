package main

import (
	"bytes"
	"encoding/hex"
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
	"github.com/deroproject/derohe/transaction"
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

	// simple way to see keys
	var k *dialog.FormDialog
	password := widget.NewPasswordEntry()
	password.SetPlaceHolder("w41137-p@55w0rd")
	// if they press enter, it is the same as clicking confirm
	password.OnSubmitted = func(s string) {
		k.Submit()
		k.Dismiss()
	}

	callback := func(b bool) {
		pass := password.Text
		password.SetText("")

		if !b { // if they cancel
			return
		}
		// check the password for all sensitive actions
		if !program.wallet.Check_Password(pass) {
			// if they get is wrong, tell them
			showError(errors.New("wrong password"), program.window)
			return
		} else { // if they get it right
			headers := []string{
				"SEED PHRASE",
				"SECRET KEY",
				"PUBLIC KEY",
			}
			data := []string{
				program.wallet.GetSeed(),
				program.wallet.Get_Keys().Secret.Text(16),
				program.wallet.Get_Keys().Public.StringHex(),
			}
			table := widget.NewTable(
				func() (rows int, cols int) { return 3, 2 },
				func() fyne.CanvasObject { return widget.NewLabel("") },
				func(tci widget.TableCellID, co fyne.CanvasObject) {
					label := co.(*widget.Label)
					switch tci.Col {
					case 0:
						label.SetText(headers[tci.Row])
					case 1:
						label.SetText(data[tci.Row])
						if tci.Row == 0 {
							label.Wrapping = fyne.TextWrapWord
						}
					}
				},
			)

			table.SetColumnWidth(0, largestMinSize(headers).Width)
			table.SetRowHeight(0, 75)
			table.SetColumnWidth(1, largestMinSize(data[1:]).Width)
			table.Refresh()
			table.OnSelected = func(id widget.TableCellID) {
				table.UnselectAll()
				if id.Col > 0 {
					program.application.Clipboard().SetContent(data[id.Row])
					showInfoFast("Copied", "Copied "+headers[id.Row], program.window)
				}
			}

			// let's make a dialog window with all the keys included
			keys := dialog.NewCustom("Keys", dismiss, table, program.window)
			size := fyne.NewSize(((program.size.Width / 10) * 8), ((program.size.Height / 2) * 1))
			keys.Resize(size)
			keys.Show()
			return

		}
	}

	// create a simple form content
	content := []*widget.FormItem{widget.NewFormItem("", password)}

	// set the content and callback
	k = dialog.NewForm("Display Keys?", confirm, dismiss, content, callback, program.window)

	k.Show()

}
func txList() {
	var lines list
	var keys, values []string

	length := func() (rows int, cols int) { return len(lines), 2 }
	create := func() fyne.CanvasObject { return widget.NewLabel("") }
	update := func(tci widget.TableCellID, co fyne.CanvasObject) {
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
	}
	// here are all the sent entries
	var s_entries wallet_entries
	s_entries = getSentTransfers(crypto.ZEROHASH)
	s_entries.sort()

	// let's make a list of transactions
	sent := new(widget.List)

	// we'll use the length of s_entries for the count of widget's to return
	sent.Length = func() int { return len(s_entries) }

	// here is the widget that we are going to use for each item of the list
	sent.CreateItem = createThreeLabels

	// then let's update the item to contain the content
	var s_table *widget.Table
	updateSent := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(s_entries) {
			return
		}

		// let's make sure the entry is bodied
		s_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		texts := []string{
			s_entries[lii].Time.Local().Format("2006-01-02 15:04"),
			truncator(s_entries[lii].TXID),
			// rpc.FormatMoney(t.Fees()),
			rpc.FormatMoney(s_entries[lii].Amount),
		}

		container := co.(*fyne.Container)

		for i, obj := range container.Objects {
			obj.(*widget.Label).SetText(texts[i])
		}

	}
	// set the update item
	sent.UpdateItem = updateSent

	// then when we select one of them, let' open it up!
	onSelected := func(id widget.ListItemID) {
		sent.Unselect(id)

		// body the s_entries
		e := s_entries[id]

		lines = strings.Split(e.String(), "\n")
		keys, values = lines.split_to_kv(": ")

		tx := getTransaction(
			rpc.GetTransaction_Params{Tx_Hashes: []string{e.TXID}},
		)

		for _, each := range tx.Txs_as_hex {
			b, err := hex.DecodeString(each)
			if err != nil {
				continue
			}
			var t transaction.Transaction
			if err := t.Deserialize(b); err != nil {
				continue
			}

			keys = append(keys, "FEES")
			fees := rpc.FormatMoney(t.Fees())
			values = append(values, fees)
		}

		s_table = widget.NewTable(length, create, update)

		width := largestMinSize(keys).Width
		s_table.SetColumnWidth(0, width)

		width = largestMinSize(values).Width
		s_table.SetColumnWidth(1, width)

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
	var r_entries wallet_entries
	r_entries = getReceivedTransfers(crypto.ZEROHASH)
	r_entries.sort()

	// let's make a list of received transactions
	received := new(widget.List)

	// we'll use the length of r_entries for the count of widget's to return
	received.Length = func() int { return len(r_entries) }

	// here is the widget that we are going to use for each item of the list
	received.CreateItem = createThreeLabels

	var r_table *widget.Table
	// then let's update the item to contain the content
	updateReceived := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(r_entries) {
			return
		}
		// let's make sure the entry is bodied
		r_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		texts := []string{
			r_entries[lii].Time.Local().Format("2006-01-02 15:04"),
			truncator(r_entries[lii].TXID),
			rpc.FormatMoney(r_entries[lii].Amount),
		}

		container := co.(*fyne.Container)
		for i, obj := range container.Objects {
			obj.(*widget.Label).SetText(texts[i])
		}
	}

	// set the update item field
	received.UpdateItem = updateReceived

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		received.Unselect(id)

		// body the r_entries
		e := r_entries[id]

		lines = strings.Split(e.String(), "\n")
		keys, values = lines.split_to_kv(": ")

		r_table = widget.NewTable(length, create, update)
		width := largestMinSize(keys).Width
		r_table.SetColumnWidth(0, width)

		width = largestMinSize(values).Width
		r_table.SetColumnWidth(1, width)

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
	var c_entries wallet_entries
	c_entries = getCoinbaseTransfers(crypto.ZEROHASH)
	c_entries.sort()

	// let's make a list of transactions
	coinbase := new(widget.List)

	// we'll use the length of c_entries for the count of widget's to return
	coinbase.Length = func() int { return len(c_entries) }

	// here is the widget that we are going to use for each item of the list
	coinbase.CreateItem = createThreeLabels

	// then let's update the item to contain the content
	var c_table *widget.Table
	updateCoins := func(lii widget.ListItemID, co fyne.CanvasObject) {

		if lii >= len(c_entries) {
			return
		}
		// let's make sure the entry is bodied
		c_entries[lii].ProcessPayload()

		// make a timestamp string in local format
		texts := []string{
			c_entries[lii].Time.Local().Format("2006-01-02 15:04"),
			truncator(c_entries[lii].BlockHash),
			rpc.FormatMoney(c_entries[lii].Amount),
		}
		container := co.(*fyne.Container)
		for i, obj := range container.Objects {
			obj.(*widget.Label).SetText(texts[i])
		}
	}
	// set the update item
	coinbase.UpdateItem = updateCoins

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		coinbase.Unselect(id)

		// body the c_entries
		e := c_entries[id]

		lines = strings.Split(e.String(), "\n")
		keys, values = lines.split_to_kv(": ")

		c_table = widget.NewTable(length, create, update)

		width := largestMinSize(keys).Width
		c_table.SetColumnWidth(0, width)

		width = largestMinSize(values).Width
		c_table.SetColumnWidth(1, width)

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
			var s wallet_entries
			s = getSentTransfers(crypto.ZEROHASH)
			s.sort()

			if search == "" {
				s_entries = s
			} else {
				s_entries = wallet_entries{}
				for _, each := range s {

					tx := getTransaction(rpc.GetTransaction_Params{
						Tx_Hashes: []string{each.TXID},
					})
					b, err := hex.DecodeString(tx.Txs_as_hex[0])
					if err != nil {
						panic(err)
					}
					var t transaction.Transaction
					if err := t.Deserialize(b); err != nil {
						panic(err)
					}

					if strings.Contains(rpc.FormatMoney(t.Fees()), search) ||
						strings.Contains(strings.ToLower(each.String()), search) {
						s_entries = append(s_entries, each)
					}
				}
			}
			sent.Refresh()
		case "Received":
			var r wallet_entries
			r = getReceivedTransfers(crypto.ZEROHASH)
			r.sort()

			if search == "" {
				r_entries = r
			} else {
				r_entries = wallet_entries{}
				for _, each := range r {
					if strings.Contains(strings.ToLower(each.String()), search) {
						r_entries = append(r_entries, each)
					}
				}
			}
			received.Refresh()
		case "Coinbase":
			var c wallet_entries
			c = getCoinbaseTransfers(crypto.ZEROHASH)
			c.sort()

			if search == "" {
				c_entries = c
			} else {
				c_entries = wallet_entries{}
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

	search_bar := createTXListSearchBar(search_entry)
	header := createTXListHeader()

	top := container.NewVBox(search_bar, header)
	content := container.NewBorder(top, nil, nil, nil, sent)
	st := container.NewTabItem("Sent", content)

	top = container.NewVBox(search_bar, header)
	content = container.NewBorder(top, nil, nil, nil, received)
	rt := container.NewTabItem("Received", content)

	top = container.NewVBox(search_bar, header)
	content = container.NewBorder(top, nil, nil, nil, coinbase)
	ct := container.NewTabItem("Coinbase", content)

	tabs = container.NewAppTabs(st, rt, ct)
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

	program.buttons.asset_scan.OnTapped = asset_scan

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
					showError(err, program.window)
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
				table.UnselectAll()
				if id.Col > 0 {
					data := values[id.Row]
					program.application.Clipboard().SetContent(values[id.Row])
					showInfo("", data+" copied to clipboard", program.window)
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
			showInfo("", scid+" copied to clipboard", program.window)
		}
		img := canvas.NewImageFromImage(setSCIDThumbnail(asset.image, float32(250), float32(250)))
		img.FillMode = canvas.ImageFillOriginal
		hash := crypto.HashHexToHash(asset.hash)
		bal, _ := program.wallet.Get_Balance_scid(hash)

		label_balance := widget.NewLabel("Balance: " + rpc.FormatMoney(bal))

		// this simple go routine will update the balance every second
		var updating bool = true
		listen_for_balance_changes := func() {
			for range time.NewTicker(time.Second * 2).C {
				if updating {
					updated, _ := program.wallet.Get_Balance_scid(hash)
					fyne.DoAndWait(func() { label_balance.SetText("Balance: " + rpc.FormatMoney(updated)) })
				} else {
					return
				}
			}
		}
		go listen_for_balance_changes()

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
				showError(err, program.window)
				return
			}

			if fl < 0 {
				showError(errors.New("cannot send less than zero"), program.window)
				return
			}

			amount := uint64(fl * atomic_units)
			if amount == 0 {
				showError(errors.New("cannot send zero"), program.window)
				return
			}

			// obviously, we can't send to no one,
			// especially not a non-validated no one
			if address.Text == "" {
				showError(errors.New("cannot send to empty address"), program.window)
				return
			}

			// get entry
			send_recipient := address.Text

			// dump the entry text
			address.SetText("")

			// let's validate on the send action

			// if less than 4 char...
			if len(send_recipient) < 4 {
				showError(errors.New("cannot be less than 5 char"), program.window)
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
						showError(errors.New("cannot send to self"), program.window)
						return
					} else {
						addr, err = rpc.NewAddress(a)
						if err != nil {
							// now what's going on here?
						}
						program.receiver = a
					}
				}
			}
			// now if the address is an integrated address...
			if addr.IsIntegratedAddress() {
				program.receiver = addr.BaseAddress().String()
			}

			// at this point, we should be fairly confident
			if program.receiver == "" {
				showError(errors.New("error obtaining receiver"), program.window)
				return
			}
			// also, would make sense to make sure that it is not self
			if strings.EqualFold(program.receiver, program.wallet.GetAddress().String()) {
				showError(errors.New("cannot send to self"), program.window)
				return
			}

			// but just to be extra sure...
			// let's see if the receiver is not registered
			if !isRegistered(program.receiver) {
				showError(errors.New("unregistered address"), program.window)
				return
			}
			// obtain their DERO balance
			bal := program.wallet.GetAccount().Balance_Mature
			// and check
			if bal < 80 {
				showError(errors.New("balance is too low, please refill wallet"), program.window)
				return
			}

			// obtain their balance
			bal = program.wallet.GetAccount().Balance[hash]
			// and check
			if bal < amount {
				showError(errors.New("balance is too low, please refill"), program.window)
				return
			}

			payload := []rpc.Transfer{{SCID: hash, Destination: program.receiver, Amount: amount}}

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
			send_asset := func() {
			try_again:
				// and let's build a transaction
				tx, err := program.wallet.TransferPayload0(
					payload, rings, send_all,
					scdata, gasstorage, dry_run,
				)
				// if this explodes...
				if err != nil {

					// show the error
					showError(err, program.window)
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
							showError(err, program.window)
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
						showInfo("", "txid copied to clipboard", program.window)
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
			}
			go send_asset()
		}
		address.OnSubmitted = func(s string) {
			callback()
		}
		address.ActionItem = widget.NewButtonWithIcon("Send", theme.MailSendIcon(), callback)
		address.SetPlaceHolder("receiver address: dero...")

		lay := &twoThirds{}
		lay.Orientation = fyne.TextAlignTrailing

		asset_scid := container.NewCenter(scid_hyperlink)
		asset_img := container.NewPadded(img)

		asset_name := container.NewCenter(widget.NewLabel(asset.name))
		asset_bal := container.NewCenter(label_balance)

		send := container.New(lay, balance, address)

		content := container.NewVBox(asset_scid, asset_img, asset_name, asset_bal, send)

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

		// fmt.Println(result)
		result := getSCValues(scid)

		content_tab := container.NewTabItem("Details", content)

		code := getSCCode(scid).Code
		contract_code := container.NewScroll(widget.NewRichTextWithText(code))
		contract_tab := container.NewTabItem("Code", contract_code)

		balance_vars := container.NewScroll(getSCIDBalancesContainer(result.Balances))
		balances_tab := container.NewTabItem("Balances", balance_vars)

		keys, values := split_scid_keys(result.VariableUint64Keys)
		uint64_vars := container.NewScroll(getSCIDUint64VarsContainer(keys, values))
		uint64_tab := container.NewTabItem("Uint64 Variables", uint64_vars)

		keys, values = split_scid_keys(result.VariableStringKeys)
		string_vars := container.NewScroll(getSCIDStringVarsContainer(keys, values))
		strings_tab := container.NewTabItem("String Variables", string_vars)

		entries_tab := container.NewTabItem("Entries", entries_list)

		remove_scid := container.NewCenter(confirm)
		remove_tab := container.NewTabItem("Remove", remove_scid)

		app_tabs := []*container.TabItem{
			content_tab,
			contract_tab,
			balances_tab,
			strings_tab,
			uint64_tab,
			entries_tab,
			remove_tab,
		}
		// let's make some tabs
		tabs := container.NewAppTabs(app_tabs...)
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
	filter.SetPlaceHolder("filter by SCID or name")
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
	filter.ActionItem = widget.NewButtonWithIcon("", theme.SearchIcon(), filterer)
	// make a new entry widget
	t := widget.NewEntry()

	// set the place holder
	t.SetPlaceHolder("add SCID to collectibles")

	token_add := func() {
		// so if the map is nil, make one
		if program.wallet.GetAccount().EntriesNative == nil {
			program.wallet.GetAccount().EntriesNative = make(map[crypto.Hash][]rpc.Entry)
		}

		if t.Text == gnomonSC { // mainnet gnomon
			showError(errors.New("cannot add gnomon sc to collectibles"), program.window)
			return
		}
		//get the hash
		hash := crypto.HashHexToHash(t.Text)
		// start a sync activity widget
		syncing := widget.NewActivity()
		syncing.Start()
		notice := makeCenteredWrappedLabel("syncing")
		content := container.NewVBox(
			layout.NewSpacer(),
			syncing,
			notice,
			layout.NewSpacer(),
		)

		// set it to a splash screen
		sync := dialog.NewCustomWithoutButtons("syncing", content, program.window)

		// resize and show
		sync.Resize(program.size)
		sync.Show()
		go func() {

			// add the token
			if err := program.wallet.TokenAdd(hash); err != nil {
				// show err if one
				fyne.DoAndWait(func() {
					showError(err, program.window)
					sync.Dismiss()
				})
				return
			} else {
				// immediately rebuild the assets
				buildAssetHashList()

				// sync the token now for good measure
				if err := program.wallet.Sync_Wallet_Memory_With_Daemon_internal(hash); err != nil {
					fyne.DoAndWait(func() {
						showError(err, program.window)
						sync.Dismiss()
					})
					return
				}

				//make a notice
				notice := truncator(hash.String()) + "has been added to your collection"

				assets = program.caches.assets

				// give notice to the user
				fyne.DoAndWait(func() {
					showInfo("Token Add", notice, program.window)
					program.lists.asset_list.Refresh()
					sync.Dismiss()
				})
			}
		}()
	}
	t.OnSubmitted = func(s string) {
		token_add()
	}

	t.ActionItem = widget.NewButtonWithIcon("", theme.ContentAddIcon(), token_add)

	// let's set the asset list into a new list
	list = container.NewBorder(
		container.NewAdaptiveGrid(3,
			t, program.buttons.asset_scan, filter,
		),
		nil,
		nil,
		nil,
		program.lists.asset_list)

	list.Refresh()

	// and we'll set the scroll into a new dialog, resize and show
	collection := dialog.NewCustom("Collectibles", dismiss, list, program.window)
	collection.Resize(program.size)
	collection.Show()
}
