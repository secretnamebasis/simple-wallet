package main

import (
	"context"
	"crypto/rand"
	"fmt"
	m_rand "math/rand"
	"os"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
)

// main caller
func main() {
	msg := "let's start with a simple design philosophy:\n\n"
	msg += "	\"When it comes to programs [...], \n"
	msg += "	both programmers and compilers should remember the advice: \n"
	text := "	\033[38;5;" + strconv.Itoa(m_rand.Intn(256)) + "mdon't be clever\033[0m"
	msg += text + ". - Credit to: https://go.dev/ref/mem"
	fmt.Println(msg)
	run() // the program
}

func run() {
	// let's make some random session so that users can run multiple wallets at once
	session_id := rand.Text()
	// let's set up our program using the name we've chosen and the session id
	program.application = app.NewWithID(program.name + "_" + session_id)
	fyne.SetCurrentApp(program.application)

	fyne.CurrentApp().SetIcon(resourceIconPng)
	program.application.SetIcon(resourceIconPng)

	// let's use a simple theme that changes only one thing
	program.application.Settings().SetTheme(customTheme{})

	// we''l assume some simple preferences
	program.preferences = program.application.Preferences()

	if fyne.CurrentDevice().IsMobile() {
		store := fyne.CurrentApp().Storage()
		dir := store.RootURI().Path()
		fmt.Println("Storage path:", dir)
		program.preferences.SetString("HOME", dir)
		globals.Arguments["--data-dir"] = dir
	}

	// let's set a simple lifecycle policy when the app starts
	program.application.Lifecycle().SetOnStarted(func() {
		// for instance, let's make sure we have at least one preference set
		program.preferences.SetBool("loggedIn", false)

		// and let's make one style change
		program.hyperlinks.home.TextStyle = fyne.TextStyle{
			Bold: true,
		}
	})

	// this is the main window
	program.window = program.application.NewWindow(program.name)

	// let's size the window, I think this is a nice size
	program.window.Resize(program.size)

	// let's center it to make things simple
	program.window.CenterOnScreen()

	// let's use a simple icon
	program.window.SetIcon(resourceIconPng)

	// let's set a simple intercept close for the window
	program.window.SetCloseIntercept(func() {
		logger.Info("closing")
		if program.loggedIn {
			program.wallet.Close_Encrypted_Wallet()
		}
		if program.wallet != nil {
			program.wallet = nil
		}
		logger.Info("closed")
		os.Exit(0)
	})

	// the app is live, initialize!
	initialize()

	// let's set a simple home container
	setContentAsHome()

	// let's simply show and run the program
	once()
	// on mobile:
	//  *** Error in Fyne call thread, this should have been called in fyne.Do[AndWait] ***
	// From: github.com/secretnamebasis/simple-wallet/main.go:92
	// doesn't seem like this caller works right
}

var once = sync.OnceFunc(func() { program.window.ShowAndRun() })

func initialize() {
	// captain's orders
	go initialize_table()

	initialize_logger() // there is no logs in mobile, but we might be able to dump logs

	// simple way to create a preferred ip endpoint file

	createPreferred()

	// test localhost first, then connect from a list of public nodes

	ctxConnection, cancelConnection = context.WithCancel(context.Background())

	// android prevents background networking,
	// would need to create a permanent notification...
	// we'll just disconnect and re-connect
	fyne.CurrentApp().Lifecycle().SetOnEnteredForeground(func() {
		ctxConnection, cancelConnection = context.WithCancel(context.Background())
		if !walletapi.Connected && program.node.list[0].ip != "" {
			go maintain_connection()
		}
	})
	if fyne.CurrentDevice().IsMobile() {
		fyne.CurrentApp().Lifecycle().SetOnExitedForeground(func() {
			walletapi.Connected = false
			cancelConnection()
		})
	}

	// let's make sure those notifications are off at start :)
	program.sliders.notifications.SetValue(.235)

	// let's start with the bottom of the application
	program.containers.bottombar = bottombar()

	// now let's fill in the top too
	program.containers.topbar = topbar()

	// let's have an easy way to see address and balances
	// but let's hide them for a moment
	program.labels.address.Hide()
	program.hyperlinks.address.Hide()
	program.hyperlinks.generate.Hide()
	program.labels.balance.Hide()

	// let's make a simple dashboard
	program.containers.dashboard = dashboard()
	// and let's hide it for the moment
	program.containers.dashboard.Hide()

	// here is a simple way to select a wallet file
	program.entries.wallet.ActionItem = widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		explorer := openWalletFile()
		explorer.Resize(program.size)
		explorer.Show()
	})

	// let's make a simple way to login
	program.hyperlinks.login.OnTapped = loginFunction

	// let's make a simple way to restore a wallet
	program.hyperlinks.restore.OnTapped = restore

	// let's make a simple way to register
	program.containers.register = register()
	// and we'll hide it for a moment
	program.containers.register.Hide()

	// let's make a simple way to send
	program.containers.send = send()
	// let's go and hide send for a moment as well
	program.containers.send.Hide()

	// let's make some tools
	program.containers.tools = tools()
	program.hyperlinks.tools.Hide()

	// and now, let's hide the toolbox
	program.containers.toolbox.Hide()

	// as a precaution, let's make sure that
	// these text fields are treated like passwords
	// that way, their visibility can be toggled
	// and fyne won't call on the app before it is launched
	program.entries.pass = widget.NewPasswordEntry()

	// almost nothing simpler than home.
	program.containers.home = home()

	// some simple configs for the time being
	program.containers.configs = configs()
	program.toggles.ws_server.OnChanged = ws_toggle
	program.toggles.indexer.OnChanged = idx_toggle

	// here is a simple way to get started
	program.preferences.SetBool("mainnet", true)
	program.sliders.network.OnChanged = slide_network
	program.sliders.network.Step = 0.0001
	program.sliders.network.SetValue(0.1337) // mainnet

	// and simple place for logging out
	program.hyperlinks.logout.OnTapped = logout
	// let's hide this for a minute
	program.hyperlinks.logout.Hide()

	// here is a simple lockscreen
	program.hyperlinks.lockscreen.OnTapped = lockScreen

	// and let's hide these for a moment
	program.hyperlinks.lockscreen.Hide()

	// previous location of connection loop
	// go maintain_connection()
}
