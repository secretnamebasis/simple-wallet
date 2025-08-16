package main

import (
	"fmt"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
)

var new_account *dialog.CustomDialog

func create() {
	program.dialogues.login.Dismiss()

	if program.entries.wallet.Text != "" || program.entries.pass.Text != "" {
		program.entries.wallet.SetText("")
		program.entries.pass.SetText("")
	}

	new_account = dialog.NewCustom("Create Wallet", "dismiss",
		container.NewVBox(
			layout.NewSpacer(),
			program.entries.wallet,
			program.entries.pass,
			program.hyperlinks.save,
			layout.NewSpacer(),
		), program.window)

	program.hyperlinks.save.Alignment = fyne.TextAlignCenter
	program.hyperlinks.save.OnTapped = save

	new_account.Resize(program.size)
	new_account.Show()
}

func save() {
	var err error
	program.wallet, err = walletapi.Create_Encrypted_Wallet(
		program.entries.wallet.Text,
		program.entries.pass.Text,
		crypto.RandomScalarBNRed(),
	)
	if err != nil {
		showError(err)
		program.entries.wallet.SetText("")
		program.entries.pass.SetText("")
	} else {
		if err := program.wallet.Save_Wallet(); err != nil {
			showError(err)
			program.entries.wallet.SetText("")
			program.entries.pass.SetText("")
			return
		} else {
			loggedIn()
			keys := dialog.NewCustom("Wallet Created", "dismiss", container.NewScroll(
				container.NewVBox(
					program.labels.seed,
					program.entries.seed,
					program.labels.public,
					program.entries.public,
					program.labels.secret,
					program.entries.secret,
				),
			), program.window)
			keys.Resize(program.size)
			keys.Show()
			new_account.Dismiss()
			updateHeader(program.hyperlinks.home)
			program.window.SetContent(program.containers.home)
			program.entries.wallet.SetText("")
			program.entries.pass.SetText("")
		}
	}
}
func register() *fyne.Container {
	// let's make a registration button
	icon := theme.UploadIcon()
	program.buttons.register = widget.NewButtonWithIcon("register", icon, nil)

	// here is what happens when we push the register button...
	program.buttons.register.OnTapped = registration

	// and here is the simple registration container
	return container.NewVBox(
		layout.NewSpacer(),
		program.activities.registration,
		program.labels.counter,
		program.buttons.register,
		layout.NewSpacer(),
	)
}
func registration() {
	// we are going to do this as a go routine so the gui doesn't lock up
	go func() {
		// there is a registration transaction
		var reg_tx = new(transaction.Transaction)

		// there is a channel of success
		var success = make(chan *transaction.Transaction)

		// we are going to track wins and fails
		var wins, fails uint64

		// while in the go routine, update the widget accordingly
		fyne.DoAndWait(func() {
			program.buttons.register.Hide()
			program.labels.counter.Show()
			program.activities.registration.Start()
			program.activities.registration.Show()
		})

		// we are going to use the max number of threads
		desired_threads := runtime.GOMAXPROCS(0)

		// we estimate that roughly 21M hashes have to be attempted,
		// it is usually less... like 7-14M
		estimate := int64(21000000)

		// we are going to set some expectations
		msg := `
Registration POW takes time 20min-2hrs...
If on battery, plug your computer in.
Please do not leave this page.
` // now, they can leave the page... but they shouldn't.

		// update the label in the go routine
		fyne.DoAndWait(func() {
			program.labels.counter.Alignment = fyne.TextAlignCenter
			program.labels.counter.SetText(msg)
		})

		// for each thread
		for range desired_threads {

			// start another go routine
			go func() {

				// for as long as we haven't won, we will persist
				for wins == 0 {

					// if the wallet isn't present or is registered... stop
					if program.wallet == nil || program.wallet.IsRegistered() {
						break
					}

					// start by making an attempt
					attempt := program.wallet.GetRegistrationTX()

					// get the hash
					hash := attempt.GetHash()

					// you win if the first 3 bytes of the hash are 0
					winner := hash[0] == 0 && hash[1] == 0 && hash[2] == 0
					if winner { // if we have a winner

						// pass the attempt down the success channel
						success <- attempt

						// record a win
						wins++

						// team break
						break
					} else {

						// decrement the estimate by 1
						estimate--

						// increment the fails by 1
						fails++

						// truncate the hash
						truncated := truncator(hash.String())

						// can't do this in the app...
						// rendering too slow...
						// show a speed gauge
						fmt.Printf(
							"\rThreads: %d "+
								"HASH: %s "+
								"ESTIMATED: %d "+
								"COUNTER: %d "+
								"Success: %d",
							desired_threads, truncated, estimate, fails, wins,
						)
					}

				}
			}()
		}

		// seeing as we have a successful attempt coming down the channel
		reg_tx = <-success

		// update the widgets in the go routine
		fyne.DoAndWait(func() {
			program.activities.registration.Stop()
			program.activities.registration.Hide()

			// change the counter
			program.labels.counter.SetText(
				"Sending Registration Tx: " + truncator(reg_tx.GetHash().String()),
			)
		})

		// ship the registration transaction over the network
		if err := program.wallet.SendTransaction(reg_tx); err != nil {
			// if an error, show it
			showError(err)
			return
		} else {
			// if successful, shout for joy!
			dialog.ShowInformation("Registration", "registration successful", program.window)

			// update the display in the go routine
			fyne.DoAndWait(func() {
				program.containers.send.Show()
				program.containers.register.Hide()
			})
			return
		}
	}()
}
