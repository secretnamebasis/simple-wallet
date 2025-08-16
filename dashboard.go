package main

import (
	"errors"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/rpc"
)

func dashboard() *fyne.Container {
	// simple way to find your keys
	program.hyperlinks.keys.OnTapped = keys

	// simple way to find all dero transfers
	program.hyperlinks.transactions.OnTapped = txList

	// simple way to review assets and their transfer histories :)
	program.hyperlinks.assets.OnTapped = assetsList

	// we'll return all this stuff into the home as a dashboard
	return container.NewVBox(
		container.NewCenter(container.NewHBox(
			container.NewCenter(program.hyperlinks.address),
			container.NewCenter(program.labels.balance),
		)),
		container.NewAdaptiveGrid(3,
			container.NewCenter(program.hyperlinks.transactions),
			container.NewCenter(program.hyperlinks.assets),
			container.NewCenter(program.hyperlinks.keys),
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
	dialog.ShowForm("Display Keys?", "confirm", "cancel",
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
						"Keys", "dismiss", scrollwindow, program.window,
					)

					keys.Resize(program.size)
					keys.Show()
					return
				}
			}
		}, program.window)

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

		// let's make a simple entry widget
		entry := widget.NewEntry()

		// make it like a text box
		entry.MultiLine = true

		// make sure the words wrap
		entry.Wrapping = fyne.TextWrapWord

		// set the test to be the stringified version of the entry
		entry.SetText(entries[id].String())

		// lock it down so they can't change it
		entry.Disable()

		// make a simple details scrolling container
		details := container.NewScroll(entry)

		// load up dialog with the container
		txs := dialog.NewCustom(
			"Transaction Detail", "dismiss",
			details, program.window,
		)

		txs.Resize(program.size)
		txs.Show()
	}

	txs := dialog.NewCustom("transactions", "dismiss", program.lists.transactions, program.window)
	txs.Resize(program.size)
	txs.Show()
}
func assetsList() {

	// let's just refresh the hash cache
	buildAssetHashList()

	// let's use the length of the hash cache for the number of objects to make
	program.lists.asset_list.Length = func() int { return len(program.caches.hashes) }

	// and here is the widget we'll use for each item in the list
	program.lists.asset_list.CreateItem = func() fyne.CanvasObject { return widget.NewLabel("") }

	// and now here is how we want each item updated
	program.lists.asset_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {

		// here is the asset details
		asset := program.caches.hashes[lii]

		// now we'll check the balance of the asset against the address
		bal, _ := program.wallet.Get_Balance_scid(asset)

		// here is the label for the list
		label := truncator(asset.String()) + " " + rpc.FormatMoney(bal)
		co.(*widget.Label).SetText(label)
	}

	// when we select an item from the list, here's what we are going to do
	program.lists.asset_list.OnSelected = func(id widget.ListItemID) {

		// here is the asset hash
		asset := program.caches.hashes[id]

		// let's get the entries
		entries := program.wallet.GetAccount().EntriesNative[asset]

		// now let's get the scid as a string
		scid := program.caches.hashes[id].String()

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
			label := time_stamp + " " + tx_type

			// and let's set the text for it
			co.(*widget.Label).SetText(label)
		}

		// so now, when we select an entry on the list we'll show the transfer details
		entries_list.OnSelected = func(id widget.ListItemID) {

			// we'll use a new entry
			e := widget.NewEntry()

			// make it like a block of text
			e.MultiLine = true

			// wrap them words
			e.Wrapping = fyne.TextWrapWord

			// lock it down
			e.Disable()

			// we'll set this transfer to the entry
			e.SetText("SCID: " + scid + "\n" + entries[id].String())

			// we'll truncate the scid for the title
			title := truncator(scid)

			// now set it, resize it and show it
			entry := dialog.NewCustom(title, "dismiss", e, program.window)
			entry.Resize(program.size)
			entry.Show()
		}

		// we'll use the truncated scid as the header for the transfers
		title := truncator(scid) + "\ntransfer history"

		// set the entries in the dialog, resize and show
		transfers := dialog.NewCustom(title, "dismiss", entries_list, program.window)
		transfers.Resize(program.size)
		transfers.Show()
	}

	// for good measure, we'll refresh the list
	program.lists.asset_list.Refresh()

	// let's set the asset list into a new scroll
	scroll := container.NewScroll(program.lists.asset_list)

	// and we'll set the scroll into a new dialog, resize and show
	collection := dialog.NewCustom("Collectibles", "dismiss", scroll, program.window)
	collection.Resize(program.size)
	collection.Show()
}
