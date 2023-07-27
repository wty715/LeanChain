package main

import (
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	putf = fmt.Printf
	puts = fmt.Println

	dbPath      = "F:/ethereum/geth/chaindata"
	ancientPath = dbPath + "/ancient"

	// bloomPath = "H:/deleted/Accounts_"
)

func doPrune(cmdline []string) {
	if len(cmdline) < 3 {
		puts("Error! Must indicate checkpoint block interval, begin blknum, and end blknum.")
		return
	}
	N, err := strconv.Atoi(cmdline[0])
	if err != nil {
		panic(err)
	}
	upNum, err := strconv.Atoi(cmdline[1])
	if err != nil {
		panic(err)
	}
	endNum, err := strconv.Atoi(cmdline[2])
	if err != nil {
		panic(err)
	}

	// Open ethereum levelDB with ancient flatten data
	ancientDb, err := rawdb.NewLevelDBDatabaseWithFreezer(dbPath, 16, 1, ancientPath, "", false)
	if err != nil {
		panic(err)
	}

	// ReadHeadHeaderHash retrieves the hash of the current canonical head header.
	currHeader := rawdb.ReadHeadHeaderHash(ancientDb)
	putf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	putf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	puts("----------------------------------------------------------------")

	// Checkpoint block state list
	var influenced_account = map[common.Address]bool{}
	var num_of_account = 0
	var file (*os.File)

	for i := endNum; i >= upNum; i-- {
		// set deleted account map and number for each block
		var deleted_account = map[common.Address]bool{}
		var total_del_account = 0

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))

		if blkHash == (common.Hash{}) {
			putf("Err: Block not found: %v\n", i)
		} else {
			putf("Etherscan url: https://etherscan.io/block/%v\n", i)
			putf("BlockHash: %x\n", blkHash)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(i))
		putf("Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))
		putf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

		// Every N blocks we maintain a checkpoint block
		if i%N == 0 {
			// Create file indicating the deleted accounts
			file, err = os.OpenFile("H:/deleted/Accounts_"+fmt.Sprint(i)+".txt", os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			// first empty the map
			for acc := range influenced_account {
				delete(influenced_account, acc)
			}
			putf("Last sliding window we have %v unique accounts.\n", num_of_account)
			num_of_account = 0

			// Then check each tx to find influenced account
			for _, tx := range blkBody.Transactions {
				// putf("tx Hash: %v\n", tx.Hash())
				txFrom := getFromAddr(tx, big.NewInt(int64(i)))
				if !influenced_account[txFrom] {
					influenced_account[txFrom] = true
					// putf("[Adding] tx From: %v\n", txFrom)
					num_of_account++
				}
				if tx.To() != nil {
					txTo := *(tx.To())
					if !influenced_account[txTo] {
						influenced_account[txTo] = true
						// putf("[Adding] tx To  : %v\n", txTo)
						num_of_account++
					}
				}
			}
		} else {
			// Perform pruning
			for _, tx := range blkBody.Transactions {
				// putf("tx Hash: %v\n", tx.Hash())
				txFrom := getFromAddr(tx, big.NewInt(int64(i)))
				if !influenced_account[txFrom] {
					influenced_account[txFrom] = true
					// putf("[Adding] tx From: %v\n", txFrom)
					num_of_account++
				} else {
					// Delete this account's state
					err = Trie.TryDeleteAccount(txFrom.Bytes())
					if err == nil {
						if !deleted_account[txFrom] {
							// putf("[NewDel] tx From  : %v\n", txFrom)
							total_del_account++
							deleted_account[txFrom] = true
						}
					} else {
						puts(err)
						putf("[ErrDel] tx From  : %v\n", txFrom)
					}
				}
				if tx.To() != nil {
					txTo := *(tx.To())
					if !influenced_account[txTo] {
						influenced_account[txTo] = true
						// putf("[Adding] tx To  : %v\n", txTo)
						num_of_account++
					} else {
						// Delete this account's state
						acc, err := Trie.TryGetAccount(txTo.Bytes())
						if err == nil {
							if acc != nil && acc.CodeHash == nil && tx.Value().Cmp(big.NewInt(0)) == 1 {
								err := Trie.TryDeleteAccount(txTo.Bytes())
								if err == nil {
									if !deleted_account[txTo] {
										// putf("[NewDel] tx To  : %v\n", txTo)
										total_del_account++
										deleted_account[txTo] = true
									}
								} else {
									puts(err)
									putf("[ErrDel] tx To  : %v\n", txTo)
								}
							}
						} else {
							puts(err)
							putf("[ErrGet] tx To  : %v\n", txTo)
						}
					}
				}
			}
		}
		putf("Block %v deleted %v accounts.\n", i, total_del_account)
		for acc := range deleted_account {
			file.WriteString(acc.String() + " ")
			delete(deleted_account, acc)
		}
		file.WriteString("BLKEND\n")
		// ReadBlock retrieves an entire block corresponding to the hash
		if blkHash != rawdb.ReadBlock(ancientDb, blkHash, uint64(i)).Hash() {
			puts("Error! blkhash doesn't match the block")
			return
		}
		// root, nodeset, err := Trie.Commit(true)
		// if err != nil {
		// 	panic(err)
		// }
		// putf("Block %v now trie root = %v\n", i, root)
		// if nodeset != nil {
		// 	mergeNS := trie.NewWithNodeSet(nodeset)
		// 	err = triedb.Update(mergeNS)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// 	err = triedb.Commit(root, false, nil)
		// 	if err != nil {
		// 		panic(err)
		// 	}
		// }
		puts("----------------------------------------------------------------")
	}
}

