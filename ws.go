package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/creachadair/jrpc2"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi/xswd"
)

func xswdServer(port int) *xswd.XSWD {
	// apps must ask for permission on initial connection
	forceAsk := false // as a result, apps can ask 'always allow' on initial connection

	noStore := []string{""} // specifiy which methods are not allowed to have their permissions stored

	return xswd.NewXSWDServerWithPort(port, program.wallet,
		forceAsk, noStore, xswdAppHandler, xswdRequestHandler,
	)
}

// this component handles the application acceptance
// the wallet is receiving application data from a source,
// and then granting permission based on the data therein
func xswdAppHandler(data *xswd.ApplicationData) bool {
	// let's serve up the data

	text := "\tID: \n" + data.Id + "\n" +
		"\tNAME: " + data.Name + "\n" +
		"\tDESCRIPTION: " + data.Description + "\n" +
		"\tURL: " + data.Url + "\n"

	// let's verify this real quick
	address, message, err := program.wallet.CheckSignature([]byte(data.Signature))
	if err != nil {
		showError(err, program.window)
		return reject
	}
	// fmt.Println(address.String(), string(message))
	text += "\tDEVELOPER: \n" + address.String()
	label := widget.NewLabel(text)

	var msg []byte
	msg, err = hex.DecodeString(string(message))
	if err != nil {
		showError(err, program.window)
		return reject
	}
	id, err := hex.DecodeString(data.Id)
	if err != nil {
		showError(err, program.window)
		return reject
	}

	if !bytes.Equal(msg, id) {
		// showError(errors.New("application signature does not match app id"), program.window)
		return reject
	}

	sig := widget.NewLabel("✅APP SIGNATURE MATCH✅")
	sig.Alignment = fyne.TextAlignCenter

	// range through the permissions if any
	app := container.NewVBox(label, sig)
	content := container.NewBorder(app, nil, nil, nil)

	if len(data.Permissions) > 0 {
		app.Add(container.NewCenter(widget.NewLabel(
			"	✋⚠️ APP PERMISSIONS REQUESTS ⚠️✋\n" +
				"this application is asking for these permissions:")))
		permit := ""
		for permission, request := range data.Permissions {
			permit += "❓ " + permission + ": " + request.String() + "\n"
		}
		p := widget.NewLabel(permit)
		p.Alignment = fyne.TextAlignCenter
		content.Add(container.NewScroll(p))
	}
	// we are going to wait on a choice
	choice := make(chan bool)

	// create a callback function
	callback := func(b bool) {
		if b { // if they hit confirm, they have accepted
			choice <- accept
		} else { // otherwise... rejected
			choice <- reject // default is to reject everything
		}
	}

	// create a pop-up like dialog
	pop := dialog.NewCustomConfirm(
		"New WebSocket Request",
		confirm, dismiss,
		content, callback,
		program.window,
	)
	pop.SetConfirmImportance(widget.WarningImportance)
	// show it
	pop.Resize(fyne.NewSize(program.size.Width/2, ((program.size.Height / 4) * 3)))
	pop.Show()
	fyne.DoAndWait(func() {
		program.window.Show()

	})
	// and block (eg. wait) for the choice
	return <-choice
}

// this is a method request that is extended to the underlying API
// we are going to make it as simple as it gets:
// do you allow it, do you reject it
func xswdRequestHandler(data *xswd.ApplicationData, r *jrpc2.Request) xswd.Permission {
	// let's serve up some content
	text := "\tID: \n" + data.Id + "\n" +
		"\tNAME: " + data.Name + "\n" +
		"\tDESCRIPTION: " + data.Description + "\n" +
		"\tURL: " + data.Url + "\n"
	// let's verify this real quick
	address, message, err := program.wallet.CheckSignature([]byte(data.Signature))
	if err != nil {
		showError(err, program.window)
		return xswd.Deny
	}
	// fmt.Println(address.String(), string(message))
	text += "\tDEVELOPER: \n" + address.String()
	label := widget.NewLabel(text)

	msg, err := hex.DecodeString(string(message))
	if err != nil {
		showError(err, program.window)
		return xswd.Deny
	}

	id, err := hex.DecodeString(data.Id)
	if err != nil {
		showError(err, program.window)
		return xswd.Deny
	}

	if !bytes.Equal(msg, id) {
		// showError(errors.New("application signature does not match app id"), program.window)
		// don't bother user with bad requests
		return xswd.AlwaysDeny
	}

	sig := widget.NewLabel("✅APP SIGNATURE MATCH✅")
	sig.Alignment = fyne.TextAlignCenter

	method := widget.NewLabel(`❓ METHOD REQUEST: ` + r.Method())
	method.Alignment = fyne.TextAlignCenter

	app := container.NewVBox(label, sig, method)
	content := container.NewBorder(app, nil, nil, nil)
	// if it has params, process them
	if r.HasParams() {
		var params rpc.EventNotification

		// un-marshal the params
		if err := r.UnmarshalParams(&params); err != nil {

			// if the params fail, serve the error
			showError(err, program.window)

			// // and then deny the request
			return xswd.Deny
		}
		// add param string to the request
		label := widget.NewLabel("")
		switch r.Method() {
		case "querykey":
			// not implemented
			break
		case "scinvoke":
			p := rpc.SC_Invoke_Params{}
			if err := json.Unmarshal([]byte(r.ParamString()), &p); err != nil {
				showError(err, program.window)
				break
			}
			pretty, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				showError(err, program.window)
				break
			}
			label.SetText(string(pretty))
			label.Wrapping = fyne.TextWrapWord
		case "transfer":
			p := rpc.Transfer_Params{}
			if err := json.Unmarshal([]byte(r.ParamString()), &p); err != nil {
				showError(err, program.window)
				break
			}
			text := ""
			if len(p.Transfers) != 0 {
				text += "TRANSFERS:\n" + fmt.Sprintf("%v", p.Transfers) + "\n"
			}
			if p.SC_Code != "" {
				text += "CODE:\n" + p.SC_Code + "\n"
			}
			if p.Fees != 0 {
				text += "FEES: " + rpc.FormatMoney(p.Fees) + " DERO"
			}
			label.SetText(text)
			label.Wrapping = fyne.TextWrapWord
		default:
			label.SetText(r.ParamString())
			label.Wrapping = fyne.TextWrapBreak
		}
		scroll := container.NewScroll(label)
		content.Add(scroll)

	}
	// we are going to wait for a choice
	choice := make(chan bool)

	// we are going to have
	callback := func(b bool) {
		if b { // if they say confirm, accept
			choice <- accept
		} else { // if they dismiss, reject
			choice <- reject
		}
	}
	// build a pop-up
	pop := dialog.NewCustomConfirm(
		"New WebSocket Request",
		confirm, dismiss,
		content, callback,
		program.window,
	)
	pop.SetConfirmImportance(widget.DangerImportance)

	// show it
	pop.Resize(fyne.NewSize(program.size.Width/2, ((program.size.Height / 4) * 3)))
	fyne.DoAndWait(func() {
		pop.Show()
		program.window.Show()
	})

	// now wait for the choice
	if <-choice { // if accepted...
		return xswd.Allow
	}

	// default is to deny
	return xswd.Deny
}
