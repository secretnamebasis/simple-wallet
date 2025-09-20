package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
)

func loggedIn() {

	// helper for ux

	// make little warbling light
	syncing := widget.NewProgressBar()

	//start it
	// syncing.Start()

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
	splash.Resize(program.size)
	splash.Show()

	// turn network on
	program.wallet.SetNetwork(program.preferences.Bool("mainnet"))

	program.wallet.SetOnlineMode()

	// make the address copiable
	address := program.wallet.GetAddress().String()
	program.hyperlinks.address.SetText(truncator(address))
	program.hyperlinks.address.OnTapped = func() {
		program.application.Clipboard().SetContent(address)
		showInfo("", "address copied to clipboard")
	}

	// update preferences
	program.preferences.SetBool("loggedIn", true)
	// show logged in
	program.labels.loggedin.SetText("WALLET: 🟢")

	// update balance every second
	go updateBalance()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// start sync with DERO history
		program.wallet.SyncHistory(crypto.ZEROHASH)
		// pull the assets list and build the cache
		buildAssetHashList()
		fyne.DoAndWait(func() {
			notice.SetText("history synced for DERO, beginning asset sync")
		})

		// sync asset histories

		// because we can't change the SyncHistory function...
		// histories are synced within limited concurrency
		// and then they have to be deduplicated

		// let's limit the number of concurrent jobs
		desired := runtime.GOMAXPROCS(0) - 2 // we reserve 1 for the os and 1 for the app
		// we will make a channel with a length of the desired number of threads
		capacity_channel := make(chan struct{}, desired)

		// let's make a wait group for this acitivity
		var assets_wg sync.WaitGroup

		// avoid concurrent writes to a map
		var mu sync.Mutex

		// let's count which jobs get done
		var completed int32
		// the total is assets plus DERO
		var total = len(program.caches.assets) + 1

		// range through the cache
		for _, asset := range program.caches.assets {

			// skip if dero is in there, just in case
			if asset.hash == crypto.ZEROHASH.String() {
				continue
			}

			hash := crypto.HashHexToHash(asset.hash)

			// separate go routine for each asset
			assets_wg.Add(1)
			go func(crypto.Hash) {
				defer assets_wg.Done()

				new_job := struct{}{}

				// load or wait for the channel to have capacity
				capacity_channel <- new_job

				// let's start a timer
				start := time.Now()

				// there is no-deduplication, de-duplicate entries immediately after
				program.wallet.SyncHistory(hash)

				// measure how long that took
				elapsed := time.Since(start)

				// let's make sure there are no duplicate entries after we have synced
				// lock this area up
				mu.Lock()

				// get the entries
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
				program.wallet.GetAccount().EntriesNative[hash] = deduped

				// and unlock this area
				mu.Unlock()

				// mark this one as completed
				atomic.AddInt32(&completed, 1)

				//update our calculation
				progress := (float64(atomic.LoadInt32(&completed)) / float64(total))

				msg := "asset: " + truncator(hash.String()) + " synced in: " + elapsed.String()
				fmt.Println(msg)
				// and update the user
				fyne.DoAndWait(func() {
					notice.SetText(msg)
					syncing.SetValue(progress)
				})

				// 'unnecessary assignment to the blank identifier (S1005)'
				// but we are doing it this way to explain what's going on better
				_ = <-capacity_channel // release capacity in the channel

			}(hash)
		}
		assets_wg.Wait()

		fyne.DoAndWait(func() {
			// be sure to turn off the syncing widget
			if syncing.Value != 1 {
				syncing.SetValue(1)
			}

			// and be sure to dismiss the splash
			splash.Dismiss()
		})
	}()

	// save wallet every second
	go isLoggedIn()

	// and while we are at it, notify me every time a new entry comes in
	go notificationNewEntry()

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

	// set keys
	program.entries.seed.SetText(program.wallet.GetSeed())
	program.entries.secret.SetText(program.wallet.Get_Keys().Secret.Text(16))
	program.entries.public.SetText(program.wallet.Get_Keys().Public.StringHex())
	// lock down keys
	program.entries.seed.Disable()
	program.entries.secret.Disable()

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

	// show buttons
	program.buttons.rpc_server.Show()
	program.buttons.assets.Show()
	program.buttons.send.Show()
	program.buttons.update_password.Show()

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
			showError(err)
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
	// first, let's check to see if we are logged-in
	if program.preferences.Bool("loggedIn") {
		// do the ole switcheroo
		program.hyperlinks.login.Hide()
		program.hyperlinks.logout.Show()

	}

	// here is a simple way to find their existing wallet
	program.entries.wallet.SetPlaceHolder("/path/to/wallet.db")
	program.entries.pass.SetPlaceHolder("w41137-p@55w0rd")

	// OnSubmitted accepts TypedKey Return as submission
	program.entries.pass.OnSubmitted = func(s string) {
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
		container.NewAdaptiveGrid(3,
			layout.NewSpacer(),
			container.NewVBox(

				container.NewVBox(program.entries.wallet),
				program.entries.pass,
				container.NewAdaptiveGrid(2,
					container.NewCenter(program.hyperlinks.create),
					container.NewCenter(program.hyperlinks.restore),
				),
			),
			layout.NewSpacer(),
		),
		layout.NewSpacer(),
	)

	// let's make a login for the wallet
	open_wallet := func(b bool) {
		// get these entries
		filename := program.entries.wallet.Text
		password := program.entries.pass.Text

		// be sure to dump the entries
		program.entries.wallet.SetText("")
		program.entries.pass.SetText("")

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
			showError(err)
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
	program.dialogues.login.Resize(program.size)
	program.dialogues.login.Show()
}
