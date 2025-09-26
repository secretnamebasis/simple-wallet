package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/block"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/ybbus/jsonrpc/v3"
	"golang.org/x/image/draw"
)

// needs to be better organized, possibly squirreled into new .go files

// simple way to truncate hashes/address
func truncator(a string) string {
	if len(a) < 18 {
		return a
	}
	return a[:6] + "......" + a[len(a)-6:]
}

// simple way to explore files
func openExplorer(e *widget.Entry, w fyne.Window) *dialog.FileDialog {
	return dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			showError(err, w)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		e.SetText(reader.URI().Path())
	}, w)
}

// simple way to explore files
func openWalletFile() *dialog.FileDialog {
	return dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			showError(err, program.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		program.entries.wallet.SetText(reader.URI().Path())
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
func createPreferred() {
	filename := "preferred"
	// let's make a simple way to have a preferred connection
	preferred_connection := struct {
		ip   string
		name string
	}{name: filename}

	if _, err := os.Stat(filename); err != nil {
		os.Create(filename)
		// really
	} else {
		file, err := os.Open(filename)
		if err != nil {

			// we shouldn't report an err here...

		}
		b, err := io.ReadAll(file)
		if err != nil {
			// again, we shouldn't report an err
			// we'll just try localhost and move on
		}
		// load the ip addres into the connection
		preferred_connection.ip = string(b)

		// now if we have an error here...
		if preferred_connection.ip != "" {
			program.node.list[0] = preferred_connection
		}
	}
}

// simple way to check if we are logged in
func isLoggedIn() {
	ticker := time.NewTicker(time.Second * 2)
	var mu sync.Mutex
	for range ticker.C {
		mu.Lock()
		if program.wallet == nil {
			program.preferences.SetBool("loggedIn", false)
			break
		}
		if program.ws_server != nil { // don't save the listeners into the wallet file
			program.preferences.SetBool("loggedIn", true)
			continue
		}
		if err := program.wallet.Save_Wallet(); err != nil {
			fyne.DoAndWait(func() {
				program.labels.loggedin.SetText("WALLET: 🟡") // signalling an error
			})
			continue
		}
		if !program.preferences.Bool("loggedIn") && program.wallet != nil {
			program.preferences.SetBool("loggedIn", true)
		}
		mu.Unlock()
	}
}
func notificationNewEntry() {
	// we are going to be a little aggressive here
	ticker := time.NewTicker(time.Second)
	// and because we aren't doing any fancy websocket stuff...
	var old_len int
	for range ticker.C { // range that ticker
		// check if we are still logged in
		if !program.preferences.Bool("loggedIn") {
			return
		}
		// check if the wallet is present
		if program.wallet == nil {
			return
		}
		// check if we are registered
		if !program.wallet.IsRegistered() {
			continue
		}

		// go get the transfers
		var current_transfers []rpc.Entry
		if program.wallet != nil { // expressly validate this
			current_transfers = getAllTransfers(crypto.ZEROHASH)
			for _, each := range program.caches.assets {
				hash := crypto.HashHexToHash(each.hash)
				current_transfers = append(current_transfers, getAllTransfers(hash)...)
			}
		} else {
			continue
		}

		// now get the length of transfers
		current_len := len(current_transfers)
		// do a diff check
		diff := current_len - old_len

		// set current as old length
		old_len = current_len

		// now if they are the same, move on
		if diff == current_len ||
			diff == 0 ||
			current_len == 0 {
			continue
		}

		// determine the inset for the slice
		inset := current_len - diff

		// to avoid runtime error: slice bounds out of range...
		if inset > len(current_transfers) {
			continue
		}

		// define the new transfers slice
		new_transfers := current_transfers[inset:]

		// now range the new transfers
		for _, each := range new_transfers {

			// only show today's transfers
			today := time.Now()
			midnight := time.Date(
				today.Year(),
				today.Month(),
				today.Day(),
				0, 0, 0, 0,
				time.UTC,
			)
			if each.Time.Before(midnight) { // maybe look in to longer timescales
				continue
			}

			// build a notification
			notification := fyne.NewNotification(
				"New Transfer", each.String(),
			)

			// ship the notification
			program.application.SendNotification(notification)
		}
	}
}

// this loop gets the wallet's balance and updates the label
func updateBalance() {
	var previous_bal uint64
	var bal uint64
	ticker := time.NewTicker(time.Second * 2)
	for range ticker.C {
		// check to see if we are logged-in first
		if !program.preferences.Bool("loggedIn") {
			fyne.DoAndWait(func() {
				program.labels.loggedin.SetText("WALLET: 🔴")
				program.labels.balance.SetText(rpc.FormatMoney(0))
				bal, previous_bal = 0, 0
			})
			return
		} else {
			// check if there is a wallet first
			if program.wallet == nil {
				return
			}

			if bal == 0 && program.wallet.IsRegistered() {
				fyne.DoAndWait(func() {
					// update it
					program.labels.balance.SetText("syncing")

				})
			} else if bal == 0 && !program.wallet.IsRegistered() {
				fyne.DoAndWait(func() {
					// update it
					program.labels.loggedin.SetText("WALLET: 🟢")
					program.labels.balance.SetText("unregistered")

				})
			}
			// check if there is a wallet first
			if program.wallet == nil {
				return
			}

			// get the balance
			if !program.preferences.Bool("loggedIn") {
				break
			} // hella sensitive
			bal, _ = program.wallet.Get_Balance()

			// check it against previous
			if previous_bal != bal {

				// update
				fyne.DoAndWait(func() {
					// obviously, we are still logged in
					program.labels.loggedin.SetText("WALLET: 🟢")

					// update it
					program.labels.balance.SetText(rpc.FormatMoney(bal))

				})
			}
		}
	}
}
func updateCaches() {

	for range time.NewTicker(time.Second * 2).C {
		program.caches.info = getDaemonInfo()
		program.caches.pool = getTxPool()
	}

}

// simple way to get all transfers
func getTransfersByHeight(min, max uint64, hash crypto.Hash, coin, in, out bool) []rpc.Entry {
	if program.wallet == nil {
		return nil
	} else {
		return program.wallet.Show_Transfers(
			hash,
			coin, in, out,
			min, max,
			"", "",
			0, 0,
		)
	}
}

// simple way to get all transfers
func getTransfers(hash crypto.Hash, coin, in, out bool) []rpc.Entry {
	return getTransfersByHeight(
		0, uint64(walletapi.Get_Daemon_Height()),
		hash,
		coin, in, out,
	)
}

// simple way to get all transfers
func getAllTransfers(hash crypto.Hash) []rpc.Entry {
	return getTransfers(hash, true, true, true)
}

// simple way to get all transfers
func getCoinbaseTransfers(hash crypto.Hash) []rpc.Entry {
	return getTransfers(hash, true, false, false)
}

// simple way to get all transfers
func getReceivedTransfers(hash crypto.Hash) []rpc.Entry {
	return getTransfers(hash, false, true, false)
}

// simple way to get all transfers
func getSentTransfers(hash crypto.Hash) []rpc.Entry {
	return getTransfers(hash, false, false, true)
}

// simple way to update all assets
func buildAssetHashList() {
	// clear out the cache
	program.caches.assets = []asset{}
	assets := program.wallet.GetAccount().EntriesNative
	var wg sync.WaitGroup
	wg.Add(len(assets))
	// range over any pre-existing entries in the account
	for a := range assets {
		go func() {
			defer wg.Done()
			// skip DERO's scid
			if !a.IsZero() {

				// load each has into the cache
				program.caches.assets = append(program.caches.assets, asset{
					name:  getSCNameFromVars(a.String()),
					hash:  a.String(),
					image: getSCIDImage(a.String()),
				})
			}
		}()
	}
	wg.Wait()
	// now sort them for consistency
	sort.Slice(program.caches.assets, func(i, j int) bool {
		return program.caches.assets[i].name > program.caches.assets[j].name
	})
}
func getSCNameFromVars(scid string) string {
	var text string
	if crypto.HashHexToHash(scid).IsZero() {
		return "DERO"
	}
	for k, v := range getSCValues(scid).VariableStringKeys {
		if !strings.Contains(k, "name") {
			continue
		}
		b, e := hex.DecodeString(v.(string))
		if e != nil {
			continue // what else can we do ?
		}
		text = string(b)
	}
	if text == "" {
		return "N/A"
	}
	return text
}
func isRegistered(s string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
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
	if err := rpcClient.CallFor(ctx, &result, method, params); err != nil {
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
	ctx, cancel := context.WithTimeout(req.Context(), timeout)

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
		} else if strings.Contains(err.Error(), "context deadline exceeded") {
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
func hashesLength() int { return len(program.caches.assets) }
func createImageLabel() fyne.CanvasObject {
	// define a canvas object
	img := canvas.NewImageFromResource(nil)

	// set a fillmode, the original is fine
	img.FillMode = canvas.ImageFillOriginal

	// make a thumbnail size
	img.SetMinSize(fyne.NewSize(25, 25))

	// set the object in a new container
	image := container.NewPadded(img)

	// resize the container to be the min size of the object
	image.Resize(img.MinSize())

	// return the adaptive container
	return container.NewAdaptiveGrid(4,
		image,
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)
}
func createLabel() fyne.CanvasObject {
	return container.NewAdaptiveGrid(3,
		widget.NewLabel(""),
		widget.NewLabel(""),
		widget.NewLabel(""),
	)
}

func makeCenteredWrappedLabel(s string) *widget.Label {
	label := widget.NewLabel(s)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	return label
}

func getTransaction(params rpc.GetTransaction_Params) rpc.GetTransaction_Result {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var result rpc.GetTransaction_Result

	// here is the method we are going to use
	var method = "DERO.GetTransaction"

	// call for the contract
	if err := rpcClient.CallFor(ctx, &result, method, params); err != nil {
		return rpc.GetTransaction_Result{}
	}
	// the code needs to be present
	if result.Status == "" {
		return rpc.GetTransaction_Result{}
	}

	return result
}
func getBlockInfo(params rpc.GetBlock_Params) rpc.GetBlock_Result {

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var result rpc.GetBlock_Result

	// here is the method we are going to use
	var method = "DERO.GetBlock"

	// call for the contract
	if err := rpcClient.CallFor(ctx, &result, method, params); err != nil {
		return rpc.GetBlock_Result{}
	}
	// the code needs to be present
	if result.Block_Header.Depth == 0 {
		return rpc.GetBlock_Result{}
	}

	return result
}

func getTxPool() rpc.GetTxPool_Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var pool rpc.GetTxPool_Result

	// here is the method we are going to use
	var method = "DERO.GetTxPool"

	// call for the contract
	if err := rpcClient.CallFor(ctx, &pool, method); err != nil {
		return rpc.GetTxPool_Result{}
	}
	// the code needs to be present
	if pool.Status == "" {
		return rpc.GetTxPool_Result{}
	}

	return pool
}
func getDaemonInfo() rpc.GetInfo_Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var info rpc.GetInfo_Result

	// here is the method we are going to use
	var method = "DERO.GetInfo"

	// call for the contract
	if err := rpcClient.CallFor(ctx, &info, method); err != nil {
		return rpc.GetInfo_Result{}
	}
	// the code needs to be present
	if info.TopoHeight == 0 {
		return rpc.GetInfo_Result{}
	}

	return info
}
func getSC(scParam rpc.GetSC_Params) rpc.GetSC_Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var sc rpc.GetSC_Result

	// here is the method we are going to use
	var method = "DERO.GetSC"

	// call for the contract
	if err := rpcClient.CallFor(ctx, &sc, method, scParam); err != nil {
		return rpc.GetSC_Result{}
	}
	// the code needs to be present
	if sc.Code == "" {
		return rpc.GetSC_Result{}
	}

	return sc
}
func getSCCode(scid string) rpc.GetSC_Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

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
	if err := rpcClient.CallFor(ctx, &sc, method, scParam); err != nil {
		return rpc.GetSC_Result{}
	}
	// the code needs to be present
	if sc.Code == "" {
		return rpc.GetSC_Result{}
	}

	return sc
}
func getSCValues(scid string) rpc.GetSC_Result {
	time := timeout
	if scid == gnomonSC {
		time = timeout * 3
	}
	ctx, cancel := context.WithTimeout(context.Background(), time)
	defer cancel()

	// get a client for the daemon's rpc
	var rpcClient = jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	// here is our results bucket
	var sc rpc.GetSC_Result

	// here is the method we are going to use
	var method = "DERO.GetSC"

	// now for some parameters
	var scParam = rpc.GetSC_Params{
		SCID:       scid,
		Code:       false,
		Variables:  true,
		TopoHeight: walletapi.Get_Daemon_Height(), // get the latest copy
	}

	// call for the contract
	if err := rpcClient.CallFor(ctx, &sc, method, scParam); err != nil {
		return rpc.GetSC_Result{}
	}
	// the code needs to be present
	if sc.VariableStringKeys == nil {
		return rpc.GetSC_Result{}
	}

	return sc
}
func setSCIDThumbnail(img image.Image, h, w float32) image.Image {
	var thumbnail = new(canvas.Image)
	thumbnail = canvas.NewImageFromResource(theme.BrokenImageIcon())
	thumbnail.SetMinSize(fyne.NewSize(w, h))
	if img == nil {
		return thumbnail.Image
	}
	thumb := image.NewNRGBA(image.Rect(0, 0, int(h), int(w)))
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, img.Bounds(), draw.Over, nil)
	thumbnail = canvas.NewImageFromImage(thumb)
	return thumbnail.Image
}
func getSCIDImage(scid string) image.Image {
	for k, v := range getSCValues(scid).VariableStringKeys {
		if strings.Contains(k, "image") ||
			strings.Contains(k, "icon") {
			b, e := hex.DecodeString(v.(string))
			if e != nil {
				fmt.Println(v, e)
				continue
			}
			value := string(b)
			uri, err := storage.ParseURI(value)
			if err != nil {
				return nil
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), timeout/3)
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
				if err != nil {
					fmt.Println(err)
					return nil
				}
				client := http.DefaultClient
				resp, err := client.Do(req)
				if err != nil || resp.StatusCode != http.StatusOK {
					return nil
				} else {
					defer resp.Body.Close()
					i, _, err := image.Decode(resp.Body)
					if err != nil {
						return nil
					}
					return i
				}
			}
		}
	}
	return nil
}
func getSCIDBalancesContainer(balances map[string]uint64) *fyne.Container {
	bals := container.NewVBox()
	balance_pairs := []struct {
		key   string
		value uint64
	}{}
	for k, v := range balances {
		balance_pairs = append(balance_pairs, struct {
			key   string
			value uint64
		}{
			key:   k,
			value: v,
		})
	}
	sort.Slice(balance_pairs, func(i, j int) bool {
		return balance_pairs[i].key < balance_pairs[j].key
	})
	for _, pair := range balance_pairs {

		bals.Add(container.NewAdaptiveGrid(5,
			layout.NewSpacer(),
			widget.NewLabel(getSCNameFromVars(pair.key)),
			widget.NewLabel(truncator(pair.key)),
			widget.NewLabel(rpc.FormatMoney(pair.value)),
			layout.NewSpacer(),
		))
	}
	return bals
}

