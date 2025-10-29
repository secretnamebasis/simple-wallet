package main

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/dvm"
	"github.com/deroproject/derohe/rpc"
)

func contracts() {
	program.contracts = fyne.CurrentApp().NewWindow(program.window.Title() + " | contracts ")
	program.contracts.SetIcon(theme.FileIcon())
	program.contracts.Resize(program.size)
	tabs := container.NewAppTabs(
		container.NewTabItem("Contract Installer", installer()),
		container.NewTabItem("Contract Interactor", interaction()),
	)
	program.contracts.SetContent(tabs)

	program.contracts.Show()
}

func installer() *fyne.Container {
	pass := widget.NewPasswordEntry()
	pass.SetPlaceHolder("w41137-p@55w0rd")
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
	file_entry := widget.NewEntry()
	// here is a simple way to select a file in general
	open := openExplorer(file_entry, program.contracts)
	file_entry.ActionItem = widget.NewButtonWithIcon("", theme.FolderOpenIcon(), func() {
		open.Resize(program.size)
		open.Show()
	})

	// let's make an simple way to open files

	file_entry.SetPlaceHolder("/path/to/contract.bas")
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
	file_entry.Validator = validate_path
	file_entry.OnChanged = func(s string) {
		if s == "" {
			return
		}
		if err := file_entry.Validate(); err != nil {
			return
		}
		b, err := os.ReadFile(file_entry.Text)
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
		pass.OnSubmitted = func(s string) {
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
			p := pass.Text

			// dump the pass entry
			pass.SetText("")

			// check the password
			if !program.wallet.Check_Password(p) {
				showError(errors.New("wrong password"), program.contracts)
				file_entry.SetText("")
				return
			} else {
				// get the filename
				filename := file_entry.Text
				// dump the entry
				file_entry.SetText("")

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
		ic = dialog.NewCustomConfirm("Confirm Password", confirm, dismiss, pass, callback, program.contracts)
		ic.Resize(password_size)
		ic.Show()
	}

	// see, notice
	notice := makeCenteredWrappedLabel("anonymous installs might affect intended SC functionality")

	form := widget.NewForm(
		widget.NewFormItem("", file_entry),
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
	pass := widget.NewPasswordEntry()
	pass.SetPlaceHolder("w41137-p@55w0rd")
	function_list := new(widget.List)

	// and then we are also going to have some dvm function
	var functions []dvm.Function

	// let's make a simple way to review the sc code
	code := widget.NewLabel("")

	// now let's make a way to enter the contract
	scid := widget.NewEntry()

	scid.SetPlaceHolder("Submit SCID here")

	// make a sctring validator
	validate := func(s string) error {

		functions = []dvm.Function{}

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
			// func_names = append(func_names, name)

			// and add each "exported" fucntion to the functions slice
			functions = append(functions, f)
		}

		function_list.Refresh()
		return nil
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

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

	function_list.Length = func() int { return len(functions) }
	function_list.CreateItem = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	function_list.UpdateItem = func(lii widget.ListItemID, co fyne.CanvasObject) {
		co.(*widget.Label).SetText(functions[lii].Name)
	}

	// make use of a on selected callback
	onSelected := func(id widget.ListItemID) {

		// after they do, unselect it
		function_list.Unselect(id) // release the object

		// we are going to put all the entries into this box
		entries := container.NewVBox()

		// these are not arguments, per se
		var addedMap = make(map[string]bool)

		// iterate over the lines of each function
		for _, lines := range functions[id].Lines {

			// if any of the lines in the function say...
			if slices.Contains(lines, "DEROVALUE") && !addedMap["DEROVALUE"] {
				addedMap["DEROVALUE"] = true
				// add this entry to the entries container
				deroValueEntry := widget.NewEntry()
				deroValueEntry.SetPlaceHolder("dero value: 123.321")
				entries.Add(deroValueEntry)

			}

			// the same is true of for assetsvalue transfers
			if slices.Contains(lines, "ASSETVALUE") && !addedMap["ASSETVALUE"] {
				addedMap["ASSETVALUE"] = true
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
				Value:    functions[id].Name,
			},
		}

		// now make a nice notice
		notice := makeCenteredWrappedLabel("anonymous interactions might affect intended SC functionality")

		// make a splash box
		splash := container.NewVBox(widget.NewLabel(functions[id].Name), notice, isAnonymous, args, entries)

		// create an interaction callback function
		callback := func(b bool) {
			// if they cancel
			if !b {
				return
			}

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
					if crypto.HashHexToHash(scid.Text).IsZero() {

						// if it is, this is a problem
						showError(errors.New("scid can not be DERO's ZEROHASH"), program.contracts)
						return
					}

					// and append to payload
					payload = append(payload, rpc.Transfer{
						Destination: randos[1],
						SCID:        crypto.HashHexToHash(scid.Text),
						Amount:      0,
						Burn:        burn, // have to burn into the contract
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
			splash := container.NewVBox(sa, pass)
			var ci *dialog.ConfirmDialog
			// if they press enter here, it is a confirmation
			pass.OnSubmitted = func(s string) {
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
				p := pass.Text

				// dump the entry
				pass.SetText("")

				if !program.wallet.Check_Password(p) {
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
