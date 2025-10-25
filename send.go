package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

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
			showInfo(title, msg, program.window)
		}
		return nil
	}

	// experience has shown that the validator is very aggressive
	// don't try to do too much or it will be slow waiting for daemon calls
	program.entries.recipient.Validator = validate_address
	btn := widget.NewButtonWithIcon("", theme.MailSendIcon(), sendForm)
	btn.IconPlacement = widget.ButtonIconTrailingText
	btn.Importance = widget.HighImportance

	program.entries.recipient.ActionItem = btn

	program.entries.recipient.OnSubmitted = func(s string) {
		sendForm()
	}
	return container.NewStack(program.entries.recipient)
}

func sendForm() {

	// obviously, we can't send to no one,
	// especially not a non-validated no one
	if program.entries.recipient.Text == "" {
		showError(errors.New("cannot send to empty address"), program.window)
		return
	}

	// get entry
	send_recipient := program.entries.recipient.Text

	// dump the entry text
	program.entries.recipient.SetText("")

	// let's validate on the send action

	// if less than 4 char...
	if len(send_recipient) < 4 {
		showError(errors.New("cannot be less than 5 char"), program.window)
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
				showError(errors.New("cannot send to self"), program.window)
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
		// if addr.Arguments.Has(rpc.Rpc, rpc.DataHash) {
		// 	showError(errors.New("currently unsupported"), program.window)
		// 	return
		// obtain the value interface
		// value := addr.Arguments.Value(rpc.RPC_ASSET, rpc.DataHash)

		// coerce the value into a crypto hash
		// asset := value.(crypto.Hash)

		// stringify the hash
		// asset_string := asset.String()

		// // set the asset string as the selection
		// program.selections.assets.SetSelected(asset_string)

		// // disable the widget to prevent error
		// program.selections.assets.Disable()
		// }

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

	// should be validated

	// but, we also can't send if DERO is under 80 deri
	// so we should create a reminder for when they try

	// obtain their balance
	bal := program.wallet.GetAccount().Balance_Mature
	// and check
	if bal < 80 {
		showError(errors.New("balance is too low, please refill wallet"), program.window)
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
	// create callback function
	callback := func(b bool) {

		if !b { // if they choose to send
			program.entries.amount.SetText("")
			program.entries.dst.SetText("")
			program.entries.comment.SetText("")
			// program.selections.assets.SetSelectedIndex(-1)
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
	}
	// create a content container
	content := []*widget.FormItem{

		// show them who they are sending to
		widget.NewFormItem("Recipient", widget.NewLabel(truncator(program.receiver))),

		// let them make a way to enter an amount
		widget.NewFormItem("Amount", program.entries.amount),

		// now use an accordion for the "advanced" options
		widget.NewFormItem("", widget.NewAccordion(
			widget.NewAccordionItem("Options", container.NewVBox(
				// program.selections.assets,
				program.entries.dst,
				program.entries.comment,
				program.checks.replyback,
			)),
		)),
	}
	// create a optionals dialog
	send := dialog.NewForm("Send DERO", "Send", "Cancel", content, callback, program.window)

	// resize and show
	send.Resize(program.size.Subtract(fyne.Size{
		Width:  program.size.Width / 4,
		Height: program.size.Height / 4,
	}))
	send.Show()
}

func conductTransfer() {
	var t *dialog.FormDialog
	program.entries.pass.OnSubmitted = func(s string) {
		t.Submit()
		t.Dismiss()
	}
	callback := func(b bool) {

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
			// program.selections.assets.SetSelectedIndex(-1)

			return
		} else if !passed {
			// show them that
			showError(errors.New("wrong password"), program.window)
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
			showError(err, program.window)
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
				showError(err, program.window)
			}

			// and let's append the dst as uint64 to our arguments
			args = append(args, rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataUint64,
				Value:    dst_uint,
			})
		}

		// again, with the comment
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
			showError(err, program.window)
			return
		}

		// let's get the reciever
		destination := program.receiver

		// dump the receiver
		program.receiver = ""

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
		// create content container
		content := container.NewVBox(notice, container.NewCenter(sync))
		// set it to a new dialog screen and show
		transact := dialog.NewCustomWithoutButtons("Transaction Dispatched", content, program.window)
		// transact.Resize(program.size)
		transact.Show()
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
				showError(err, program.window)
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
						sync.Stop()
						transact.Dismiss()
					})
				}
				// and return

			} else { // if we have success
				start := time.Now().Add(time.Second * 600)
				for searching := range time.NewTicker(time.Second * 2).C {
					if searching.After(start) {
						sync.Stop()
						transact.Dismiss()
						showError(errors.New("manually confirm transfer"), program.window)
						return
					}
					if len(program.caches.pool.Tx_list) > 0 {

						fmt.Println("searching", searching.After(start), searching.String(), program.caches.pool.Tx_list)

						if slices.Contains(program.caches.pool.Tx_list, tx.GetHash().String()) {
							in_pool := time.Now().Add(time.Second * 600)
							for on_chain := range time.NewTicker(time.Second * 2).C {
								if on_chain.After(in_pool) {
									sync.Stop()
									transact.Dismiss()
									showError(errors.New("manually confirm transfer"), program.window)
									return
								}

								result := getTransaction(rpc.GetTransaction_Params{
									Tx_Hashes: []string{tx.GetHash().String()},
								})
								for _, each := range result.Txs_as_hex {
									b, _ := hex.DecodeString(each)
									var tr transaction.Transaction
									tr.Deserialize(b)
									t := tr.GetHash().String()
									if strings.Contains(t, tx.GetHash().String()) {
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
											sync.Stop()
											transact.Dismiss()
											// set it to a new dialog screen and show
											dialog.ShowCustom(
												"Transaction Dispatched", "dismissed",
												container.NewVBox(txid), program.window,
											)
										})
										return
									}
								}
							}
						}
					}
				}
			}
		}()

		// and now fit it into the window
	}

	// make a simple form
	content := []*widget.FormItem{widget.NewFormItem("", program.entries.pass)}

	// make them confirm with password
	t = dialog.NewForm("Password Confirmation", confirm, "Cancel", content, callback, program.window)

	// resize and show
	t.Resize(password_size)
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
