// this is the largest part of the repo, and the most fun.

package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/block"
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
	program.buttons.explorer.OnTapped = explorer
	program.buttons.encryption.OnTapped = encryption
	program.buttons.contracts.OnTapped = contracts

	// and then set them
	program.containers.toolbox = container.NewVBox(
		program.buttons.encryption,
		program.buttons.explorer,
		program.buttons.contracts,
	)

	// and now, let's hide them
	program.containers.toolbox.Hide()

	return container.NewVBox(
		program.containers.topbar,
		layout.NewSpacer(),
		container.NewAdaptiveGrid(3,
			layout.NewSpacer(),
			program.containers.toolbox,
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
		program.containers.bottombar,
	)

}

func encryption() {
	program.encryption = fyne.CurrentApp().NewWindow(program.window.Title() + " | encryption ")
	program.encryption.SetIcon(theme.VisibilityOffIcon())
	program.encryption.Resize(fyne.NewSize(program.size.Width/2, program.size.Height))
	tabs := container.NewAppTabs(
		container.NewTabItem("File Sign / Verify",
			filesign(),
		),
		container.NewTabItem("Self Crypt",
			self_crypt(),
		),
		container.NewTabItem("Recipient Crypt",
			recipient_crypt(),
		),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	program.encryption.SetContent(container.NewAdaptiveGrid(1, tabs))

	program.encryption.Show()
}

func contracts() {
	// program.buttons.contract_installer.OnTapped = installer
	// program.buttons.contract_interactor.OnTapped = interaction
	program.contracts = fyne.CurrentApp().NewWindow(program.window.Title() + " | contracts ")
	program.contracts.SetIcon(theme.FileIcon())
	program.contracts.Resize(program.size)
	tabs := container.NewAppTabs(
		container.NewTabItem("Contract Installer", installer()),
		container.NewTabItem("Contract interactor", interaction()),
	)
	program.contracts.SetContent(tabs)

	program.contracts.Show()
}

// this is a pretty under-rated feature
func filesign() *fyne.Container {
	// here is a simple way to select a file in general
	program.dialogues.open = openExplorer(program.encryption)

	// let's make an simple way to open files
	program.entries.file.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		program.dialogues.open.Show()
	})

	// let's make it noticeable that you can select the file
	program.entries.file.SetPlaceHolder("/path/to/file.txt")

	// now let's make a sign hyperlink
	sign := widget.NewHyperlink("filesign", nil)

	// and when the user taps it
	onTapped := func() {
		var fs *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			fs.Submit()
			fs.Dismiss()
		}
		// create a callback function
		callback := func(b bool) {
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
				showError(errors.New("wrong password"), program.encryption)

				// dump the filepath
				program.entries.file.SetText("")
				return
			} else {
				// get the filename
				filename := program.entries.file.Text

				//dump the entry
				program.entries.file.SetText("")

				// read the file
				file, err := os.ReadFile(filename)
				if err != nil {
					showError(err, program.encryption)

					return
				}

				// sign the file into bytes
				data := program.wallet.SignData(file)

				// it is possible to sign data as an unregistered user
				if !isRegistered(program.wallet.GetAddress().String()) {
					notice := "you have signed a file as an unregistered user"
					// notify the user, but continue anyway
					showInfo("NOTICE", notice, program.encryption)

				}

				// make a filename
				save_path := filename + ".signed"

				// write the file to disc
				os.WriteFile(save_path, data, default_file_permissions)

				msg := "File successfully signed\n\n" +
					"Located in " + save_path

				// notify the user
				showInfo("Filesign", msg, program.encryption)

			}
		}
		// now create a simple form
		content := []*widget.FormItem{widget.NewFormItem("", program.entries.pass)}

		// set the content and the callback
		fs = dialog.NewForm("Sign File?", confirm, dismiss, content, callback, program.encryption)
		fs.Resize(password_size)
		fs.Show()
	}

	// now set the on tapped
	sign.OnTapped = onTapped

	// we are going to do the same thing, but the reverse direction

	// make a link to verify a file
	verify := widget.NewHyperlink("fileverify", nil)

	// when they click the link
	onTapped = func() {
		var v *dialog.FormDialog
		program.entries.pass.OnSubmitted = func(s string) {
			v.Submit()
			v.Dismiss()
		}

		// create a callback
		callback := func(b bool) {
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
				showError(errors.New("wrong password"), program.encryption)

				//dump entry
				program.entries.file.SetText("")

			} else {
				// get the filename
				filename := program.entries.file.Text
				program.entries.file.SetText("")

				// check if the file is a .signed file
				if !strings.HasSuffix(filename, ".signed") {

					// display error
					showError(errors.New("not a .signed file"), program.encryption)

					return
				}

				// if everything is good so far, read the files
				file, err := os.ReadFile(filename)
				if err != nil {
					showError(err, program.encryption)

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
					showError(err, program.encryption)

					return
				}

				// it is possible to sign data as an unregistered user
				if !isRegistered(sign.String()) {
					notice := "an unregistered user has signed this data"
					// notify the user, but continue
					showInfo("NOTICE", notice, program.encryption)

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
				fv := dialog.NewInformation("FileVerify", notice, program.encryption)
				fv.Show()
				return
			}

		}

		// create a simple form
		content := []*widget.FormItem{widget.NewFormItem("", program.entries.pass)}

		// set the content and the callback
		v = dialog.NewForm("Verify File?", confirm, dismiss, content, callback, program.encryption)
		v.Resize(password_size)
		v.Show()
	}

	// set on tapped
	verify.OnTapped = onTapped

	// now let's make another notice
	filesign := "filesign creates `.signed` files."
	fileverify := "fileverify verifies `.signed` data."

	// let's load all the widgets into a container inside a dialog
	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewVBox(program.entries.file),
		container.NewAdaptiveGrid(2,
			container.NewCenter(sign),
			container.NewCenter(verify),
		),
		widget.NewRichTextFromMarkdown(filesign),
		widget.NewRichTextFromMarkdown(fileverify),
		layout.NewSpacer(),
	)
	return content
}
func self_crypt() *fyne.Container {
	pass := widget.NewPasswordEntry()
	pass.SetPlaceHolder("w41137-p@55w0rd")

	entry := widget.NewEntry()
	entry.SetPlaceHolder("text to encrypt/decrypt")
	entry.MultiLine = true

	// another round of make sure this works XD
	file_entry := widget.NewEntry()
	file_entry.SetPlaceHolder("/path/to/file.txt")
	// here is a simple way to select a file in general
	program.dialogues.open = openExplorer(program.encryption)

	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		program.dialogues.open.Resize(program.size)
		program.dialogues.open.Show()
	})

	// let's make an simple way to open files
	entry.OnChanged = func(s string) {
		if s == "" {
			file_entry.Enable()
			return
		} else {
			file_entry.Disable()
			return
		}
	}
	file_entry.OnChanged = func(s string) {
		if s == "" {
			entry.Enable()
			return
		} else {
			entry.Disable()
			return
		}
	}

	// let's encrypt data
	encrypt := widget.NewHyperlink("encrypt", nil)

	// when the user clicks here...
	onTapped := func() {
		var e *dialog.FormDialog
		pass.OnSubmitted = func(s string) {
			e.Submit()
			e.Dismiss()
		}
		// create a callback function
		callback := func(b bool) {
			// if they cancel
			if !b {
				return
			}
			// let's get the password
			p := pass.Text

			//dump the entry
			pass.SetText("")

			// check the password
			if !program.wallet.Check_Password(p) {
				// notify them when wrong
				showError(errors.New("wrong password"), program.encryption)
			} else {
				if entry.Disabled() {
					// get the filename
					filename := file_entry.Text

					// dump the entry
					file_entry.SetText("")

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						// display error if there is one
						showError(err, program.encryption)

						return
					}

					// encrypt the data
					data, err := program.wallet.Encrypt(file)
					if err != nil {
						showError(err, program.encryption)

						return
					}

					// made a save path
					save_path := filename + ".enc"

					// write file to disk
					os.WriteFile(save_path, []byte(base64.StdEncoding.EncodeToString(data)), default_file_permissions)

					// make a success notice
					notice := "File successfully encrypted\n" +
						"Message saved as " + save_path

					// load it , and show it
					e := dialog.NewInformation("Encrypt", notice, program.encryption)
					e.Resize(program.size)
					e.Show()
					return
				} else if !entry.Disabled() {
					text := entry.Text
					// encrypt the data
					data, err := program.wallet.Encrypt([]byte(text))
					if err != nil {
						showError(err, program.encryption)

						return
					}

					entry.SetText(base64.StdEncoding.EncodeToString(data))
					entry.Refresh()
				}
			}
		}

		// create a simple form
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

		// set the content and the callback
		e = dialog.NewForm("Encrypt?", confirm, dismiss, content, callback, program.encryption)
		e.Resize(password_size)
		e.Show()
	}
	// now set the on tapped
	encrypt.OnTapped = onTapped

	// now let's decrypt
	decrypt := widget.NewHyperlink("decrypt", nil)

	// here's what we are going to do
	onTapped = func() {
		var d *dialog.FormDialog
		pass.OnSubmitted = func(s string) {
			d.Submit()
			d.Dismiss()
		}
		// create a callback function
		callback := func(b bool) {
			// if they cancel
			if !b {
				return
			}
			// get the password
			p := pass.Text

			// dump the password
			pass.SetText("")

			// check the password
			if !program.wallet.Check_Password(p) {

				// notify the user
				showError(errors.New("wrong password"), program.encryption)

			} else {
				if entry.Disabled() {

					// get the file name
					filename := file_entry.Text

					// dump the entry
					file_entry.SetText("")

					// check if this is an .enc file
					if !strings.HasSuffix(filename, ".enc") {

						// notify the user
						showError(errors.New("not a .enc file"), program.encryption)

						return
					}

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err, program.encryption)

						return
					}

					data, err := base64.RawStdEncoding.DecodeString(string(file))
					if err != nil {
						showError(err, program.encryption)

						return
					}

					// decrypt the file
					data, err = program.wallet.Decrypt(data)
					if err != nil {
						showError(err, program.encryption)

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
					d := dialog.NewInformation("Decrypt", notice, program.encryption)
					d.Resize(program.size)
					d.Show()
					return
				} else if !entry.Disabled() {
					text := entry.Text
					// decrypt the file

					data, err := base64.StdEncoding.DecodeString(text)
					if err != nil {
						showError(err, program.encryption)

						return
					}

					data, err = program.wallet.Decrypt(data)
					if err != nil {
						showError(err, program.encryption)

						return
					}
					entry.SetText(string(data))
					entry.Refresh()
				}
			}
		}

		// create a simple form
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

		// set the content and the callback
		d = dialog.NewForm("Decrypt?", confirm, dismiss, content, callback, program.encryption)
		d.Resize(password_size)
		d.Show()
	}

	// set the callback for on tapped
	decrypt.OnTapped = onTapped

	// let's make another notice
	notice := "Symmetrically encrypt/decrypt files or text. "
	notice += "Select file, or freetype text, that is to be encrypted / decrypted using wallet's secretkey. "
	notice += "Encrypted content is base64Encoded, eg. VxE/12ZXvZzBaI3Sj1qcHcC18qRz/dNyQfihbRkz/Yg="

	// create a label widget
	label := makeCenteredWrappedLabel(notice)

	content := container.NewVBox(
		layout.NewSpacer(),
		widget.NewForm(
			widget.NewFormItem("", file_entry),
			widget.NewFormItem("", entry),
		),
		container.NewAdaptiveGrid(2,
			container.NewCenter(encrypt),
			container.NewCenter(decrypt),
		),
		label,
		layout.NewSpacer(),
	)

	return content
}
func recipient_crypt() *fyne.Container {
	entry := widget.NewEntry()
	entry.MultiLine = true
	entry.SetPlaceHolder("text to be encrypted / decrypted")

	// let's make a simple way to open a file
	file_entry := widget.NewEntry()
	file_entry.SetPlaceHolder("/path/to/file.txt")
	// here is a simple way to select a file in general
	program.dialogues.open = openExplorer(program.encryption)

	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		program.dialogues.open.Resize(program.size)
		program.dialogues.open.Show()
	})

	entry.OnChanged = func(s string) {
		if s == "" {
			file_entry.Enable()
			return
		} else {
			file_entry.Disable()
			return
		}
	}

	file_entry.OnChanged = func(s string) {
		if s == "" {
			entry.Enable()
			return
		} else {
			entry.Disable()
			return
		}
	}

	counterparty := widget.NewEntry()
	counterparty.SetPlaceHolder("counterparty address: dero...")
	pass := widget.NewPasswordEntry()
	pass.SetPlaceHolder("w41137-p@55w0rd")

	// now we are going to encrypt a file
	encrypt := widget.NewHyperlink("encrypt", nil)

	// create an onTapped function
	onTapped := func() {
		var e *dialog.FormDialog
		pass.OnSubmitted = func(s string) {
			e.Submit()
			e.Dismiss()
		}
		callback := func(b bool) {
			// if they cancel
			if !b {
				return
			}
			// let's validate the address real quick
			if err := counterparty.Validate(); err != nil {
				showError(err, program.encryption)

				return
			}

			// get the pass
			p := pass.Text

			// dump the entry
			pass.SetText("")

			// check the password
			if !program.wallet.Check_Password(p) {
				showError(errors.New("wrong password"), program.encryption)
				counterparty.SetText("")
				file_entry.SetText("")
			} else {
				if entry.Disabled() {

					//get the filename
					filename := file_entry.Text

					// dump the entry
					file_entry.SetText("")

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err, program.encryption)

						counterparty.SetText("")
						return
					}

					// let's check the receiver
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						// show the user the error
						showError(err, program.encryption)

						counterparty.SetText("")
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
					os.WriteFile(save_path, []byte(base64.StdEncoding.EncodeToString(file)), default_file_permissions)

					// build a notice
					notice := "File successfully encrypted\n" +
						"Message saved as " + save_path

					// load it into the dialog
					e := dialog.NewInformation("Encrypt", notice, program.encryption)

					// resize and show
					e.Resize(program.size)
					e.Show()
					return
				} else if !entry.Disabled() {
					text := entry.Text
					// let's check the receiver
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						// show the user the error
						showError(err, program.encryption)

						counterparty.SetText("")
						return
					}

					// get your secret as big int
					secret_key := program.wallet.Get_Keys().Secret.BigInt()

					// get the recipient pub key
					reciever_pub_key := addr.PublicKey.G1()

					// make a shared key
					shared_key := crypto.GenerateSharedSecret(secret_key, reciever_pub_key)

					// encrypt the file using the shared key
					text_bytes := []byte(text)
					crypto.EncryptDecryptUserData(shared_key, text_bytes)

					entry.SetText(base64.StdEncoding.EncodeToString(text_bytes))
				}
			}
			// use the main window for the encrypt
		}

		// create a simple form
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

		// set the content and the callback
		e = dialog.NewForm("Encrypt File?", confirm, dismiss, content, callback, program.encryption)
		e.Resize(password_size)
		e.Show()
	}
	// set the function
	encrypt.OnTapped = onTapped

	// let's decrypt a file
	decrypt := widget.NewHyperlink("decrypt", nil)

	// reassign the on tapped
	onTapped = func() {
		var d *dialog.FormDialog
		pass.OnSubmitted = func(s string) {
			d.Submit()
			d.Dismiss()
		}

		callback := func(b bool) {
			// if they cancel
			if !b {
				return
			}
			// let's validate the address real quick
			if err := counterparty.Validate(); err != nil {
				showError(err, program.encryption)

				return
			}
			// get the pass
			p := pass.Text

			// dump the password
			pass.SetText("")

			// check the password
			if !program.wallet.Check_Password(p) {
				showError(errors.New("wrong password"), program.encryption)

				file_entry.SetText("")
			} else {
				if entry.Disabled() {

					// get the filename
					filename := file_entry.Text

					// check if it is an .enc file
					if !strings.HasSuffix(filename, ".enc") {
						showError(errors.New("not a .enc file"), program.encryption)

						return
					}

					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err, program.encryption)

						return
					}

					// check the receiver address
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						showError(err, program.encryption)

						return
					}

					// get the wallet's secret key as a big int
					secret_key := program.wallet.Get_Keys().Secret.BigInt()

					// use the reciever pub key
					reciever_pub_key := addr.PublicKey.G1()

					// create a shared key
					shared_key := crypto.GenerateSharedSecret(secret_key, reciever_pub_key)

					data, err := base64.StdEncoding.DecodeString(string(file))

					if err != nil {
						showError(err, program.encryption)

						return
					}

					// decrypt the file with the key
					crypto.EncryptDecryptUserData(shared_key, data)

					// trim the .enc suffix
					save_path := strings.TrimSuffix(filename, ".enc")

					// write the file to disk
					os.WriteFile(save_path, data, default_file_permissions)

					// let's make another notice
					notice := "File successfully decrypted\n" +
						"Message saved as " + save_path

					// load the notice in the dialog
					e := dialog.NewInformation("Decrypt",
						notice,
						program.encryption)

					//resize and show
					e.Resize(program.size)
					e.Show()
					return
				} else if !entry.Disabled() {
					text := entry.Text
					// let's check the receiver
					addr, err := rpc.NewAddress(program.receiver)
					if err != nil {
						// show the user the error
						showError(err, program.encryption)

						counterparty.SetText("")
						return
					}
					text_bytes, err := base64.StdEncoding.DecodeString(text)
					if err != nil {
						showError(err, program.encryption)

						return
					}
					// get your secret as big int
					secret_key := program.wallet.Get_Keys().Secret.BigInt()

					// get the recipient pub key
					reciever_pub_key := addr.PublicKey.G1()

					// make a shared key
					shared_key := crypto.GenerateSharedSecret(secret_key, reciever_pub_key)

					// encrypt the file using the shared key
					crypto.EncryptDecryptUserData(shared_key, text_bytes)

					entry.SetText(string(text_bytes))
				}
			}
		}

		// use a simple form
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

		// set callback and content
		d = dialog.NewForm("Decrypt File?", confirm, dismiss, content, callback, program.encryption)
		d.Resize(password_size)
		d.Show()
	}

	// set the onTapped callback
	decrypt.OnTapped = onTapped

	// let's make sure that we validate the address we use
	counterparty.Validator = addressValidator

	// let's also make a notice
	notice := "Asymetrically encrypt/decrypt files and text. "
	notice += "Select file, or freetype text. to encrypt/decrypt and enter the address of the counterparty user. "
	notice += "Text is base64Encoded, eg. 5vIlTk1XpQM3OOSkhw== "

	// make the label
	label := makeCenteredWrappedLabel(notice)

	// let's make a nice content screen
	content := container.NewVBox(
		layout.NewSpacer(),
		widget.NewForm(
			widget.NewFormItem("", entry),
			widget.NewFormItem("", file_entry),
			widget.NewFormItem("", counterparty),
		),
		container.NewAdaptiveGrid(2,
			container.NewCenter(encrypt),
			container.NewCenter(decrypt),
		),
		label,
		layout.NewSpacer(),
	)
	return content
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

	// create a callback function
	callback := func(b bool) {
		// if they cancel
		if !b {
			return
		}

		// start a sync activity widget
		// syncing := widget.NewActivity()
		// syncing.Start()
		notice := makeCenteredWrappedLabel("Beginning Scan")
		prog := widget.NewProgressBar()
		content := container.NewVBox(
			layout.NewSpacer(),
			prog,
			// syncing,
			notice,
			layout.NewSpacer(),
		)
		// set it to a splash screen
		syncro := dialog.NewCustomWithoutButtons("syncing", content, program.window)

		// resize and show
		syncro.Resize(program.size)
		syncro.Show()

		// rebuild the hash list
		buildAssetHashList()

		prog.Min = 0
		prog.Max = 1

		// as a go routine
		go func() {

			// clean the wallet
			program.wallet.Clean()

			// it will be helpful to know when we are done
			var done bool

			// as a go routine...
			go func() {

				// keep track of the start
				var start int

				// now we are going to have this spin every second
				ticker := time.NewTicker(time.Millisecond * 300)
				for range ticker.C {
					h := getDaemonInfo().Height

					// if we are done, break this loop
					if done {
						break
					}

					// get transfers
					transfers := getTransfersByHeight(
						uint64(start), uint64(h),
						crypto.ZEROHASH,
						true, true, true,
					)

					// measure them
					current_len := len(transfers)

					if current_len == 0 {
						continue
					}

					// set the start higher up the chain
					end_of_index := current_len - 1
					start = int(transfers[end_of_index].Height)
					ratio := float64(start) / float64(h)
					fyne.DoAndWait(func() {
						prog.SetValue(ratio)
					})

					// now spin through the transfers at the point of difference
					for _, each := range transfers {

						// update the notice
						fyne.DoAndWait(func() {
							notice.SetText("Blockheight: " + strconv.Itoa(int(each.Height)) + " Timestamp: " + each.Time.String())
						})

						// take a small breather between updates
						time.Sleep(time.Millisecond)
					}

					// set notice to a default
					// fyne.DoAndWait(func() {
					// 	notice.SetText("Retrieving more txs")
					// })
				}
			}()

			// then sync the wallet for DERO
			if err := program.wallet.Sync_Wallet_Memory_With_Daemon(); err != nil {
				// if there is an error, notify the user
				showError(err, program.window)
				return
			} else {
				// now range through each token in the cache one at a time
				desired := 1
				capacity_channel := make(chan struct{}, desired)
				var wg sync.WaitGroup
				wg.Add(len(program.caches.assets))
				for _, asset := range program.caches.assets {
					go func() {
						new_job := struct{}{}

						capacity_channel <- new_job
						defer wg.Done()
						// assume there could be an error
						var err error

						// then add each scid back to the map
						hash := crypto.HashHexToHash(asset.hash)
						if err = program.wallet.TokenAdd(hash); err != nil {
							// if err, show it
							showError(err, program.window)
							// but don't stop, just continue the loop
							return
						}

						// and then sync scid internally with the daemon
						if err = program.wallet.Sync_Wallet_Memory_With_Daemon_internal(hash); err != nil {
							// if err, show it
							showError(err, program.window)
							// but don't stop, just continue the loop
							return
						}

						<-capacity_channel
					}()
				}
				wg.Wait()
			}
			// when done, shut down the sync status in the go routine
			fyne.DoAndWait(func() {
				done = true
				// syncing.Stop()
				syncro.Dismiss()
			})
		}()

	}

	// here is the rescan dialog
	rescan := dialog.NewConfirm("Balance Rescan", big_notice, callback, program.window) // set to the main window

	// resize and show
	rescan.Resize(program.size)
	rescan.Show()
}

