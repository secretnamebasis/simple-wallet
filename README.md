# simple-wallet

This is a simple [Dero](https://deroproject.io/) wallet built using Fyne.

**This software is alpha stage software, use only for testing and evaluation purposes.**

# Features
<details>
<summary><strong>Status & Connectivity</strong></summary>

- Topbar indicators:
  - **BLOCK** – Block height
  - **NODE** – Node connection
  - **WALLET** – Wallet logged-in state
  - **WS** – WebSocket server
  - **RPC** – RPC server
  - **IDX** – Indexer connection
- Public node auto-connect
- WS server
- RPC server
- Indexer passthrough over WebSocket

</details>

<details>
<summary><strong>Security & Access</strong></summary>

- 🔒 Lockscreen button
- Create / Login / Restore wallets
- Seed phrase & public/secret key reveal
- New entry notification settings

</details>

<details>
<summary><strong>Wallet & Assets</strong></summary>

- Transaction history
- Asset collections with history
- Token adding
- Integrated address generation
- Sending (including token assets)
- Balance rescanning

</details>

<details>
<summary><strong>Indexing & Scanning</strong></summary>

- Asset scanner via **Gnomon Indexer**
- Simulator launcher

</details>

<details>
<summary><strong>DERO Tools</strong></summary>

**Encryption**
- File signing & verification
- Self-encryption / decryption
- Recipient encryption / decryption

**Blockchain Exploration**
- Difficulty graph

**Smart Contracts**
- Contract installer
- Contract interactor

</details>


# Installation

You will need Go, and a newer version is recommended. Fyne has its own set of dependencies: [https://docs.fyne.io/started/](https://docs.fyne.io/started/)

Be sure to check out the [releases](https://github.com/secretnamebasis/simple-wallet/releases) for linux and windows binaries; maybe one day I will be able to afford a mac...  

If you would like to build from source:
```sh
git clone https://github.com/secretnamebasis/simple-wallet
cd simple-wallet
go build .
./simple-wallet
```

Or you can simply run the program:
```sh
git clone https://github.com/secretnamebasis/simple-wallet
cd simple-wallet
go run .
```

Or, assuming you have a properly configured `go` environment:
```sh
go install github.com/secretnamebasis/simple-wallet@latest
simple-wallet
```

# Custom Methods

<details>
  <summary>With the addition of XSWD, the wallet has custom methods: {</summary>

  `"GetAssets"`, 
  
  `"GetAssetBalance"`,
  
  `"GetTXEstimate"`,
  
  `"AttemptEPOCHWithAddr"`
  
  `"Gnomon.GetAllOwnersAndSCIDs"`
  
  `"Gnomon.GetAllSCIDVariableDetails"`

  </details>
}

Read more in [docs](./docs/methods.md).

# Development

Most of the dev process has been to imbue a GUI with as much of the present wallet client tools as possible, while also introducing some components that make it easier to dev on the DERO blockchain. 

There's a rough-outline of a [roadmap](./docs/roadmap.md) in the docs.

# Image Gallery

You can find an [image gallery](./docs/image_gallery.md) for the application in the docs.

# Contributing
There's really only one rule for contributing to projects I maintain: have fun learning! Anyone is welcome to contribute as much as they'd like, or they can fork the project at any time to create their own version of simple-wallet.

# SPONSORSHIP
Like the proof of work? That's cool. 

Want to privately and anonymously sponsor the project? Rad! Pls send dero:
```dero
deroi1qyc96tgvz8fz623snpfwjgdhlqznamcsuh8rahrh2yvsf2gqqxdljqdzvfp4xatnd9khqmr994mkzmrvv46zqum4wpcx7un5vfz92qg8up0zt
```
# License
You are welcome to do whatever you want with this code, as long as you first respect the RESEARCH LICENSE of Derohe (restrictive) and then observe the BSD 3-Clause license of Fyne (permissive). Please see LICENSE for more details. But the most important thing to remember:

TECHNOLOGY IS PROVIDED "AS IS", WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED INCLUDING, WITHOUT LIMITATION, WARRANTIES THAT ANY SUCH TECHNOLOGY IS FREE OF DEFECTS, MERCHANTABLE, FIT FOR A PARTICULAR PURPOSE, OR NON-INFRINGING OF THIRD PARTY RIGHTS. YOU AGREE THAT YOU BEAR THE ENTIRE RISK IN CONNECTION WITH YOUR USE AND DISTRIBUTION OF ANY AND ALL TECHNOLOGY UNDER THIS LICENSE.
