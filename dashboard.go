package main

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
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
	k = dialog.NewForm("Display Keys?", confirm, dismiss,
		[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
		func(b bool) {
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
					keys := dialog.NewCustom(
						"Keys", dismiss, scrollwindow, program.window,
					)

					keys.Resize(program.size)
					keys.Show()
					return
				}
			}
		}, program.window)

	k.Show()

	// dump password when done
	program.entries.pass.SetText("")
}
func txList() {
	// here are all the entries
	entries := allTransfers()

	// let's make a list of transactions
	program.lists.transactions = new(widget.List)

	// we'll use the length of entries for the count of widget's to return
	program.lists.transactions.Length = func() int { return len(entries) }

	// here is the widget that we are going to use for each item of the list
	program.lists.transactions.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }

	// then let's update the item to contain the content
	program.lists.transactions.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {
		// let's make sure the entry is bodied
		entries[lii].ProcessPayload()

		// let's split the entry up
		parts := strings.Split(entries[lii].String(), "\n")

		// get the tx type from the header
		tx_type := parts[0]

		// make a timestamp string in local format
		time_stamp := entries[lii].Time.Local().Format("2006-01-02 15:04")

		// here's the simple label
		label := time_stamp + " " + tx_type

		// and let's set the text for it
		co.(*widget.Label).SetText(label)
	}

	// then when we select one of them, let' open it up!
	program.lists.transactions.OnSelected = func(id widget.ListItemID) {

		program.lists.transactions.Unselect(id)

		// body the entries
		entries[id].ProcessPayload()

		e := entries[id]

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

	txs := dialog.NewCustom("transactions", dismiss, program.lists.transactions, program.window)
	txs.Resize(program.size)
	txs.Show()
}
func hashesLength() int { return len(program.caches.assets) }
func createLabel() fyne.CanvasObject {
	return container.NewAdaptiveGrid(3,
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)
}
func assetsList() {

	// let's just refresh the hash cache
	// buildAssetHashList()

	var scroll *container.Scroll

	if hashesLength() == 0 {
		scroll = container.NewVScroll(container.NewVBox(
			layout.NewSpacer(),
			program.buttons.token_add,
			layout.NewSpacer(),
		))
	} else {

		program.lists.asset_list.HideSeparators = true

		// let's use the length of the hash cache for the number of objects to make
		program.lists.asset_list.Length = hashesLength

		// and here is the widget we'll use for each item in the list
		program.lists.asset_list.CreateItem = createLabel

		// and now here is how we want each item updated
		program.lists.asset_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {

			// here is the asset details
			asset := program.caches.assets[lii]

			// here is the label for the list
			container := co.(*fyne.Container)
			text := asset.name
			label := container.Objects[0].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)

			text = truncator(asset.hash)
			label = container.Objects[1].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)

			// now we'll check the balance of the asset against the address
			bal, _ := program.wallet.Get_Balance_scid(
				crypto.HashHexToHash(asset.hash),
			)
			text = rpc.FormatMoney(bal)
			label = container.Objects[2].(*widget.Label)
			label.Alignment = fyne.TextAlignCenter
			label.SetText(text)
		}

		// when we select an item from the list, here's what we are going to do
		program.lists.asset_list.OnSelected = func(id widget.ListItemID) {

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

			// here is how we'll update the item to look
			entries_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {

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

			// so now, when we select an entry on the list we'll show the transfer details
			entries_list.OnSelected = func(id widget.ListItemID) {
				entries_list.Unselect(id)

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
			var smart_contract_details *dialog.CustomDialog

			// we'll use the truncated scid as the header for the transfers
			title := truncator(scid)

			scid_hyperlink := widget.NewHyperlink(scid, nil)
			scid_hyperlink.OnTapped = func() {
				program.application.Clipboard().SetContent(scid)
				showInfo("", scid+" copied to clipboard")
			}

			stuff := container.NewAdaptiveGrid(1,
				getSCIDImageThumbnailContainer(scid),
				container.NewCenter(widget.NewLabel(asset.name)),
				container.NewCenter(scid_hyperlink),
			)

			confirm := widget.NewHyperlink("Are You Sure?", nil)
			confirm.OnTapped = func() {

				// delete the item from the EntriesNative Map
				delete(program.wallet.GetAccount().EntriesNative, crypto.HashHexToHash(scid))

				// rebuild the asset cache
				buildAssetHashList()

				// for good measure, we'll refresh the list
				program.lists.asset_list.Refresh()
				smart_contract_details.Dismiss()
			}

			// let's make some tabs
			tabs := container.NewAppTabs(
				container.NewTabItem("Details",
					stuff,
				),
				container.NewTabItem("Code",
					container.NewScroll(
						widget.NewRichTextWithText(getSCCode(scid).Code),
					),
				),
				container.NewTabItem("Balances",
					container.NewScroll(
						getSCIDBalancesContainer(scid),
					),
				),
				container.NewTabItem("String Variables",
					container.NewScroll(
						getSCIDStringVarsContainer(scid),
					),
				),
				container.NewTabItem("Uint64 Variables",
					container.NewScroll(
						getSCIDUint64VarsContainer(scid),
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

		// for good measure, we'll refresh the list
		program.lists.asset_list.Refresh()

		// let's set the asset list into a new scroll
		scroll = container.NewScroll(program.lists.asset_list)
	}

	// and we'll set the scroll into a new dialog, resize and show
	collection := dialog.NewCustom("Collectibles", dismiss, scroll, program.window)
	collection.Resize(program.size)
	collection.Show()
}
