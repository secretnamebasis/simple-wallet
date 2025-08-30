// this is the largest part of the repo, and the most fun.

package main

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
)

func tools() *fyne.Container {

	// let's do this when we click on tool
	program.hyperlinks.tools.OnTapped = func() {
		updateHeader(program.hyperlinks.tools)
		program.window.SetContent(program.containers.tools)
	}

	// let's set the functions for each of these links
	program.buttons.filesign.OnTapped = filesign
	program.buttons.self_encrypt_decrypt.OnTapped = self_crypt
	program.buttons.recipient_encrypt_decrypt.OnTapped = recipient_crypt
	program.buttons.integrated.OnTapped = integrated_address_generator
	program.buttons.balance_rescan.OnTapped = balance_rescan
	program.buttons.contract_installer.OnTapped = installer
	program.buttons.contract_interactor.OnTapped = interaction
	program.buttons.token_add.OnTapped = add_token

	// and then set them
	toolButtons := []fyne.CanvasObject{
		container.NewVBox(program.buttons.filesign),
		container.NewVBox(program.buttons.integrated),
		container.NewVBox(program.buttons.self_encrypt_decrypt),
		container.NewVBox(program.buttons.recipient_encrypt_decrypt),
		container.NewVBox(program.buttons.token_add),
		container.NewVBox(program.buttons.balance_rescan),
		container.NewVBox(program.buttons.contract_installer),
		container.NewVBox(program.buttons.contract_interactor),
	}

	// Wrap them in a responsive container
	program.containers.toolbox = container.New(&responsiveGrid{},
		toolButtons...,
	)

	// and now, let's hide them
	program.containers.toolbox.Hide()

	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		program.containers.toolbox,
		layout.NewSpacer(),
		program.containers.bottombar,
	)

}

