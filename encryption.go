package main

import (
	"encoding/base64"
	"errors"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
)

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

// this is a pretty under-rated feature
func filesign() *fyne.Container {
	file_entry := widget.NewEntry()
	// let's make an simple way to open files
	open := openExplorer(file_entry, program.encryption)
	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		open.Show()
	})

	// let's make it noticeable that you can select the file
	file_entry.SetPlaceHolder("/path/to/file.txt")
	// here is a simple way to select a file in general

	pass := widget.NewPasswordEntry()
	pass.SetPlaceHolder("w41137-p@55w0rd")

	// now let's make a sign hyperlink
	sign := widget.NewHyperlink("filesign", nil)

	// and when the user taps it
	onTapped := func() {
		var fs *dialog.FormDialog
		pass.OnSubmitted = func(s string) {
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
			p := pass.Text

			// dump the text
			pass.SetText("")

			// check the password first
			if !program.wallet.Check_Password(p) {

				// let them know if they were wrong
				showError(errors.New("wrong password"), program.encryption)

				// dump the filepath
				file_entry.SetText("")
				return
			} else {
				// get the filename
				filename := file_entry.Text

				//dump the entry
				file_entry.SetText("")

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
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

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
		pass.OnSubmitted = func(s string) {
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
			p := pass.Text

			// dump the text
			pass.SetText("")

			// check the password, every time
			if !program.wallet.Check_Password(p) {

				// show and error when wrong
				showError(errors.New("wrong password"), program.encryption)

				//dump entry
				file_entry.SetText("")

			} else {
				// get the filename
				filename := file_entry.Text
				file_entry.SetText("")

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
		content := []*widget.FormItem{widget.NewFormItem("", pass)}

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
		container.NewVBox(file_entry),
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
	entry.Wrapping = fyne.TextWrapWord
	// another round of make sure this works XD
	file_entry := widget.NewEntry()
	file_entry.SetPlaceHolder("/path/to/file.txt")
	// here is a simple way to select a file in general
	open := openExplorer(file_entry, program.encryption)

	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		open.Resize(program.size)
		open.Show()
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
	entry.Wrapping = fyne.TextWrapWord

	// let's make a simple way to open a file
	file_entry := widget.NewEntry()
	file_entry.SetPlaceHolder("/path/to/file.txt")
	// here is a simple way to select a file in general
	open := openExplorer(file_entry, program.encryption)

	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		open.Resize(program.size)
		open.Show()
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
	// let's make sure that we validate the address we use
	counterparty.Validator = addressValidator

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
			recipient := program.receiver
			program.receiver = ""

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
					addr, err := rpc.NewAddress(recipient)
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
					addr, err := rpc.NewAddress(recipient)
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

			recipient := program.receiver
			program.receiver = ""

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
					addr, err := rpc.NewAddress(recipient)
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
					addr, err := rpc.NewAddress(recipient)
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
