package main

import (
	"errors"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
)

func send() *fyne.Container {

	program.entries.recipient.SetPlaceHolder("receiver address: dero...")

	// validate if they are requesting an address first
	validate_address := func(s string) error {
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
		if addr.Arguments.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataUint64) {
			title := "Notice"
			msg := "This address is requesting a reply back address to be sent with the transaction"
			showInfo(title, msg)
		}
		return nil
	}

	// experience has shown that the validator is very aggressive
	// don't try to do too much or it will be slow waiting for daemon calls
	program.entries.recipient.Validator = validate_address

	// build out simple options for the send action
	program.buttons.send.SetIcon(theme.MailSendIcon())
	program.buttons.send.OnTapped = sendForm

	return container.New(&twoThirds{},
		program.entries.recipient,
		program.buttons.send,
	)
}

func sendForm() {

	// obviously, we can't send to no one,
	// especially not a non-validated no one
	if program.entries.recipient.Text == "" {
		showError(errors.New("cannot send to empty address"))
		return
	}

	// get entry
	send_recipient := program.entries.recipient.Text

	// dump the entry text
	program.entries.recipient.SetText("")

	// let's validate on the send action

	// if less than 4 char...
	if len(send_recipient) < 4 {
		showError(errors.New("cannot be less than 5 char"))
		return
	}

	// any changes to the string should immediately update the receiver string
	// program.receiver = ""

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

		// now if the address is an integrated address...
	} else if addr.IsIntegratedAddress() {

		// the base of that address is what we'll use as the receiver
		program.receiver = addr.BaseAddress().String()

		// while we are here... let's process the address's arguments

		// if they are asking for a replyback, notify the user
		if addr.Arguments.Has(rpc.RPC_NEEDS_REPLYBACK_ADDRESS, rpc.DataUint64) {
			program.checks.replyback.SetChecked(true)
			program.checks.replyback.Disable()
		}

		// if they are asking for a specific asset, let's use it
		if addr.Arguments.Has(rpc.RPC_ASSET, rpc.DataHash) {
			// obtain the value interface
			value := addr.Arguments.Value(rpc.RPC_ASSET, rpc.DataHash)

			// coerce the value into a crypto hash
			asset := value.(crypto.Hash)

			// stringify the hash
			asset_string := asset.String()

			// set the asset string as the selection
			program.selections.assets.SetSelected(asset_string)

			// disable the widget to prevent error
			program.selections.assets.Disable()
		}

		// if they have a specific port... use it
		if addr.Arguments.Has(rpc.RPC_DESTINATION_PORT, rpc.DataUint64) {

			// obtain the value interface
			value := addr.Arguments.Value(rpc.RPC_DESTINATION_PORT, rpc.DataUint64)

			// coerce the value into a uint64
			port := value.(uint64)

			// convert the port into a string
			port_string := strconv.Itoa(int(port))

			// set string to the widget
			program.entries.dst.SetText(port_string)

			// lock down the widget to prevent error
			program.entries.dst.Disable()
		}

		// if the addr has a comment...
		if addr.Arguments.Has(rpc.RPC_COMMENT, rpc.DataString) {
			// obtain the value
			value := addr.Arguments.Value(rpc.RPC_COMMENT, rpc.DataString)

			// coerce the value to string
			comment := value.(string)

			// set the comment to the widget
			program.entries.comment.SetText(comment)

			// disable the widget to prevent error
			program.entries.comment.Disable()
		}

		// and if the addr has a specific amount...
		if addr.Arguments.Has(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64) {

			// obtain the value
			value := addr.Arguments.Value(rpc.RPC_VALUE_TRANSFER, rpc.DataUint64)

			// coerce the value into uint64
			amount := value.(uint64)

			// stringify the amount
			formatted := rpc.FormatMoney(amount)

			// set the formatted amount to the widget
			program.entries.amount.SetText(formatted)

			// lock the widget to prevent error
			program.entries.amount.Disable()
		}
	} else if addr.String() != "" && !addr.IsIntegratedAddress() {
		// if the addr isn't empty
		// now if it is not an integrated address

		// set the receiver
		program.receiver = addr.String()
	}

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

	// should be validated

	// but, we also can't send if DERO is under 80 deri
	// so we should create a reminder for when they try

	// obtain their balance
	bal := program.wallet.GetAccount().Balance_Mature
	// and check
	if bal < 80 {
		showError(errors.New("balance is too low, please refill wallet"))
		return
	}

	// set place holders
	program.entries.amount.SetPlaceHolder("816.80085")
	program.entries.dst.SetPlaceHolder("dst port")

	// they can't send incorrect amounts
	program.entries.amount.Validator = func(s string) error {

		// parse the float
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return err
		}
		// coerce it into uint64 as atomic units
		a := uint64(f * atomic_units)

		// compare it against the balance
		if a > bal {
			return errors.New(
				"cannot send more than balance " + rpc.FormatMoney(bal),
			)
		}
		return nil
	}

	// let them know they can't send more than like... 100 char
	notice := "optional comment\n"
	notice += `max 100 characters`

	// set the notice
	program.entries.comment.SetPlaceHolder(notice)

	// as a text block
	program.entries.comment.MultiLine = true

	// wrap the word
	program.entries.comment.Wrapping = fyne.TextWrapWord

	// and make sure that it is less than 100 char
	program.entries.comment.Validator = func(s string) error {
		if len(s) > 100 {
			return errors.New("100 character limit")
		}
		return nil
	}

	// let's make a list of assets on hand
	var scids []string

	// refresh the hash list
	buildAssetHashList()

	// iterate over the hash list for each
	for _, asset := range program.caches.assets {

		// dynamically obtain the asset balance for each hash
		asset_balance := func() string {

			// obtain the balance of the hash
			bal, _ := program.wallet.Get_Balance_scid(
				crypto.HashHexToHash(asset.hash),
			)

			// return a formatted string of the balance
			return rpc.FormatMoney(bal)
		}

		// build a label for each hash with its balance
		label := truncator(asset.hash) + " " + asset_balance()

		// append them to the scids
		scids = append(scids, label)
	}

	// load up the scids and asset options in the selector
	program.selections.assets.Options = scids

	// create a placeholder that makes sense
	program.selections.assets.PlaceHolder = "DERO by default, or select asset"

	// create a optionals dialog
	send := dialog.NewForm("Send DERO", "Send", "Cancel",
		[]*widget.FormItem{

			// show them who they are sending to
			widget.NewFormItem("Recipient", widget.NewLabel(truncator(program.receiver))),

			// let them make a way to enter an amount
			widget.NewFormItem("Amount", program.entries.amount),

			// now use an accordion for the "advanced" options
			widget.NewFormItem("", widget.NewAccordion(
				widget.NewAccordionItem("Options", container.NewVBox(
					program.selections.assets,
					program.entries.dst,
					program.entries.comment,
					program.checks.replyback,
				)),
			)),
		}, func(b bool) {

			if !b { // if they choose to send
				program.entries.amount.SetText("")
				program.entries.dst.SetText("")
				program.entries.comment.SetText("")
				program.selections.assets.SetSelectedIndex(-1)
				// clear recipient on send action
				program.entries.recipient.SetText("")
				program.checks.replyback.SetChecked(false)
				enableOptionals()

				return
			}
			conductTransfer() // ride it to the end

			// now let's make sure each of these are re-enabled
			enableOptionals()

			// fit it into the window
		}, program.window)

	// resize and show
	send.Resize(program.size)
	send.Show()
}

