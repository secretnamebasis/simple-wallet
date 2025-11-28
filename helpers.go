package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"maps"
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
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/mnemonics"
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

	if _, err := os.Stat(filename + ".conf"); err != nil {
		os.Create(filename + ".conf")
		// really
	} else {
		file, err := os.Open(filename + ".conf")
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
	var now, height int64
	now = walletapi.Get_Daemon_TopoHeight()
	for range ticker.C {
		height = walletapi.Get_Daemon_TopoHeight()
		if now < height {
			now = height
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
}
func notificationNewEntry() {
	// we are going to be a little aggressive here
	ticker := time.NewTicker(time.Second)
	// and because we aren't doing any fancy websocket stuff...
	var old_len int
	for range ticker.C { // range that ticker
		if !program.preferences.Bool("notifications") {
			continue
		}
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
		var current_transfers wallet_entries
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
	callback := func() {
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
					program.labels.balance.SetText(rpc.FormatMoney(0))

				})
			} else if bal == 0 && !program.wallet.IsRegistered() {
				fyne.DoAndWait(func() {
					// update it
					program.labels.loggedin.SetText("WALLET: ✅")
					program.labels.balance.SetText("unregistered")

				})
			}
			// check if there is a wallet first
			if program.wallet == nil {
				return
			}

			// get the balance
			if !program.preferences.Bool("loggedIn") {
				return
			}

			// hella sensitive
			bal, _ = program.wallet.Get_Balance()

			// check it against previous
			if previous_bal != bal {

				// update
				fyne.DoAndWait(func() {
					// obviously, we are still logged in
					program.labels.loggedin.SetText("WALLET: ✅")

					// update it
					program.labels.balance.SetText(rpc.FormatMoney(bal))

				})
			}
		}
	}
	callback()
	ticker := time.NewTicker(time.Second * 2)
	new := int64(0)
	for range ticker.C {
		height := walletapi.Get_Daemon_TopoHeight()
		if new < height {
			new = height
			callback()
		} else {
			continue
		}
	}
}
func updateCaches() {
	marker := walletapi.Get_Daemon_TopoHeight()
	for range time.NewTicker(time.Second * 2).C {
		height := walletapi.Get_Daemon_TopoHeight()
		if marker < height {
			marker = height
			program.node.info = getDaemonInfo()
			program.node.pool = getTxPool()
		}
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
	// range over any pre-existing entries in the account
	for a := range assets {
		// skip DERO's scid
		if crypto.HashHexToHash(a.String()).IsZero() {
			continue
		}
		wg.Add(1)
		go func(scid string) {
			defer wg.Done()
			keys := getSCValues(scid).VariableStringKeys
			t := asset{
				name:  getSCNameFromVars(keys),
				hash:  scid,
				image: getSCIDImage(keys),
			}
			logger.Info("asset cache", "loading into the cache", truncator(scid))
			// load each has into the cache
			program.caches.assets = append(program.caches.assets, t)
		}(a.String())
	}
	wg.Wait()
	// now sort them for consistency
	program.caches.assets.sort()
}
func getSCNameFromVars(keys map[string]interface{}) string {
	var text string

	for k, v := range keys {
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

func getTxsAndTransactions(hashes []crypto.Hash) (txs []rpc.GetTransaction_Result, transactions []transaction.Transaction) {
	for _, each := range hashes {
		if _, ok := program.node.transactions[each.String()]; !ok {
			// update the cache
			program.node.transactions[each.String()] = getTransaction(
				rpc.GetTransaction_Params{
					Tx_Hashes: []string{each.String()},
				},
			)
		}

		if len(program.node.transactions[each.String()].Txs_as_hex) == 0 {
			continue // there is nothing here ?
		}
		tx := program.node.transactions[each.String()]
		txs = append(txs, tx)
		var transaction transaction.Transaction
		b, err := hex.DecodeString(tx.Txs_as_hex[0])
		if err != nil {
			logger.Error(err, "lol")
			continue
		}
		transaction.Deserialize(b)
		transactions = append(transactions, transaction)
	}
	return
}
func isRegistered(s string) bool {
	result := callRPC("DERO.GetEncryptedBalance",
		rpc.GetEncryptedBalance_Params{
			Address:    s,  // the address to check
			TopoHeight: -1, // the top of the chain
		},
		func(r rpc.GetEncryptedBalance_Result) bool {
			return r.Registration != 0
		})

	// if there is no error and the registration isn't 0
	return result.Registration != 0
}

// simple way to test http connection to derod
func testConnection(s string) error {

	// make a new request
	req, err := http.NewRequest("GET", "http://"+s, nil)
	if err != nil {
		logger.Error(err, "connection error")
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
		logger.Error(err, "connection error")
		return err
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
		err := errors.New("body does not contain DERO")
		logger.Error(err, string(body)) // might not be a bad idea to know what they are saying

		return err
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
func createTXListSearchBar(e *widget.Entry) fyne.CanvasObject {
	return container.NewAdaptiveGrid(3, layout.NewSpacer(), e, layout.NewSpacer())
}
func createTXListHeader() fyne.CanvasObject {
	sent_header := createThreeLabels()
	sent_header.(*fyne.Container).Objects[0].(*widget.Label).SetText("DATE")
	sent_header.(*fyne.Container).Objects[1].(*widget.Label).SetText("TXID")
	sent_header.(*fyne.Container).Objects[2].(*widget.Label).SetText("AMOUNT")
	return sent_header
}
func createOneLabel() fyne.CanvasObject {
	return createLabels(1)
}
func createTwoLabels() fyne.CanvasObject {
	return createLabels(2)
}
func createThreeLabels() fyne.CanvasObject {
	return createLabels(3)
}
func createFourLabels() fyne.CanvasObject {
	return createLabels(4)
}

func createLabels(c int) fyne.CanvasObject {
	cont := container.NewAdaptiveGrid(c)
	for range c {
		cont.Add(widget.NewLabel(""))
	}
	return cont
}

func makeCenteredWrappedLabel(s string) *widget.Label {
	label := widget.NewLabel(s)
	label.Alignment = fyne.TextAlignCenter
	label.Wrapping = fyne.TextWrapWord
	return label
}
func callRPC[t any](method string, params any, validator func(t) bool) t {
	result, err := handleResult[t](method, params)
	if err != nil {
		logger.Error(err, "RPC error", "method", method)
		var zero t
		return zero
	}

	if !validator(result) {
		logger.Error(errors.New("failed validation"), method)
		var zero t
		return zero
	}

	return result
}
func handleResult[T any](method string, params any) (T, error) {
	var result T
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	if method == "DERO.GetSC" {
		ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(deadline))
	}
	defer cancel()

	rpcClient := jsonrpc.NewClient("http://" + walletapi.Daemon_Endpoint + "/json_rpc")

	var err error
	if params == nil {
		err = rpcClient.CallFor(ctx, &result, method) // no params argument
	} else {
		err = rpcClient.CallFor(ctx, &result, method, params)
	}

	if err != nil {
		var zero T
		return zero, err
	}

	return result, nil
}

func getTransaction(params rpc.GetTransaction_Params) rpc.GetTransaction_Result {
	validator := func(r rpc.GetTransaction_Result) bool {
		return r.Status != ""
	}
	result := callRPC("DERO.GetTransaction", params, validator)
	return result
}

func getBlockInfo(params rpc.GetBlock_Params) rpc.GetBlock_Result {
	validator := func(r rpc.GetBlock_Result) bool {
		return r.Block_Header.Depth != 0
	}
	result := callRPC("DERO.GetBlock", params, validator)
	return result
}

func getTxPool() rpc.GetTxPool_Result {
	validator := func(r rpc.GetTxPool_Result) bool {
		return r.Status != ""
	}
	result := callRPC("DERO.GetTxPool", nil, validator)
	return result
}

func getDaemonInfo() rpc.GetInfo_Result {
	validator := func(r rpc.GetInfo_Result) bool {
		return r.TopoHeight != 0
	}
	result := callRPC("DERO.GetInfo", nil, validator)
	return result
}

func getSC(scParam rpc.GetSC_Params) rpc.GetSC_Result {
	validator := func(r rpc.GetSC_Result) bool {
		if scParam.Code {
			return r.Code != ""
		}
		return true
	}
	result := callRPC("DERO.GetSC", scParam, validator)
	return result
}

func getSCCode(scid string) rpc.GetSC_Result {
	return getSC(rpc.GetSC_Params{
		SCID:       scid,
		Code:       true,
		Variables:  false,
		TopoHeight: walletapi.Get_Daemon_Height(),
	})
}

func getSCValues(scid string) rpc.GetSC_Result {
	return getSC(rpc.GetSC_Params{
		SCID:       scid,
		Code:       false,
		Variables:  true,
		TopoHeight: walletapi.Get_Daemon_Height(),
	})
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
func getSCIDImage(keys map[string]interface{}) image.Image {
	for k, v := range keys {
		if !strings.Contains(k, "image") && !strings.Contains(k, "icon") {
			continue
		}
		encoded := v.(string)
		b, e := hex.DecodeString(encoded)
		if e != nil {
			logger.Error(e, encoded)
			continue
		}
		value := string(b)
		logger.Info("scid", "key", k, "value", value)
		uri, err := storage.ParseURI(value)
		if err != nil {
			logger.Error(err, value)
			return nil
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, "GET", uri.String(), nil)
			if err != nil {
				logger.Error(err, "get error")
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

		scid := pair.key
		amnt := pair.value
		keys := getSCValues(scid).VariableStringKeys

		bals.Add(container.NewAdaptiveGrid(5,
			layout.NewSpacer(),
			widget.NewLabel(getSCNameFromVars(keys)),
			widget.NewLabel(truncator(scid)),
			widget.NewLabel(rpc.FormatMoney(amnt)),
			layout.NewSpacer(),
		))
	}
	return bals
}
func split_scid_keys(keys any) (k []string, v []string) {
	switch pairs := keys.(type) {
	case map[uint64]any:
		uint64_pairs := []struct {
			key   uint64
			value any
		}{}
		for k, v := range pairs {
			uint64_pairs = append(uint64_pairs, struct {
				key   uint64
				value any
			}{key: k, value: v})
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
		return keys, values
	case map[string]any:
		string_pairs := []struct {
			key   string
			value any
		}{}
		for k, v := range pairs {
			string_pairs = append(string_pairs, struct {
				key   string
				value any
			}{key: k, value: v})
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
				if p.key == "C" {
					value = v // large strings break fyne
				} else {
					b, e := hex.DecodeString(v)
					if e != nil {
						continue
					}
					value = string(b)
				}

			case uint64:
				value = strconv.Itoa(int(v))
			case float64:
				value = strconv.FormatFloat(v, 'f', 0, 64)
			}
			keys = append(keys, p.key)
			values = append(values, value)
		}
		return keys, values
	default:
		return nil, nil
	}
}
func getSCIDStringVarsContainer(keys, values []string) *fyne.Container {
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
		switch id.Col {
		case 0:
			data = keys[id.Row]
		case 1:
			data = values[id.Row]
			if keys[id.Row] == "C" { // we truncate the c value because it is big...
				data = truncator(values[id.Row])
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

func getSCIDUint64VarsContainer(keys, values []string) *fyne.Container {
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
func showError(e error, w fyne.Window) {
	logger.Error(e, "showing error")
	dialog.ShowError(e, w)
}

func showInfo(t, m string, w fyne.Window) {
	logger.Info(t, "msg", m)
	i := dialog.NewInformation(t, m, w)
	i.Resize(fyne.NewSize(
		program.size.Width/2,
		program.size.Height/3,
	))
	i.Show()
}
func showInfoFast(t, m string, w fyne.Window) {
	logger.Info(t, "msg", m)
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
	program.preferences.SetBool("isLocked", true)
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
		program.preferences.SetBool("isLocked", false)
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
	// let's make a clone in case it changes while making the graph
	hd := make(map[int]int, len(hd_map))
	maps.Copy(hd, hd_map)

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

	if len(hd) == 0 {
		return canvas.NewText("No data", color.Black)
	}

	// we are goind to index the heights
	heights := []int{}

	// need some min maxing while we are at it
	var first_height, last_height, max_difficulty int

	// range, append and process
	for h, d := range hd {
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
		difficulty := hd[height]

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
		if i%50 != 0 { // reports of stalls?
			continue
		}

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
	quick := widget.NewHyperlink("new random address", nil)
	quick.Alignment = fyne.TextAlignCenter
	quick.Wrapping = fyne.TextWrapBreak
	quick.OnTapped = func() {
		if quick.Text == "new random address" {
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
			quick.SetText(addr.String())
			quick.OnTapped = func() {
				program.application.Clipboard().SetContent(addr.String())
				showInfo("Copied", addr.String()+"\ncopied to clipboard", program.window)
			}
		} else {
			// is there?
		}
	}
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
			return
		}

		// if the following is empty and unchecked...
		if dst.Text == "" &&
			value.Text == "" &&
			comment.Text == "" &&
			!needs_replyback.Checked {

			// generate a random address
			result := program.wallet.GetRandomIAddress8()

			quick.SetText(result.String())

			quick.OnTapped = func() {
				program.application.Clipboard().SetContent(result.String())
				showInfo("Copied", result.String()+"\ncopied to clipboard", program.window)
			}
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

			quick.SetText(result.String())
			quick.OnTapped = func() {
				program.application.Clipboard().SetContent(result.String())
				showInfo("Copied", result.String()+"\ncopied to clipboard", program.window)
			}
		}

	}

	form := widget.NewForm([]*widget.FormItem{
		widget.NewFormItem("", value),
		widget.NewFormItem("", dst),
		widget.NewFormItem("", comment),
		widget.NewFormItem("", needs_replyback),
	}...)

	form.OnSubmit = func() {
		callback(true)
	}
	form.SubmitText = "Generate New"
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
	cancel := make(chan struct{})
	syncro.SetButtons([]fyne.CanvasObject{widget.NewButton("cancel", func() {
		label.SetText("Cancel initiated, pls wait")
		close(cancel)
	})})
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
					select {
					case <-cancel:
						return
					default:
					}
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
			finish := func() {
				buildAssetHashList()
				fyne.DoAndWait(func() {
					syncing.Stop()
					syncing.Hide()
					program.lists.asset_list.Refresh()
					syncro.Dismiss()
				})
			}
			select {
			case <-cancel:
				finish()
				return
			default:
				finish()
				fyne.DoAndWait(func() {
					showInfo("Asset Scan", "Scan complete", program.window)
				})

			}

		}()
	}
	notice := `
this function will search through every smart contract in the network for a balance or entries and add the token to your collectibles.

it is recommended that you use a full node for best success.`
	scan = dialog.NewConfirm("Asset Scan", notice, callback, program.window)
	scan.Resize(program.size)
	scan.Show()

}

// for count append random words with a separator
func randomWords(count int, sep string) (words string) {
	for i := range count {
		r := rand.Intn(len(mnemonics.Mnemonics_English.Words) - 1)
		words += mnemonics.Mnemonics_English.Words[r]
		if i != count-1 {
			words += sep
		}
	}
	return
}

func setText(txt string, text *widget.Label) {
	var opts string
	// let's show off a list
	switch {
	case program.sliders.network.Value < 0.33:
		opts = "preferred if set, then localhost, then the fastest public node:\n\n"
	case program.sliders.network.Value > 0.33 && program.sliders.network.Value < 0.66:
		opts = program.node.current
	case program.sliders.network.Value > 0.66:
		opts = program.node.current
	}

	text.SetText(txt + opts)
}

func slide_network(f float64) {
	var msg string = "Auto-connects to "
	if program.sliders.network.Value >= 0 && program.sliders.network.Value < 0.33 {
		program.labels.mainnet.TextStyle.Bold = true
		program.labels.testnet.TextStyle.Bold = false
		program.labels.simulator.TextStyle.Bold = false
		program.labels.mainnet.Refresh()
		program.labels.testnet.Refresh()
		program.labels.simulator.Refresh()
		program.sliders.network.SetValue(0.1337)
		program.tables.connections.Show()
		globals.Arguments["--testnet"] = false
		globals.Arguments["--simulator"] = false
		program.preferences.SetBool("mainnet", true)
		program.entries.node.PlaceHolder = "127.0.0.1:10102"
		program.node.current = program.node.list[0].ip // if not set up, it will roll through
		program.labels.current_node.SetText("Current Node: " + program.node.current)
		program.entries.node.Refresh()
		globals.InitNetwork()
		// lets create data directories
		if err := os.MkdirAll(globals.GetDataDirectory(), 0750); err != nil {
			panic(err)
		}

		setText(msg, program.labels.notice)
	}
	if program.sliders.network.Value > 0.33 && program.sliders.network.Value < 0.66 {
		program.labels.mainnet.TextStyle.Bold = false
		program.labels.testnet.TextStyle.Bold = true
		program.labels.simulator.TextStyle.Bold = false
		program.labels.mainnet.Refresh()
		program.labels.testnet.Refresh()
		program.labels.simulator.Refresh()

		program.sliders.network.SetValue(0.5)
		program.tables.connections.Hide()
		globals.Arguments["--testnet"] = true
		globals.Arguments["--simulator"] = false
		program.preferences.SetBool("mainnet", false)
		program.node.current = "127.0.0.1:40402"
		program.entries.node.PlaceHolder = program.node.current
		program.labels.current_node.SetText("Current Node: " + program.node.current)
		program.entries.node.Refresh()
		globals.InitNetwork()
		// lets create data directories
		if err := os.MkdirAll(globals.GetDataDirectory(), 0750); err != nil {
			panic(err)
		}

		setText(msg, program.labels.notice)
	}
	if program.sliders.network.Value > 0.66 && program.sliders.network.Value <= 1 {
		program.labels.mainnet.TextStyle.Bold = false
		program.labels.testnet.TextStyle.Bold = false
		program.labels.simulator.TextStyle.Bold = true
		program.labels.mainnet.Refresh()
		program.labels.testnet.Refresh()
		program.labels.simulator.Refresh()
		program.sliders.network.SetValue(0.85)
		program.tables.connections.Hide()
		globals.Arguments["--testnet"] = true
		globals.Arguments["--simulator"] = true
		program.preferences.SetBool("mainnet", false)
		program.node.current = "127.0.0.1:20000"
		program.entries.node.PlaceHolder = program.node.current
		program.labels.current_node.SetText("Current Node: " + program.node.current)
		program.entries.node.Refresh()
		globals.InitNetwork()
		// lets create data directories
		if err := os.MkdirAll(globals.GetDataDirectory(), 0750); err != nil {
			panic(err)
		}

		setText(msg, program.labels.notice)
	}
}

func addressValidator(s string) (err error) {

	// any changes to the string should immediately update the receiver string
	program.receiver = ""

	switch {
	case s == "":
		return nil
	case len(s) != 66:
		if len(s) < 5 {
			return errors.New("cannot be less than 5 char, sry capt")
		}
		// check to see if it is a name
		a, err := program.wallet.NameToAddress(s)
		if err != nil && strings.Contains(err.Error(), "leaf not found") {
			return errors.New("invalid DERO NameAddress")
		}
		if a != "" {
			program.receiver = a
		} else {
			return errors.New("invalid DERO NameAddress")
		}
	case len(s) == 66:
		addr, err := rpc.NewAddress(s)
		if err != nil {
			return errors.New("invalid DERO address")
		}
		if addr.IsIntegratedAddress() {

			// the base of that address is what we'll use as the receiver
			program.receiver = addr.BaseAddress().String()

		} else if addr.String() != "" && // if the addr isn't empty
			!addr.IsIntegratedAddress() { // now if it is not an integrated address

			// set the receiver
			program.receiver = addr.String()
		}
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