// this is going to be a rudimentary explorer at first
func explorer() {
	// let's start with the stats tab
	stats := []string{
		strconv.Itoa(int(program.caches.info.Height)),
		strconv.Itoa(int(program.caches.info.AverageBlockTime50)),
		strconv.Itoa(int(program.caches.info.Tx_pool_size)),
		strconv.Itoa(int(program.caches.info.Difficulty) / 1000000),
		strconv.Itoa(int(program.caches.info.Total_Supply)),
		program.caches.info.Status,
	}
	diff := widget.NewLabel("Network Height: " + stats[0])
	average_blocktime := widget.NewLabel("Network Blocktime: " + stats[1] + " seconds")
	mem_pool := widget.NewLabel("Mempool Size: " + stats[2])
	hash_rate := widget.NewLabel("Hash Rate: " + stats[3] + " MH/s")
	supply := widget.NewLabel("Total Supply: " + stats[4])
	network_status := widget.NewLabel("Network Status: " + stats[5])

	// we are going to need this for our graph
	diff_map := map[int]int{}
	updateDiffData := func() {
		// don't do more than this...
		const limit = 100

		// concurrency!
		var wg sync.WaitGroup
		var mu sync.RWMutex
		var threads = runtime.GOMAXPROCS(0) - 2
		var capacity = make(chan struct{}, threads)

		wg.Add(limit)
		for i := range limit {
			func(i int) {
				defer wg.Done()
				capacity <- struct{}{}
				h := uint64(getDaemonInfo().TopoHeight) - (uint64(i))

				_, exists := diff_map[int(h)]

				if exists {
					mu.Lock()
					if len(diff_map) >= limit {
						delete(diff_map, int(h)-limit)
					}
					mu.Unlock()
					return
				}

				tx := getBlockInfo(rpc.GetBlock_Params{
					Height: h,
				})

				b, err := hex.DecodeString(tx.Blob)
				if err != nil {
					return
				}

				var bl block.Block
				err = bl.Deserialize(b)
				if err != nil {
					return
				}

				i, e := strconv.Atoi(tx.Block_Header.Difficulty)
				if e != nil {
					return
				}
				diff_map[int(bl.Height)] = i
				<-capacity
			}(i)
		}
		wg.Wait()
	}

	updateDiffData()
	if len(diff_map) <= 0 {
		showError(errors.New("failed to collect data, please check connection and try again"), program.window)
		return
	}
	g := &graph{hd_map: diff_map}
	g.ExtendBaseWidget(g)
	diff_graph := g
	contains_stats := container.NewBorder(
		container.NewVBox(container.NewAdaptiveGrid(3,
			diff,
			average_blocktime,
			mem_pool,
			hash_rate,
			supply,
			network_status,
		)),
		nil,
		container.NewStack(diff_graph),
		nil, nil,
	)
	tab_stats := container.NewTabItem("STATS", contains_stats)

	// speaking of tabs...
	var tabs *container.AppTabs
	var searchTab *container.TabItem

	var searchData, searchHeaders []string
	var results_table *widget.Table

	searchBlockchain := func(s string) {
		results_table.ScrollToTop()
		results_table.ScrollToLeading()
		// You should probably log or handle this error
		// results_index = nil
		searchHeaders = []string{"NO BLOCK DATA"}
		searchData = []string{"NO DATA"}

		buildBlockResults := func(r rpc.GetBlock_Result) {
			var bl block.Block
			b, _ := hex.DecodeString(r.Blob)
			bl.Deserialize(b)

			searchHeaders = search_headers_block

			var previous_block string
			if len(r.Block_Header.Tips) > 0 {
				previous_block = r.Block_Header.Tips[0]
			}

			var size = uint64(len(bl.Serialize()))
			var hashes []string
			var types []string
			if r.Block_Header.TXCount != 0 {
				for _, each := range bl.Tx_hashes {
					r := getTransaction(rpc.GetTransaction_Params{
						Tx_Hashes: []string{each.String()},
					})
					for _, each := range r.Txs_as_hex {
						b, e := hex.DecodeString(each)
						if e != nil {
							continue
						}
						var tx transaction.Transaction
						if err := tx.Deserialize(b); err != nil {
							continue
						}
						size += uint64(len(tx.Serialize()))
						types = append(types, tx.TransactionType.String())
						hashes = append(hashes, tx.GetHash().String())
					}
				}
			}

			searchData = []string{
				strconv.Itoa(int(r.Block_Header.TopoHeight)),
				strconv.Itoa(int(r.Block_Header.Height)),
				r.Block_Header.Hash,
				previous_block,
				strconv.Itoa(int(r.Block_Header.Timestamp)),
				time.Unix(0, int64(r.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
				time.Duration((uint64(time.Now().UTC().UnixMilli()) - r.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
				strconv.Itoa(int(r.Block_Header.Major_Version)) + "." + strconv.Itoa(int(r.Block_Header.Minor_Version)),
				fmt.Sprintf("%0.5f", float64(r.Block_Header.Reward)/atomic_units),
				fmt.Sprintf("%.03f", float32(size)/float32(kilobyte)),
				strconv.Itoa(len(bl.MiniBlocks)),
				strconv.Itoa(int(r.Block_Header.Depth)),
			}

			searchHeaders = append(searchHeaders, "TX COUNT")
			searchData = append(searchData, strconv.Itoa(int(r.Block_Header.TXCount)))
			searchHeaders = append(searchHeaders, types...)
			searchData = append(searchData, hashes...)

			searchHeaders = append(searchHeaders, []string{
				"MINING OUTPUTS",
			}...)

			miners := []string{}
			miners = append(miners, r.Block_Header.Miners...)
			var rewards uint64
			if len(miners) != 0 {
				for range len(miners) {
					searchHeaders = append(searchHeaders, " ")
				}
			} else { // here you go, capt...

				var acckey crypto.Point
				if err := acckey.DecodeCompressed(bl.Miner_TX.MinerAddress[:]); err != nil {
					panic(err)
				}

				address := rpc.NewAddressFromKeys(&acckey)
				address.Mainnet = program.preferences.Bool("mainnet")
				miners = append(miners, address.String())
				rewards += bl.Miner_TX.Value
			}

			searchData = append(searchData, []string{
				fmt.Sprintf("%0.5f", float64(rewards+r.Block_Header.Reward)/atomic_units),
				"MINER ADDRESS",
			}...)
			searchData = append(searchData, miners...)

		}
		//pre-processing
		switch len(s) {
		case 64: // this is basically any hash
			r := getBlockInfo(
				rpc.GetBlock_Params{
					Hash: s,
				},
			)
			// -32098: file does not exist
			// at this point we have to determine if...
			if len(r.Block_Header.Miners) != 0 {
				buildBlockResults(r)
			} else {
				// not a mined transaction
				r := getTransaction(
					rpc.GetTransaction_Params{
						Tx_Hashes: []string{s},
					},
				)
				// fmt.Printf("tx: %+v\n", r)
				if len(r.Txs_as_hex) < 1 {
					// goto end
				}
				b, err := hex.DecodeString(r.Txs_as_hex[0])
				if err != nil {
					// goto end
				}
				var tx transaction.Transaction
				if err := tx.Deserialize(b); err != nil {
					// goto end
				}

				// we should encapsulate this logic
				block_info := getBlockInfo(rpc.GetBlock_Params{
					Hash: r.Txs[0].ValidBlock,
				})

				bl := getBlockDeserialized(block_info.Blob)

				switch tx.TransactionType {
				case transaction.PREMINE:
				case transaction.REGISTRATION:

					searchHeaders = search_headers_registration

					var acckey crypto.Point
					if err := acckey.DecodeCompressed(tx.MinerAddress[:]); err != nil {
						panic(err)
					}

					address := rpc.NewAddressFromKeys(&acckey)
					address.Mainnet = program.preferences.Bool("mainnet")

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(),
						bl.GetHash().String(),
						address.String(),
						"TRUE",
					}

					results_table.Refresh()
					// searchHeaders = append(searchHeaders, )
				case transaction.COINBASE: // these aren't shown in the explorer; and sc interactions are...
				case transaction.NORMAL:

					searchHeaders = search_headers_normal

					var ring_members []string
					var outputs []string
					for i, each := range r.Txs[0].Ring {
						searchHeaders = append(searchHeaders, "OUTPUT "+strconv.Itoa(i+1))
						outputs = append(outputs, "RING MEMBERS")
						for i, member := range each {
							searchHeaders = append(searchHeaders, "Ring Member "+strconv.Itoa(i+1))
							ring_members = append(ring_members, member)
						}
					}

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(),
						r.Txs[0].ValidBlock,
						fmt.Sprintf("%x", tx.BLID),
						strconv.Itoa(int(tx.Height)),
						fmt.Sprintf("%x", tx.Payloads[0].Statement.Roothash[:]),
						strconv.Itoa(int(block_info.Block_Header.Timestamp)),
						time.Unix(0, int64(block_info.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
						time.Duration((uint64(time.Now().Local().UnixMilli()) - block_info.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
						strconv.Itoa(int(block_info.Block_Header.TopoHeight)),
						fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
						fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/float32(kilobyte)),
						strconv.Itoa(int(tx.Version)),
						strconv.Itoa(int(block_info.Block_Header.Depth)),
						"DERO_HOMOMORPHIC",
						strconv.Itoa(int(len(r.Txs[0].Ring))),
						strconv.Itoa(int(float64(len(ring_members)) / float64(len(r.Txs[0].Ring)))),
					}
					for i, each := range outputs {
						searchData = append(searchData, each)
						searchData = append(searchData, r.Txs[0].Ring[i]...)
					}

					results_table.Refresh()
				case transaction.BURN_TX:
					// I haven't seen any of these yet...
				case transaction.SC_TX:

					searchHeaders = search_headers_sc_prefix

					// headers := []string{}
					sc := getSC(rpc.GetSC_Params{
						SCID:      s,
						Code:      true,
						Variables: true,
					})

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(), //
						block_info.Block_Header.Hash,
						"ATOMIC AMOUNTS",
					}

					for k, v := range sc.Balances {
						searchHeaders = append(searchHeaders, k)
						searchData = append(searchData, strconv.Itoa(int(v)))
					}
					searchHeaders = append(searchHeaders, "STRING VARS")
					searchData = append(searchData, "STRING VALUES")
					type string_pair struct {
						k string
						v string
					}
					var string_pairs []string_pair
					for k, v := range sc.VariableStringKeys {
						var value string
						switch val := v.(type) {
						case string:
							if k != "C" {
								b, e := hex.DecodeString(val)
								if e != nil {
									continue
								}
								value = string(b)
							} else {
								value = truncator(val)
							}
						case uint64:
							value = strconv.Itoa(int(val))
						case float64:
							value = strconv.FormatFloat(val, 'f', 0, 64)
						}
						string_pairs = append(string_pairs, string_pair{
							k: k,
							v: value,
						})
					}
					sort.Slice(string_pairs, func(i, j int) bool {
						return string_pairs[i].k > string_pairs[j].k
					})
					for _, each := range string_pairs {
						searchHeaders = append(searchHeaders, each.k)
						searchData = append(searchData, each.v)
					}
					searchHeaders = append(searchHeaders, "UINT64 VARS")
					searchData = append(searchData, "UINT64 VALUES")
					type uint64_pair struct {
						k string
						v string
					}
					var uint64_pairs []uint64_pair
					for k, v := range sc.VariableUint64Keys {
						var value string
						switch val := v.(type) {
						case string:
							b, e := hex.DecodeString(val)
							if e != nil {
								continue
							}
							value = string(b)
						case uint64:
							value = strconv.Itoa(int(val))
						case float64:
							value = strconv.FormatFloat(val, 'f', 0, 64)
						}
						uint64_pairs = append(uint64_pairs, uint64_pair{
							k: strconv.Itoa(int(k)),
							v: value,
						})
					}
					sort.Slice(uint64_pairs, func(i, j int) bool {
						return uint64_pairs[i].k > uint64_pairs[j].k
					})
					for _, each := range uint64_pairs {
						searchHeaders = append(searchHeaders, each.k)
						searchData = append(searchData, each.v)
					}

					searchHeaders = append(searchHeaders, search_headers_sc_body...)

					var ring_members []string

					for _, each := range r.Txs[0].Ring {
						ring_members = append(ring_members, each...)
					}

					searchData = append(searchData, []string{
						fmt.Sprintf("%x", tx.BLID),
						fmt.Sprintf("%x", tx.Payloads[0].Statement.Roothash[:]),
						strconv.Itoa(int(block_info.Block_Header.Height)),
						strconv.Itoa(int(block_info.Block_Header.Timestamp)),
						time.Unix(0, int64(block_info.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
						time.Duration((uint64(time.Now().Local().UnixMilli()) - block_info.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
						strconv.Itoa(int(block_info.Block_Header.TopoHeight)),
						fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
						fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/float32(kilobyte)), // we need to break this down as before
						strconv.Itoa(int(tx.Version)),
						strconv.Itoa(int(block_info.Block_Header.Depth)),
						"DERO_HOMOMORPHIC",
						strconv.Itoa(len(ring_members)),
						r.Txs[0].Signer,
						"RING MEMBERS",
					}...)
					for range ring_members {
						searchHeaders = append(searchHeaders, "")
					}
					searchData = append(searchData, ring_members...)
					searchHeaders = append(searchHeaders, "SC BALANCE") // in DERO
					searchData = append(searchData, rpc.FormatMoney(sc.Balance))

					searchHeaders = append(searchHeaders, []string{
						"SC CODE",
						"SC ARGS",
					}...)
					searchData = append(searchData, []string{
						sc.Code,
						fmt.Sprintf("%+v", tx.SCDATA),
					}...)
					results_table.Refresh()

				default:
				}
			}
		case 66: // dero1 addresses?
		default: // this is going to be a height, or a wallet address or... I mean, what do we search for on the blockchain?
			// I mean, now we are having to search things like tela...

			i, err := strconv.Atoi(s)
			if err != nil {
				return
			}
			r := getBlockInfo(
				rpc.GetBlock_Params{
					Height: uint64(i),
				},
			)
			if r.Blob != "" {
				buildBlockResults(r)
			}
		}

		results_table.SetColumnWidth(0, largestMinSize(searchHeaders).Width)
		results_table.SetColumnWidth(1, largestMinSize(searchData).Width)
		results_table.Refresh()
	}

	pool_label_data := [][]string{}

	updatePoolCache := func() {

		pool_label_data = [][]string{}
		pool := program.caches.pool
		if len(pool.Tx_list) <= 0 {
			return
		}
		for i := range pool.Tx_list {

			transfer := getTransaction(rpc.GetTransaction_Params{
				Tx_Hashes: []string{pool.Tx_list[i]},
			})
			var tx transaction.Transaction
			decoded, _ := hex.DecodeString(transfer.Txs_as_hex[0])

			if err := tx.Deserialize(decoded); err != nil {
				continue
			}
			var size int
			for _, each := range transfer.Txs {
				size += len(each.Ring)
			}

			// Build data row
			pool_label_data = append(pool_label_data, []string{
				strconv.Itoa(int(tx.Height)),
				tx.GetHash().String(),
				fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
				strconv.Itoa(size),
				fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/1024),
			})
		}
	}
	updatePoolCache()
	var pool_table *widget.Table

	lengthPool := func() (rows int, cols int) {
		return len(program.caches.pool.Tx_list), len(pool_headers)
	}

	createPool := func() fyne.CanvasObject {
		return container.NewStack(
			widget.NewLabel(""), // For regular text
			container.NewScroll(widget.NewHyperlink("", nil)), // For clickable hash
		)
	}

	updatePool := func(tci widget.TableCellID, co fyne.CanvasObject) {
		pool_data := pool_label_data
		cell := co.(*fyne.Container)
		label := cell.Objects[0].(*widget.Label)
		scroll := cell.Objects[1].(*container.Scroll)
		link := scroll.Content.(*widget.Hyperlink)
		if len(pool_data) == 0 {
			label.SetText("")
			label.Hide()
			link.SetText("")
			link.Hide()
			return
		}
		switch tci.Col {
		case 0, 2, 3, 4:
			label.Show()
			scroll.Hide()
			if tci.Row >= len(pool_data) {
				label.SetText("")
			} else {
				label.SetText(pool_data[tci.Row][tci.Col])
			}

		case 1:
			label.Hide()
			scroll.Show()
			if tci.Row >= len(pool_data) {
				link.SetText("")
			} else {
				link.SetText(pool_data[tci.Row][tci.Col])
				link.OnTapped = func() {
					hash := pool_data[tci.Row][tci.Col]
					searchBlockchain(hash)
					results_table.Refresh()
					tabs.Select(searchTab)
				}
			}

		}

	}

	pool_table = widget.NewTable(lengthPool, createPool, updatePool)
	pool_table.ShowHeaderRow = true
	pool_table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}

	pool_table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		// fmt.Println(id)
		if id.Col >= 0 && id.Col < len(pool_headers) {

			template.(*widget.Label).SetText(pool_headers[id.Col])
		}
	}

	for i := range pool_headers {
		pool_table.SetColumnWidth(i, largestMinSize(pool_headers).Width)
	}

	block_label_data := [][]string{}
	limit := 10

	var block_table *widget.Table
	updateBlocksData := func() {

		block_label_data = [][]string{}
		height := program.caches.info.TopoHeight
		// we are going to take the last ten blocks,
		// like... the last 3 minutes

		for i := 1; i <= limit; i++ {
			h := uint64(height) - uint64(i)

			tx_label_data := [][]string{}

			result := getBlockInfo(rpc.GetBlock_Params{Height: h})
			var bl block.Block
			b, err := hex.DecodeString(result.Blob)
			if err != nil {
				continue
			}
			bl.Deserialize(b)

			tx_results, transactions := func() (txs []rpc.GetTransaction_Result, transactions []transaction.Transaction) {
				for _, each := range bl.Tx_hashes {
					tx := getTransaction(
						rpc.GetTransaction_Params{
							Tx_Hashes: []string{each.String()},
						},
					)
					if len(tx.Txs_as_hex) == 0 {
						continue // there is nothing here ?
					}

					txs = append(txs, tx)
					var transaction transaction.Transaction
					b, err := hex.DecodeString(tx.Txs_as_hex[0])
					if err != nil {
						fmt.Println(err)
						continue // lol
					}
					transaction.Deserialize(b)
					transactions = append(transactions, transaction)
				}
				return
			}()
			size := uint64(len(bl.Serialize()))
			if len(tx_results) != 0 {
				for i := range tx_results {
					size += uint64(len(transactions[i].Serialize()))
					var rings uint64
					if len(transactions[i].Payloads) > 0 { // not sure when this wouldn't be the case...
						rings = uint64(len(tx_results[i].Txs[0].Ring[0]))
					}
					tx_label_data = append(tx_label_data, []string{
						"", "", "", "", "",
						transactions[i].GetHash().String(),
						transactions[i].TransactionType.String(),
						fmt.Sprintf("%0.5f", float64(transactions[i].Fees())/atomic_units),
						strconv.Itoa(int(rings)),
						fmt.Sprintf("%0.3f", float64(len(transactions[i].Serialize()))/kilobyte),
					})
				}
			}
			block_label_data = append(block_label_data, []string{
				strconv.Itoa(int(result.Block_Header.Height)),
				strconv.Itoa(int(result.Block_Header.TopoHeight)),
				time.Duration((uint64(time.Now().Local().UnixMilli()) - bl.Timestamp) * uint64(time.Millisecond)).String(),
				strconv.Itoa(len(bl.MiniBlocks)),
				fmt.Sprintf("%0.3f", float64(size)/kilobyte),
				bl.GetHash().String(),
				"BLOCK",
				fmt.Sprintf("%0.5f", float64(result.Block_Header.Reward)/atomic_units),
				"N/A",
				fmt.Sprintf("%0.3f", float64(len(bl.Miner_TX.Serialize()))/kilobyte),
			})
			if len(tx_label_data) > 0 {
				block_label_data = append(block_label_data, tx_label_data...)
			}
		}
	}
	updateBlocksData()

	lengthBlocks := func() (rows int, cols int) {
		return len(block_label_data), len(block_headers)
	}
	createBlocks := func() fyne.CanvasObject {
		return container.NewStack(
			widget.NewLabel(""),
			container.NewScroll(widget.NewHyperlink("", nil)),
		)
	}
	updateBlocks := func(tci widget.TableCellID, co fyne.CanvasObject) {
		block_data := block_label_data
		cell := co.(*fyne.Container)
		label := cell.Objects[0].(*widget.Label)
		scroll := cell.Objects[1].(*container.Scroll)
		link := scroll.Content.(*widget.Hyperlink)
		if len(block_label_data) == 0 {
			label.SetText("")
			label.Hide()
			link.SetText("")
			link.Hide()
			return
		}
		switch tci.Col {
		case 0, 1, 2, 3, 4, 6, 7, 8, 9:
			label.SetText(block_data[tci.Row][tci.Col])
			label.Show()
			scroll.Hide()
		case 5:
			label.Hide()
			scroll.Show()
			link.SetText(block_data[tci.Row][tci.Col])
			link.OnTapped = func() {
				hash := block_data[tci.Row][tci.Col]
				searchBlockchain(hash)
				results_table.Refresh()
				tabs.Select(searchTab)
			}
		}

	}

	block_table = widget.NewTable(lengthBlocks, createBlocks, updateBlocks)
	block_table.ShowHeaderRow = true
	block_table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	block_table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		if id.Col >= 0 && id.Col < len(block_headers) {
			template.(*widget.Label).SetText(block_headers[id.Col])
		}
	}
	for i := range block_headers {
		block_table.SetColumnWidth(i, largestMinSize(block_headers).Width)
	}

	search := widget.NewEntry()

	lengthSearch := func() (rows int, cols int) { return len(searchData), 2 }
	createSearch := func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.SetText("THIS IS A PLACEHOLDER FOR THE APPLICATION")
		l.Wrapping = fyne.TextWrapOff
		return container.NewStack(l)
	}
	updateSearch := func(id widget.TableCellID, template fyne.CanvasObject) {
		box := template.(*fyne.Container)
		l := box.Objects[0].(*widget.Label)

		switch id.Col {
		case 0:
			if id.Row >= len(searchHeaders) {
				l.SetText("")
			} else {
				text := searchHeaders[id.Row]
				l.SetText(text)
				l.Refresh()
			}
		case 1:
			if id.Row >= len(searchData) {
				l.SetText("")
			} else {
				text := searchData[id.Row]
				l.SetText(text)
				l.Refresh()
				if id.Row < len(searchHeaders) && (searchHeaders[id.Row] == "SC CODE" || searchHeaders[id.Row] == "SC ARGS") {
					sizing := l.MinSize().Height + (theme.Padding() * 2)
					results_table.SetRowHeight(id.Row, sizing)
					l.Refresh()
				}
			}
		default:
			l.SetText("ERROR")
		}
	}
	results_table = widget.NewTable(
		lengthSearch, createSearch, updateSearch,
	)
	results_table.OnSelected = func(id widget.TableCellID) {
		var data string
		if id.Col == 0 {
			data = searchHeaders[id.Row]
			program.application.Clipboard().SetContent(data)
			results_table.UnselectAll()
			results_table.Refresh()
			showInfoFast("Copied", data, program.explorer)
		} else {
			data = searchData[id.Row]
			program.application.Clipboard().SetContent(data)
			results_table.UnselectAll()
			results_table.Refresh()
			showInfoFast("Copied", data, program.explorer)
		}
	}

	tapped := func() {
		if search.Text == "" {
			return
		}
		s := search.Text
		search.SetText("")
		searchBlockchain(s)
	}

	search.ActionItem = widget.NewButtonWithIcon("search", theme.SearchIcon(), tapped)
	search.OnSubmitted = func(s string) {
		tapped()
	}
	searchBar := container.NewVBox(search)

	var updating bool = true

	go func() {
		height := program.caches.info.TopoHeight
		for range time.NewTicker(time.Second * 2).C {
			if updating {
				if height != program.caches.info.TopoHeight {
					height = program.caches.info.TopoHeight

					updateDiffData()

					newGraph := &graph{hd_map: diff_map}
					g.ExtendBaseWidget(g)

					// Replace content in graph container
					diff_graph = newGraph

					updatePoolCache()

					updateBlocksData()

					fyne.DoAndWait(func() {
						diff_graph.Refresh()
						pool_table.Refresh()
						block_table.Refresh()
					})

					stats = []string{
						strconv.Itoa(int(program.caches.info.Height)),
						strconv.Itoa(int(program.caches.info.AverageBlockTime50)),
						strconv.Itoa(int(program.caches.info.Tx_pool_size)),
						strconv.Itoa(int(program.caches.info.Difficulty) / 1000000),
						strconv.Itoa(int(program.caches.info.Total_Supply)),
						program.caches.info.Status,
					}
					fyne.DoAndWait(func() {
						diff.SetText("Network Height: " + stats[0])
						average_blocktime.SetText("Network Blocktime: " + stats[1] + " seconds")
						mem_pool.SetText("Mempool Size: " + stats[2])
						hash_rate.SetText("Hash Rate: " + stats[3] + " MH/s")
						supply.SetText("Total Supply: " + stats[4])
						network_status.SetText("Network Status: " + stats[5])
					})

				}
			} else {
				return
			}
		}
	}()
	searchTab = container.NewTabItem("Search", container.NewBorder(
		searchBar,     // top
		nil,           // bottom
		nil,           // left
		nil,           // right
		results_table, // center
	))
	tabs = container.NewAppTabs(
		tab_stats,
		container.NewTabItem("TX Pool", container.NewAdaptiveGrid(1,
			pool_table,
		)),
		container.NewTabItem("Recent Blocks", container.NewAdaptiveGrid(1,
			block_table,
		)),
		searchTab,
	)

	tabs.SetTabLocation(container.TabLocationLeading)
	program.explorer = fyne.CurrentApp().NewWindow(program.name + " | viewer ")
	program.explorer.Resize(program.size)
	program.explorer.SetIcon(theme.SearchIcon())
	explore := dialog.NewCustomWithoutButtons("Explorer",
		tabs,
		program.explorer,
	)
	program.explorer.SetOnClosed(func() {
		updating = false
		explore.Dismiss()
	})
	program.explorer.Show()
	explore.Resize(program.size)
	explore.Show()
}

func installer() *fyne.Container {

	// let's make it easy write a contract on the fly

	entry := widget.NewEntry()
	entry.SetPlaceHolder("...or free write contract here")
	entry.MultiLine = true
	entry.SetMinRowsVisible(10)
	validate_string := func(s string) error {

		if s == "" {
			return nil
		}

		// if it is a parsable smart contract, green light it
		if _, _, err := dvm.ParseSmartContract(s); err != nil {

			// otherwise, return err
			return err
		}

		// validated
		return nil
	}
	entry.Validator = validate_string
	importer := widget.NewEntry()
	importer.SetPlaceHolder("import code using SCID")
	importer.Validator = func(s string) error {
		if s == "" {
			return nil
		}
		r := getSC(rpc.GetSC_Params{
			SCID: s,
			Code: true,
		})
		if r.Code != "" {
			// if it is a parsable smart contract, green light it
			if _, _, err := dvm.ParseSmartContract(r.Code); err != nil {

				// otherwise, return err
				return err
			}
		}
		return nil
	}
	importer.OnChanged = func(s string) {
		if s == "" {
			return
		}
		if err := importer.Validate(); err != nil {
			return
		}
		r := getSC(rpc.GetSC_Params{
			SCID: s,
			Code: true,
		})
		entry.SetText(r.Code)
	}
	// here is a simple way to select a file in general
	program.dialogues.open = openExplorer(program.contracts)

	// let's make an simple way to open files
	program.entries.file.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		program.dialogues.open.Resize(program.size)
		program.dialogues.open.Show()
	})

	program.entries.file.SetPlaceHolder("/path/to/contract.bas")
	// let's validate that file, shall we?
	validate_path := func(s string) error {

		if s == "" {
			return nil
		}

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

	// and set it
	program.entries.file.Validator = validate_path
	program.entries.file.OnChanged = func(s string) {
		if s == "" {
			return
		}
		if err := program.entries.file.Validate(); err != nil {
			return
		}
		b, err := os.ReadFile(program.entries.file.Text)
		if err != nil {
			return
		}
		text := string(b)
		entry.SetText(text)
	}
	// we just use this because simplicity
	ringsize := uint64(2)

	// we do enable this, and we notify the user
	isAnonymous := widget.NewCheck("is anonymous?", func(b bool) {
		if b {
			ringsize = 16
		}
	})

	// let's walk through install as a callback
	callback := func() {

		var ic *dialog.ConfirmDialog
		// if they press enter, we assume it means confirm
		program.entries.pass.OnSubmitted = func(s string) {
			ic.Confirm()
			ic.Dismiss()
		}

		// create a callback
		callback := func(b bool) {
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
				showError(errors.New("wrong password"), program.contracts)
				program.entries.file.SetText("")
				return
			} else {
				// get the filename
				filename := program.entries.file.Text
				// dump the entry
				program.entries.file.SetText("")

				var upload string
				if filename == "" && entry.Text == "" {
					return
				} else if filename != "" && entry.Text == "" {
					// read the file
					file, err := os.ReadFile(filename)
					if err != nil {
						showError(err, program.contracts)
						return
					}

					// coerce the file into string
					upload = string(file)
				} else if filename == "" && entry.Text != "" {
					upload = entry.Text
				}

				// parse the file for an error; yes, I know we validated...
				if _, _, err := dvm.ParseSmartContract(upload); err != nil {

					// show them an error if one
					showError(err, program.contracts)
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
					showError(err, program.contracts)
					return
				}

				// ship the tx to the daemon
				if err := program.wallet.SendTransaction(tx); err != nil {

					// notify the user if err
					showError(err, program.contracts)
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
					showInfo("", "txid copied to clipboard", program.contracts)
				}

				// center it
				txid.Alignment = fyne.TextAlignCenter

				// walk the user through success
				success := dialog.NewCustom("Contract Installer", "dissmiss",
					container.NewVBox(
						widget.NewLabel("Contract successfully installed"),
						txid,
					), program.contracts)

				// resize and show
				success.Resize(program.size)
				success.Show()
			}

		}

		// put a window in a window...
		ic = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, program.entries.pass, callback, program.contracts)
		ic.Resize(password_size)
		ic.Show()
	}

	// see, notice
	notice := makeCenteredWrappedLabel("anonymous installs might effect intended SC functionality")

	form := widget.NewForm(
		widget.NewFormItem("", program.entries.file),
		widget.NewFormItem("", container.NewCenter(widget.NewLabel("or"))),
		widget.NewFormItem("", importer),
		widget.NewFormItem("", container.NewCenter(widget.NewLabel("or"))),
		widget.NewFormItem("", entry))

	form.OnSubmit = callback
	form.SubmitText = "install"

	// let's make some content
	content := container.NewVBox(
		layout.NewSpacer(),
		form,
		container.NewCenter(isAnonymous),
		notice,
		layout.NewSpacer(),
	)
	return content
}
func interaction() *fyne.Container {
	function_list := new(widget.List)

	// we are going to have some func names
	var func_names []string

	// and then we are also going to have some dvm function
	var functions []dvm.Function

	// let's make a simple way to review the sc code
	code := widget.NewLabel("")

	// now let's make a way to enter the contract
	scid := widget.NewEntry()

	scid.SetPlaceHolder("Submit SCID here")

	// make a sctring validator
	validate := func(s string) error {

		func_names = []string{}

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

		function_list.Refresh()
		return nil
	}

	// let's validate it
	scid.Validator = validate

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
	function_list.Length = func() int { return len(func_names) }
	function_list.CreateItem = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	function_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {
		co.(*widget.Label).SetText(func_names[lii])
	}

	// make use of a on selected callback
	onSelected := func(id widget.ListItemID) {

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
				showError(errors.New("type is either invalid or none"), program.contracts)
				return
			default:
				showError(errors.New("unknown type"), program.contracts)
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

		// create an interaction callback function
		callback := func(b bool) {
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
							showError(errors.New("multiple token assets not implemented"), program.contracts)
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
						showError(err, program.contracts)
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
						showError(err, program.contracts)
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
						showError(err, program.contracts)
						return
					}

					// coerce the float into an burnable amount
					burn := uint64(float) * atomic_units

					// now check if the best guess is ZEROHASH
					if best_guess.IsZero() {

						// if it is, this is a problem
						showError(errors.New("please report this error to the develop"), program.contracts)
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
				showError(err, program.contracts)
				// but keep going
			}

			// let's get the args for the user to review
			sa := makeCenteredWrappedLabel(string(string_args))

			// load up the splash and a password entry
			splash := container.NewVBox(sa, program.entries.pass)
			var ci *dialog.ConfirmDialog
			// if they press enter here, it is a confirmation
			program.entries.pass.OnSubmitted = func(s string) {
				ci.Confirm()
				ci.Dismiss()
			}

			// create a callback function
			callback := func(b bool) {
				// if they cancel
				if !b {
					return
				}

				// get the pass
				pass := program.entries.pass.Text

				// dump the entry
				program.entries.pass.SetText("")

				if !program.wallet.Check_Password(pass) {
					showError(errors.New("wrong password"), program.contracts)
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
						showError(err, program.contracts)
						return
					}

					// submit the transfer to the daemon
					if err := program.wallet.SendTransaction(tx); err != nil {
						showError(err, program.contracts)
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
						showInfo("", "txid copied to clipboard", program.contracts)
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
						), program.contracts)
					// resize and show
					success.Resize(program.size)
					success.Show()

				}
			}

			// load it to the main window
			ci = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, splash, callback, program.contracts)
			ci.Resize(password_size)
			ci.Show()
		}

		// and walk the user through the argument process
		arguments := dialog.NewCustomConfirm("Interact", confirm, dismiss, splash, callback, program.contracts)

		// resize and show
		arguments.Resize(program.size)
		arguments.Show()
	}

	// let the user select the fuction
	function_list.OnSelected = onSelected

	// we want users to be happy with the code before they interact with it
	notice := "If you are satisfied with the code, "
	notice += "please select the function to be interacted with to be forwarded to the next screen"

	confirmation := makeCenteredWrappedLabel(notice)
	content := container.NewBorder(
		container.NewVBox(scid, confirmation),
		nil, nil, nil,
		container.NewAdaptiveGrid(1,
			container.NewScroll(code),
			function_list,
		),
	)

	return content
}

func addressValidator(s string) (err error) {

	// any changes to the string should immediately update the receiver string
	program.receiver = ""

	// if less than 4 char...
	if s != "" && len(s) < 4 {
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