// this is a pretty under-rated feature
func filesign() {

	// let's make it noticeable that you can select the file
	program.entries.file.entry.SetPlaceHolder("/path/to/file.txt")

	// now let's make a sign hyperlink
	sign := widget.NewHyperlink("filesign", nil)

	// and when the user taps it
	sign.OnTapped = func() {
		var fs *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			fs.Submit()
			fs.Dismiss()
		}
		fs = dialog.NewForm("Sign File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
			func(b bool) {
				// if they cancel
				if !b {
					return
				}

				// copy the password
				pass := program.entries.pass.Text

				// dump the text
				program.entries.pass.SetText("")

				// check the password first
				if !program.wallet.Check_Password(pass) {

					// let them know if they were wrong
					showError(errors.New("wrong password"))

					// dump the filepath
					program.entries.file.entry.SetText("")
					return
				} else {
					// get the filename
					filename := program.entries.file.entry.Text

					//dump the entry
					program.entries.file.entry.SetText("")

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err)
						return
					}

					// sign the file into bytes
					data := program.wallet.SignData(file)

					// it is possible to sign data as an unregistered user
					if !isRegistered(program.wallet.GetAddress().String()) {
						notice := "you have signed a file as an unregistered user"
						// notify the user, but continue anyway
						showInfo("NOTICE", notice)
					}

					// make a filename
					save_path := filename + ".signed"

					// write the file to disc
					os.WriteFile(save_path, data, default_file_permissions)

					// notify the user
					showInfo("Filesign", "File successfully signed")

				}
				// display to main window
			}, program.window)
		fs.Show()
	}

	// we are going to do the same thing, but the reverse direction

	// make a link to verify a file
	verify := widget.NewHyperlink("fileverify", nil)

	// when they click the link
	verify.OnTapped = func() {
		var v *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			v.Submit()
			v.Dismiss()
		}
		v = dialog.NewForm("Verify File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
			func(b bool) {
				// if they cancel
				if !b {
					return
				}

				// copy the password
				pass := program.entries.pass.Text

				// dump the text
				program.entries.pass.SetText("")

				// check the password, every time
				if !program.wallet.Check_Password(pass) {

					// show and error when wrong
					showError(errors.New("wrong password"))

					//dump entry
					program.entries.file.entry.SetText("")

				} else {
					// get the filename
					filename := program.entries.file.entry.Text
					program.entries.file.entry.SetText("")

					// check if the file is a .signed file
					if !strings.HasSuffix(filename, ".signed") {

						// display error
						showError(errors.New("not a .signed file"))

						return
					}

					// if everything is good so far, read the files
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err)
						return
					}

					// now parse the file to get the details
					sign, // this is the signer
						contents, // this is the contents
						err :=    // as well as an error
						program.wallet.CheckSignature(file)

					// if there is an error
					if err != nil {

						// show the user
						showError(err)
						return
					}

					// it is possible to sign data as an unregistered user
					if !isRegistered(sign.String()) {
						notice := "an unregistered user has signed this data"
						// notify the user, but continue
						showInfo("NOTICE", notice)
					}

					// now trim the .signed from the filename
					save_path := strings.TrimSuffix(filename, ".signed")

					// write the contents to disk
					os.WriteFile(save_path, contents, default_file_permissions)

					// notify the user
					notice := "File successfully verified\n" +
						"Signed by: " + truncator(sign.String()) + "\n" +
						"Message saved as " + save_path

					// load the notice into the dialog
					fv := dialog.NewInformation("FileVerify", notice, program.window)
					fv.Show()
					return
				}

				// load it into the main window
			}, program.window)
		v.Show()
	}

	// now let's make another notice
	notice := "filesign and fileverify are novel DERO features. "
	notice += "filesign allows users to sign data in a verifiable way. "
	notice += "fileverify allows users to verify data was signed by another user."

	// load the notice into a label
	label := makeCenteredWrappedLabel(notice)

	// let's load all the widgets into a container inside a dialog
	file := dialog.NewCustom("filesign/fileverify", dismiss,
		container.NewVBox(
			layout.NewSpacer(),
			container.NewVBox(program.entries.file),
			container.NewAdaptiveGrid(2,
				container.NewCenter(sign),
				container.NewCenter(verify),
			),
			label,
			layout.NewSpacer(),
		), program.window)

	//resize and show
	file.Resize(program.size)
	file.Show()
}
func self_crypt() {
	// another round of make sure this works XD
	program.entries.file.entry.SetPlaceHolder("/path/to/file.txt")

	// let's encrypt data
	encrypt := widget.NewHyperlink("encrypt", nil)

	// when the user clicks here...
	encrypt.OnTapped = func() {
		var e *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			e.Submit()
			e.Dismiss()
		}
		e = dialog.NewForm("Encrypt File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)}, func(b bool) {
				// if they cancel
				if !b {
					return
				}
				// let's get the password
				pass := program.entries.pass.Text

				//dump the entry
				program.entries.pass.SetText("")

				// check the password
				if !program.wallet.Check_Password(pass) {
					// notify them when wrong
					showError(errors.New("wrong password"))

					// dump the entry
					program.entries.file.entry.SetText("")
				} else {

					// get the filename
					filename := program.entries.file.entry.Text

					// dump the entry
					program.entries.file.entry.SetText("")

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						// display error if there is one
						showError(err)
						return
					}

					// encrypt the data
					data, err := program.wallet.Encrypt(file)
					if err != nil {
						showError(err)
						return
					}

					// made a save path
					save_path := filename + ".enc"

					// write file to disk
					os.WriteFile(save_path, data, default_file_permissions)

					// make a success notice
					notice := "File successfully encrypted\n" +
						"Message saved as " + save_path

					// load it , and show it
					e := dialog.NewInformation("Encrypt", notice, program.window)
					e.Resize(program.size)
					e.Show()
					return

				}
			}, program.window)
		e.Show()
	}

	// now let's decrypt
	decrypt := widget.NewHyperlink("decrypt", nil)

	// here's what we are going to do
	decrypt.OnTapped = func() {
		var d *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			d.Submit()
			d.Dismiss()
		}

		d = dialog.NewForm("Decrypt File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
			func(b bool) {
				// if they cancel
				if !b {
					return
				}
				// get the password
				pass := program.entries.pass.Text

				// dump the password
				program.entries.pass.SetText("")

				// check the password
				if !program.wallet.Check_Password(pass) {

					// notify the user
					showError(errors.New("wrong password"))

					// dump the file path
					program.entries.file.entry.SetText("")

				} else {

					// get the file name
					filename := program.entries.file.entry.Text

					// dump the entry
					program.entries.file.entry.SetText("")

					// check if this is an .enc file
					if !strings.HasSuffix(filename, ".enc") {

						// notify the user
						showError(errors.New("not a .enc file"))
						return
					}

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err)
						return
					}

					// decrypt the file
					data, err := program.wallet.Decrypt(file)
					if err != nil {
						showError(err)
						return
					}

					// trim the suffix off
					save_path := strings.TrimSuffix(filename, ".enc")

					// write the decrypted file
					os.WriteFile(save_path, data, default_file_permissions)

					// build a notice
					notice := "File successfully decrypted\n" +
						"Message saved as " + save_path

					// load the notice and show it
					d := dialog.NewInformation("Decrypt", notice, program.window)
					d.Resize(program.size)
					d.Show()
					return
				}
				// show decrypt in the window
			}, program.window)
		d.Show()
	}

	// let's make another notice
	notice := "DERO wallets allow users to symmetrically encrypt/decrypt information. "
	notice += "Select file to encrypt, or decrypt, for self"

	// create a label widget
	label := makeCenteredWrappedLabel(notice)

	// load the widgets and dialog
	self_crypt := dialog.NewCustom("Self Encrypt/Decrypt", dismiss,
		container.NewVBox(
			layout.NewSpacer(),
			container.NewVBox(program.entries.file),
			container.NewAdaptiveGrid(2,
				container.NewCenter(encrypt),
				container.NewCenter(decrypt),
			),
			label,
			layout.NewSpacer(),
		),
		// load it in the main window
		program.window)

	// resize and show
	self_crypt.Resize(program.size)
	self_crypt.Show()
}
func recipient_crypt() {
	// let's make a simple way to open a file
	program.entries.file.entry.SetPlaceHolder("/path/to/file.txt")
	program.entries.counterparty.SetPlaceHolder("counterparty address: dero...")

	// now we are going to encrypt a file
	encrypt := widget.NewHyperlink("encrypt", nil)
	encrypt.OnTapped = func() {
		var e *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			e.Submit()
			e.Dismiss()
		}
		e = dialog.NewForm("Encrypt File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
			func(b bool) {
				// if they cancel
				if !b {
					return
				}
				// let's validate the address real quick
				if err := program.entries.counterparty.Validate(); err != nil {
					showError(err)
					return
				}

				// get the pass
				pass := program.entries.pass.Text

				// dump the entry
				program.entries.pass.SetText("")

				// check the password
				if !program.wallet.Check_Password(pass) {
					showError(errors.New("wrong password"))
					program.entries.counterparty.SetText("")
					program.entries.file.entry.SetText("")
				} else {

					//get the filename
					filename := program.entries.file.entry.Text

					// dump the entry
					program.entries.file.entry.SetText("")

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err)
						program.entries.counterparty.SetText("")
						return
					}

					// let's check the receiver
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						// show the user the error
						showError(err)
						program.entries.counterparty.SetText("")
						return
					}

					// get your secret as big int
					secret_key := program.wallet.Get_Keys().Secret.BigInt()

					// get the recipient pub key
					reciever_pub_key := addr.PublicKey.G1()

					// make a shared key
					shared_key := crypto.GenerateSharedSecret(secret_key, reciever_pub_key)

					// encrypt the file using the shared key
					crypto.EncryptDecryptUserData(shared_key, file)

					// use the .enc suffix
					save_path := filename + ".enc"

					// write the file to disk
					os.WriteFile(save_path, file, default_file_permissions)

					// build a notice
					notice := "File successfully encrypted\n" +
						"Message saved as " + save_path

					// load it into the dialog
					e := dialog.NewInformation("Encrypt", notice, program.window)

					// resize and show
					e.Resize(program.size)
					e.Show()
					return

				}
				// use the main window for the encrypt
			}, program.window)
		e.Show()
	}

	// let's decrypt a file
	decrypt := widget.NewHyperlink("decrypt", nil)
	decrypt.OnTapped = func() {
		var d *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			d.Submit()
			d.Dismiss()
		}
		d = dialog.NewForm("Decrypt File?", confirm, dismiss,
			[]*widget.FormItem{widget.NewFormItem("", program.entries.pass)},
			func(b bool) {
				// if they cancel
				if !b {
					return
				}
				// let's validate the address real quick
				if err := program.entries.counterparty.Validate(); err != nil {
					showError(err)
					return
				}
				// get the pass
				pass := program.entries.pass.Text

				// dump the password
				program.entries.pass.SetText("")

				// check the password
				if !program.wallet.Check_Password(pass) {
					showError(errors.New("wrong password"))
					program.entries.file.entry.SetText("")
				} else {

					// get the filename
					filename := program.entries.file.entry.Text

					// check if it is an .enc file
					if !strings.HasSuffix(filename, ".enc") {
						showError(errors.New("not a .enc file"))
						return
					}

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err)
						return
					}

					// check the receiver address
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						showError(err)
						return
					}

					// get the wallet's secret key as a big int
					secret_key := program.wallet.Get_Keys().Secret.BigInt()

					// use the reciever pub key
					reciever_pub_key := addr.PublicKey.G1()

					// create a shared key
					shared_key := crypto.GenerateSharedSecret(secret_key, reciever_pub_key)

					// decrypt the file with the key
					crypto.EncryptDecryptUserData(shared_key, file)

					// trim the .enc suffix
					save_path := strings.TrimSuffix(filename, ".enc")

					// write the file to disk
					os.WriteFile(save_path, file, default_file_permissions)

					// let's make another notice
					notice := "File successfully decrypted\n" +
						"Message saved as " + save_path

					// load the notice in the dialog
					e := dialog.NewInformation("Decrypt",
						notice,
						program.window)

					//resize and show
					e.Resize(program.size)
					e.Show()
					return
				}
			}, program.window)
		d.Show()
	}

	// let's make sure that we validate the address we use
	program.entries.counterparty.Validator = addressValidator

	// let's also make a notice
	notice := "Asymetrically encrypt/decrypt files. "
	notice += "Select the file to encrypt/decrypt and enter the address of the user who sent it or is to receive. "

	// make the label
	label := makeCenteredWrappedLabel(notice)

	// let's make a nice spash screen
	recipient_crypt := dialog.NewCustom("Recipient Encrypt/Decrypt", dismiss,
		container.NewVBox(
			layout.NewSpacer(),
			container.NewVBox(program.entries.file),
			program.entries.counterparty,
			container.NewAdaptiveGrid(2,
				container.NewCenter(encrypt),
				container.NewCenter(decrypt),
			),
			label,
			layout.NewSpacer(),
		), program.window)

	// resize it and show it
	recipient_crypt.Resize(program.size)
	recipient_crypt.Show()
}
func integrated_address_generator() {

	// let's start with a set of arguments
	args := rpc.Arguments{}

	// we are going to get a destination port
	dst := widget.NewEntry()

	// make a user friendly placeholder
	dst.SetPlaceHolder("destination port: 123456789012345678")

	// let's validate the destination port
	dst.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		// parse the string
		i, err := strconv.Atoi(s)

		// if it isn't a number...
		if err != nil {
			return err
		}
		if i >= math.MaxInt64-1 { // 'no value of type int is greater than math.MaxInt64'
			// yeah... but if it is...
			return errors.New("can't be more than 18 decimals")
		}
		// just in case they think they can
		if i < 1 {
			return errors.New("can't be a zero or a negative number")
		}
		// load up the value into the args
		args = append(args, rpc.Argument{
			Name:     rpc.RPC_DESTINATION_PORT,
			DataType: rpc.DataUint64,
			Value:    uint64(i),
		})
		return nil
	}

	// let's get a value from the user
	value := widget.NewEntry()

	// here is a place holder
	value.SetPlaceHolder("value: 816.80085")

	// and now let's parse it
	value.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		// let's assume it is float...
		f, err := strconv.ParseFloat(s, 64)
		// error if not
		if err != nil {
			return err
		}

		// that float can't be more than 21M
		if f > 21000000 {
			return errors.New("nope, you can't do that")
		}
		// and it can't me less than 0
		if f <= 0 {
			return errors.New("can't be a zero or a negative number")
		}

		// let's coerce the float into a value of atomic units
		v := uint64(f * atomic_units)

		// an append it into the arguments
		args = append(args, rpc.Argument{
			Name:     rpc.RPC_VALUE_TRANSFER,
			DataType: rpc.DataUint64,
			Value:    v,
		})
		return nil
	}

	// let's see if there is a comment
	comment := widget.NewEntry()

	// make a placeholder
	comment.SetPlaceHolder("send comment with transaction: barking-mad-war-dogs")

	// validate the comment
	comment.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		// seems simple enought
		if len(s) > 100 {
			return errors.New("can't be more than 100 characters")
		}

		// load the string into the args
		args = append(args, rpc.Argument{
			Name:     rpc.RPC_COMMENT,
			DataType: rpc.DataString,
			Value:    s,
		})
		return nil
	}

	// now if a reply back is necessary
	needs_replyback := widget.NewCheck("Needs Replyback Address?", func(b bool) {
		args = append(args, rpc.Argument{
			Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			DataType: rpc.DataUint64,
			Value:    uint64(1),
		})
	})

	// so they clicked on generate did they?
	generate := widget.NewHyperlink("generate random dst", nil)
	generate.OnTapped = func() {
		dst.SetText(strconv.Itoa(rand.Int()))
	}

	// well..
	callback := func(b bool) {
		// in case they cancel
		if !b {
			return
		}
		// let's validate; what about assets?
		if dst.Validate() != nil &&
			comment.Validate() != nil &&
			value.Validate() != nil {

			// show them the error
			showError(errors.New("something isn't working"))
		}

		// make an address entry
		address := widget.NewEntry()

		// block text, she big
		address.MultiLine = true

		// wrap the word, looks better that way
		address.Wrapping = fyne.TextWrapWord

		// disable so there is no error there
		address.Disable()

		// if the following is empty and unchecked...
		if dst.Text == "" &&
			value.Text == "" &&
			comment.Text == "" &&
			!needs_replyback.Checked {

			// generate a random address
			address.SetText(program.wallet.GetRandomIAddress8().String())
		} else { // otherwise
			// get the wallet address
			addr := program.wallet.GetAddress()

			// make a new address struct
			result, err := rpc.NewAddress(addr.String())
			if err != nil {
				showError(err)
				return
			}

			// check the pack of the args
			if _, err := args.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
				showError(err)
				return
			}
			// the result arguments are now the args
			result.Arguments = args

			// set the block text entry to the integrated addr
			address.SetText(result.String())
		}

		// set the address into a splash
		integrated_address := dialog.NewCustom("Integrated Address", dismiss, container.NewVBox(address), program.window)

		// resize and show
		integrated_address.Resize(program.size)
		integrated_address.Show()
	}

	items := []*widget.FormItem{
		widget.NewFormItem("", value),
		widget.NewFormItem("", dst),
		widget.NewFormItem("", generate),
		widget.NewFormItem("", comment),
		widget.NewFormItem("", needs_replyback),
	}

	// put all the fun stuff into a dialog
	integrated := dialog.NewForm("Integrated Address", confirm, dismiss, items, callback,
		program.window) // send it to the main window
	// resize and show
	integrated.Resize(program.size)
	integrated.Show()
}
func balance_rescan() {
	// nice big notice
	big_notice :=
		"This action will clear all transfer history and balances. " +
			"Balances are nearly instant in resync; however... " +
			"Tx history depends on node status, eg pruned/full... " +
			"Some txs may not be available at your node connection. " +
			"For full history, and privacy, run a full node starting at block 0 upto current topoheight. " +
			"This operation could take a long time with many token assets and transfers. "
	// here is the rescan dialog
	rescan := dialog.NewConfirm("Balance Rescan", big_notice,
		func(b bool) {
			// if they cancel
			if !b {
				return
			}

			// start a sync activity widget
			syncing := widget.NewActivity()
			syncing.Start()
			notice := makeCenteredWrappedLabel("Beginning Scan")
			// set it to a splash screen
			sync := dialog.NewCustomWithoutButtons("syncing",
				container.NewVBox(layout.NewSpacer(), syncing, notice, layout.NewSpacer()),
				program.window)

			// resize and show
			sync.Resize(program.size)
			sync.Show()

			// rebuild the hash list
			buildAssetHashList()

			// as a go routine
			go func() {

				// clean the wallet
				program.wallet.Clean()
				var done bool
				go func() {

					var old int
					ticker := time.NewTicker(time.Second)
					for range ticker.C {
						if done {
							break
						}

						transfers := allTransfers()
						current := len(transfers)
						if current == old {
							continue
						}
						diff := current - old
						old = current
						for _, each := range transfers[diff:] {
							fyne.DoAndWait(func() {
								notice.SetText("BlockHash: " + each.BlockHash)
							})
							time.Sleep(time.Second)
							fyne.DoAndWait(func() {
								notice.SetText("Retrieving more txs")
							})
						}
					}
				}()

				// then sync the wallet for DERO
				if err := program.wallet.Sync_Wallet_Memory_With_Daemon(); err != nil {
					// if there is an error, notify the user
					showError(err)
					return
				} else {
					// now range through each token in the cache one at a time
					for _, asset := range program.caches.assets {
						// assume there could be an error
						var err error

						// then add each scid back to the map
						if err = program.wallet.TokenAdd(
							crypto.HashHexToHash(asset.hash),
						); err != nil {
							// if err, show it
							showError(err)
							// but don't stop, just continue the loop
							continue
						}

						// and then sync scid internally with the daemon
						if err = program.wallet.Sync_Wallet_Memory_With_Daemon_internal(
							crypto.HashHexToHash(asset.hash),
						); err != nil {
							// if err, show it
							showError(err)
							// but don't stop, just continue the loop
							continue
						}
					}
					// when done, shut down the sync status in the go routine
					fyne.DoAndWait(func() {
						done = true
						syncing.Stop()
						sync.Dismiss()
					})
				}
			}()

		}, program.window) // set to the main window

	// resize and show
	rescan.Resize(program.size)
	rescan.Show()
}
func installer() {

	// let's validate that file, shall we?
	program.entries.file.entry.Validator = func(s string) error {
		// read the file
		b, err := os.ReadFile(s)

		// return an err, if one
		if err != nil {
			return err
		}

		// if it is a parsable smart contract, green light it
		if _, _, err = dvm.ParseSmartContract(string(b)); err != nil {

			// otherwise, return err
			return err
		}

		// validated
		return nil
	}

	// we just use this because simplicity
	ringsize := uint64(2)

	// we do enable this, and we notify the user
	isAnonymous := widget.NewCheck("is anonymous?", func(b bool) {
		if b {
			ringsize = 16
		}
	})

	// see, notice
	notice := makeCenteredWrappedLabel("anonymous installs might effect intended SC functionality")

	// let's make a splash screen
	splash := container.NewVBox(
		layout.NewSpacer(),
		container.NewVBox(program.entries.file),
		isAnonymous,
		notice,
		layout.NewSpacer(),
	)

	// let's walk throught install
	install := dialog.NewCustomConfirm("Install Contract", confirm, dismiss, splash,
		func(b bool) {
			// if they cnacel
			if !b {
				return
			}

			var ic *dialog.ConfirmDialog
			// if they press enter, we assume it means confirm
			program.entries.pass.OnSubmitted = func(s string) {
				ic.Confirm()
				ic.Dismiss()
			}
			ic = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, program.entries.pass,
				func(b bool) {
					// if they cnacel
					if !b {
						return
					}
					// get the password
					pass := program.entries.pass.Text

					// dump the pass entry
					program.entries.pass.SetText("")

					// check the password
					if !program.wallet.Check_Password(pass) {
						showError(errors.New("wrong password"))
						program.entries.file.entry.SetText("")
						return
					} else {
						// get the filename
						filename := program.entries.file.entry.Text

						// dump the entry
						program.entries.file.entry.SetText("")

						// read the file
						file, err := os.ReadFile(filename)
						if err != nil {
							showError(err)
							return
						}
						// coerce the file into string
						upload := string(file)

						// parse the file for an error; yes, I know we validated...
						if _, _, err = dvm.ParseSmartContract(upload); err != nil {

							// show them an error if one
							showError(err)
							return
						}

						// let's get a list of random addresses for the install
						randos := program.wallet.Random_ring_members(crypto.ZEROHASH)

						// let's grab the first one on the list
						rando := randos[0]

						// let's make sure it isn't the wallet's address
						if rando == program.wallet.GetAddress().String() {
							// the liklihood that 0 && 1 are the same addr is...
							// funny
							rando = randos[1]
						}

						// create a payload
						payload := []rpc.Transfer{
							{
								SCID:        crypto.ZEROHASH,
								Destination: rando,
								Amount:      0,
							},
						}

						// no, don't transfer all; still not implemented anyway
						transfer_all := false

						// here it the sc data
						sc_data := rpc.Arguments{
							rpc.Argument{
								Name:     rpc.SCACTION,
								DataType: rpc.DataUint64,
								Value:    uint64(1), //  eg. 'yes, it does'
							},
							rpc.Argument{
								Name:     rpc.SCCODE,
								DataType: rpc.DataString,
								Value:    upload, // eg. the contract
							},
						}

						// we aren't storing anything
						gasstorage := uint64(0)

						// this is not a drill!
						dry_run := false

						// let's create a transaction
						tx, err := program.wallet.TransferPayload0(
							payload, ringsize, transfer_all,
							sc_data, gasstorage, dry_run,
						)

						// if it errors out...
						if err != nil {

							// notify the user
							showError(err)
							return
						}

						// ship the tx to the daemon
						if err := program.wallet.SendTransaction(tx); err != nil {

							// notify the user if err
							showError(err)
							return
						}

						// get the txid and truncate it
						truncated_hash := truncator(tx.GetHash().String())

						// attach it to a hyperlink widget
						txid := widget.NewHyperlink(truncated_hash, nil)

						// set the clipboard function to it
						txid.OnTapped = func() {
							program.application.Clipboard().SetContent(tx.GetHash().String())

							// notify the user
							showInfo("", "txid copied to clipboard")
						}

						// center it
						txid.Alignment = fyne.TextAlignCenter

						// walk the user through success
						success := dialog.NewCustom("Contract Installer", "dissmiss",
							container.NewVBox(
								widget.NewLabel("Contract successfully installed"),
								txid,
								// I heard you like windows...
							), program.window)

						// resize and show
						success.Resize(program.size)
						success.Show()
					}

					// so I put a window in your window...
				}, program.window)
			ic.Show()
			// so you can enjoy windows... XD
		}, program.window)

	// resize and show
	install.Resize(program.size)
	install.Show()
}
func interaction() {

	// we are going to have some func names
	var func_names []string

	// and then we are also going to have some dvm function
	var functions []dvm.Function

	// let's make a simple way to review the sc code
	code := widget.NewEntry()

	// make it a code block
	code.MultiLine = true

	// use like 10 rows
	code.SetMinRowsVisible(10)

	// wrap the words
	// code.Wrapping = fyne.TextWrapWord // I think it looks better off

	// and here is a place holder
	code.SetPlaceHolder("sc code seen here")

	// lock it down
	code.Disable()

	// now let's make a way to enter the contract
	scid := widget.NewEntry()

	// let's validate it
	scid.Validator = func(s string) error {

		// the code needs to be present
		sc := getSCCode(s)

		if sc.Status == "" {
			code.SetText("")
			return errors.New("sc status is empty")
		}

		if sc.Code == "" {
			code.SetText("")
			return errors.New("contract code is empty")
		}

		// set the entry with the code
		code.SetText(sc.Code)

		// now parse the smart contract code function map
		smart_contract, _, err := dvm.ParseSmartContract(sc.Code)
		if err != nil {
			code.SetText("")
			return err
		}

		// now range throught the functionse
		for name, f := range smart_contract.Functions {

			// coerce name to rune
			char := []rune(name)

			// get the first char
			first_char := char[0]

			// if the first char is not upper, skip it
			if !unicode.IsUpper(first_char) {
				continue
			}

			// append each "exported" function name to the func_names slice
			func_names = append(func_names, name)

			// and add each "exported" fucntion to the functions slice
			functions = append(functions, f)
		}
		return nil
	}
	// we should assume rings size 2 for simplicty
	ringsize := uint64(2)

	// but the user can make the choice
	isAnonymous := widget.NewCheck("is anonymous?", func(b bool) {
		if b {
			ringsize = 16
		}
	})

	// sorting does nothing because these calls are made asynchronously
	// they are done as they are done.
	function_list := new(widget.List)
	function_list.Length = func() int { return len(func_names) }
	function_list.CreateItem = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	function_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {
		co.(*widget.Label).SetText(func_names[lii])
	}

	// let the user select the fuction
	function_list.OnSelected = func(id widget.ListItemID) {

		// after they do, unselect it
		function_list.Unselect(id) // release the object

		// we are going to put all the entries into this box
		entries := container.NewVBox()

		// these are not arguments, per se

		// iterate over the lines of each function
		for _, lines := range functions[id].Lines {

			// if any of the lines in the function say...
			if slices.Contains(lines, "DEROVALUE") {

				// add this entry to the entries container
				deroValueEntry := widget.NewEntry()
				deroValueEntry.SetPlaceHolder("dero value: 123.321")
				entries.Add(deroValueEntry)

			}

			// the same is true of for assetsvalue transfers
			if slices.Contains(lines, "ASSETVALUE") {

				// add this entry to the entries container
				assetValueEntry := widget.NewEntry()
				assetValueEntry.SetPlaceHolder("asset value: 123.321")
				entries.Add(assetValueEntry)

			}

			// lastly, if there is the word...
			if slices.Contains(lines, "SIGNER") {

				// make this un-usable
				isAnonymous.SetChecked(false)

				// and lock it down
				isAnonymous.Disable()
			}
		}

		// we are going to add args to a box
		args := container.NewVBox()

		// each of them have a type
		var arg_type string

		// iterate over the parameters of each funciton
		for _, param := range functions[id].Params {

			// switch on the type
			switch param.Type {
			case dvm.String:
				arg_type = "string"
			case dvm.Uint64:
				arg_type = "uint64"
			case dvm.Invalid, dvm.None:
				showError(errors.New("type is either invalid or none"))
				return
			default:
				showError(errors.New("unknown type"))
				return
			} // pretty self explanatory

			// now add the this entry to the args box
			arg := widget.NewEntry()

			// make a note worthy label for them
			arg.SetPlaceHolder(param.Name + " " + arg_type)
			args.Add(arg)
		}

		// we are going to build a payload
		payload := []rpc.Transfer{}

		// first, the preliminary sc_args
		sc_args := rpc.Arguments{
			{ // what is being done
				Name:     rpc.SCACTION,
				DataType: rpc.DataUint64,
				Value:    uint64(rpc.SC_CALL),
			},
			{ // where it is going
				Name:     rpc.SCID,
				DataType: rpc.DataHash,
				Value:    crypto.HashHexToHash(scid.Text),
			},
			{ // which entrypoint
				Name:     "entrypoint",
				DataType: rpc.DataString,
				Value:    func_names[id],
			},
		}

		// now make a nice notice
		notice := makeCenteredWrappedLabel("anonymous interactions might effect intended SC functionality")

		// make a splash box
		splash := container.NewVBox(widget.NewLabel(func_names[id]), notice, isAnonymous, args, entries)

		// and walk the user through the argument process
		arguments := dialog.NewCustomConfirm("Interact",
			confirm, dismiss,
			splash, func(b bool) {
				// if they cancel
				if !b {
					return
				}

				// parse user data to make a best guess
				var best_guess crypto.Hash

				// range over the arg objects to get the final touches
				for i, obj := range args.Objects {

					// we are making an argument
					var arg rpc.Argument

					// switch on type
					switch functions[id].Params[i].Type {
					case dvm.String:

						// get the parameter
						param := functions[id].Params[i].Name

						// get the value from the entry
						value := obj.(*widget.Entry).Text

						arg = rpc.Argument{ // load up the goods
							Name:     param,
							DataType: rpc.DataString,
							Value:    value,
						}

						// we are going to make a crazy assumption here...
						// there is almost never a case when a contract creator would ask for ZEROHASH
						// so we make an assumption that:
						//  of the args objects
						// 	of which are string
						//  of which aren't crypto.ZEROHASH
						//  will be our best guess
						// 	and we will only guess once
						hash := crypto.HashHexToHash(value)
						if hash != crypto.ZEROHASH {
							var once bool
							if !once {
								// we can best guess this scenario
								best_guess = crypto.HashHexToHash(obj.(*widget.Entry).Text)
								once = true
							} else {
								showError(errors.New("multiple token assets not implemented"))
								return
							}
						}
					case dvm.Uint64:
						// little more straight forward
						value := obj.(*widget.Entry).Text

						// convert string to int
						integer, err := strconv.Atoi(value)
						if err != nil {

							// show err if so
							showError(err)
							return
						}

						// get the param name
						param := functions[id].Params[i].Name

						// load up the arg
						arg = rpc.Argument{
							Name:     param,
							DataType: rpc.DataUint64,
							Value:    uint64(integer),
						}
					}

					// append the applicable arg to the sc args
					sc_args = append(sc_args, arg)
				}

			reroll:

				// get some random ring members to chill with
				randos := program.wallet.Random_ring_members(crypto.ZEROHASH)

				// check your self before you wreck yourself
				self := program.wallet.GetAddress().String()

				// top three dudes, can't be self... can they?
				if randos[0] == self && randos[1] == self && randos[2] == self {
					fmt.Println("easter egg unlocked: do a barrel roll XD")
					goto reroll
				}

				// anyway, iterate over the entry objects
				for _, obj := range entries.Objects {

					// this is a nifty trick
					// grab the placeholder
					placeholder := obj.(*widget.Entry).PlaceHolder

					// if it has the 'dero value:' place holder
					if strings.Contains(placeholder, "dero value:") {

						// get the value fromt the entry
						value := obj.(*widget.Entry).Text

						// parse the float
						float, err := strconv.ParseFloat(value, 64)
						if err != nil {
							showError(err)
							return
						}

						// coerce the float to a burn value
						burn := uint64(float) * atomic_units

						// load it into the payload
						payload = append(payload, rpc.Transfer{
							Destination: randos[0], // rando numero uno
							Amount:      0,         // don't send anything to them
							Burn:        burn,      // burning the dero from crypto.ZEROHASH ledger into the SCID ledger
						})
					} else if strings.Contains(placeholder, "asset value:") {

						// get the value
						value := obj.(*widget.Entry).Text

						// parse it
						float, err := strconv.ParseFloat(value, 64)

						// show err if so
						if err != nil {
							showError(err)
							return
						}

						// coerce the float into an burnable amount
						burn := uint64(float) * atomic_units

						// now check if the best guess is ZEROHASH
						if best_guess.IsZero() {

							// if it is, this is a problem
							showError(errors.New("please report this error to the develop"))
							return
						}

						// and if it isn't ZEROHASH,
						// well... load er' up
						scid := best_guess

						// and append to payload
						payload = append(payload, rpc.Transfer{
							Destination: randos[1],
							SCID:        scid,
							Amount:      0,
							Burn:        burn,
						})
					}

				}

				// in the off chance that we don't have a payload...
				if len(payload) < 1 {

					// append one
					payload = append(payload, rpc.Transfer{
						Destination: randos[2],
					})
				}

				string_args, err := sc_args.MarshalBinary()
				if err != nil {
					//show the err
					showError(err)
					// but keep going
				}

				// let's get the args for the user to review
				sa := makeCenteredWrappedLabel(string(string_args))

				// load up the splash and a password entry
				splash := container.NewVBox(sa, program.entries.password)
				var ci *dialog.ConfirmDialog
				// if they press enter here, it is a confirmation
				program.entries.pass.OnSubmitted = func(s string) {
					ci.Confirm()
					ci.Dismiss()
				}
				ci = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, splash, func(b bool) {
					// if they cancel
					if !b {
						return
					}

					// get the pass
					pass := program.entries.pass.Text

					// dump the entry
					program.entries.pass.SetText("")

					if !program.wallet.Check_Password(pass) {
						showError(errors.New("wrong password"))
						return
					} else {

						// easy to read details
						send_all := false
						gasstorage := uint64(0)
						dry_run := false

						// create the transaction
						tx, err := program.wallet.TransferPayload0(
							payload, ringsize, send_all,
							sc_args, // hold on to your butts
							gasstorage, dry_run,
						)

						// if we have an err, show it
						if err != nil {
							showError(err)
							return
						}

						// submit the transfer to the daemon
						if err := program.wallet.SendTransaction(tx); err != nil {
							showError(err)
							return
						}

						// get the hash and truncate it
						truncated_hash := truncator(tx.GetHash().String())

						// make it into a new hyperlink
						txid := widget.NewHyperlink(truncated_hash, nil)

						// align it center
						txid.Alignment = fyne.TextAlignCenter

						// make it copiable
						txid.OnTapped = func() {
							program.application.Clipboard().SetContent(tx.GetHash().String())
							showInfo("", "txid copied to clipboard")
						}

						// make a nice big mesesage
						msg := "Contract successfully interacted\n\n" +
							"Please note: a successful interaction does not mean a successful contract operation." +
							"Please review verify all interactions"

						// attach it to a notice
						notice := makeCenteredWrappedLabel(msg)

						success := dialog.NewCustom("Contract Installer", "dissmiss",
							container.NewVBox(
								notice,
								txid,
							), program.window)
						// resize and show
						success.Resize(program.size)
						success.Show()

					}
					// load it to the main window
				}, program.window)

				ci.Show()

			}, program.window)

		// resize and show
		arguments.Resize(program.size)
		arguments.Show()
	}

	// we want users to be happy with the code before they interact with it
	notice := "If you are satisfied with the code, "
	notice += "please confirm to be forwarded to the contract functions"

	confirmation := makeCenteredWrappedLabel(notice)

	interact := dialog.NewCustomConfirm("Interact with Contract",
		confirm, dismiss,
		container.NewVBox(scid, code, confirmation),
		func(b bool) {
			// if they cancel
			if !b {
				return
			}

			// the final countdown!
			action := dialog.NewCustom("contract interaction", dismiss,
				function_list,
				program.window)

			//resize and show
			action.Resize(program.size)
			action.Show()

			// windows in windows...
		}, program.window)

	// resize and show
	interact.Resize(program.size)
	interact.Show()
}

