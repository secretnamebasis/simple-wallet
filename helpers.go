package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/ybbus/jsonrpc"
	"go.etcd.io/bbolt"
	// "go.etcd.io/bbolt"
)

// simple way to truncate hashes/address
func truncator(a string) string { return a[:6] + "......." + a[len(a)-6:] }

// simple way to explore files
func openExplorer() *dialog.FileDialog {
	return dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, program.window)
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
				program.labels.loggedin.SetText("💰: 🔴")
			})
		}
	}
}

// this loop gets the wallet's balance and updates the label
func updateBalance() {
	var previous_bal uint64
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// check to see if we are logged-in first
		if !program.preferences.Bool("loggedIn") {
			fyne.DoAndWait(func() {
				program.labels.loggedin.SetText("💰: 🔴")
			})
		} else {

			if strings.Contains(program.labels.balance.Text, "balance") {
				fyne.DoAndWait(func() {
					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("DERO: %s", "syncing"))
				})
			}

			// sync with network
			if err := program.wallet.Sync_Wallet_Memory_With_Daemon(); err != nil {
				dialog.ShowError(err, program.window)
				return
			}

			// get the balance
			bal, _ := program.wallet.Get_Balance()

			// check it against previous
			if previous_bal != bal {

				// set the old as the new
				previous_bal = bal

				// update
				fyne.DoAndWait(func() {
					// obviously, we are still logged in
					program.labels.loggedin.SetText("💰: 🟢")

					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("DERO: %s", rpc.FormatMoney(bal)))

					program.labels.balance.Refresh()
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

func startDB() {

	// let's make a simple pong db

	// let's name our db
	address := program.wallet.GetAddress().String()
	last_eight_chars := len(address) - 8
	truncated := address[last_eight_chars:]
	db_name := truncated + ".bbolt.db"

	var err error

	// and let's open it
	program.db, err = bbolt.Open(db_name, 0600, nil)
	if err != nil {
		dialog.ShowError(err, program.window)
		return
	}

	// also, let's have a bucket for pings we are watching for
	err = program.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("PING"))
		return err
	})
	if err != nil {
		dialog.ShowError(err, program.window)
		return
	}

	// now let'd make sure there is a pong bucket for outgoing
	err = program.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("PONG"))
		return err
	})

	if err != nil {
		dialog.ShowError(err, program.window)
		return
	}
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

type queue struct {
	Item  listing
	Entry rpc.Entry
	Tx    *transaction.Transaction
}

var sends []*transaction.Transaction

var pongs []queue
var refunds []queue

func processTXQueues() {
	for program.preferences.Bool("pong_server") {
		old_height := walletapi.Get_Daemon_Height()
		for i, queue := range refunds {
			height := walletapi.Get_Daemon_Height()
			if height == old_height {
				continue
			}
			old_height = walletapi.Get_Daemon_Height()
			// fmt.Println(queue)
			if !program.preferences.Bool("pong_server") {
				return
			}
			// attempt
			if err := program.wallet.SendTransaction(queue.Tx); err != nil {
				fyne.DoAndWait(func() {
					// where ever the user is, notify them
					dialog.ShowError(err, program.window)
				})
				// dump the refund and try again in the higher loop
				refunds = slices.Delete(refunds, i, i+1)
				continue
			}

			// we actually need to refund it... so we record that?
			if err := program.db.Update(func(tx *bbolt.Tx) error {
				b := tx.Bucket([]byte("PONG"))

				// make a new entry
				pong, err := json.Marshal(pong{
					Time:    time.Now(),
					Txid:    queue.Entry.TXID,
					Status:  "refunded",
					Address: queue.Item.Address,
				})

				// if this screws up...
				if err != nil {
					return err
				}

				// put to db
				return b.Put([]byte(queue.Entry.TXID), pong)
			}); err != nil {
				// where ever the user is, notify them
				dialog.ShowError(err, program.window)
				continue
			}
			msg := "refund has been sent for " + truncator(queue.Item.Address)
			fyne.DoAndWait(func() {
				dialog.ShowInformation("Pong Server", msg, program.window)
				program.entries.pongs.Refresh()
			})
			refunds = slices.Delete(refunds, i, i+1)
		}

		for i, queue := range pongs {

			// fmt.Println(queue)
			if !program.preferences.Bool("pong_server") {
				return
			}

			if err := program.wallet.SendTransaction(queue.Tx); err != nil {

				fyne.DoAndWait(func() {

					// where ever the user is, notify them
					dialog.ShowError(err, program.window)
				})

				// dump the tx and try again at a higher loop
				pongs = slices.Delete(pongs, i, i+1)
				continue
			} else { // update the db
				if err := program.db.Update(func(tx *bbolt.Tx) error {

					b := tx.Bucket([]byte("PING"))

					new_supply := queue.Item.Supply - 1
					// hopefully we don't experience failure
					updated_supply := listing{
						Name:        queue.Item.Name,
						Description: queue.Item.Description,
						Address:     queue.Item.Address,
						DST:         queue.Item.DST,
						Replyback:   queue.Item.Replyback,
						Token:       queue.Item.Token,
						Sendback:    queue.Item.Sendback,
						Supply:      new_supply,
					}

					bytes, err := json.Marshal(updated_supply)
					if err != nil {
						return err
					}

					if err := b.Put([]byte(queue.Item.Address), bytes); err != nil {
						return err
					}

					// go to the pong bucket
					b = tx.Bucket([]byte("PONG"))

					// make a new entry
					pong, err := json.Marshal(pong{
						Time:    time.Now(),
						Txid:    queue.Entry.TXID,
						Status:  "done",
						Address: queue.Item.Address,
					})

					// if this screws up...
					if err != nil {
						return err
					}

					// put to db
					return b.Put([]byte(queue.Entry.TXID), pong)
				}); err != nil {
					// if there is a problem, let us know
					dialog.ShowError(err, program.window)
					continue
				}
				msg := "pong has been sent for " + truncator(queue.Item.Address)

				fyne.DoAndWait(func() {
					dialog.ShowInformation("Pong Server", msg, program.window)
					program.entries.pongs.Refresh()
				})
				pongs = slices.Delete(pongs, i, i+1)
			}
		}
	}
}

func makeCenteredWrappedLabel(s string) *widget.Label {
	label := widget.NewLabel(s)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	return label
}
