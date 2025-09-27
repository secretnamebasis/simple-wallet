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

func create() {
	program.dialogues.login.Dismiss()
	pass := widget.NewPasswordEntry()
	if program.entries.wallet.Text != "" || pass.Text != "" {
		program.entries.wallet.SetText("")
	}

	pass.SetText(randomWords(4, "-"))

	content := container.NewVBox(
		layout.NewSpacer(),
		program.entries.wallet,
		pass,
		program.hyperlinks.save,
		layout.NewSpacer(),
	)

	new_account := dialog.NewCustom("Create Wallet", dismiss,
		content, program.window)

	// if they press enter, it is as if they pressed save
	pass.OnSubmitted = func(s string) {
		new_account.Dismiss()
		create_account(s)
		// dump entries
		program.entries.wallet.SetText("")
		pass.SetText("")
	}

	program.hyperlinks.save.Alignment = fyne.TextAlignCenter
	program.hyperlinks.save.OnTapped = func() {
		create_account(pass.Text)
		// dump entries
		program.entries.wallet.SetText("")
		pass.SetText("")
		new_account.Dismiss()

	}

	new_account.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/3))
	new_account.SetOnClosed(func() {
		// don't want to write over every password field
		pass.SetText("")
	})
	new_account.Show()
}

func create_account(password string) {
	var err error
	// get entries
	filename := program.entries.wallet.Text

	program.wallet, err = walletapi.Create_Encrypted_Wallet(
		filename,
		password,
		crypto.RandomScalarBNRed(),
	)
	if err != nil {
		showError(err, program.window)

	} else {
		// now save the wallet
		if err := program.wallet.Save_Wallet(); err != nil {
			// if that doesn't work...
			showError(err, program.window)
			return

		} else { // follow logged in workflow
			loggedIn()
			updateHeader(program.hyperlinks.home)
			setContentAsHome()
		}
	}
}
func register() *fyne.Container {
	// let's make a registration button
	icon := theme.UploadIcon()
	program.buttons.register = widget.NewButtonWithIcon("REGISTER", icon, nil)

	// here is what happens when we push the register button...
	program.buttons.register.OnTapped = registration

	// and here is the simple registration container
	return container.NewVBox(
		program.activities.registration,
		program.buttons.register,
		program.labels.counter,
	)
}
func registration() {
	// we are going to do this as a go routine so the gui doesn't lock up
	go func() {

		// while in the go routine, update the widget accordingly
		fyne.DoAndWait(func() {
			program.buttons.register.Hide()
			program.labels.counter.Show()
			program.activities.registration.Start()
			program.activities.registration.Show()
		})

		// we are going to set some expectations

		// there is a registration transaction
		var reg_tx = new(transaction.Transaction)

		// there is a channel of success
		var success = make(chan *transaction.Transaction)

		// we are going to track wins and fails
		var wins, fails uint64
		var os_thread, app_thread int = 1, 1
		// reserve 1 thread for os management
		// reserve 1 thread for app management

		// we are going to use almost all threads
		max_threads := runtime.GOMAXPROCS(0)
		desired_threads := max_threads - os_thread - app_thread

		// we estimate that roughly 21M hashes have to be attempted,
		// it is usually less... like 7-14M
		estimate := int64(21000000)

		// for each thread
		for range desired_threads {

			// start another go routine
			go func() {

				// for as long as we haven't won, we will persist
				for wins == 0 {

					// if the wallet isn't present or is registered... stop
					if program.wallet == nil ||
						program.wallet.IsRegistered() ||
						!program.activities.registration.Visible() {
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
			showError(err, program.window)
			return
		} else {
			// if successful, shout for joy!
			showInfo("Registration", "registration successful", program.window)

			// update the display in the go routine
			fyne.DoAndWait(func() {
				program.containers.send.Show()
				program.buttons.assets.Enable()
				program.buttons.transactions.Enable()
				program.containers.register.Hide()
			})
			return
		}
	}()
}