func getSCIDStringVarsContainer(stringKeys map[string]any) *fyne.Container {

	string_pairs := []struct {
		key   string
		value any
	}{}
	for k, v := range stringKeys {
		string_pairs = append(string_pairs, struct {
			key   string
			value any
		}{
			key:   k,
			value: v,
		})
	}
	sort.Slice(string_pairs, func(i, j int) bool {
		return string_pairs[i].key < string_pairs[j].key
	})

	keys := []string{}
	values := []string{}

	for _, p := range string_pairs {

		var value string
		switch v := p.value.(type) {
		case string:
			if p.key != "C" {
				b, e := hex.DecodeString(v)
				if e != nil {
					continue
				}
				value = string(b)
			} else {
				value = truncator(v)
			}

		case uint64:
			value = strconv.Itoa(int(v))
		case float64:
			value = strconv.FormatFloat(v, 'f', 0, 64)
		}
		keys = append(keys, p.key)
		values = append(values, value)
	}
	if len(keys) == 0 {
		keys = []string{"NO DATA"}
		values = []string{"NO DATA"}
	}
	table := widget.NewTable(
		func() (rows int, cols int) { return len(keys), 2 },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			switch tci.Col {
			case 0:
				label.SetText(keys[tci.Row])
			case 1:
				label.SetText(values[tci.Row])
			}
		},
	)
	table.OnSelected = func(id widget.TableCellID) {
		var data string
		if id.Col == 0 {
			data = keys[id.Row]

		} else if id.Col == 1 {
			data = values[id.Row]
			if keys[id.Row] == "C" { // we truncated it for ease of viewing
				data = stringKeys["C"].(string)
			}
		}
		if data != "" {
			program.application.Clipboard().SetContent(data)
			table.UnselectAll()
			table.Refresh()
			showInfoFast("Copied", data, program.window)
		}
	}
	table.SetColumnWidth(0, largestMinSize(keys).Width)
	table.SetColumnWidth(1, largestMinSize(values).Width)

	return container.NewAdaptiveGrid(1, table)
}

