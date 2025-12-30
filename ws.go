package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/civilware/epoch"
	"github.com/civilware/tela"
	"github.com/creachadair/jrpc2"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi/xswd"
	structures "github.com/secretnamebasis/simple-gnomon/structs"
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

	// reject all connection attempts while screen is locked.
	if program.preferences.Bool("isLocked") {
		return reject
	}
	var label, sig = widget.NewLabel(""), widget.NewLabel("")
	// let's serve up the data
	text := "\tID: \n" + data.Id + "\n" +
		"\tNAME: " + data.Name + "\n" +
		"\tDESCRIPTION: " + data.Description + "\n" +
		"\tURL: " + data.Url + "\n"

	// let's verify this real quick
	if data.Signature == nil {
		deny := make(chan bool, 1)
		dialog.ShowConfirm(
			"App Signature is missing",
			(data.Name + " does not contain an application signature.\nPlease be advised."),
			func(b bool) {
				if !b {
					deny <- true
					return
				}
				deny <- false
				label.Alignment = fyne.TextAlignCenter
				label.SetText(text)
				sig.Alignment = fyne.TextAlignCenter
				sig.SetText("✋⚠️ APP SIGNATURE MISSING ⚠️✋")
			},
			program.window,
		)
		if <-deny {
			return reject
		}
	} else {
		// let's verify this real quick
		address, message, err := program.wallet.CheckSignature([]byte(data.Signature))
		if err != nil {
			showError(fmt.Errorf("app authorization signature resulted in err:\n%s", err), program.window)
			return reject
		}
		// fmt.Println(address.String(), string(message))
		text += "\tDEVELOPER: \n" + address.String()
		label = widget.NewLabel(text)

		var msg []byte
		msg, err = hex.DecodeString(string(message))
		if err != nil {
			showError(fmt.Errorf("app authorization message decoding resulted in err:\n%s", err), program.window)
			return reject
		}
		id, err := hex.DecodeString(data.Id)
		if err != nil {
			showError(fmt.Errorf("app authorization data.Id resulted in err:\n%s", err), program.window)
			return reject
		}

		if !bytes.Equal(msg, id) {
			// showError(errors.New("application signature does not match app id"), program.window)
			return reject
		}

		sig = widget.NewLabel("✅APP SIGNATURE MATCH✅")
		sig.Alignment = fyne.TextAlignCenter
	}

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
	// reject all connection attempts while screen is locked.
	if program.preferences.Bool("isLocked") {
		return xswd.Deny
	}

	var label, sig = widget.NewLabel(""), widget.NewLabel("")
	// let's serve up some content
	text := "\tID: \n" + data.Id + "\n" +
		"\tNAME: " + data.Name + "\n" +
		"\tDESCRIPTION: " + data.Description + "\n" +
		"\tURL: " + data.Url + "\n"

	// let's verify this real quick
	if data.Signature == nil {

		label.Alignment = fyne.TextAlignCenter
		label.SetText(text)
		sig.Alignment = fyne.TextAlignCenter
		sig.SetText("✋⚠️ APP SIGNATURE MISSING ⚠️✋")

	} else {

		address, message, err := program.wallet.CheckSignature([]byte(data.Signature))
		if err != nil {
			showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
			return xswd.Deny
		}
		// fmt.Println(address.String(), string(message))
		text += "\tDEVELOPER: \n" + address.String()
		label = widget.NewLabel(text)

		msg, err := hex.DecodeString(string(message))
		if err != nil {
			showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
			return xswd.Deny
		}

		id, err := hex.DecodeString(data.Id)
		if err != nil {
			showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
			return xswd.Deny
		}

		if !bytes.Equal(msg, id) {
			// showError(errors.New("application signature does not match app id"), program.window)
			// don't bother user with bad requests
			return xswd.AlwaysDeny
		}

		sig = widget.NewLabel("✅APP SIGNATURE MATCH✅")
		sig.Alignment = fyne.TextAlignCenter
	}

	method := widget.NewLabel(`❓ METHOD REQUEST: ` + r.Method())
	method.Alignment = fyne.TextAlignCenter

	app := container.NewVBox(label, sig, method)
	content := container.NewBorder(app, nil, nil, nil)
	// if it has params, process them
	if r.HasParams() {
		// var params rpc.EventNotification

		// un-marshal the params
		// if err := r.UnmarshalParams(&params); err != nil {

		// 	// if the params fail, serve the error
		// 	showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)

		// 	// // and then deny the request
		// 	return xswd.Deny
		// }
		// add param string to the request
		label := widget.NewLabel("")
		switch r.Method() {
		case "querykey":
			// not implemented
			break
		case "scinvoke":
			p := rpc.SC_Invoke_Params{}
			if err := json.Unmarshal([]byte(r.ParamString()), &p); err != nil {
				showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
				break
			}
			pretty, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
				break
			}
			label.SetText(string(pretty))
			label.Wrapping = fyne.TextWrapWord
		case "transfer":
			p := rpc.Transfer_Params{}
			if err := json.Unmarshal([]byte(r.ParamString()), &p); err != nil {
				showError(fmt.Errorf("app request resulted in err:\n%s", err), program.window)
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

type getTXEstimateResult struct {
	Fees uint64 `json:"fees"`
}

// because the GetGasEstimater doesn't do gas estimates for transfers...
func getTXEstimate(ctx context.Context, params rpc.Transfer_Params) getTXEstimateResult {
	if len(params.Transfers) == 0 {
		return getTXEstimateResult{Fees: 0}
	}

	dry_run := true
	transfer_all := false
	tx, err := program.wallet.TransferPayload0(
		params.Transfers,
		params.Ringsize,
		transfer_all, //not implemented
		params.SC_RPC,
		0, // dry_run does not require gasstorage
		dry_run,
	)
	if err != nil {
		return getTXEstimateResult{Fees: 0}
	}

	return getTXEstimateResult{Fees: tx.Fees()}
}

type getAllOwnersAndSCIDsResult struct {
	Result map[string]any `json:"allOwners"`
}

func getAllOwnersAndSCIDs(ctx context.Context) (getAllOwnersAndSCIDsResult, error) {

	msg := map[string]any{
		"method": "GetAllOwnersAndSCIDs",
		"id":     "1",
	}

	var err error

	if indexer_connection == nil {
		err = errors.New("indexer connection not established")
		showError(err, program.window)
		return getAllOwnersAndSCIDsResult{}, err
	}

	if err := indexer_connection.WriteJSON(msg); err != nil {
		return getAllOwnersAndSCIDsResult{}, errors.New("failed to write")
	}

	_, b, err := indexer_connection.ReadMessage()
	if err != nil {
		return getAllOwnersAndSCIDsResult{}, errors.New("failed to read")
	}

	var r structures.JSONRpcResp
	if err := json.Unmarshal(b, &r); err != nil {
		return getAllOwnersAndSCIDsResult{}, errors.New("failed to unmarshal")
	}

	return getAllOwnersAndSCIDsResult{r.Result.(map[string]any)}, nil
}

type getAllSCIDVariableDetailsResult struct {
	Result []any `json:"allVariables"`
}

type getAllSCIDVariableDetailsParams struct {
	SCID string
}

func getAllSCIDVariableDetails(ctx context.Context, params getAllSCIDVariableDetailsParams) (getAllSCIDVariableDetailsResult, error) {

	msg := map[string]any{
		"method": "GetAllSCIDVariableDetails",
		"id":     "1",
		"params": params,
	}

	var err error
	if indexer_connection == nil {
		err = errors.New("indexer connection not established")
		showError(err, program.window)
		return getAllSCIDVariableDetailsResult{}, err
	}
	if err := indexer_connection.WriteJSON(msg); err != nil {
		return getAllSCIDVariableDetailsResult{}, errors.New("failed to write")
	}

	_, b, err := indexer_connection.ReadMessage()
	if err != nil {
		return getAllSCIDVariableDetailsResult{}, errors.New("failed to read")
	}

	var r structures.JSONRpcResp
	if err := json.Unmarshal(b, &r); err != nil {
		return getAllSCIDVariableDetailsResult{}, errors.New("failed to unmarshal")
	}
	if r.Error != nil {
		return getAllSCIDVariableDetailsResult{}, errors.New("for fucks sake")

	}

	return getAllSCIDVariableDetailsResult{r.Result.([]any)}, nil
}

type getSCIDsResult struct {
	SCIDS []string `json:"scids"`
}

func getAssets(ctx context.Context) (getSCIDsResult, error) {
	scids := []string{}
	for each := range program.wallet.GetAccount().EntriesNative {
		scids = append(scids, each.String())
	}
	return getSCIDsResult{scids}, nil
}

type getAssetBalanceParams struct {
	Height int64
	SCID   string
}
type getAssetBalanceResult struct {
	Balance uint64 `json:"balance"`
}

func getAssetBalance(ctx context.Context, params getAssetBalanceParams) (getAssetBalanceResult, error) {

	hash := crypto.HashHexToHash(params.SCID)

	height := params.Height // -1 is current topoheight

	address := program.wallet.GetAddress()

	bal, _, err := program.wallet.GetDecryptedBalanceAtTopoHeight(hash, height, address.String())

	if err != nil {
		return getAssetBalanceResult{}, err
	}

	return getAssetBalanceResult{bal}, nil
}

type telaLink_Params struct {
	TelaLink string `json:"telaLink"` // format is target://<arg>/<arg>/...
}

type telaLink_Display struct {
	Name     string              `json:"nameHdr,omitempty"`
	Descr    string              `json:"descrHdr,omitempty"`
	DURL     string              `json:"dURL,omitempty"`
	TelaLink string              `json:"telaLink"` // format is target://<arg>/<arg>/...
	Rating   *tela.Rating_Result `json:"rating,omitempty"`
}

type telaLink_Result struct {
	TelaLinkResult string `json:"telaLinkResult"`
}

type getAttemptEpochParams struct {
	Hashes  int    `json:"hashes"`
	Address string `json:"address"`
}

func handleTELALinks(ctx context.Context, params telaLink_Params) (result telaLink_Result, err error) {

	target, args, err := tela.ParseTELALink(params.TelaLink)
	if err != nil {
		err = fmt.Errorf("could not parse tela link: %s", err)
		return
	}

	if target != "tela" {
		err = fmt.Errorf("unknown tela target %q", args[0])
		return
	}

	if args[0] != "open" {
		err = fmt.Errorf("unsupport tela argument %q", args[0])
		return
	}
	var wait = make(chan bool, 1)
	if !tela.UpdatesAllowed() {
		dialog.ShowConfirm(
			"TELA LINK UPDATED",
			"This tela link's content has changed since first install.\nIf you would like to proceed anyway, please confirm.\nPlease be advised.",
			func(b bool) {
				if !b {
					wait <- !b
				}
				tela.AllowUpdates(true)
				wait <- tela.UpdatesAllowed()
			},
			program.window,
		)
		<-wait
	}

	var link string
	link, err = tela.OpenTELALink(params.TelaLink, program.node.current)
	if err != nil {
		return
	}

	var url *url.URL
	url, err = url.Parse(link)
	if err != nil {
		err = fmt.Errorf("could not parse URL: %s", err)
		return
	}

	err = fyne.CurrentApp().OpenURL(url)
	if err != nil {
		err = fmt.Errorf("could not open tela link: %s", err)
		return
	}

	result.TelaLinkResult = link
	// case "gnomon":

	return
}

func attemptEPOCHWithAddr(ctx context.Context, params getAttemptEpochParams) (epoch.EPOCH_Result, error) {

	reserve := 2 // one for the app and one for the os
	threads := runtime.GOMAXPROCS(0)
	maximum := threads - reserve

	epoch.SetMaxThreads(maximum)
	addr, err := rpc.NewAddress(params.Address)
	if err != nil {
		return epoch.EPOCH_Result{}, errors.New("invalid address")
	}

	endpoint := program.node.current

	err = epoch.StartGetWork(addr.String(), endpoint)
	if err != nil {
		return epoch.EPOCH_Result{}, errors.New("failed start get work server")
	}
	defer epoch.StopGetWork()

	timeout := time.Second * 10

	err = epoch.JobIsReady(timeout)
	if err != nil {
		return epoch.EPOCH_Result{}, errors.New("failed get job before timeout")
	}

	// the smaller of the two
	hashes := min(params.Hashes, epoch.LIMIT_MAX_HASHES)

	return epoch.AttemptHashes(hashes)
}
