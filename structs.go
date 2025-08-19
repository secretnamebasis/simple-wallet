package main

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/walletapi"
	"github.com/deroproject/derohe/walletapi/rpcserver"
	"go.etcd.io/bbolt"
)

// leveraging go's strong types is extremely helpful

type (
	components struct {
		db          *bbolt.DB
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

	caches struct {
		hashes []crypto.Hash
	}

	dialogues struct {
		open  *dialog.FileDialog
		login *dialog.ConfirmDialog
	}
	checks struct {
		replyback *widget.Check
	}

	toggles struct {
		server *widget.RadioGroup
		pong   *widget.RadioGroup
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
		pong_server,
		rpc_server,
		contract_installer,
		contract_interactor,
		generate,
		assets,
		token_add,
		restore,
		connections,
		create,
		open_wallet,
		address,
		send,
		save,
		keys,
		transactions,
		balance_rescan,
		integrated,
		filesign,
		fileverify,
		self_encrypt_decrypt,
		recipient_encrypt_decrypt *widget.Hyperlink
	}
	buttons struct {
		register *widget.Button
		open_file *widget.Button
	}
	selections struct {
		assets *widget.Select
	}
	lists struct {
		transactions *widget.List
		asset_list   *widget.List
	}
	activities struct {
		registration *widget.Activity
	}
	entries struct {
		node,
		username, password,
		wallet,
		file,
		pass,
		seed,
		pongs,
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
		pong,
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

type listing struct {
	Name        string      // the name of the thing
	Description string      // the stuff about the thing
	Address     string      // the thing being watched
	DST         uint64      // the destination
	Replyback   string      // the comment being sent back
	Token       crypto.Hash // the token used
	Sendback    uint64      // the count of token used
	Supply      int         // the count of times we can do this
}
type pong struct {
	Time    time.Time
	Txid    string
	Status  string
	Address string
}