func add_token() {

	// make a new entry widget
	t := widget.NewEntry()

	// set the place holder
	t.SetPlaceHolder("SCID TOKEN ADDRESS")

	// walk the user through adding a token
	add := dialog.NewCustomConfirm("Add Token", confirm, dismiss,
		container.NewVBox(
			layout.NewSpacer(),
			t,
			layout.NewSpacer(),
		),
		func(b bool) {
			// if they cancel
			if !b {
				return
			}
			// so if the map is nil, make one
			if program.wallet.GetAccount().EntriesNative == nil {
				program.wallet.GetAccount().EntriesNative = make(map[crypto.Hash][]rpc.Entry)
			}

			//get the hash
			hash := crypto.HashHexToHash(t.Text)
			// start a sync activity widget
			syncing := widget.NewActivity()
			syncing.Start()
			notice := makeCenteredWrappedLabel("syncing")

			// set it to a splash screen
			sync := dialog.NewCustomWithoutButtons("syncing",
				container.NewVBox(layout.NewSpacer(), syncing, notice, layout.NewSpacer()),
				program.window)

			// resize and show
			sync.Resize(program.size)
			sync.Show()
			go func() {

				// add the token
				if err := program.wallet.TokenAdd(hash); err != nil {

					// show err if one
					fyne.DoAndWait(func() {

						showError(err)
						sync.Dismiss()
					})
					return
				} else {

					// sync the token now for good measure
					if err := program.wallet.Sync_Wallet_Memory_With_Daemon_internal(hash); err != nil {
						fyne.DoAndWait(func() {

							showError(err)
							sync.Dismiss()
						})
						return
					}

					//make a notice
					notice := truncator(hash.String()) + "has been added to your collection"

					// give notice to the user
					fyne.DoAndWait(func() {

						showInfo("Token Add", notice)
						sync.Dismiss()
					})
				}
			}()

		}, program.window)

	// resize and show
	add.Resize(program.size)
	add.Show()
}

