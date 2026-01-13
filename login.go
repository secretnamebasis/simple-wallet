package main

import (
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
)

func loggedIn() {
	defer logger.Info("login", "status", "complete")
	// helper for ux

	// make little warbling light
	syncing := widget.NewActivity()

	//start it
	syncing.Start()

	// make some notice
	notice := makeCenteredWrappedLabel("Wallet is syncing with network\n\nPls hodl")

	// set the widgets to a container
	syncro := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(syncing),
		notice,
		layout.NewSpacer(),
	)

	// set the sync into a splash dialog
	splash := dialog.NewCustomWithoutButtons("Opening Wallet", syncro, program.window)
	splash.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/3))
	splash.Show()

	// turn network on
	program.wallet.SetNetwork(program.preferences.Bool("mainnet"))

	program.wallet.SetOnlineMode()

	// make the address copiable
	address := program.wallet.GetAddress().String()
	program.hyperlinks.address.SetText(truncator(address))
	program.hyperlinks.address.OnTapped = func() {
		program.application.Clipboard().SetContent(address)
		showInfoFast("", "Address copied to clipboard", program.window)
	}

	// update preferences
	program.preferences.SetBool("loggedIn", true)

	// update balance every second
	go updateBalance()
	logger.Info("balance loop", "status", "initiated")
	go updateCaches()
	logger.Info("cache loop", "status", "initiated")

	// and while we are at it, notify me every time a new entry comes in
	go notificationNewEntry()
	logger.Info("notification loop", "status", "initiated")

	program.wallet.SyncHistory(crypto.ZEROHASH)
	logger.Info("sync history", "status", "initiated")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// start sync with DERO history
		if program.wallet == nil {
			return
		}

		// pull the assets list and build the cache
		buildAssetHashList()
		logger.Info("asset cahce", "status", "complete")
		fyne.DoAndWait(func() {
			notice.SetText("DERO sync initiated\nAsset sync initiating")
		})

		// sync asset histories

		// because SyncHistory function can't change...
		// histories are synced within limited concurrency
		// and then deduplicated

		// let's start a timer
		start := time.Now()

		// limit the number of concurrent jobs
		desired := 1 // 1 at a time to limit network traffic

		// make a channel with a length of the desired number of capacity
		capacity_channel := make(chan struct{}, desired)

		// let's count which jobs get done
		var completed int32

		// range through the cache
		for _, asset := range program.caches.assets {

			// skip if dero is in there, just in case
			if asset.hash == crypto.ZEROHASH.String() {
				continue
			}

			hash := crypto.HashHexToHash(asset.hash)

			// separate go routine for each asset
			go func(crypto.Hash) {

				new_job := struct{}{}

				// load or wait for the channel to have capacity
				capacity_channel <- new_job

				// there is no-deduplication, de-duplicate entries immediately after
				if program.wallet == nil {
					return
				}
				program.wallet.SyncHistory(hash)

				// measure how long that took
				elapsed := time.Since(start)

				// let's make sure there are no duplicate entries after we have synced

				// get the entries
				if program.wallet == nil {
					return
				}
				entries := program.wallet.GetAccount().EntriesNative[hash]

				// make a seen map
				seen := make(map[string]bool)

				// make a new bucket for the entries
				var deduped []rpc.Entry

				// range through the entries from the sync process
				for _, entry := range entries {

					// check if we have not seen them
					if !seen[entry.TXID] {

						// insert the txid in the map and mark it as true
						seen[entry.TXID] = true

						// load the entry into the deduped bucket
						deduped = append(deduped, entry)
					}
				}
				// load the deduped entries into the wallet's hash map
				if program.wallet == nil {
					return
				}
				program.wallet.GetAccount().EntriesNative[hash] = deduped

				// mark this one as completed
				atomic.AddInt32(&completed, 1)

				//update our calculation
				logger.Info("syncing asset",
					"asset:", strconv.Itoa(int(completed)),
					" hash:", truncator(hash.String()),
					" synced in:", elapsed.String(),
				)

				<-capacity_channel // release capacity for the channel

			}(hash)
		}

		fyne.DoAndWait(func() {
			// be sure to turn off the syncing widget
			syncing.Stop()

			// and be sure to dismiss the splash
			splash.Dismiss()
		})
	}()

	// save wallet every second
	go isLoggedIn()
	logger.Info("wallet save loop", "status", "initiated")

	// check for registration
	if program.wallet.Wallet_Memory.IsRegistered() {
		// show them where to send
		program.containers.send.Show()

		// review assets
		program.buttons.assets.Show()
		program.buttons.assets.Enable()
		program.buttons.transactions.Enable()
		// they don't need to register
		program.containers.register.Hide()
	} else {
		program.buttons.assets.Disable()
		program.buttons.transactions.Disable()
		program.containers.register.Show()
		program.containers.send.Hide()
	}

	// hide login and notice
	program.hyperlinks.login.Hide()
	program.labels.notice.Hide()

	// show labels
	program.labels.address.Show()
	program.labels.balance.Show()

	// show links
	program.hyperlinks.tools.Show()
	program.hyperlinks.logout.Show()
	program.hyperlinks.lockscreen.Show()
	program.hyperlinks.address.Show()
	program.hyperlinks.generate.Show()

	// show buttons
	if !fyne.CurrentDevice().IsMobile() {
		program.buttons.rpc_server.Show()
		program.buttons.ws_server.Show()
	}

	program.buttons.assets.Show()
	program.buttons.send.Show()
	program.buttons.update_password.Show()
	program.buttons.balance_rescan.Show()

	// show containers
	program.containers.toolbox.Show()
	program.containers.dashboard.Show()

	// nice to be in a refreshed home
	program.containers.home.Refresh()
	program.containers.topbar.Refresh()

	// update the header to be home
	updateHeader(program.hyperlinks.home)

	// set the stage
	setContentAsHome()
}