func doQuery(cmdline []string) {
	if len(cmdline) < 4 {
		puts("Error! Must indicate at least the account hash, the block interval, and start/end block number.")
		return
	}
	inter, err := strconv.Atoi(cmdline[1])
	if err != nil {
		panic(err)
	}
	upNum, err := strconv.Atoi(cmdline[2])
	if err != nil {
		panic(err)
	}
	endNum, err := strconv.Atoi(cmdline[3])
	if err != nil {
		panic(err)
	}
	if len(cmdline) == 5 {
		// do the range query
		rangeint, err := strconv.Atoi(cmdline[4])
		if err != nil {
			panic(err)
		}
		originRangeQuery(cmdline[0], upNum, endNum, rangeint)
		puts("------------------------------------------------------------------")
		prunedRangeQuery(inter, cmdline[0], upNum, endNum, rangeint)
	} else {
		// do the point query
		originPointQuery(cmdline[0], upNum, endNum)
		puts("------------------------------------------------------------------")
		prunedPointQuery(inter, cmdline[0], upNum, endNum)
	}
}

func originPointQuery(account string, upNum int, endNum int) {
	// Open ethereum levelDB with ancient flatten data
	ancientDb, err := rawdb.NewLevelDBDatabaseWithFreezer(dbPath, 16, 1, ancientPath, "", false)
	if err != nil {
		panic(err)
	}
	defer ancientDb.Close()

	// ReadHeadHeaderHash retrieves the hash of the current canonical head header.
	currHeader := rawdb.ReadHeadHeaderHash(ancientDb)
	putf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	putf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	puts("----------------------Origin Point Query----------------------")
	startTime := time.Now()

	var longestime = time.Duration(0)
	var long2ndtime = time.Duration(0)
	var shortestime = time.Duration(10000000)
	var short2ndtime = time.Duration(10000000)
	for i := upNum; i <= endNum; i++ {
		roundTime := time.Now()

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))

		if blkHash == (common.Hash{}) {
			putf("Err: Block not found: %v\n", i)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(i))
		// putf("Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}
		_, err = Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		// acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		if err != nil {
			puts(err)
		}
		// else {
		// 	putf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), acc.Balance, i)
		// }

		roundElapsed := time.Since(roundTime) / time.Microsecond
		if roundElapsed > longestime {
			long2ndtime = longestime
			longestime = roundElapsed
		} else if roundElapsed > long2ndtime {
			long2ndtime = roundElapsed
		}
		if roundElapsed < shortestime {
			short2ndtime = shortestime
			shortestime = roundElapsed
		} else if roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			putf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	putf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration(endNum-upNum))
	putf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
}

