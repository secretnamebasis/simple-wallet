package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
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
func notificationNewEntry() {
	// we are going to be a little aggressive here
	ticker := time.NewTicker(time.Second)
	// and because we aren't doing any fancy websocket stuff...
	var old_len int
	for range ticker.C { // range that ticker
		// check if we are still logged in
		if !program.preferences.Bool("loggedIn") {
			break
		}
		// check if the wallet is present
		if program.wallet == nil {
			continue
		}
		// check if we are registered
		if !program.wallet.IsRegistered() {
			continue
		}

		// go get the transfers
		var current_transfers []rpc.Entry
		if program.wallet != nil { // expressly validate this
			current_transfers = getAllTransfers()
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
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		// check to see if we are logged-in first
		if !program.preferences.Bool("loggedIn") {
			fyne.DoAndWait(func() {
				program.labels.loggedin.SetText("WALLET: 🔴")
				program.labels.balance.SetText(
					fmt.Sprintf("BALANCE: %s", rpc.FormatMoney(0)))
				bal, previous_bal = 0, 0
			})
		} else {
			if bal == 0 && program.wallet.IsRegistered() {
				fyne.DoAndWait(func() {
					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("BALANCE: %s", "syncing"))

				})
			} else if bal == 0 && !program.wallet.IsRegistered() {
				fyne.DoAndWait(func() {
					// update it
					program.labels.balance.SetText(
						fmt.Sprintf("BALANCE: %s", "unregistered"))

				})
			}
			// check if there is a wallet first
			if program.wallet == nil {
				return
			}

			// get the balance
			bal, _ = program.wallet.Get_Balance()

			// check it against previous
			if previous_bal != bal {

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
func getTransfers(coin, in, out bool) []rpc.Entry {
	return program.wallet.Show_Transfers(
		crypto.ZEROHASH,
		coin, in, out,
		0, uint64(walletapi.Get_Daemon_Height()),
		"", "",
		0, 0,
	)
}

// simple way to get all transfers
func getAllTransfers() []rpc.Entry {
	return getTransfers(true, true, true)
}

// simple way to get all transfers
func getCoinbaseTransfers() []rpc.Entry {
	return getTransfers(true, false, false)
}

// simple way to get all transfers
func getReceivedTransfers() []rpc.Entry {
	return getTransfers(false, true, false)
}

// simple way to get all transfers
func getSentTransfers() []rpc.Entry {
	return getTransfers(false, false, true)
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
		fmt.Println(err)
		return rpc.GetInfo_Result{}
	}
	// the code needs to be present
	if info.TopoHeight == 0 {
		return rpc.GetInfo_Result{}
	}

	return info
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
func setSCIDThumbnail(img image.Image, h, w float32) *canvas.Image {
	var thumbnail = new(canvas.Image)
	thumbnail = canvas.NewImageFromResource(theme.BrokenImageIcon())
	thumbnail.SetMinSize(fyne.NewSize(w, h))
	if img == nil {
		return thumbnail
	}
	thumb := image.NewNRGBA(image.Rect(0, 0, int(h), int(w)))
	draw.ApproxBiLinear.Scale(thumb, thumb.Bounds(), img, img.Bounds(), draw.Over, nil)
	thumbnail = canvas.NewImageFromImage(thumb)
	return thumbnail
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
					} else {
						return i
					}
				}
			}
		}
	}
	return nil
}
func getSCIDBalancesContainer(scid string) *fyne.Container {
	balances := container.NewVBox()
	balance_pairs := []struct {
		key   string
		value uint64
	}{}
	for k, v := range getSCValues(scid).Balances {
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

		balances.Add(container.NewAdaptiveGrid(5,
			layout.NewSpacer(),
			widget.NewLabel(getSCNameFromVars(pair.key)),
			widget.NewLabel(truncator(pair.key)),
			widget.NewLabel(rpc.FormatMoney(pair.value)),
			layout.NewSpacer(),
		))
	}
	return balances
}

func getSCIDStringVarsContainer(scid string) *fyne.Container {
	string_vars := container.NewVBox()

	if len(getSCValues(scid).VariableStringKeys) != 0 {
		string_pairs := []struct {
			key   string
			value any
		}{}

		for k, v := range getSCValues(scid).VariableStringKeys {
			if k == "C" {
				continue
			}
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

		for _, pair := range string_pairs {
			var value string
			switch v := pair.value.(type) {
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
			string_vars.Add(container.NewAdaptiveGrid(2,
				widget.NewLabel(pair.key),
				widget.NewLabel(value),
			))
		}
	} else {
		string_vars.Add(widget.NewLabel("N/A"))
	}
	return string_vars
}

func getSCIDUint64VarsContainer(scid string) *fyne.Container {
	uint64_vars := container.NewVBox()

	if len(getSCValues(scid).VariableUint64Keys) != 0 {
		uint64_pairs := []struct {
			key   uint64
			value any
		}{}
		for k, v := range getSCValues(scid).VariableUint64Keys {

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

		for _, pair := range uint64_pairs {
			var value string
			switch v := pair.value.(type) {
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
			uint64_vars.Add(container.NewAdaptiveGrid(2,
				widget.NewLabel(strconv.Itoa(int(pair.key))),
				widget.NewLabel(value),
			))
		}
	} else {
		uint64_vars.Add(widget.NewLabel("N/A"))
	}
	return uint64_vars
}

// simple way to show error
func showError(e error) { dialog.ShowError(e, program.window) }

func showInfo(t, m string) { dialog.ShowInformation(t, m, program.window) }

// simple way to go home
func setContentAsHome() { program.window.SetContent(program.containers.home) }

func lockScreen() {

	content := container.NewVBox(
		layout.NewSpacer(),
		program.entries.pass,
		program.hyperlinks.unlock,
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
			showError(errors.New("wrong password"))
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
