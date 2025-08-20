package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/ybbus/jsonrpc"
)

// simple way to truncate hashes/address
func truncator(a string) string { return a[:6] + "......." + a[len(a)-6:] }

// simple way to explore files
func openExplorer() *dialog.FileDialog {
	return dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			showError(err)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		program.entries.file.SetText(reader.URI().Path())
	}, program.window)
}

// simple way to update the header
func updateHeader(bold *widget.Hyperlink) {
	for _, link := range []*widget.Hyperlink{
		program.hyperlinks.home,
		program.hyperlinks.tools,
		program.hyperlinks.configs,
		program.hyperlinks.logout,
	} {
		if link == bold {
			link.TextStyle = fyne.TextStyle{
				Bold: true,
			}
		} else {
			link.TextStyle = fyne.TextStyle{
				Bold: false,
			}
		}
	}
}

// simple way to check if we are logged in
func isLoggedIn() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		if err := program.wallet.Save_Wallet(); err != nil {
			fyne.DoAndWait(func() {
				program.preferences.SetBool("loggedIn", false)
				program.labels.loggedin.SetText("WALLET: 🔴")
			})
		}
	}
}

// this loop gets the wallet's balance and updates the label
func updateBalance() {
	var previous_bal uint64
	var bal uint64
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// check to see if we are logged-in first
		if !program.preferences.Bool("loggedIn") {
			fyne.DoAndWait(func() {
				program.labels.loggedin.SetText("WALLET: 🔴")
			})
		} else {
			if bal == 0 {
				fyne.DoAndWait(func() {
					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("BALANCE: %s", "syncing"))
				})
			}

			// sync with network
			if err := program.wallet.Sync_Wallet_Memory_With_Daemon(); err != nil {
				showError(err)
				return
			}

			if program.wallet == nil {
				return
			}

			bal, _ = program.wallet.Get_Balance()
			// get the balance

			// check it against previous
			if previous_bal != bal {

				// set the old as the new
				previous_bal = bal

				// update
				fyne.DoAndWait(func() {
					// obviously, we are still logged in
					program.labels.loggedin.SetText("WALLET: 🟢")

					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("BALANCE: %s", rpc.FormatMoney(bal)))

				})
			}
		}
	}
}

// simple way to get all transfers
func allTransfers() []rpc.Entry {
	return program.wallet.Show_Transfers(
		crypto.ZEROHASH,
		true, true, true,
		0, uint64(walletapi.Get_Daemon_Height()),
		"", "",
		0, 0,
	)
}

// simple way to update all assets
func buildAssetHashList() {
	// clear out the cache
	program.caches.hashes = []crypto.Hash{}

	// range over any pre-existing entries in the account
	for hash := range program.wallet.GetAccount().EntriesNative {

		// skip DERO's scid
		if !hash.IsZero() {

			// load each has into the cache
			program.caches.hashes = append(program.caches.hashes, hash)
		}
	}
}

func isRegistered(s string) bool {
	// make a new client
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// make a bucket to collect the result
	var result rpc.GetEncryptedBalance_Result

	// define the method to use
	var method = "DERO.GetEncryptedBalance"

	// set some particulars for the request
	var params = rpc.GetEncryptedBalance_Params{
		Address:    s,  // the address to check
		TopoHeight: -1, // the top of the chain
	}

	// if we have an error here..
	if err := rpcClient.CallFor(&result, method, params); err != nil {
		return false
	}
	if result.Registration == 0 {
		return false
	}
	// if there is no error and the registration isn't 0
	return true
}

// simple way to test http connection to derod
func testConnection(s string) error {

	// make a new request
	req, err := http.NewRequest("GET", "http://"+s, nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// set a timeout context
	ctx, cancel := context.WithTimeout(req.Context(), time.Second*1)

	// defer the cancel of the request
	defer cancel()

	// add the timeout context to the request
	req = req.WithContext(ctx)

	// make a new client
	client := http.DefaultClient

	// make the request and get the response
	resp, err := client.Do(req)

	// if there is an error
	if err != nil {
		// return the error
		if strings.Contains(err.Error(), "connect: connection refused") {
			return err
		} else {
			// show these errors in the terminal just because
			fmt.Println("unhandled err", err)
			return err
		}
	}

	// defer closing the body
	defer resp.Body.Close()

	// now parse the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// if the address doesn't say DERO ... return an err
	if !strings.Contains(string(body), "DERO") {
		fmt.Println(string(body)) // might not be a bad idea to know what they are saying
		return errors.New("body does not contain DERO")
	}
	return nil
}

func makeCenteredWrappedLabel(s string) *widget.Label {
	label := widget.NewLabel(s)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	return label
}

func getSCCode(scid string) rpc.GetSC_Result {

	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var sc rpc.GetSC_Result

	// here is the method we are going to use
	var method = "DERO.GetSC"

	// now for some parameters
	var scParam = rpc.GetSC_Params{
		SCID:       scid,
		Code:       true, // we are getting the code
		Variables:  false,
		TopoHeight: walletapi.Get_Daemon_Height(), // get the latestest copy
	}

	// call for the contract
	if err := rpcClient.CallFor(&sc, method, scParam); err != nil {
		return rpc.GetSC_Result{}
	}
	// the code needs to be present
	if sc.Code == "" {
		return rpc.GetSC_Result{}
	}

	return sc
}

// simple way to show error
func showError(e error) { dialog.ShowError(e, program.window) }

func showInfo(t, m string) { dialog.ShowInformation(t, m, program.window) }

// simple way to go home
func setContentAsHome() { program.window.SetContent(program.containers.home) }