func originRangeQuery(account string, upNum int, endNum int, rangeint int) {
	// Open ethereum levelDB with ancient flatten data
	ancientDb, err := rawdb.NewLevelDBDatabaseWithFreezer(dbPath, 16, 1, ancientPath, "", false)
	if err != nil {
		panic(err)
	}
	defer ancientDb.Close()

	// ReadHeadHeaderHash retrieves the hash of the current canonical head header.
	currHeader := rawdb.ReadHeadHeaderHash(ancientDb)
	putf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	putf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	puts("----------------------Origin Range Query----------------------")
	startTime := time.Now()

	var longestime = time.Duration(0)
	var long2ndtime = time.Duration(0)
	var shortestime = time.Duration(10000000)
	var short2ndtime = time.Duration(10000000)
	for i := upNum; i <= endNum; i += rangeint {
		roundTime := time.Now()

		for j := i; j <= i+rangeint && j <= endNum; j++ {
			// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
			blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(j))

			if blkHash == (common.Hash{}) {
				putf("Err: Block not found: %v\n", j)
			}

			// ReadHeader retrieves the block header corresponding to the hash.
			blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(j))
			// putf("Block state root: 0x%x\n", blkHeader.Root)

			// Retrieve state root and construct the trie accordingly
			Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
			if err != nil {
				panic(err)
			}
			_, err = Trie.TryGetAccount(common.HexToAddress(account).Bytes())
			// acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
			if err != nil {
				puts(err)
			}
			// else {
			// 	putf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), acc.Balance, i)
			// }
		}

		roundElapsed := time.Since(roundTime) / time.Microsecond
		if roundElapsed > longestime {
			long2ndtime = longestime
			longestime = roundElapsed
		} else if roundElapsed > long2ndtime {
			long2ndtime = roundElapsed
		}
		if roundElapsed < shortestime {
			short2ndtime = shortestime
			shortestime = roundElapsed
		} else if roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			putf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	putf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration((endNum-upNum)/rangeint+1))
	putf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
}

