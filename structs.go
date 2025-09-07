package main

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
)

// leveraging go's strong types is extremely helpful

type (
	components struct {
		wallet      *walletapi.Wallet_Disk
		rpc_server  *rpcserver.RPCServer
		node        nodes
		caches      caches
		application fyne.App
		preferences fyne.Preferences
		selections  selections
		dialogues   dialogues
		activities  activities
		toggles     toggles
		checks      checks
		lists       lists
		entries     entries
		containers  containers
		hyperlinks  hyperlinks
		labels      labels
		buttons     buttons
		window      fyne.Window
		loggedIn    bool
		name        string
		receiver    string
		size        fyne.Size
	}

	nodes struct {
		list []struct {
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
		open  *dialog.FileDialog
		login *dialog.ConfirmDialog
	}
	checks struct {
		replyback *widget.Check
	}

	toggles struct {
		server  *widget.RadioGroup
		network *widget.RadioGroup
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
		connections,
		rpc_server,
		update_password,
		filesign,
		fileverify,
		self_encrypt_decrypt,
		recipient_encrypt_decrypt,
		token_add,
		balance_rescan,
		integrated,
		contract_installer,
		contract_interactor *widget.Button
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
		secret,
		public,
		seed_placeholder,
		secret_placeholder *widget.Entry
	}
	labels struct {
		height,
		connection,
		rpc_server,
		balance,
		counter,
		notice,
		loggedin,
		address,
		seed,
		secret,
		public *widget.Label
	}
)