func getSCIDUint64VarsContainer(uint64Keys map[uint64]any) *fyne.Container {

	uint64_pairs := []struct {
		key   uint64
		value any
	}{}
	for k, v := range uint64Keys {
		uint64_pairs = append(uint64_pairs, struct {
			key   uint64
			value any
		}{
			key:   k,
			value: v,
		})
	}
	sort.Slice(uint64_pairs, func(i, j int) bool {
		return uint64_pairs[i].key < uint64_pairs[j].key
	})

	keys := []string{}
	values := []string{}

	for _, p := range uint64_pairs {

		var value string
		switch v := p.value.(type) {
		case string:

			b, e := hex.DecodeString(v)
			if e != nil {
				continue
			}
			value = string(b)

		case uint64:
			value = strconv.Itoa(int(v))
		case float64:
			value = strconv.FormatFloat(v, 'f', 0, 64)
		}
		keys = append(keys, strconv.Itoa(int(p.key)))
		values = append(values, value)
	}
	if len(keys) == 0 {
		keys = []string{"NO DATA"}
		values = []string{"NO DATA"}
	}
	table := widget.NewTable(
		func() (rows int, cols int) { return len(keys), 2 },
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(tci widget.TableCellID, co fyne.CanvasObject) {
			label := co.(*widget.Label)
			switch tci.Col {
			case 0:
				label.SetText(keys[tci.Row])
			case 1:
				label.SetText(values[tci.Row])
			}
		},
	)
	table.OnSelected = func(id widget.TableCellID) {
		var data string
		if id.Col == 0 {
			data = keys[id.Row]
		} else if id.Col == 1 {
			data = values[id.Row]
		}
		if data != "" {
			program.application.Clipboard().SetContent(data)
			table.UnselectAll()
			table.Refresh()
			showInfoFast("Copied", data, program.window)
		}
	}
	table.SetColumnWidth(0, largestMinSize(keys).Width)
	table.SetColumnWidth(1, largestMinSize(values).Width)

	return container.NewAdaptiveGrid(1, table)
}
func getBlockDeserialized(blob string) block.Block {

	var bl block.Block
	b, err := hex.DecodeString(blob)
	if err != nil {
		// should probably log or handle this error
		fmt.Println(err.Error())
		return block.Block{}
	}
	bl.Deserialize(b)
	return bl
}