func prunedPointQuery(inter int, account string, upNum int, endNum int) {
	// Open ethereum levelDB with ancient flatten data
	ancientDb, err := rawdb.NewLevelDBDatabaseWithFreezer(dbPath, 16, 1, ancientPath, "", false)
	if err != nil {
		panic(err)
	}
	defer ancientDb.Close()

	// ReadHeadHeaderHash retrieves the hash of the current canonical head header.
	currHeader := rawdb.ReadHeadHeaderHash(ancientDb)
	putf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	putf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	// Read bloom filter
	var prunedAddresses []map[common.Address]bool
	// prunedAddresses = append(prunedAddresses, map[common.Address]bool{})
	cpBlockNum := upNum - upNum%inter

	// file, err := os.OpenFile(bloomPath+fmt.Sprint(cpBlockNum+200)+".txt", os.O_RDONLY, 0666)
	// if err != nil {
	// 	panic(err)
	// }
	// defer file.Close()

	// scanner := bufio.NewScanner(file)
	// scanner.Split(bufio.ScanWords)
	// var i = 0
	// for scanner.Scan() {
	// 	str := scanner.Text()
	// 	if str == "BLKEND" {
	// 		i++
	// 		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})
	// 	} else {
	// 		prunedAddresses[i][common.HexToAddress(str)] = true
	// 	}
	// }
	// putf("prunedAddress length = %d\n", len(prunedAddresses))

	for i := cpBlockNum + 1; i <= endNum; i++ {
		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))
		if blkHash == (common.Hash{}) {
			putf("Err: Internal Block not found: %v\n", i)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))

		// Retrieve transactions and perform rebuilding
		for _, tx := range blkBody.Transactions {
			// putf("tx Hash: %v\n", tx.Hash())
			txFrom := getFromAddr(tx, big.NewInt(int64(i)))
			prunedAddresses[i-cpBlockNum-1][txFrom] = true
			if tx.To() != nil {
				txTo := *(tx.To())
				prunedAddresses[i-cpBlockNum-1][txTo] = true
			}
		}
	}

	puts("----------------------Pruned Point Query----------------------")
	startTime := time.Now()

	var longestime = time.Duration(0)
	var long2ndtime = time.Duration(0)
	var shortestime = time.Duration(10000000)
	var short2ndtime = time.Duration(10000000)
	for i := upNum; i <= endNum; i += inter {
		localCpBlockNum := i - i%inter
		roundTime := time.Now()

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(localCpBlockNum))

		if blkHash == (common.Hash{}) {
			putf("Err: Checkpoint Block not found: %v\n", localCpBlockNum)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(localCpBlockNum))
		// putf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}
		acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		if err != nil {
			puts(err)
		}

		var nowBalance = acc.Balance
		for j := localCpBlockNum + 1; j <= i; j++ {
			if prunedAddresses[j-cpBlockNum-1][common.HexToAddress(account)] {
				// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
				blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(j))

				if blkHash == (common.Hash{}) {
					putf("Err: Internal Block not found: %v\n", j)
				}

				// ReadBody retrieves the block body corresponding to the hash.
				blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(j))
				// putf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

				// Retrieve transactions and perform rebuilding
				for _, tx := range blkBody.Transactions {
					// putf("tx Hash: %v\n", tx.Hash())
					txFrom := getFromAddr(tx, big.NewInt(int64(j)))
					if txFrom == common.HexToAddress(account) {
						nowBalance.Sub(nowBalance, tx.Value())
						nowBalance.Sub(nowBalance, tx.GasPrice().Mul(tx.GasPrice(), big.NewInt(21000)))
					} else if tx.To() != nil {
						txTo := *(tx.To())
						if txTo == common.HexToAddress(account) {
							nowBalance.Add(nowBalance, tx.Value())
						}
					}
				}
			}
		}
		// putf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), nowBalance, i)

		roundElapsed := time.Since(roundTime) / time.Microsecond
		if roundElapsed > longestime {
			long2ndtime = longestime
			longestime = roundElapsed
		} else if roundElapsed > long2ndtime {
			long2ndtime = roundElapsed
		}
		if roundElapsed < shortestime {
			short2ndtime = shortestime
			shortestime = roundElapsed
		} else if roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			putf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	putf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration((endNum-upNum)/inter+1))
	putf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
}

