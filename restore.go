package main

import (
	"bytes"
	"encoding/hex"
	"errors"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
)

var restore_wallet *dialog.ConfirmDialog

func restore() {
	// let's open these up so that we can fill them in
	program.entries.seed.Enable()
	program.entries.secret.Enable()

	// let's make sure the login splash is dismissed
	program.dialogues.login.Dismiss()

	// let's make some placeholders to make things easier to understand
	program.entries.seed.SetPlaceHolder("seed phrase goes here... ")
	program.entries.secret.SetPlaceHolder("secret hex string goes here...")
	program.entries.pass.SetPlaceHolder("N3w-w41137-p@$$")

	// we'll use this restore link to preform a restoration
	restore := widget.NewHyperlink("restore", nil)

	restore.Alignment = fyne.TextAlignCenter

	// here is a simple way do a wallet restoration
	restore.OnTapped = restoration
	or := widget.NewLabel("OR")
	or.Alignment = fyne.TextAlignCenter
	// make some content
	content := container.NewVBox(
		program.entries.seed, or, program.entries.secret,
		layout.NewSpacer(),
		program.entries.pass,
		layout.NewSpacer(),
	)
	// let's have our selves a restore wallet dialog
	callback := func(b bool) {
		if !b {
			return
		}
		restore_wallet.Dismiss()
		restoration()
	}
	restore_wallet = dialog.NewCustomConfirm("Restore Wallet", "restore", dismiss, content, callback, program.window)

	// if they press enter... it's as if they clicked restore
	program.entries.pass.OnSubmitted = func(s string) { callback(true) }
	restore_wallet.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/2))
	restore_wallet.Show()
}

func restoration() {
	// there are two ways that a user can fill in their wallet

	// as a seed phrase
	isSeed := (program.entries.seed.Text != "")

	// or as a secret hex key
	isSecret := (program.entries.secret.Text != "")

	// get the password
	pass := program.entries.pass.Text

	// dump the password
	program.entries.pass.SetText("")

	// let's assume an error is possible here
	var err error

	// when using a seed
	if isSeed {

		// get the seed
		seed := program.entries.seed.Text

		// dump the seed entry... don't worry we put it back in the loggedIn
		program.entries.seed.SetText("")

		// attempt to create a wallet using the seed
		fp := filepath.Join(globals.GetDataDirectory(), "wallet.db")
		program.wallet, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(fp, pass, seed)

		// if it screws up, please show an error
		if err != nil {
			showError(err, program.window)
			return
		}

		// attempt to save the wallet
		if err = program.wallet.Save_Wallet(); err != nil {
			// if it screws up, please show an error
			showError(err, program.window)
			return
		}

		// proceed with log in stuff
		loggedIn()

		// and dismiss the splash screen
		restore_wallet.Dismiss()

	} else if isSecret { // in the event that they use a secret hex key

		var network string
		if program.sliders.network.Value == 0.1337 {
			// use the base network
			network = "0"
		}

		// get the secret
		secret := program.entries.secret.Text

		// dump the entry's text
		program.entries.secret.SetText("")

		// build a string using the network and the secret key
		s := network + secret // ;)

		// decode the string
		snb, err := hex.DecodeString(s)
		if len(s) >= 65 || err != nil {
			showError(err, program.window)
			return
		}

		// now create the wallet
		fp := filepath.Join(globals.GetDataDirectory(), "wallet.db")
		program.wallet, err = walletapi.Create_Encrypted_Wallet(fp, pass,
			// make a big number from the secret key bytes
			new(crypto.BNRed).SetBytes(snb),
		)
		// show the error if there is one
		if err != nil {
			showError(err, program.window)
			return
		}
		// save the wallet
		if err := program.wallet.Save_Wallet(); err != nil {
			// show the error if there is one
			showError(err, program.window)
			return
		}
		// give the user some feedback
		showInfo("Restore Wallet", "successfully saved as wallet.db", program.window)
		program.wallet.SetNetwork(true)
		program.wallet.SetOnlineMode()
		program.wallet.SyncHistory(crypto.ZEROHASH)
		logger.Info("sync history", "status", "initiated")
		if program.wallet.IsRegistered() {
			time.Sleep(time.Second * 1)
		}

		// proceed with the usuals
		loggedIn()

		// dismiss the splash
		restore_wallet.Dismiss()
	} else if isSeed && isSecret { // there is the off chance they do both

		// let's create the wallet from the seed
		fp := filepath.Join(globals.GetDataDirectory(), "wallet.db")
		seed, err := walletapi.Create_Encrypted_Wallet_From_Recovery_Words(fp, pass, program.entries.seed.Text)
		// if there is an error show it
		if err != nil {
			showError(err, program.window)
			return
		}

		var network string
		if program.sliders.network.Value == 0.1337 {
			// use the base network
			network = "0"
		}

		// get the secret hex key
		secret := program.entries.secret.Text

		// dump the key from the entry
		program.entries.secret.SetText("")

		// build a string
		s := network + secret

		// decode the string into bytes
		snb, err := hex.DecodeString(s) // ;)
		if len(s) >= 65 || err != nil {
			showError(err, program.window)
			return
		}

		// now create the wallet from the secret key
		fp = filepath.Join(globals.GetDataDirectory(), "wallet.db")
		program.wallet, err = walletapi.Create_Encrypted_Wallet(fp, pass,
			new(crypto.BNRed).SetBytes(snb),
		)
		// show there error if there is one
		if err != nil {
			showError(err, program.window)
			return
		}

		// if the bytes of the two secrets are not equal...
		if !bytes.Equal(seed.Secret, program.wallet.Secret) {
			// show the error
			showError(errors.New("seed and secret are not the same"), program.window)
			return
		}
		// dump the seed
		seed = nil

		// save the wallet , or show an error
		if err := program.wallet.Save_Wallet(); err != nil {
			showError(err, program.window)
			return
		}

		// tell the user they have succeeded
		showInfo("Restore Wallet", "successfully saved as wallet.db", program.window)

		// do the loggedIn dance
		loggedIn()

		// close the splash
		restore_wallet.Dismiss()
	} else {
		showError(errors.New("can't restore from empty fields"), program.window)
		return
	}
}
