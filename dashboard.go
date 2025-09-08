package main

import (
	"errors"
	"sort"
	"strings"

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
	return container.NewVBox(
		container.NewAdaptiveGrid(3,
			container.NewVBox(program.buttons.transactions),
			container.NewVBox(program.buttons.assets),
			container.NewVBox(program.buttons.keys),
		),
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

	// we'll use the length of entries for the count of widget's to return
	sent.Length = func() int { return len(s_entries) }

	// here is the widget that we are going to use for each item of the list
	sent.CreateItem = createLabel
	// then let's update the item to contain the content
	updateItem := func(lii widget.ListItemID, co fyne.CanvasObject) {

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
	sent.UpdateItem = updateItem

	// then when we select one of them, let' open it up!
	onSelected := func(id widget.ListItemID) {

		sent.Unselect(id)

		// body the entries
		s_entries[id].ProcessPayload()

		e := s_entries[id]

		lines := strings.SplitSeq(e.String(), "\n")
		tx_details := container.NewVBox()

		for line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			key_entry := widget.NewLabel(key)
			value := pair[1]
			var v string = value
			if key != "Time" {
				if len(value) > 16 {
					v = truncator(value)
				}
			}
			value_hyperlink := widget.NewHyperlink(v, nil)
			value_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(value)
				showInfo("", value+" copied to clipboard")
			}
			tx_details.Add(container.NewAdaptiveGrid(2, key_entry, value_hyperlink))
		}

		// make a simple details scrolling container
		details := container.NewScroll(tx_details)

		// load up dialog with the container
		txs := dialog.NewCustom(
			"Transaction Detail", dismiss,
			details, program.window,
		)

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

	// we'll use the length of entries for the count of widget's to return
	received.Length = func() int { return len(r_entries) }

	// here is the widget that we are going to use for each item of the list
	received.CreateItem = createLabel

	// then let's update the item to contain the content
	updateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {

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
	received.UpdateItem = updateItem

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		received.Unselect(id)

		// body the entries
		r_entries[id].ProcessPayload()

		e := r_entries[id]

		lines := strings.SplitSeq(e.String(), "\n")
		tx_details := container.NewVBox()

		for line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			key_entry := widget.NewLabel(key)
			value := pair[1]
			var v string = value
			if key != "Time" {
				if len(value) > 16 {
					v = truncator(value)
				}
			}
			value_hyperlink := widget.NewHyperlink(v, nil)
			value_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(value)
				showInfo("", value+" copied to clipboard")
			}
			tx_details.Add(container.NewAdaptiveGrid(2, key_entry, value_hyperlink))
		}

		// make a simple details scrolling container
		details := container.NewScroll(tx_details)

		// load up dialog with the container
		txs := dialog.NewCustom(
			"Transaction Detail", dismiss,
			details, program.window,
		)

		txs.Resize(program.size)
		txs.Show()
	}

	// set the on selected field
	received.OnSelected = onSelected

	// here are all the coinbase entries
	coins := getCoinbaseTransfers()

	sort.Slice(coins, func(i, j int) bool {
		return coins[i].Height > coins[j].Height
	})

	// let's make a list of transactions
	coinbase := new(widget.List)

	// we'll use the length of entries for the count of widget's to return
	coinbase.Length = func() int { return len(coins) }

	// here is the widget that we are going to use for each item of the list
	coinbase.CreateItem = createLabel

	// then let's update the item to contain the content
	updateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {

		// let's make sure the entry is bodied
		coins[lii].ProcessPayload()

		// make a timestamp string in local format
		time_stamp := coins[lii].Time.Local().Format("2006-01-02 15:04")
		txid := truncator(coins[lii].BlockHash)
		amount := coins[lii].Amount
		container := co.(*fyne.Container)
		container.Objects[0].(*widget.Label).SetText(time_stamp)
		container.Objects[1].(*widget.Label).SetText(txid)
		container.Objects[2].(*widget.Label).SetText(rpc.FormatMoney(amount))
	}
	// set the update item
	coinbase.UpdateItem = updateItem

	// then when we select one of them, let' open it up!
	onSelected = func(id widget.ListItemID) {

		coinbase.Unselect(id)

		// body the entries
		coins[id].ProcessPayload()

		e := coins[id]

		lines := strings.SplitSeq(e.String(), "\n")
		tx_details := container.NewVBox()

		for line := range lines {
			if line == "" {
				continue
			}
			pair := strings.Split(line, ": ")
			key := pair[0]
			key_entry := widget.NewLabel(key)
			value := pair[1]
			var v string = value
			if key != "Time" {
				if len(value) > 16 {
					v = truncator(value)
				}
			}
			value_hyperlink := widget.NewHyperlink(v, nil)
			value_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(value)
				showInfo("", value+" copied to clipboard")
			}
			tx_details.Add(container.NewAdaptiveGrid(2, key_entry, value_hyperlink))
		}

		// make a simple details scrolling container
		details := container.NewScroll(tx_details)

		// load up dialog with the container
		txs := dialog.NewCustom(
			"Transaction Detail", dismiss,
			details, program.window,
		)

		txs.Resize(program.size)
		txs.Show()
	}
	// set the field
	coinbase.OnSelected = onSelected

	tabs := container.NewAppTabs(
		container.NewTabItem("Sent", sent),
		container.NewTabItem("Received", received),
		container.NewTabItem("Coinbase", coinbase),
	)
	tabs.SetTabLocation(container.TabLocationLeading)
	txs := dialog.NewCustom("transactions", dismiss, tabs, program.window)
	txs.Resize(program.size)
	txs.Show()
}