func prunedRangeQuery(inter int, account string, upNum int, endNum int, rangeint int) {
	// Open ethereum levelDB with ancient flatten data
	ancientDb, err := rawdb.NewLevelDBDatabaseWithFreezer(dbPath, 16, 1, ancientPath, "", false)
	if err != nil {
		panic(err)
	}
	defer ancientDb.Close()

	// ReadHeadHeaderHash retrieves the hash of the current canonical head header.
	currHeader := rawdb.ReadHeadHeaderHash(ancientDb)
	putf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	putf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	// Read bloom filter
	var prunedAddresses []map[common.Address]bool
	// prunedAddresses = append(prunedAddresses, map[common.Address]bool{})
	cpBlockNum := upNum - upNum%inter

	// file, err := os.OpenFile(bloomPath+fmt.Sprint(cpBlockNum+200)+".txt", os.O_RDONLY, 0666)
	// if err != nil {
	// 	panic(err)
	// }
	// defer file.Close()

	// scanner := bufio.NewScanner(file)
	// scanner.Split(bufio.ScanWords)
	// var i = 0
	// for scanner.Scan() {
	// 	str := scanner.Text()
	// 	if str == "BLKEND" {
	// 		i++
	// 		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})
	// 	} else {
	// 		prunedAddresses[i][common.HexToAddress(str)] = true
	// 	}
	// }
	// putf("prunedAddress length = %d\n", len(prunedAddresses))

	for i := cpBlockNum + 1; i <= endNum; i++ {
		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))
		if blkHash == (common.Hash{}) {
			putf("Err: Internal Block not found: %v\n", i)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))

		// Retrieve transactions and perform rebuilding
		for _, tx := range blkBody.Transactions {
			// putf("tx Hash: %v\n", tx.Hash())
			txFrom := getFromAddr(tx, big.NewInt(int64(i)))
			prunedAddresses[i-cpBlockNum-1][txFrom] = true
			if tx.To() != nil {
				txTo := *(tx.To())
				prunedAddresses[i-cpBlockNum-1][txTo] = true
			}
		}
	}

	puts("----------------------Pruned Range Query----------------------")
	startTime := time.Now()

	var longestime = time.Duration(0)
	var long2ndtime = time.Duration(0)
	var shortestime = time.Duration(10000000)
	var short2ndtime = time.Duration(10000000)
	for i := upNum; i <= endNum; i += rangeint {
		localCpBlockNum := i - i%inter
		roundTime := time.Now()

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(localCpBlockNum))

		if blkHash == (common.Hash{}) {
			putf("Err: Checkpoint Block not found: %v\n", localCpBlockNum)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(localCpBlockNum))
		// putf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}
		acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		if err != nil {
			puts(err)
		}

		var nowBalance = acc.Balance
		for j := localCpBlockNum + 1; j <= i+rangeint && j <= endNum; j += inter {
			var k = j
			for ; k < j+inter-1 && k <= endNum; k++ {
				if prunedAddresses[k-cpBlockNum-1][common.HexToAddress(account)] {
					// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
					blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(k))

					if blkHash == (common.Hash{}) {
						putf("Err: Internal Block not found: %v\n", k)
					}

					// ReadBody retrieves the block body corresponding to the hash.
					blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(k))
					// putf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

					// Retrieve transactions and perform rebuilding
					for _, tx := range blkBody.Transactions {
						// putf("tx Hash: %v\n", tx.Hash())
						txFrom := getFromAddr(tx, big.NewInt(int64(i)))
						if txFrom == common.HexToAddress(account) {
							nowBalance.Sub(nowBalance, tx.Value())
							nowBalance.Sub(nowBalance, tx.GasPrice().Mul(tx.GasPrice(), big.NewInt(21000)))
						} else if tx.To() != nil {
							txTo := *(tx.To())
							if txTo == common.HexToAddress(account) {
								nowBalance.Add(nowBalance, tx.Value())
							}
						}
					}
				}
			}
			if k == j+inter-1 {
				// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
				blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(k))

				if blkHash == (common.Hash{}) {
					putf("Err: Checkpoint Block not found: %v\n", k)
				}

				// ReadHeader retrieves the block header corresponding to the hash.
				blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(k))
				// putf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

				// Retrieve state root and construct the trie accordingly
				Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
				if err != nil {
					panic(err)
				}
				acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
				if err != nil {
					puts(err)
				}
				nowBalance = acc.Balance
			}
		}

		roundElapsed := time.Since(roundTime) / time.Microsecond
		if roundElapsed > longestime {
			long2ndtime = longestime
			longestime = roundElapsed
		} else if roundElapsed > long2ndtime {
			long2ndtime = roundElapsed
		}
		if roundElapsed < shortestime {
			short2ndtime = shortestime
			shortestime = roundElapsed
		} else if roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			putf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	putf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration((endNum-upNum)/rangeint+1))
	putf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
}

func main() {
	if len(os.Args) < 2 {
		puts("Error! Must indicate the option: $go run main.go [prune/query]")
	}

	switch os.Args[1] {
	case "prune":
		doPrune(os.Args[2:])
	case "query":
		doQuery(os.Args[2:])
	default:
		puts("Error! Must indicate the option: $go run main.go [prune/query]")
	}
}

func getFromAddr(tx *types.Transaction, num *big.Int) common.Address {
	var signer types.Signer = types.MakeSigner(params.MainnetChainConfig, num)

	from, err := types.Sender(signer, tx)
	if err != nil {
		panic(err)
	}

	return from
}