func loginOpenFile() {
	callback := func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			showError(err, program.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()
		program.entries.wallet.SetText(reader.URI().Path())
	}
	file := dialog.NewFileOpen(callback, program.window)
	file.Resize(program.size)
	file.Show()
}

func loginFunction() {

	updateHeader(program.hyperlinks.login)

	pass := widget.NewPasswordEntry()
	// first, let's check to see if we are logged-in
	if program.preferences.Bool("loggedIn") {
		// do the ole switcheroo
		program.hyperlinks.login.Hide()
		program.hyperlinks.logout.Show()

	}

	// here is a simple way to find their existing wallet
	program.entries.wallet.SetPlaceHolder("/path/to/wallet.db")
	if fyne.CurrentDevice().IsMobile() {
		program.entries.wallet.SetPlaceHolder("wallet.db")
	}
	pass.SetPlaceHolder("w41137-p@55w0rd")

	// OnSubmitted accepts TypedKey Return as submission
	pass.OnSubmitted = func(s string) {
		program.dialogues.login.Confirm()
	}

	// if they don't know where it is they can find it graphically
	program.buttons.open_wallet.SetText("find wallet in explorer")
	program.buttons.open_wallet.OnTapped = loginOpenFile

	// 	let's make a simple way to create a new wallet in case they don't have one
	program.hyperlinks.create.SetText("create new wallet")
	program.hyperlinks.create.OnTapped = create

	// this will be our simple login container
	login_screen := container.NewVBox(
		layout.NewSpacer(),
		container.NewVBox(

			container.NewVBox(program.entries.wallet),
			pass,
			container.NewGridWithColumns(2,
				container.NewCenter(program.hyperlinks.create),
				container.NewCenter(program.hyperlinks.restore),
			),
		),
		layout.NewSpacer(),
	)

	// let's make a login for the wallet
	open_wallet := func(b bool) {
		// get these entries
		filename := program.entries.wallet.Text
		if fyne.CurrentDevice().IsMobile() {
			filename = filepath.Join(globals.GetDataDirectory(), filename)
		}
		password := pass.Text

		// be sure to dump the entries
		program.entries.wallet.SetText("")
		pass.SetText("")

		if !b { // in case they cancel
			return
		}

		var err error

		// open the wallet using the wallet path and password
		program.wallet, err = walletapi.Open_Encrypted_Wallet(filename, password)

		// if there is a problem...
		if err != nil || program.wallet == nil {

			// go home
			updateHeader(program.hyperlinks.home)
			setContentAsHome()

			// show the error
			showError(err, program.window)
			return
		}

		// simple way to know which file you are working with
		program.window.SetTitle(program.name + " | " + filename)

		// and then do the loggedIn thing
		loggedIn()
	}

	// load the login screen into the login dialog in order to open the wallet
	program.dialogues.login = dialog.NewCustomConfirm("", "login", dismiss,
		login_screen, open_wallet, program.window,
	)
	program.dialogues.login.Resize(fyne.NewSize(program.size.Width/3, program.size.Height/3))
	program.dialogues.login.Show()
}
