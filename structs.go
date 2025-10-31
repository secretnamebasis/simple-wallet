package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/blockchain"
	derodrpc "github.com/deroproject/derohe/cmd/derod/rpc"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
	"github.com/deroproject/derohe/walletapi/xswd"
)

// leveraging go's strong types is extremely helpful

type (
	components struct {
		wallet           *walletapi.Wallet_Disk
		ws_server        *xswd.XSWD
		rpc_server       *rpcserver.RPCServer
		simulator_server *derodrpc.RPCServer

		node        nodes
		caches      caches
		application fyne.App
		preferences fyne.Preferences
		selections  selections
		tables      tables
		dialogues   dialogues
		activities  activities
		toggles     toggles
		sliders     sliders
		checks      checks
		lists       lists
		entries     entries
		containers  containers
		hyperlinks  hyperlinks
		labels      labels
		buttons     buttons
		window      fyne.Window
		encryption  fyne.Window
		contracts   fyne.Window
		explorer    fyne.Window
		loggedIn    bool
		name        string
		receiver    string
		size        fyne.Size
	}

	nodes struct {
		pool                 rpc.GetTxPool_Result
		info                 rpc.GetInfo_Result
		simulator_chain      *blockchain.Blockchain
		simulator_wallets    []*walletapi.Wallet_Disk
		simulator_rpcservers []*rpcserver.RPCServer
		transactions         map[string]rpc.GetTransaction_Result
		blocks               map[uint64]rpc.GetBlock_Result
		list                 []struct {
			ip   string
			name string
		}
		current string
	}
	asset struct {
		name  string
		hash  string
		image image.Image
	}
	caches struct {
		assets []asset
	}

	dialogues struct {
		login *dialog.ConfirmDialog
	}
	checks struct {
		replyback *widget.Check
	}

	sliders struct {
		notifications,
		network *widget.Slider
	}
	toggles struct {
		rpc_server *widget.RadioGroup
		ws_server  *widget.RadioGroup
		simulator  *widget.RadioGroup
	}
	containers struct {
		topbar,
		home,
		tools,
		configs,
		bottombar,
		// supplemental
		dashboard,
		send,
		register,
		toolbox *fyne.Container
	}
	hyperlinks struct {
		home,
		tools,
		configs,
		login, logout,
		// supplemental
		lockscreen,
		unlock,
		generate,
		restore,
		create,
		address,
		save *widget.Hyperlink
	}
	buttons struct {
		register,
		open_wallet,
		open_file,
		transactions,
		assets,
		keys,
		send,
		simulator,
		simulation,
		connections,
		ws_server,
		rpc_server,
		update_password,
		contracts,
		encryption,
		token_add,
		balance_rescan,
		notifications,
		asset_scan,
		explorer *widget.Button
	}
	tables struct {
		connections *widget.Table
	}
	selections struct {
		assets *widget.Select
	}
	lists struct {
		transactions,
		asset_list *widget.List
	}
	activities struct {
		registration *widget.Activity
	}
	entries struct {
		wallet,
		file,
		node,
		username, password,
		pass,
		seed,
		counterparty,
		recipient,
		amount,
		dst,
		comment,
		port,
		secret,
		public *widget.Entry
	}
	labels struct {
		height,
		connection,
		current_node,
		rpc_server,
		ws_server,
		balance,
		counter,
		notice,
		mainnet,
		testnet,
		simulator,
		loggedin,
		address,
		seed,
		secret,
		public *widget.Label
	}
)

type wallet_entries []rpc.Entry
type list []string