func conductTransfer() {
	var t *dialog.FormDialog
	program.entries.pass.OnSubmitted = func(s string) {
		t.Submit()
		t.Dismiss()
	}
	// make them confirm with password
	t = dialog.NewForm("Password Confirmation", confirm, "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("", program.entries.pass),
		},
		func(b bool) {

			// get the pass
			pass := program.entries.pass.Text

			// dump the pass
			program.entries.pass.SetText("")

			passed := program.wallet.Check_Password(pass)

			// if they get the password wrong
			// in case they cancel
			if !b {

				// clear recipient on send action
				program.entries.recipient.SetText("")
				// dump all the data of the transfer, and reset

				program.entries.amount.SetText("")
				program.entries.dst.SetText("")
				program.entries.comment.SetText("")
				program.selections.assets.SetSelectedIndex(-1)

				return
			} else if !passed {
				// show them that
				showError(errors.New("wrong password"))
				return
			}

			// let's start with a set of arguments
			args := rpc.Arguments{}

			// and let's get that amount
			amnt := program.entries.amount.Text

			// dump the amount text
			program.entries.amount.SetText("")

			// let's parse that amount so that we can work with it
			flo, err := strconv.ParseFloat(amnt, 64)
			if err != nil {
				showError(err)
			}

			// now let's coerce it into a dero atomic units
			value := uint64(flo * atomic_units) // atomic

			// now let's go get the destination port
			dst := program.entries.dst.Text

			// now dump the text
			program.entries.dst.SetText("")

			// if it is not empty...
			if dst != "" {

				// parse the string to uint
				dst_uint, err := strconv.ParseUint(dst, 10, 64)
				if err != nil {
					showError(err)
				}

				// and let's append the dst as uint64 to our arguments
				args = append(args, rpc.Argument{
					Name:     rpc.RPC_DESTINATION_PORT,
					DataType: rpc.DataUint64,
					Value:    dst_uint,
				})
			}

			// again, with the copy the comment
			comment := program.entries.comment.Text

			// dump the text from memory
			program.entries.comment.SetText("")

			// if it isn't empty...
			if comment != "" {
				// append the comment onto the arguments
				args = append(args, rpc.Argument{
					Name:     rpc.RPC_COMMENT,
					DataType: rpc.DataString,
					Value:    comment,
				})
			}

			reply := program.checks.replyback.Checked
			program.checks.replyback.SetChecked(false)

			// then...
			if reply {

				// append the wallet address to the arguments
				args = append(args, rpc.Argument{
					Name:     rpc.RPC_REPLYBACK_ADDRESS,
					DataType: rpc.DataAddress,
					Value:    program.wallet.GetAddress(),
				})
			}

			// before we ship it, check the pack to make sure it can go
			if _, err := args.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {

				// show error if it can't
				showError(err)
				return
			}

			// let's get the reciever
			destination := program.receiver

			// dump the receiver
			// clear recipient on send action
			program.entries.recipient.SetText("")

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

			// the scid is defaulted to DERO
			scid := crypto.ZEROHASH

			// let's check if they selected another asset to send

			// first let's get the current selected index
			index := program.selections.assets.SelectedIndex()

			// dump the selection
			program.selections.assets.SetSelectedIndex(-1)

			// If they selected something, it is an asset
			isAsset := index > -1

			if isAsset {
				// thusly, well get the asset from the cache
				asset := program.caches.assets[index]

				// and now the scid is the asset
				scid = crypto.HashHexToHash(asset.hash)
			}

			// now, let's build the payload
			payload := []rpc.Transfer{
				{
					SCID:        scid,
					Destination: destination,
					Amount:      value,
					Payload_RPC: args, // if any
				},
			}

			notice := makeCenteredWrappedLabel("dispatching transfer.... should recieve txid soon")
			sync := widget.NewActivity()

			sync.Start()
			// set it to a new dialog screen and show
			transaction := dialog.NewCustomWithoutButtons(
				"Transaction Dispatched",
				container.NewVBox(notice, container.NewCenter(sync)), program.window,
			)
			transaction.Resize(program.size)
			transaction.Show()
			var retries int
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
							sync.Stop()
							transaction.Dismiss()
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
						sync.Stop()
						transaction.Dismiss()
						// set it to a new dialog screen and show
						dialog.ShowCustom(
							"Transaction Dispatched", "dismissed",
							container.NewVBox(txid), program.window,
						)
					})
				}
			}()

			// and now fit it into the window
		}, program.window)

	// resize and show
	t.Resize(program.size)
	t.Show()
}

func enableOptionals() {
	if program.entries.amount.Disabled() {
		program.entries.amount.Enable()
	}
	if program.entries.dst.Disabled() {
		program.entries.dst.Enable()
	}
	if program.entries.comment.Disabled() {
		program.entries.comment.Enable()
	}
	if program.checks.replyback.Disabled() {
		program.checks.replyback.Enable()
	}
}