// simple way to show error
func showError(e error, w fyne.Window) { dialog.ShowError(e, w) }

func showInfo(t, m string, w fyne.Window) { dialog.ShowInformation(t, m, w) }
func showInfoFast(t, m string, w fyne.Window) {
	s := dialog.NewInformation(t, m, w)
	s.Show()
	go func() {
		time.Sleep(time.Millisecond * 800)
		fyne.DoAndWait(func() {
			s.Dismiss()
		})
	}()
}

// simple way to go home
func setContentAsHome() { program.window.SetContent(program.containers.home) }

func lockScreen() {

	content := container.NewVBox(
		layout.NewSpacer(),
		container.NewAdaptiveGrid(3,
			layout.NewSpacer(),
			container.NewVBox(
				program.entries.pass,
				program.hyperlinks.unlock,
			),
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
	)

	lockscreen := dialog.NewCustomWithoutButtons("locked screen", content, program.window)

	program.hyperlinks.unlock.Alignment = fyne.TextAlignCenter

	submit := func() {
		// get the password
		pass := program.entries.pass.Text

		// dump password
		program.entries.pass.SetText("")

		if !program.wallet.Check_Password(pass) {
			showError(errors.New("wrong password"), program.window)
			return
		}

		lockscreen.Dismiss()
	}

	program.entries.pass.OnSubmitted = func(s string) {
		submit()
	}

	program.hyperlinks.unlock.OnTapped = submit

	lockscreen.Resize(program.size)
	lockscreen.Show()
}

func makeGraph(hd_map map[int]int, w, h float32) fyne.CanvasObject {
	// the graph is upside down...
	// remember?
	// 0,0 +1 +2 +3 +4 >
	// -1
	// -2
	// -3
	//  ▼
	invert := float32(-1.0)

	var ( // up for interpretation, change as necessary
		left_padding   = float32(60)
		right_padding  = float32(0)
		top_padding    = float32(20)
		bottom_padding = float32(40)
	)

	graph_width := w - left_padding - right_padding
	graph_height := h - top_padding - bottom_padding

	if len(hd_map) == 0 {
		return canvas.NewText("No data", color.Black)
	}

	// we are goind to index the heights
	heights := []int{}

	// need some min maxing while we are at it
	var first_height, last_height, max_difficulty int

	// range, append and process
	for h, d := range hd_map {
		heights = append(heights, h)
		max_difficulty = max(d, max_difficulty)
	}

	// get sorted
	sort.Ints(heights)

	// the first graph_height is the lowest
	first_height = heights[0]
	last_height = heights[len(heights)-1]

	height_range := float32(last_height - first_height)
	if height_range == 0 {
		height_range = 1 // avoid divide by zero
	}

	// so that the top of the graph isn't the max difficulty
	buffer := float32(1.5) // take it up 150% just for a clearer view
	difficulty_range := float32(max_difficulty) * (buffer)
	if difficulty_range == 0 {
		difficulty_range = 1
	}

	// now let's make an slice of objects
	objects := []fyne.CanvasObject{}

	// when plotting a graph, always start with the x axis first
	x_axis := canvas.NewLine(theme.Color(theme.ColorNameForeground))

	// then start by establishing where x_axis start/end x,y co-ordinates are
	x_axis.Position1 = fyne.NewPos( // left-top 'position of the Line'
		left_padding,
		(graph_height + top_padding), // just so happens to be a negative number
	)

	x_axis.Position2 = x_axis.Position1.Add(
		fyne.Position{ // right-bottom 'position of the Line'
			X: graph_width, // straight across
			Y: 0,           // not angled
		})

	// now, let's graph the y axis line
	y_axis := canvas.NewLine(theme.Color(theme.ColorNameForeground))

	// we'll use the starting position of the x_axis as the starting position for y
	y_axis.Position1 = x_axis.Position1 // left-top 'position of the Line'

	// then we'll subtract to make the line go up, -n minus -n = -n plus n = 0
	y_axis.Position2 = y_axis.Position1.Subtract(fyne.Position{ // right-bottom 'position of the Line'
		// X: 0, // straight
		Y: graph_height,
	})

	// let's add the lines into the bucket of objects
	objects = append(objects, x_axis, y_axis)

	// for scaling purposes
	horizontal_scaling := graph_width / height_range
	vertical_scaling := graph_height / difficulty_range

	var prevX, prevY float32
	for i, height := range heights {

		// place it on the x plane
		x := (float32(i) * horizontal_scaling) + (left_padding)

		// let's get the difficulty from the map
		difficulty := hd_map[height]

		// now place it on the y plane
		y := (invert * float32(difficulty) * vertical_scaling) + (graph_height + top_padding)

		// let's make a dot to represent the data point
		dot := canvas.NewCircle(color.RGBA{R: 0, G: 150, B: 255, A: 255})

		// the dot has two features:
		// stroke, or the edge
		// fill, or the center

		// paint it
		dot.StrokeColor = theme.Color(theme.ColorNameForeground)
		dot.StrokeWidth = 1 // give it depth

		// fill it
		dot.FillColor = theme.Color(theme.ColorNameBackground)
		dot.Resize(fyne.NewSize(2, 2)) // give it depth

		// move it into place
		dot.Move(fyne.NewPos(x, y))

		// might as well, load it in the bucket
		objects = append(objects, dot)

		// make a line for each datapoint after first height
		if i > 0 {
			// start by making a line
			line := canvas.NewLine(theme.Color(theme.ColorNameForeground))
			// feel free to color it
			line.StrokeColor = theme.Color(theme.ColorNameForeground)

			// and here is your brush stroke
			line.StrokeWidth = 1

			// we are going to use a previous iteration's values
			line.Position1 = fyne.NewPos(prevX, prevY)

			// and then add the difference between these values and previous
			line.Position2 = line.Position1.Add(fyne.Position{
				X: (x - prevX),
				Y: (y - prevY),
			})

			// and append
			objects = append(objects, line)
		}

		// pick this up on the way to the next iteration
		prevX, prevY = x, y
	}

	ticks := 10 // feel free to change it
	for i := range ticks {

		// there is no x value because we are on the y axis plane
		x := float32(0.0)

		// find the smallest tick value
		smallest := (difficulty_range / float32(ticks))

		// multiply it by the tick index
		value := float32(i) * smallest

		// to obtain y...
		//invert the value and multiply by scaling factor
		y := (invert * value * vertical_scaling) + // then add paddings
			graph_height + top_padding

		// let's make a tick line
		tick := canvas.NewLine(color.Gray{Y: 100})
		tick.StrokeWidth = 1
		// tick.Resize(fyne.NewSize(5, 5)) // did it do anything?
		x += left_padding - theme.Padding()

		// set the position of the tick
		tick.Position1 = fyne.NewPos(x, y)

		// add back the padding
		tick.Position2 = tick.Position1.Add(fyne.Position{
			X: theme.Padding(),
			// Y: 0,
		})

		// append the tick onto the board
		objects = append(objects, tick)

		// now make a label
		txt := fmt.Sprintf("%.0fMH/s", value/1000000)
		label := canvas.NewText(txt, theme.Color(theme.ColorNameForeground))
		label.TextSize = 10
		label.Move(fyne.NewPos(theme.Padding(), (y - theme.Padding())))
		objects = append(objects, label)
	}

	// re-assign the tick value
	ticks = int(max(1.0, float32((last_height-first_height)/len(heights))))
	for i := first_height; i <= last_height; i += ticks {

		// continue if height modulus 10 is not 0... essentially, all but the tenth block
		// if i%10 != 0 { // reports of stalls?
		// 	continue
		// }

		// get the working value
		value := float32((i - first_height))

		// to obtain x, take value times scaling and add in padding
		x := (value * horizontal_scaling) + left_padding

		// let's make a tick
		tick := canvas.NewLine(color.Gray{Y: 100})

		// establish the first position
		tick.Position1 = fyne.NewPos(x, (graph_height + top_padding))

		// add in some padding
		tick.Position2 = tick.Position1.Add(fyne.Position{
			Y: theme.Padding(),
		})

		// append the objects
		objects = append(objects, tick)

		// make a label
		txt := fmt.Sprintf("%d", i)
		label := canvas.NewText(txt, theme.Color(theme.ColorNameForeground))
		label.TextSize = 10
		label.Move(fyne.NewPos((x - top_padding), (graph_height + top_padding + theme.Padding())))
		objects = append(objects, label)
	}
	// now let's make a nice label for the block heights
	bottom_text := canvas.NewText("Block Height", theme.Color(theme.ColorNameForeground))
	bottom_text.TextStyle = fyne.TextStyle{Bold: true}

	// the width of the graph in half
	half_way := (w / 2)
	// almost touching the bottom
	near_the_bottom := (h - top_padding) + theme.Padding()
	// position it
	bottom_text.Move(fyne.NewPos(half_way, near_the_bottom))

	// and append
	objects = append(objects, bottom_text)

	// here is the left side
	left_text := canvas.NewText("Difficulty", theme.Color(theme.ColorNameForeground))
	left_text.TextStyle = fyne.TextStyle{Bold: true}
	left_text.TextSize = 12 // smaller...

	// we need to find out what half way is for this object
	width := fyne.MeasureText(left_text.Text, left_text.TextSize, left_text.TextStyle).Width
	half_way = (left_padding - (width / 2))
	// and set it right on top of the y-axis
	left_text.Move(fyne.NewPos(half_way, theme.Padding()))

	// append object
	objects = append(objects, left_text)

	// let them all lay where we said...
	graph := container.NewWithoutLayout(objects...)
	// resize the graph
	graph.Resize(fyne.NewSize(w, h))

	// gimme gimme gimme
	return graph
}

func largestMinSize(s []string) fyne.Size {
	var largest = theme.Padding()
	// fmt.Println(largest)
	speed := make(map[float32]string)
	for _, e := range s {
		size := float32(len(e))
		if size <= 1 {
			continue
		}
		largest = max(largest, size)
		speed[size] = e
		// fmt.Println("current largest", speed[largest], largest)
	}
	l := widget.NewLabel(speed[largest])
	l.Wrapping = fyne.TextWrapOff
	return l.MinSize()
}

func integrated_address_generator() {

	// let's start with a set of arguments
	args := rpc.Arguments{}

	// we are going to get a destination port
	dst := widget.NewEntry()

	// make a user friendly placeholder
	dst.SetPlaceHolder("destination port: 123456789012345678")
	dst.ActionItem = widget.NewButtonWithIcon("random", theme.SearchReplaceIcon(), func() {
		dst.SetText(strconv.Itoa(rand.Int()))
	})

	// let's validate the destination port
	validate_string := func(s string) error {
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
	dst.Validator = validate_string

	// let's get a value from the user
	value := widget.NewEntry()

	// here is a place holder
	value.SetPlaceHolder("value: 816.80085")

	// and now let's parse it
	validate_string = func(s string) error {
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
	value.Validator = validate_string

	// let's see if there is a comment
	comment := widget.NewEntry()

	// make a placeholder
	comment.SetPlaceHolder("send comment with transaction: barking-mad-war-dogs")

	// validate the comment
	validate_string = func(s string) error {
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
	comment.Validator = validate_string

	// now if a reply back is necessary
	needs_replyback := widget.NewCheck("Needs Replyback Address?", func(b bool) {
		args = append(args, rpc.Argument{
			Name:     rpc.RPC_NEEDS_REPLYBACK_ADDRESS,
			DataType: rpc.DataUint64,
			Value:    uint64(1),
		})
	})

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
			showError(errors.New("something isn't working"), program.window)

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
				showError(err, program.window)
				return
			}

			// check the pack of the args
			if _, err := args.CheckPack(transaction.PAYLOAD0_LIMIT); err != nil {
				showError(err, program.window)
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
	quick := widget.NewHyperlink("new random address", nil)
	quick.Alignment = fyne.TextAlignCenter
	quick.OnTapped = func() {
		addr, _ := rpc.NewAddress(program.wallet.GetAddress().String())
		rand.NewSource(time.Now().UnixNano())
		n := rand.Intn(100000000)
		addr.Arguments = rpc.Arguments{
			rpc.Argument{
				Name:     rpc.RPC_DESTINATION_PORT,
				DataType: rpc.DataString,
				Value:    strconv.Itoa(n),
			},
		}
		program.application.Clipboard().SetContent(addr.String())
		showInfoFast("Copied", truncator(addr.String())+"\ncopied to clipboard", program.window)
	}

	items := []*widget.FormItem{
		widget.NewFormItem("", value),
		widget.NewFormItem("", dst),
		widget.NewFormItem("", comment),
		widget.NewFormItem("", needs_replyback),
	}
	form := widget.NewForm(items...)
	form.OnSubmit = func() {
		callback(true)
	}
	form.SubmitText = "Create"
	advanced := widget.NewAccordion(
		widget.NewAccordionItem("Advanced", form),
	)
	content := container.NewVBox(
		quick,
		advanced,
	)
	// put all the fun stuff into a dialog
	integrated := dialog.NewCustom("Integrated Address", dismiss, content, program.window)
	// resize and show
	integrated.Resize(fyne.NewSize(((program.size.Width / 3) * 2), (((program.size.Height / 3) * 2) + (theme.Padding() * 2))))
	integrated.Show()
}
func asset_scan() {
	var scan *dialog.ConfirmDialog
	syncing := widget.NewActivity()
	scids := widget.NewProgressBar()
	label := makeCenteredWrappedLabel("Gathering Gnomon Smart Contract Data")
	content := container.NewVBox(
		layout.NewSpacer(),
		syncing,
		scids,
		label,
		layout.NewSpacer(),
	)
	syncro := dialog.NewCustomWithoutButtons("syncing", content, program.window)
	callback := func(b bool) {
		// if they cancel
		if !b {
			return
		}
		syncing.Start()
		scids.Hide()
		syncro.Resize(program.size)
		syncro.Show()
		label.SetText("Syncing with gnomon smart contract")
		go func() {

			var list_of_scids []string

			big_map := getSCValues(gnomonSC).VariableStringKeys
			lenMap := len(big_map)
			if lenMap == 0 {
				showError(errors.New("gnomon values are not in memory"), program.window)
				return
			}
			for k := range big_map {
				if strings.Contains(k, "owner") ||
					strings.Contains(k, "height") ||
					strings.Contains(k, "C") ||
					len(k) < 64 {
					continue
				}
				list_of_scids = append(list_of_scids, k)
			}

			scid_count := len(list_of_scids)

			// start a sync activity widget
			fyne.DoAndWait(func() {
				syncing.Stop()
				syncing.Hide()
				scids.Show()
				label.SetText("Scanning SCIDs")
			})
			scid_chan := make(chan string, len(list_of_scids))
			for _, scid := range list_of_scids {
				scid_chan <- scid
			}
			close(scid_chan)

			var wg sync.WaitGroup
			var counter int
			work := func() {
				defer wg.Done()
				for scid := range scid_chan {
					counter++
					fyne.DoAndWait(func() { scids.SetValue(float64(counter) / float64(scid_count)) })
					hash := crypto.HashHexToHash(scid)
					bal, _, err := program.wallet.GetDecryptedBalanceAtTopoHeight(hash, -1, program.wallet.GetAddress().String())
					if err != nil {
						continue
					}
					if bal != 0 {
						text := "ASSET FOUND: " + truncator(scid) + " Balance: " + rpc.FormatMoney(bal)
						fyne.DoAndWait(func() { label.SetText(text) })
						if err := program.wallet.TokenAdd(hash); err != nil {
							// obviously already in the map
						}
						// we are just going to set this now...
						program.wallet.GetAccount().Balance[hash] = bal

						// if there is a "better" balance, we'll let it happen here
						if err := program.wallet.Sync_Wallet_Memory_With_Daemon_internal(hash); err != nil {
							showError(err, program.window)
							continue
						} // seems like there isn't an error

					}
				}
			}
			var os_thread, app_thread int = 1, 1
			// reserve 1 thread for os management
			// reserve 1 thread for app management

			// we are going to use almost all threads
			max_threads := runtime.GOMAXPROCS(0)
			desired_threads := max_threads - os_thread - app_thread
			wg.Add(desired_threads)
			for range desired_threads {
				go work()
			}
			wg.Wait()
			fyne.DoAndWait(func() {
				scids.Hide()
				label.SetText("Rebuilding cache")
				syncing.Show()
				syncing.Start()
			})
			buildAssetHashList()
			fyne.DoAndWait(func() {
				showInfo("Asset Scan", "Scan complete", program.window)
				syncing.Stop()
				syncing.Hide()
				syncro.Dismiss()
			})
		}()
	}
	notice := `
this function will search through every smart contract in the network for a balance or entries and add the token to your collectibles.

it is recommended that you use a full node for best success.`
	scan = dialog.NewConfirm("Asset Scan", notice, callback, program.window)
	scan.Resize(program.size)
	scan.Show()

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