func addressValidator(s string) (err error) {

	// any changes to the string should immediately update the receiver string
	program.receiver = ""

	// if empty...
	if s == "" {
		return errors.New("recipient cannot be empty")
	}

	// if less than 4 char...
	if len(s) < 4 {
		return errors.New("cannot be less than 5 char")
	}

	// first check to see if it is an address
	addr, err := rpc.NewAddress(s)
	// if it is not an address...
	if err != nil {
		// check to see if it is a name
		a, err := program.wallet.NameToAddress(s)
		if err != nil {
			// she barks
			// fmt.Println(err)
		}
		// if a valid , they are the receiver
		if a != "" {
			program.receiver = a
		}

		// now if the address is an integrated address...
	} else if addr.IsIntegratedAddress() {

		// the base of that address is what we'll use as the receiver
		program.receiver = addr.BaseAddress().String()

	} else if addr.String() != "" && // if the addr isn't empty
		!addr.IsIntegratedAddress() { // now if it is not an integrated address

		// set the receiver
		program.receiver = addr.String()
	}

	// at this point, we should be fairly confident
	if program.receiver == "" {
		err = errors.New("error obtaining receiver")

		return err
	}

	// but just to be extra sure...
	// let's see if the receiver is not registered
	if !isRegistered(program.receiver) {
		err = errors.New("unregistered address")

		return err
	}

	// also, would make sense to make sure that it is not self
	if strings.EqualFold(program.receiver, program.wallet.GetAddress().String()) {
		err = errors.New("cannot send to self")

		return err
	}

	// should be validated
	return nil
}
