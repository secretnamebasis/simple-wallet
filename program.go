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
		pong:   widget.NewRadioGroup([]string{}, nil),
	},
	checks: checks{
		replyback: widget.NewCheck("replyback?", nil),
	},
	hyperlinks: hyperlinks{
		// header
		home:    widget.NewHyperlink("home", nil),
		tools:   widget.NewHyperlink("tools", nil),
		configs: widget.NewHyperlink("configs", nil),
		logout:  widget.NewHyperlink("logout", nil),
		// supplemental
		lockscreen:                widget.NewHyperlink("🔒", nil),
		unlock:                    widget.NewHyperlink("unlock", nil),
		pong_server:               widget.NewHyperlink("pong server", nil),
		rpc_server:                widget.NewHyperlink("rpc server", nil),
		contract_installer:        widget.NewHyperlink("contract installer", nil),
		contract_interactor:       widget.NewHyperlink("contract interactor", nil),
		create:                    widget.NewHyperlink("create", nil),
		assets:                    widget.NewHyperlink("assets", nil),
		generate:                  widget.NewHyperlink("generate", nil),
		restore:                   widget.NewHyperlink("restore wallet", nil),
		connections:               widget.NewHyperlink("connections", nil),
		open_wallet:               widget.NewHyperlink("open wallet", nil),
		open_file:                 widget.NewHyperlink("open file", nil),
		address:                   widget.NewHyperlink("address", nil),
		send:                      widget.NewHyperlink("send", nil),
		login:                     widget.NewHyperlink("login", nil),
		save:                      widget.NewHyperlink("save", nil),
		keys:                      widget.NewHyperlink("keys", nil),
		transactions:              widget.NewHyperlink("tx history", nil),
		balance_rescan:            widget.NewHyperlink("balance rescan", nil),
		token_add:                 widget.NewHyperlink("token add", nil),
		integrated:                widget.NewHyperlink("integrated address", nil),
		filesign:                  widget.NewHyperlink("filesign", nil),
		fileverify:                widget.NewHyperlink("fileverify", nil),
		self_encrypt_decrypt:      widget.NewHyperlink("self crypt", nil),
		recipient_encrypt_decrypt: widget.NewHyperlink("recipient crypt", nil),
	},
	labels: labels{
		height:     widget.NewLabel("⬡: 0000000"),
		connection: widget.NewLabel("🌐: 🔴"),
		loggedin:   widget.NewLabel("💰: 🔴"),
		rpc_server: widget.NewLabel("📡: 🔴"),
		pong:       widget.NewLabel("🏓: 🔴"),
		notice:     widget.NewLabel(""),
		balance:    widget.NewLabel("balance"),
		counter:    widget.NewLabel("counter"),
		address:    widget.NewLabel("address: "),
		seed:       widget.NewLabel("seed"),
		secret:     widget.NewLabel("secret"),
		public:     widget.NewLabel("public"),
	},
	entries: entries{
		username:           widget.NewEntry(),
		password:           widget.NewEntry(),
		node:               widget.NewEntry(),
		wallet:             widget.NewEntry(),
		file:               widget.NewEntry(),
		pass:               widget.NewEntry(),
		pongs:              widget.NewEntry(),
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
		425,
		425,
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
