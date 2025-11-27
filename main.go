package main

import (
	"crypto/rand"
	"encoding/gob"
	"fmt"
	m_rand "math/rand"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/chzyer/readline"
	"github.com/deroproject/derohe/globals"
	"github.com/deroproject/derohe/walletapi"
	"github.com/go-logr/logr"
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

	// let's use a simple theme that changes only one thing
	program.application.Settings().SetTheme(customTheme{})

	// we''l assume some simple preferences
	program.preferences = program.application.Preferences()

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
	program.window.SetIcon(theme.AccountIcon())

	// let's set a simple intercept close for the window
	program.window.SetCloseIntercept(func() {
		logger.Info("closing")
		if program.loggedIn {
			program.wallet.Close_Encrypted_Wallet()
		}
		if program.wallet != nil {
			program.wallet = nil
		}
		os.Exit(0)
	})

	// the app is live, initialize!
	initialize()

	// let's set a simple home container
	setContentAsHome()

	// let's simply show and run the program
	program.window.ShowAndRun()
}

func initialize() {
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

	// here is a simple way to get started
	program.preferences.SetBool("mainnet", true)
	program.sliders.network.OnChanged = slide_network
	slide_network(0.1337) // mainnet

	// and simple place for logging out
	program.hyperlinks.logout.OnTapped = logout
	// let's hide this for a minute
	program.hyperlinks.logout.Hide()

	// here is a simple lockscreen
	program.hyperlinks.lockscreen.OnTapped = lockScreen

	// and let's hide these for a moment
	program.hyperlinks.lockscreen.Hide()

	// captain's orders
	go initialize_table()

	initialize_logger()

	// simple way to create a preferred ip endpoint file
	createPreferred()

	// test localhost first, then connect from a list of public nodes
	go maintain_connection()
}
func initialize_logger() {
	// We need to initialize readline first, so it changes stderr to ansi processor on windows
	l, err := readline.NewEx(&readline.Config{
		Prompt: "\033[92mDERO:\033[32m»\033[0m",
		// Prompt:          prompt,
		HistoryFile: "", // wallet never saves any history file anywhere, to prevent any leakage
		// AutoComplete:    completer,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold: true,
		// FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()

	// parse arguments and setup logging , print basic information
	f, err := os.Create(filepath.Join(globals.GetDataDirectory(), program.name+".log"))
	if err != nil {
		fmt.Printf("Error while opening log file err: %s filename %s\n", err, program.name+".log")
		return
	}
	globals.InitializeLog(l.Stdout(), f)
	logger = globals.Logger.WithName(program.name)

}

var logger logr.Logger

var only_once sync.Once

// let's make sure this is only loaded once
func initialize_table() {
	only_once.Do(func() {
		big_table := os.Getenv("USE_BIG_TABLE")
		handleTable := func(s string, i int) {
			logger.Info("Please wait, generating precompute table.... ")
			if loadTable(s) {
				logger.Info("Loaded lookup table from disk.")
			} else {
				msg := "(1<<21)"
				if big_table != "" {
					msg = "(1<<24)"
				}
				logger.Info("Generating lookup table " + msg + "... this may take a while.")
				walletapi.Initialize_LookupTable(1, i)
				saveTable(s, walletapi.Balance_lookup_table)
			}

			tables := len(*walletapi.Balance_lookup_table)
			if walletapi.Balance_lookup_table != nil && tables > 0 {
				tableLen := float64(len((*walletapi.Balance_lookup_table)[0]))
				logger.Info(fmt.Sprintf("Lookup table info: %d table, with %.f entries (~%.f MiB)",
					tables, tableLen, tableLen*8/kilobyte/kilobyte))
			}
		}
		if big_table != "" {
			handleTable("big_table.gob", 1<<24)
		} else {
			handleTable("small_table.gob", 1<<21)
		}
		logger.Info("Precompute table loaded into memory")
	})
}

func loadTable(filename string) bool {
	f, err := os.Open(filepath.Join(globals.GetDataDirectory(), filename))
	if err != nil {
		return false // file not found
	}
	defer f.Close()
	if err := gob.NewDecoder(f).Decode(&walletapi.Balance_lookup_table); err != nil {
		fmt.Println("Failed to decode lookup table:", err)
		return false
	}
	return true
}

func saveTable(filename string, table any) {
	f, err := os.Create(filepath.Join(globals.GetDataDirectory(), filename))
	if err != nil {
		fmt.Println("Failed to create table file:", err)
		return
	}
	defer f.Close()
	if err := gob.NewEncoder(f).Encode(table); err != nil {
		fmt.Println("Failed to encode lookup table:", err)
	}
}