func assetsList() {

	// let's just refresh the hash cache
	// buildAssetHashList()

	var scroll *container.Scroll

	if hashesLength() == 0 {
		scroll = container.NewVScroll(container.NewVBox(
			layout.NewSpacer(),
			makeCenteredWrappedLabel("Please visit tools, and Add SCID to your collection"),
			layout.NewSpacer(),
		))
	} else {

		program.lists.asset_list.HideSeparators = true

		// let's use the length of the hash cache for the number of objects to make
		program.lists.asset_list.Length = hashesLength

		// and here is the widget we'll use for each item in the list
		program.lists.asset_list.CreateItem = createImageLabel

		updateItem := func(lii widget.ListItemID, co fyne.CanvasObject) {
			// here is the asset details
			asset := program.caches.assets[lii]

			// here is the container
			contain := co.(*fyne.Container)
			// here is the container holding the image
			padded := contain.Objects[0].(*fyne.Container)

			// here is the image object
			img := padded.Objects[0].(*canvas.Image)

			// we'll use the asset image when not nil
			if asset.image != nil {
				img.Image = asset.image
			} else { // otherwise, set the resource to be the broken icon
				img.Resource = theme.BrokenImageIcon()
			}

			text := asset.name
			label := contain.Objects[1].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)

			text = truncator(asset.hash)
			label = contain.Objects[2].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)

			// now we'll check the balance of the asset against the address
			bal, _ := program.wallet.Get_Balance_scid(
				crypto.HashHexToHash(asset.hash),
			)
			text = rpc.FormatMoney(bal)
			label = contain.Objects[3].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)
		}

		// and now here is how we want each item updated
		program.lists.asset_list.UpdateItem = updateItem

		//
		onSelected := func(id widget.ListItemID) {

			program.lists.asset_list.Unselect(id)

			// here is the asset hash
			asset := program.caches.assets[id]

			// let's get the entries
			entries := program.wallet.GetAccount().EntriesNative[crypto.HashHexToHash(asset.hash)]

			// now let's get the scid as a string
			scid := asset.hash

			// again, let's make a list
			entries_list := new(widget.List)

			// set the length based on entries
			entries_list.Length = func() int { return len(entries) }

			// we'll use a label for the list item
			entries_list.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }

			updateItem := func(lii widget.ListItemID, co fyne.CanvasObject) {

				// let's make sure the entry is bodied
				entries[lii].ProcessPayload()

				// let's split the entry up
				parts := strings.Split(entries[lii].String(), "\n")

				// get the tx type from the header
				tx_type := parts[0]

				// make a timestamp string in local format
				time_stamp := entries[lii].Time.Local().Format("2006-01-02 15:04")

				// here's the simple label
				text := time_stamp + " " + tx_type

				label := co.(*widget.Label)

				// and let's set the text for it
				label.SetText(text)
			}
			// here is how we'll update the item to look
			entries_list.UpdateItem = updateItem

			// so now, when we select an entry on the list we'll show the transfer details
			onSelected := func(id widget.ListItemID) {
				entries_list.Unselect(id)
				sort.Slice(entries, func(i, j int) bool {
					return entries[i].Time.After(entries[j].Time)
				})

				lines := strings.SplitSeq(entries[id].String(), "\n")
				asset_details := container.NewVBox()
				scid_label := widget.NewLabel("SCID")
				scid_hyperlink := widget.NewHyperlink(truncator(scid), nil)
				scid_hyperlink.OnTapped = func() {
					program.application.Clipboard().SetContent(scid)
					showInfo("", scid+" copied to clipboard")
				}
				asset_details.Add(container.NewAdaptiveGrid(2, scid_label, scid_hyperlink))
				for line := range lines {
					if line == "" {
						continue
					}
					pair := strings.Split(line, ": ")
					key := pair[0]
					value := pair[1]
					var v string = value
					if key != "Time" {
						if len(value) > 16 {
							v = truncator(value)
						}
					}
					value_hyperlink := widget.NewHyperlink(v, nil)
					value_hyperlink.OnTapped = func() {
						program.application.Clipboard().SetContent(value)
						showInfo("", value+" copied to clipboard")
					}
					asset_details.Add(
						container.NewAdaptiveGrid(2,
							widget.NewLabel(key),
							value_hyperlink,
						),
					)
				}

				// we'll truncate the scid for the title
				title := truncator(scid)

				details := container.NewScroll(asset_details)

				// now set it, resize it and show it
				entry := dialog.NewCustom(title, dismiss, details, program.window)
				entry.Resize(program.size)
				entry.Show()
			}
			entries_list.OnSelected = onSelected

			var smart_contract_details *dialog.CustomDialog

			// we'll use the truncated scid as the header for the transfers
			title := truncator(scid)

			scid_hyperlink := widget.NewHyperlink(scid, nil)
			scid_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(scid)
				showInfo("", scid+" copied to clipboard")
			}

			img := setSCIDThumbnail(asset.image, 250, 250)
			img.FillMode = canvas.ImageFillOriginal
			contain := container.NewPadded(img)
			content := container.NewAdaptiveGrid(1,
				contain,
				container.NewCenter(widget.NewLabel(asset.name)),
				container.NewCenter(scid_hyperlink),
			)

			confirm := widget.NewHyperlink("Are You Sure?", nil)

			onTapped := func() {

				// delete the item from the EntriesNative Map
				delete(program.wallet.GetAccount().EntriesNative, crypto.HashHexToHash(scid))

				// rebuild the asset cache
				buildAssetHashList()

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

			// set the entries in the dialog, resize and show
			smart_contract_details = dialog.NewCustom(title, dismiss, tabs, program.window)
			smart_contract_details.Resize(program.size)
			smart_contract_details.Show()
		}

		program.lists.asset_list.OnSelected = onSelected

		// let's set the asset list into a new scroll
		scroll = container.NewScroll(program.lists.asset_list)

		scroll.Refresh()

	}

	// and we'll set the scroll into a new dialog, resize and show
	collection := dialog.NewCustom("Collectibles", dismiss, scroll, program.window)
	collection.Resize(program.size)
	collection.Show()
}
