package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/walletapi"
)

func loggedIn() {

	// helper for ux

	// make little warbling light
	syncing := widget.NewActivity()

	//start it
	syncing.Start()

	// make some notice
	notice := makeCenteredWrappedLabel("Wallet is syncing with network\n\nPls hodl")

	// set the widgets to a container
	sync := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(syncing),
		notice,
		layout.NewSpacer(),
	)

	// set the sync into a splash dialog
	splash := dialog.NewCustomWithoutButtons("Opening Wallet", sync, program.window)
	splash.Resize(program.size)
	splash.Show()

	// turn network on
	program.wallet.SetNetwork(true)
	program.wallet.SetOnlineMode()

	go func() { // catch up with the network

		// while in the go routine...
		fyne.DoAndWait(func() {

			// be sure to turn off the syncing widget
			syncing.Stop()

			// and be sure to dismiss the splash
			splash.Dismiss()
		})
	}()

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
	program.labels.loggedin.SetText("💰: 🟢")

	// update balance every second
	go updateBalance()

	// build the cache
	go buildAssetHashList()

	// save wallet every second
	go isLoggedIn()

	// // let's make a simple way to have a pong db
	go startDB()

	// check for registration
	if program.wallet.Wallet_Memory.IsRegistered() {
		// show them where to send
		program.containers.send.Show()

		// review assets
		program.hyperlinks.assets.Show()

		// they don't need to register
		program.containers.register.Hide()
	} else {
		program.containers.register.Show()
		program.hyperlinks.assets.Hide()
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

	// show links
	program.hyperlinks.rpc_server.Show()
	program.hyperlinks.tools.Show()
	program.hyperlinks.logout.Show()
	program.hyperlinks.send.Show()
	program.hyperlinks.pong_server.Show()
	program.hyperlinks.lockscreen.Show()

	// show containers
	program.containers.toolbox.Show()
	program.containers.dashboard.Show()

	// nice to be in a refreshed home
	program.containers.home.Refresh()

	// update the header to be home
	updateHeader(program.hyperlinks.home)

	// set the stage
	setContentAsHome()
}

func loginOpenFile() {
	file := dialog.NewFileOpen(
		func(reader fyne.URIReadCloser, err error) {
			if err != nil {
				showError(err)
				return
			}
			if reader == nil {
				return
			}
			defer reader.Close()
			program.entries.wallet.SetText(reader.URI().Path())
		},
		program.window,
	)
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

	// if they don't know where it is they can find it graphically
	program.hyperlinks.open_wallet.SetText("find wallet in explorer")
	program.hyperlinks.open_wallet.OnTapped = loginOpenFile

	// 	let's make a simple way to create a new wallet in case they don't have one
	program.hyperlinks.create.SetText("create new wallet")
	program.hyperlinks.create.OnTapped = create

	// this will be our simple login container
	login_screen := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(program.hyperlinks.open_wallet),
		program.entries.wallet,
		program.entries.pass,
		container.NewAdaptiveGrid(2,
			container.NewCenter(program.hyperlinks.create),
			container.NewCenter(program.hyperlinks.restore),
		),
		layout.NewSpacer(),
	)

	// let's make a login for the wallet
	open_wallet := func(b bool) {
		if b {
			var err error

			// open the wallet using the wallet path and password
			program.wallet, err = walletapi.Open_Encrypted_Wallet(
				program.entries.wallet.Text, program.entries.pass.Text,
			)

			// if there is a problem...
			if err != nil || program.wallet == nil {

				// clear the password
				program.entries.pass.SetText("")

				// go home
				updateHeader(program.hyperlinks.home)
				setContentAsHome()

				// show the error
				showError(err)
				return
			}

			// be sure to dump the entries
			program.entries.wallet.SetText("")
			program.entries.pass.SetText("")

			// and then do the loggedIn thing
			loggedIn()
		}
	}
	// load the login screen into the login dialog in order to open the wallet
	program.dialogues.login = dialog.NewCustomConfirm("", "login", dismiss,
		login_screen, open_wallet, program.window,
	)
	program.dialogues.login.Resize(program.size)
	program.dialogues.login.Show()
}
