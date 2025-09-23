package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

// simple way to compress byte size
const kilobyte = float64(1024)

// a simple way to convert units
const atomic_units = 100000

// simple way to set file permissions
const default_file_permissions = 0644

// simple way to set dismiss
const dismiss = `dismiss`

// simple way to set confirm
const confirm = `confirm`

// simple way to set timeouts
const timeout = time.Second * 9 // the world is a really big place

// simple way to identify gnomon
const gnomonSC = `a05395bb0cf77adc850928b0db00eb5ca7a9ccbafd9a38d021c8d299ad5ce1a4`

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
		server:    widget.NewRadioGroup([]string{}, nil),
		network:   widget.NewRadioGroup([]string{}, nil),
		simulator: widget.NewRadioGroup([]string{}, nil),
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
		transactions:              widget.NewButton("TXS", nil),
		rpc_server:                widget.NewButton("RPC SERVER", nil),
		update_password:           widget.NewButton("UPDATE PASSWORD", nil),
		contract_installer:        widget.NewButton("SC INSTALLER", nil),
		contract_interactor:       widget.NewButton("SC INTERACTOR", nil),
		simulator:                 widget.NewButton("SIMULATOR", nil),
		connections:               widget.NewButton("CONNECTIONS", nil),
		balance_rescan:            widget.NewButton("BALANCE RESCAN", nil),
		asset_scan:                widget.NewButton("ASSET SCAN", nil),
		token_add:                 widget.NewButton("TOKEN ADD", nil),
		explorer:                  widget.NewButton("EXPLORER", nil),
		integrated:                widget.NewButton("I-ADDRESSES", nil),
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
		generate:   widget.NewHyperlink("generate address", nil),
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
		balance:    widget.NewLabel("0.00000"),
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
		wallet:             widget.NewEntry(),
		file:               widget.NewEntry(),
		username:           widget.NewEntry(),
		password:           widget.NewEntry(),
		node:               widget.NewEntry(),
		pass:               widget.NewEntry(),
		seed:               widget.NewEntry(),
		secret:             widget.NewEntry(),
		public:             widget.NewEntry(),
		counterparty:       widget.NewEntry(),
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
var password_size = fyne.NewSize(program.size.Width/3, program.size.Height/4)

// it would be ideal to have... like 20, or a callable list
var node_list = []struct {
	ip   string
	name string
}{
	{
		ip:   "",
		name: "preferred",
	},
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
var pool_headers = []string{
	"height built",
	"tx hash",
	"fee",
	"ring size",
	"tx size [kB]",
}
var block_headers = []string{
	"height",
	"topo height",
	"age",
	"miniblocks", // why would we need to know if there was less than 10?
	"size [kiB]",
	"tx hash",
	"type",
	"fees",
	"ring size",
	"tx size [kB]",
}
var search_headers_block = []string{
	"TOPO HEIGHT",
	"BUILD HEIGHT",
	"BLID",
	"PREVIOUS",
	"UNIX TIME",
	"UTC TIME",
	"AGE",
	"MAJOR.MINIOR VERSION",
	"REWARD",
	"SIZE kB",
	"MINIBLOCKS",
	"CONFIRMATIONS",
}
var search_headers_registration = []string{
	"TXID",
	"TYPE",
	"BLOCK",
	"ADDRESS",
	"VALID",
}
var search_headers_normal = []string{
	"TXID",
	"TYPE",
	"BLOCK",
	"BLID",
	"BUILD HEIGHT",
	"ROOT HASH",
	"UNIX TIME",
	"UTC TIME",
	"AGE",
	"TOPO HEIGHT",
	"FEES",
	"SIZE kB",
	"VERSION",
	"CONFIRMATIONS",
	"TYPE",
	"OUTPUTS",
	"RING SIZE",
}
var search_headers_sc_prefix = []string{
	"TXID",
	"TYPE",
	"BLOCK",
	"SCID RESERVES", // this is a k/v pair
}
var search_headers_sc_body = []string{
	"BLID",
	"ROOT HASH",
	"BUILD HEIGHT",
	"UNIX TIME",
	"UTC TIME",
	"AGE",
	"TOPO HEIGHT",
	"FEES",
	"SIZE kB",
	"VERSION",
	"CONFIRMATIONS",
	"SIGNATURE TYPE",
	"RING SIZE",
	"SENDER",
	"RING MEMBERS",
}
