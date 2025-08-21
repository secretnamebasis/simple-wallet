package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

// a simple way to convert units
const atomic_units = 100000

// simple way to set file permissions
const default_file_permissions = 0644

// simple way to set dismiss
const dismiss = `dismiss`

// simple way to set confirm
const confirm = `confirm`

// not to be confused with an app, this is a program:
var program = components{
	activities: activities{
		registration: widget.NewActivity(),
	},
	lists: lists{
		transactions: new(widget.List),
		asset_list:   new(widget.List),
	},
	toggles: toggles{
		server: widget.NewRadioGroup([]string{}, nil),
	},
	checks: checks{
		replyback: widget.NewCheck("replyback?", nil),
	},

	buttons: buttons{
		open_file:                 widget.NewButton("", nil),
		open_wallet:               widget.NewButton("open wallet", nil),
		send:                      widget.NewButton("SEND", nil),
		assets:                    widget.NewButton("ASSETS", nil),
		keys:                      widget.NewButton("KEYS", nil),
		transactions:              widget.NewButton("TRANSACTIONS", nil),
		rpc_server:                widget.NewButton("RPC SERVER", nil),
		contract_installer:        widget.NewButton("CONTRACT INSTALLER", nil),
		contract_interactor:       widget.NewButton("CONTRACT INTERACTOR", nil),
		connections:               widget.NewButton("CONNECTIONS", nil),
		balance_rescan:            widget.NewButton("BALANCE RESCAN", nil),
		token_add:                 widget.NewButton("TOKEN ADD", nil),
		integrated:                widget.NewButton("INTEGRATED ADDRESSES", nil),
		filesign:                  widget.NewButton("FILESIGN", nil),
		fileverify:                widget.NewButton("FILEVERIFY", nil),
		self_encrypt_decrypt:      widget.NewButton("SELF CRYPT", nil),
		recipient_encrypt_decrypt: widget.NewButton("RECIPIENT CRYPT", nil),
	},

	hyperlinks: hyperlinks{
		// header
		home:    widget.NewHyperlink("home", nil),
		tools:   widget.NewHyperlink("tools", nil),
		configs: widget.NewHyperlink("configs", nil),
		logout:  widget.NewHyperlink("logout", nil),
		// supplemental
		lockscreen: widget.NewHyperlink(" 🔒", nil),
		unlock:     widget.NewHyperlink("unlock", nil),
		create:     widget.NewHyperlink("create", nil),
		generate:   widget.NewHyperlink("generate", nil),
		restore:    widget.NewHyperlink("restore wallet", nil),
		address:    widget.NewHyperlink("address", nil),
		login:      widget.NewHyperlink("login", nil),
		save:       widget.NewHyperlink("save", nil),
	},
	labels: labels{
		height:     widget.NewLabel("BLOCK: 0000000"),
		connection: widget.NewLabel("NODE: 🔴"),
		loggedin:   widget.NewLabel("WALLET: 🔴"),
		rpc_server: widget.NewLabel("RPC: 🔴"),
		notice:     widget.NewLabel(""),
		balance:    widget.NewLabel("BALANCE"),
		counter: makeCenteredWrappedLabel(`
Registration POW takes time 20min-2hrs...
If on battery, plug your computer in.
Please do not leave this page.
			`),
		address: widget.NewLabel("ADDRESS: "),
		seed:    widget.NewLabel("SEED PHRASE"),
		secret:  widget.NewLabel("SECRET KEY"),
		public:  widget.NewLabel("PUBLIC KEY"),
	},
	entries: entries{
		username:           widget.NewEntry(),
		password:           widget.NewEntry(),
		node:               widget.NewEntry(),
		wallet:             widget.NewEntry(),
		file:               widget.NewEntry(),
		pass:               widget.NewEntry(),
		seed:               widget.NewEntry(),
		secret:             widget.NewEntry(),
		public:             widget.NewEntry(),
		recipient:          widget.NewEntry(),
		amount:             widget.NewEntry(),
		dst:                widget.NewEntry(),
		comment:            widget.NewEntry(),
		seed_placeholder:   widget.NewEntry(),
		secret_placeholder: widget.NewEntry(),
	},
	selections: selections{
		assets: widget.NewSelect([]string{""}, func(s string) {}),
	},
	rpc_server: new(rpcserver.RPCServer),
	wallet:     new(walletapi.Wallet_Disk),
	node: nodes{
		list: node_list,
	},

	name: "simple wallet",
	size: fyne.NewSize(
		900,
		600,
	),
}

// it would be ideal to have... like 20, or a callable list
var node_list = []struct {
	ip   string
	name string
}{
	{
		ip:   "127.0.0.1:10102",
		name: "localhost",
	},
	{
		ip:   "173.208.130.94:11012",
		name: "node.derofoundation.org",
	},
	{
		ip:   "64.226.81.37:10102",
		name: "dero-node.net",
	},
	{
		ip:   "51.81.96.25:10102",
		name: "dero.geeko.cloud",
	},
}
