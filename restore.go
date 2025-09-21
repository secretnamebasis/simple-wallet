package main

import (
	"bytes"
	"encoding/hex"
	"errors"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/walletapi"
)

var restore_wallet *dialog.CustomDialog

func restore() {
	// let's open these up so that we can fill them in
	program.entries.seed.Enable()
	program.entries.secret.Enable()

	// let's make sure the login splash is dismissed
	program.dialogues.login.Dismiss()

	// let's make some placeholders to make things easier to understand
	program.entries.seed.SetPlaceHolder("seed phrase goes here... ")
	program.entries.secret.SetPlaceHolder("secret hex string goes here...")

	// we'll use this restore link to preform a restoration
	restore := widget.NewHyperlink("restore", nil)

	restore.Alignment = fyne.TextAlignCenter

	// here is a simple way do a wallet restoration
	restore.OnTapped = restoration

	// make some content
	content := container.NewVBox(
		program.entries.seed,
		widget.NewLabel("OR"),
		program.entries.secret,
		widget.NewLabel("New Password"),
		program.entries.pass,
		restore,
	)
	// let's have our selves a restore wallet dialog
	restore_wallet = dialog.NewCustom("Restore Wallet", dismiss, content, program.window)

	// if they press enter... it's as if they clicked restore
	program.entries.pass.OnSubmitted = func(s string) {
		restore_wallet.Dismiss()
		restoration()
	}
	restore_wallet.Resize(program.size)
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
		program.wallet, err = walletapi.Create_Encrypted_Wallet_From_Recovery_Words(
			"wallet.db", pass, seed,
		)

		// if it screws up, please show an error
		if err != nil {
			showError(err)
			return
		}

		// attempt to save the wallet
		if err = program.wallet.Save_Wallet(); err != nil {
			// if it screws up, please show an error
			showError(err)
			return
		}

		// proceed with log in stuff
		loggedIn()

		// and dismiss the splash screen
		restore_wallet.Dismiss()

	} else if isSecret { // in the event that they use a secret hex key

		var network string
		if program.toggles.network.Selected == "mainnet" {
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
			showError(err)
			return
		}

		// now create the wallet
		program.wallet, err = walletapi.Create_Encrypted_Wallet(
			"wallet.db", pass,
			// make a big number from the secret key bytes
			new(crypto.BNRed).SetBytes(snb),
		)
		// show the error if there is one
		if err != nil {
			showError(err)
			return
		}
		// save the wallet
		if err := program.wallet.Save_Wallet(); err != nil {
			// show the error if there is one
			showError(err)
			return
		}
		// give the user some feedback
		showInfo("Restore Wallet", "successfully saved as wallet.db")

		// proceed with the usuals
		loggedIn()

		// dismiss the splash
		restore_wallet.Dismiss()
	} else if isSeed && isSecret { // there is the off chance they do both

		// let's create the wallet from the seed
		seed, err := walletapi.Create_Encrypted_Wallet_From_Recovery_Words(
			"wallet.db", pass, program.entries.seed.Text,
		)
		// if there is an error show it
		if err != nil {
			showError(err)
			return
		}

		var network string
		if program.toggles.network.Selected == "mainnet" {
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
			showError(err)
			return
		}

		// now create the wallet from the secret key
		program.wallet, err = walletapi.Create_Encrypted_Wallet(
			"wallet.db", pass,
			new(crypto.BNRed).SetBytes(snb),
		)
		// show there error if there is one
		if err != nil {
			showError(err)
			return
		}

		// if the bytes of the two secrets are not equal...
		if !bytes.Equal(seed.Secret, program.wallet.Secret) {
			// show the error
			showError(errors.New("seed and secret are not the same"))
			return
		}
		// dump the seed
		_ = seed

		// save the wallet , or show an error
		if err := program.wallet.Save_Wallet(); err != nil {
			showError(err)
			return
		}

		// tell the user they have succeeded
		showInfo("Restore Wallet", "successfully saved as wallet.db")

		// do the loggedIn dance
		loggedIn()

		// close the splash
		restore_wallet.Dismiss()
	} else {
		showError(errors.New("can't restore from empty fields"))
		return
	}
}
