package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/deroproject/derohe/block"
	"github.com/deroproject/derohe/cryptography/crypto"
	"github.com/deroproject/derohe/rpc"
	"github.com/deroproject/derohe/transaction"
	"github.com/deroproject/derohe/walletapi"
)

// this is going to be a rudimentary explorer at first
func explorer() {

	logger.Info("explorer", "status", "initiated")

	program.explorer = fyne.CurrentApp().NewWindow(program.name + " | viewer ")
	program.explorer.Resize(program.size)
	program.explorer.SetIcon(theme.SearchIcon())
	notice := makeCenteredWrappedLabel("LOADING EXPLORER...\npls hodl")
	program.explorer.SetContent(container.NewAdaptiveGrid(1,
		notice,
	))
	program.explorer.Show()
	// let's start with the stats tab
	stats := []string{
		strconv.Itoa(int(program.node.info.Height)),
		strconv.Itoa(int(program.node.info.AverageBlockTime50)),
		strconv.Itoa(int(program.node.info.Tx_pool_size)),
		strconv.Itoa(int(program.node.info.Difficulty) / 1000000),
		strconv.Itoa(int(program.node.info.Total_Supply)),
		program.node.info.Status,
	}
	diff := widget.NewLabel("Network Height: " + stats[0])
	average_blocktime := widget.NewLabel("Network Blocktime: " + stats[1] + " seconds")
	mem_pool := widget.NewLabel("Mempool Size: " + stats[2])
	hash_rate := widget.NewLabel("Hash Rate: " + stats[3] + " MH/s")
	supply := widget.NewLabel("Total Supply: " + stats[4])
	network_status := widget.NewLabel("Network Status: " + stats[5])

	// we are going to need this for our graph
	diff_map := map[int]int{}
	updateDiffData := func() {
		// don't do more than this...
		const limit = 300

		// concurrency!
		var wg sync.WaitGroup
		var mu sync.RWMutex

		wg.Add(limit)
		for i := range limit {
			func(i int) {
				defer wg.Done()

				h := uint64(walletapi.Get_Daemon_TopoHeight()) - (uint64(i))

				_, exists := diff_map[int(h)]

				_, havBlock := program.node.blocks[h]

				if exists {
					mu.Lock()
					defer mu.Unlock()
					x := int(walletapi.Get_Daemon_TopoHeight()) - limit
					if len(diff_map) >= limit {
						delete(diff_map, x)
					}
					if havBlock {
						if len(program.node.blocks) >= limit {
							delete(program.node.blocks, uint64(x))
							logger.Info("explorer", "block removed from map", strconv.Itoa(int(x)))
						}
					}
					return
				}

				r := getBlockInfo(rpc.GetBlock_Params{
					Height: h,
				})

				b, err := hex.DecodeString(r.Blob)
				if err != nil {
					return
				}

				var bl block.Block
				err = bl.Deserialize(b)
				if err != nil {
					return
				}

				d, e := strconv.Atoi(r.Block_Header.Difficulty)
				if e != nil {
					return
				}
				mu.Lock()
				if !exists {
					diff_map[int(bl.Height)] = d
				}
				if !havBlock {
					program.node.blocks[h] = r
					logger.Info("explorer", "block added to map", strconv.Itoa(int(h)))
				}
				mu.Unlock()
			}(i)
		}
		wg.Wait()
	}

	updateDiffData()
	if len(diff_map) <= 0 {
		showError(errors.New("failed to collect data, please check connection and try again"), program.window)
		return
	}
	g := &graph{hd_map: diff_map}
	g.ExtendBaseWidget(g)
	diff_graph := g
	contains_stats := container.NewBorder(
		container.NewVBox(container.NewAdaptiveGrid(3,
			diff,
			average_blocktime,
			mem_pool,
			hash_rate,
			supply,
			network_status,
		)),
		nil,
		container.NewStack(diff_graph),
		nil, nil,
	)
	tab_stats := container.NewTabItem("STATS", contains_stats)

	// speaking of tabs...
	var tabs *container.AppTabs
	var tab_search *container.TabItem

	var searchData, searchHeaders []string
	var results_table *widget.Table

	searchBlockchain := func(s string) {
		results_table.ScrollToTop()
		results_table.ScrollToLeading()
		searchHeaders = []string{"NO BLOCK DATA"}
		searchData = []string{"NO DATA"}

		buildBlockResults := func(r rpc.GetBlock_Result) {
			var bl block.Block
			b, _ := hex.DecodeString(r.Blob)
			bl.Deserialize(b)

			searchHeaders = search_headers_block

			var previous_block string
			if len(r.Block_Header.Tips) > 0 {
				previous_block = r.Block_Header.Tips[0]
			}

			var size = uint64(len(bl.Serialize()))
			var hashes []string
			var types []string
			if r.Block_Header.TXCount != 0 {
				for _, each := range bl.Tx_hashes {
					r := getTransaction(rpc.GetTransaction_Params{
						Tx_Hashes: []string{each.String()},
					})
					for _, each := range r.Txs_as_hex {
						b, e := hex.DecodeString(each)
						if e != nil {
							continue
						}
						var tx transaction.Transaction
						if err := tx.Deserialize(b); err != nil {
							continue
						}
						size += uint64(len(tx.Serialize()))
						types = append(types, tx.TransactionType.String())
						hashes = append(hashes, tx.GetHash().String())
					}
				}
			}

			searchData = []string{
				strconv.Itoa(int(r.Block_Header.TopoHeight)),
				strconv.Itoa(int(r.Block_Header.Height)),
				r.Block_Header.Hash,
				previous_block,
				strconv.Itoa(int(r.Block_Header.Timestamp)),
				time.Unix(0, int64(r.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
				time.Duration((uint64(time.Now().UTC().UnixMilli()) - r.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
				strconv.Itoa(int(r.Block_Header.Major_Version)) + "." + strconv.Itoa(int(r.Block_Header.Minor_Version)),
				fmt.Sprintf("%0.5f", float64(r.Block_Header.Reward)/atomic_units),
				fmt.Sprintf("%.03f", float32(size)/float32(kilobyte)),
				strconv.Itoa(len(bl.MiniBlocks)),
				strconv.Itoa(int(r.Block_Header.Depth)),
			}

			searchHeaders = append(searchHeaders, "TX COUNT")
			searchData = append(searchData, strconv.Itoa(int(r.Block_Header.TXCount)))
			searchHeaders = append(searchHeaders, types...)
			searchData = append(searchData, hashes...)

			searchHeaders = append(searchHeaders, []string{"MINING OUTPUTS"}...)

			miners := []string{}
			miners = append(miners, r.Block_Header.Miners...)
			var rewards uint64
			if len(miners) != 0 {
				for range len(miners) {
					searchHeaders = append(searchHeaders, " ")
				}
			} else { // here you go, capt...

				var acckey crypto.Point
				if err := acckey.DecodeCompressed(bl.Miner_TX.MinerAddress[:]); err != nil {
					panic(err)
				}

				address := rpc.NewAddressFromKeys(&acckey)
				address.Mainnet = program.preferences.Bool("mainnet")
				miners = append(miners, address.String())
				rewards += bl.Miner_TX.Value
			}

			searchData = append(searchData, []string{
				fmt.Sprintf("%0.5f", float64(rewards+r.Block_Header.Reward)/atomic_units),
				"MINER ADDRESS",
			}...)
			searchData = append(searchData, miners...)

		}
		//pre-processing
		switch len(s) {
		case 64: // this is basically any hash
			var result rpc.GetBlock_Result

			for _, each := range program.node.blocks {
				if s == each.Block_Header.Hash {
					result = each
					break
				}
			}

			if result.Block_Header.Hash != s {
				result = getBlockInfo(rpc.GetBlock_Params{Hash: s})
				// -32098: file does not exist
			}

			// at this point we have to determine if...
			if len(result.Block_Header.Miners) != 0 {
				buildBlockResults(result)
			} else {
				var r rpc.GetTransaction_Result
				if _, ok := program.node.transactions[s]; ok {
					r = program.node.transactions[s]
				} else {
					// not a mined transaction
					r = getTransaction(
						rpc.GetTransaction_Params{
							Tx_Hashes: []string{s},
						},
					)
				}

				// fmt.Printf("tx: %+v\n", r)
				if len(r.Txs_as_hex) < 1 {
					// goto end
				}
				b, err := hex.DecodeString(r.Txs_as_hex[0])
				if err != nil {
					// goto end
				}
				var tx transaction.Transaction
				if err := tx.Deserialize(b); err != nil {
					// goto end
				}

				// we should encapsulate this logic
				block_info := getBlockInfo(rpc.GetBlock_Params{
					Hash: r.Txs[0].ValidBlock,
				})

				bl := getBlockDeserialized(block_info.Blob)

				switch tx.TransactionType {
				case transaction.PREMINE:
				case transaction.REGISTRATION:

					searchHeaders = search_headers_registration

					var acckey crypto.Point
					if err := acckey.DecodeCompressed(tx.MinerAddress[:]); err != nil {
						panic(err)
					}

					address := rpc.NewAddressFromKeys(&acckey)
					address.Mainnet = program.preferences.Bool("mainnet")

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(),
						bl.GetHash().String(),
						address.String(),
						"TRUE",
					}

					results_table.Refresh()
					// searchHeaders = append(searchHeaders, )
				case transaction.COINBASE: // these aren't shown in the explorer; and sc interactions are...
				case transaction.NORMAL:

					searchHeaders = search_headers_normal

					var ring_members []string
					var outputs []string
					for i, each := range r.Txs[0].Ring {
						searchHeaders = append(searchHeaders, "OUTPUT "+strconv.Itoa(i+1))
						outputs = append(outputs, "RING MEMBERS")
						for i, member := range each {
							searchHeaders = append(searchHeaders, "Ring Member "+strconv.Itoa(i+1))
							ring_members = append(ring_members, member)
						}
					}

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(),
						r.Txs[0].ValidBlock,
						fmt.Sprintf("%x", tx.BLID),
						strconv.Itoa(int(tx.Height)),
						fmt.Sprintf("%x", tx.Payloads[0].Statement.Roothash[:]),
						strconv.Itoa(int(block_info.Block_Header.Timestamp)),
						time.Unix(0, int64(block_info.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
						time.Duration((uint64(time.Now().Local().UnixMilli()) - block_info.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
						strconv.Itoa(int(block_info.Block_Header.TopoHeight)),
						fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
						fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/float32(kilobyte)),
						strconv.Itoa(int(tx.Version)),
						strconv.Itoa(int(block_info.Block_Header.Depth)),
						"DERO_HOMOMORPHIC",
						strconv.Itoa(int(len(r.Txs[0].Ring))),
						strconv.Itoa(int(float64(len(ring_members)) / float64(len(r.Txs[0].Ring)))),
					}
					for i, each := range outputs {
						searchData = append(searchData, each)
						searchData = append(searchData, r.Txs[0].Ring[i]...)
					}

					results_table.Refresh()
				case transaction.BURN_TX:
					// I haven't seen any of these yet...
				case transaction.SC_TX:

					searchHeaders = search_headers_sc_prefix

					// headers := []string{}
					sc := getSC(rpc.GetSC_Params{
						SCID:      s,
						Code:      true,
						Variables: true,
					})

					searchData = []string{
						tx.GetHash().String(),
						tx.TransactionType.String(), //
						block_info.Block_Header.Hash,
						"ATOMIC AMOUNTS",
					}

					for k, v := range sc.Balances {
						searchHeaders = append(searchHeaders, k)
						searchData = append(searchData, strconv.Itoa(int(v)))
					}
					searchHeaders = append(searchHeaders, "STRING VARS")
					searchData = append(searchData, "STRING VALUES")
					type string_pair struct {
						k string
						v string
					}
					var string_pairs []string_pair
					for k, v := range sc.VariableStringKeys {
						var value string
						switch val := v.(type) {
						case string:
							if k != "C" {
								b, e := hex.DecodeString(val)
								if e != nil {
									continue
								}
								value = string(b)
							} else {
								value = truncator(val)
							}
						case uint64:
							value = strconv.Itoa(int(val))
						case float64:
							value = strconv.FormatFloat(val, 'f', 0, 64)
						}
						string_pairs = append(string_pairs, string_pair{
							k: k,
							v: value,
						})
					}
					sort.Slice(string_pairs, func(i, j int) bool {
						return string_pairs[i].k > string_pairs[j].k
					})
					for _, each := range string_pairs {
						searchHeaders = append(searchHeaders, each.k)
						searchData = append(searchData, each.v)
					}
					searchHeaders = append(searchHeaders, "UINT64 VARS")
					searchData = append(searchData, "UINT64 VALUES")
					type uint64_pair struct {
						k string
						v string
					}
					var uint64_pairs []uint64_pair
					for k, v := range sc.VariableUint64Keys {
						var value string
						switch val := v.(type) {
						case string:
							b, e := hex.DecodeString(val)
							if e != nil {
								continue
							}
							value = string(b)
						case uint64:
							value = strconv.Itoa(int(val))
						case float64:
							value = strconv.FormatFloat(val, 'f', 0, 64)
						}
						uint64_pairs = append(uint64_pairs, uint64_pair{
							k: strconv.Itoa(int(k)),
							v: value,
						})
					}
					sort.Slice(uint64_pairs, func(i, j int) bool {
						return uint64_pairs[i].k > uint64_pairs[j].k
					})
					for _, each := range uint64_pairs {
						searchHeaders = append(searchHeaders, each.k)
						searchData = append(searchData, each.v)
					}

					searchHeaders = append(searchHeaders, search_headers_sc_body...)

					var ring_members []string

					for _, each := range r.Txs[0].Ring {
						ring_members = append(ring_members, each...)
					}

					searchData = append(searchData, []string{
						fmt.Sprintf("%x", tx.BLID),
						fmt.Sprintf("%x", tx.Payloads[0].Statement.Roothash[:]),
						strconv.Itoa(int(block_info.Block_Header.Height)),
						strconv.Itoa(int(block_info.Block_Header.Timestamp)),
						time.Unix(0, int64(block_info.Block_Header.Timestamp*uint64(time.Millisecond))).Format("2006-01-02 15:04:05"),
						time.Duration((uint64(time.Now().Local().UnixMilli()) - block_info.Block_Header.Timestamp) * uint64(time.Millisecond)).String(),
						strconv.Itoa(int(block_info.Block_Header.TopoHeight)),
						fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
						fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/float32(kilobyte)), // we need to break this down as before
						strconv.Itoa(int(tx.Version)),
						strconv.Itoa(int(block_info.Block_Header.Depth)),
						"DERO_HOMOMORPHIC",
						strconv.Itoa(len(ring_members)),
						r.Txs[0].Signer,
						"RING MEMBERS",
					}...)
					for range ring_members {
						searchHeaders = append(searchHeaders, "")
					}
					searchData = append(searchData, ring_members...)
					searchHeaders = append(searchHeaders, "SC BALANCE") // in DERO
					searchData = append(searchData, rpc.FormatMoney(sc.Balance))

					searchHeaders = append(searchHeaders, []string{
						"SC CODE",
						"SC ARGS",
					}...)
					searchData = append(searchData, []string{
						sc.Code,
						fmt.Sprintf("%+v", tx.SCDATA),
					}...)
					results_table.Refresh()

				default:
				}
			}
		case 66: // dero1 addresses?
		default: // this is going to be a height, or a wallet address or... I mean, what do we search for on the blockchain?
			// I mean, now we are having to search things like tela...

			i, err := strconv.Atoi(s)
			if err != nil {
				return
			}
			r := getBlockInfo(
				rpc.GetBlock_Params{
					Height: uint64(i),
				},
			)
			if r.Blob != "" {
				buildBlockResults(r)
			}
		}
		scaling := float32(1.2)
		results_table.SetColumnWidth(0, (largestMinSize(searchHeaders).Width * scaling))
		results_table.SetColumnWidth(1, largestMinSize(searchData).Width)
		results_table.Refresh()
	}

	pool_label_data := [][]string{}

	updatePoolCache := func() {

		pool_label_data = [][]string{}
		pool := program.node.pool
		if len(pool.Tx_list) <= 0 {
			return
		}
		for txid := range program.node.transactions {
			if !slices.Contains(pool.Tx_list, txid) {
				delete(program.node.transactions, txid)
				logger.Info("explorer", "status", "tx removed from pool")
			}
		}
		for i := range pool.Tx_list {
			if _, ok := program.node.transactions[pool.Tx_list[i]]; !ok {
				logger.Info("explorer", "status", "tx added to pool")
				program.node.transactions[pool.Tx_list[i]] = getTransaction(rpc.GetTransaction_Params{
					Tx_Hashes: []string{pool.Tx_list[i]},
				})
			}

			var tx transaction.Transaction
			decoded, _ := hex.DecodeString(program.node.transactions[pool.Tx_list[i]].Txs_as_hex[0])

			if err := tx.Deserialize(decoded); err != nil {
				continue
			}
			var size int
			for _, each := range program.node.transactions[pool.Tx_list[i]].Txs {
				size += len(each.Ring)
			}

			// Build data row
			pool_label_data = append(pool_label_data, []string{
				strconv.Itoa(int(tx.Height)),
				tx.GetHash().String(),
				fmt.Sprintf("%0.5f", float64(tx.Fees())/atomic_units),
				strconv.Itoa(size),
				fmt.Sprintf("%.03f", float32(len(tx.Serialize()))/1024),
			})
		}
	}
	updatePoolCache()
	var pool_table *widget.Table

	lengthPool := func() (rows int, cols int) {
		return len(program.node.pool.Tx_list), len(pool_headers)
	}

	createPool := func() fyne.CanvasObject {
		return container.NewStack(
			widget.NewLabel(""), // For regular text
			container.NewScroll(widget.NewHyperlink("", nil)), // For clickable hash
		)
	}

	updatePool := func(tci widget.TableCellID, co fyne.CanvasObject) {
		pool_data := pool_label_data
		cell := co.(*fyne.Container)
		label := cell.Objects[0].(*widget.Label)
		scroll := cell.Objects[1].(*container.Scroll)
		link := scroll.Content.(*widget.Hyperlink)
		if len(pool_data) == 0 {
			label.SetText("")
			label.Hide()
			link.SetText("")
			link.Hide()
			return
		}
		switch tci.Col {
		case 0, 2, 3, 4:
			label.Show()
			scroll.Hide()
			if tci.Row >= len(pool_data) {
				label.SetText("")
			} else {
				label.SetText(pool_data[tci.Row][tci.Col])
			}

		case 1:
			label.Hide()
			scroll.Show()
			if tci.Row >= len(pool_data) {
				link.SetText("")
			} else {
				link.SetText(pool_data[tci.Row][tci.Col])
				link.OnTapped = func() {
					hash := pool_data[tci.Row][tci.Col]
					searchBlockchain(hash)
					results_table.Refresh()
					tabs.Select(tab_search)
				}
			}

		}

	}

	pool_table = widget.NewTable(lengthPool, createPool, updatePool)
	pool_table.ShowHeaderRow = true
	pool_table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}

	pool_table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		// fmt.Println(id)
		if id.Col >= 0 && id.Col < len(pool_headers) {

			template.(*widget.Label).SetText(pool_headers[id.Col])
		}
	}

	for i := range pool_headers {
		pool_table.SetColumnWidth(i, largestMinSize(pool_headers).Width)
	}

	block_label_data := [][]string{}
	const limit = 10

	var block_table *widget.Table
	updateBlocksData := func() {

		block_label_data = [][]string{}
		height := program.node.info.TopoHeight
		// we are going to take the last ten blocks,
		// like... the last 3 minutes

		for i := 1; i <= limit; i++ {
			h := uint64(height) - uint64(i)
			for txid, each := range program.node.transactions {
				if each.Txs[0].Block_Height < int64(int(h)-limit) {
					delete(program.node.transactions, txid)
				}
			}

			tx_label_data := [][]string{}

			result := program.node.blocks[h]
			var bl block.Block
			b, err := hex.DecodeString(result.Blob)
			if err != nil {
				continue
			}
			bl.Deserialize(b)

			tx_results, transactions := getTxsAndTransactions(bl.Tx_hashes)
			size := uint64(len(bl.Serialize()))

			tx_results_callback := func(tx_results []rpc.GetTransaction_Result) {
				for i := range tx_results {
					size += uint64(len(transactions[i].Serialize()))
					var rings uint64
					if len(transactions[i].Payloads) > 0 { // not sure when this wouldn't be the case...
						rings = uint64(len(tx_results[i].Txs[0].Ring[0]))
					}
					tx_label_data = append(tx_label_data, []string{
						"", "", "", "", "",
						transactions[i].GetHash().String(),
						transactions[i].TransactionType.String(),
						fmt.Sprintf("%0.5f", float64(transactions[i].Fees())/atomic_units),
						strconv.Itoa(int(rings)),
						fmt.Sprintf("%0.3f", float64(len(transactions[i].Serialize()))/kilobyte),
					})
				}
			}

			if len(tx_results) != 0 {
				tx_results_callback(tx_results)
			}

			block_label_data = append(block_label_data, []string{
				strconv.Itoa(int(result.Block_Header.Height)),
				strconv.Itoa(int(result.Block_Header.TopoHeight)),
				time.Duration((uint64(time.Now().Local().UnixMilli()) - bl.Timestamp) * uint64(time.Millisecond)).String(),
				strconv.Itoa(len(bl.MiniBlocks)),
				fmt.Sprintf("%0.3f", float64(size)/kilobyte),
				bl.GetHash().String(),
				"BLOCK",
				fmt.Sprintf("%0.5f", float64(result.Block_Header.Reward)/atomic_units),
				"N/A",
				fmt.Sprintf("%0.3f", float64(len(bl.Miner_TX.Serialize()))/kilobyte),
			})
			if len(tx_label_data) > 0 {
				block_label_data = append(block_label_data, tx_label_data...)
			}
		}
	}
	updateBlocksData()

	lengthBlocks := func() (rows int, cols int) {
		return len(block_label_data), len(block_headers)
	}
	createBlocks := func() fyne.CanvasObject {
		return container.NewStack(widget.NewLabel(""), container.NewScroll(widget.NewHyperlink("", nil)))
	}
	updateBlocks := func(tci widget.TableCellID, co fyne.CanvasObject) {
		block_data := block_label_data
		cell := co.(*fyne.Container)
		label := cell.Objects[0].(*widget.Label)
		scroll := cell.Objects[1].(*container.Scroll)
		link := scroll.Content.(*widget.Hyperlink)
		if len(block_label_data) == 0 {
			label.SetText("")
			label.Hide()
			link.SetText("")
			link.Hide()
			return
		}
		switch tci.Col {
		case 0, 1, 2, 3, 4, 6, 7, 8, 9:
			label.SetText(block_data[tci.Row][tci.Col])
			label.Show()
			scroll.Hide()
		case 5:
			label.Hide()
			scroll.Show()
			link.SetText(block_data[tci.Row][tci.Col])
			link.OnTapped = func() {
				hash := block_data[tci.Row][tci.Col]
				searchBlockchain(hash)
				results_table.Refresh()
				tabs.Select(tab_search)
			}
		}

	}

	block_table = widget.NewTable(lengthBlocks, createBlocks, updateBlocks)
	block_table.ShowHeaderRow = true
	block_table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabel("")
	}
	block_table.UpdateHeader = func(id widget.TableCellID, template fyne.CanvasObject) {
		if id.Col >= 0 && id.Col < len(block_headers) {
			template.(*widget.Label).SetText(block_headers[id.Col])
		}
	}
	for i := range block_headers {
		block_table.SetColumnWidth(i, largestMinSize(block_headers).Width)
	}

	search := widget.NewEntry()

	lengthSearch := func() (rows int, cols int) { return len(searchData), 2 }
	createSearch := func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.SetText("THIS IS A PLACEHOLDER FOR THE APPLICATION")
		l.Wrapping = fyne.TextWrapOff
		return container.NewStack(l)
	}
	updateSearch := func(id widget.TableCellID, template fyne.CanvasObject) {
		box := template.(*fyne.Container)
		l := box.Objects[0].(*widget.Label)

		switch id.Col {
		case 0:
			if id.Row >= len(searchHeaders) {
				l.SetText("")
			} else {
				text := searchHeaders[id.Row]
				l.SetText(text)
				l.Refresh()
			}
		case 1:
			if id.Row >= len(searchData) {
				l.SetText("")
			} else {
				text := searchData[id.Row]
				l.SetText(text)
				l.Refresh()
				if id.Row < len(searchHeaders) && (searchHeaders[id.Row] == "SC CODE" || searchHeaders[id.Row] == "SC ARGS") {
					sizing := l.MinSize().Height + (theme.Padding() * 2)
					results_table.SetRowHeight(id.Row, sizing)
					l.Refresh()
				}
			}
		default:
			l.SetText("ERROR")
		}
	}
	results_table = widget.NewTable(
		lengthSearch, createSearch, updateSearch,
	)
	results_table.OnSelected = func(id widget.TableCellID) {
		var data string
		if id.Col == 0 {
			data = searchHeaders[id.Row]
			program.application.Clipboard().SetContent(data)
			results_table.UnselectAll()
			results_table.Refresh()
			showInfoFast("Copied", data, program.explorer)
		} else {
			data = searchData[id.Row]
			program.application.Clipboard().SetContent(data)
			results_table.UnselectAll()
			results_table.Refresh()
			showInfoFast("Copied", data, program.explorer)
		}
	}

	tapped := func() {
		if search.Text == "" {
			return
		}
		s := search.Text
		search.SetText("")
		searchBlockchain(s)
	}

	search.ActionItem = widget.NewButtonWithIcon("search", theme.SearchIcon(), tapped)
	search.OnSubmitted = func(s string) {
		tapped()
	}
	search.SetPlaceHolder("search txid/scid/blockid/blockheight")
	searchBar := container.NewVBox(search)

	var updating bool = true
	update := func() {
		height := program.node.info.TopoHeight
		for range time.NewTicker(time.Second * 2).C {
			if !updating {
				return
			}
			if height != program.node.info.TopoHeight {
				height = program.node.info.TopoHeight

				updateDiffData()

				newGraph := &graph{hd_map: diff_map}
				g.ExtendBaseWidget(g)

				// Replace content in graph container
				diff_graph = newGraph

				updatePoolCache()

				updateBlocksData()

				fyne.DoAndWait(func() {
					diff_graph.Refresh()
					pool_table.Refresh()
					block_table.Refresh()
				})

				stats = []string{
					strconv.Itoa(int(program.node.info.Height)),
					strconv.Itoa(int(program.node.info.AverageBlockTime50)),
					strconv.Itoa(int(program.node.info.Tx_pool_size)),
					strconv.Itoa(int(program.node.info.Difficulty) / 1000000),
					strconv.Itoa(int(program.node.info.Total_Supply)),
					program.node.info.Status,
				}
				fyne.DoAndWait(func() {
					diff.SetText("Network Height: " + stats[0])
					average_blocktime.SetText("Network Blocktime: " + stats[1] + " seconds")
					mem_pool.SetText("Mempool Size: " + stats[2])
					hash_rate.SetText("Hash Rate: " + stats[3] + " MH/s")
					supply.SetText("Total Supply: " + stats[4])
					network_status.SetText("Network Status: " + stats[5])
				})

			}
		}
	}
	go update()

	search_window := container.NewBorder(searchBar, nil, nil, nil, results_table)
	tab_search = container.NewTabItem("Search", search_window)

	pool := container.NewAdaptiveGrid(1, pool_table)
	tab_pool := container.NewTabItem("TX Pool", pool)

	blocks := container.NewAdaptiveGrid(1, block_table)
	tab_blocks := container.NewTabItem("Recent Blocks", blocks)

	tabs = container.NewAppTabs(tab_stats, tab_pool, tab_blocks, tab_search)

	tabs.SetTabLocation(container.TabLocationLeading)

	program.explorer.SetOnClosed(func() {
		logger.Info("explorer", "status", "close")
		updating = false
	})
	program.explorer.SetContent(tabs)

}
