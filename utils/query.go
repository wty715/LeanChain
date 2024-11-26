package utils

import (
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie"
)

func DoQuery(cmdline []string) {
	if len(cmdline) < 4 {
		fmt.Println("Error! Must indicate at least the account hash, the block interval, and start/end block number.")
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
		fmt.Println("------------------------------------------------------------------")
		prunedRangeQuery(inter, cmdline[0], upNum, endNum, rangeint)
	} else {
		// do the point query
		originPointQuery(cmdline[0], upNum, endNum)
		fmt.Println("------------------------------------------------------------------")
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
	fmt.Printf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	fmt.Printf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	fmt.Println("----------------------Origin Point Query----------------------")
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
			fmt.Printf("Err: Block not found: %v\n", i)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(i))
		// fmt.Printf("Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}
		_, err = Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		// acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		if err != nil {
			fmt.Println(err)
		}
		// else {
		// 	fmt.Printf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), acc.Balance, i)
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
		} else if roundElapsed > shortestime && roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			fmt.Printf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	fmt.Printf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration(endNum-upNum+1))
	fmt.Printf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
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
	fmt.Printf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	fmt.Printf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	fmt.Println("----------------------Origin Range Query----------------------")
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
				fmt.Printf("Err: Block not found: %v\n", j)
			}

			// ReadHeader retrieves the block header corresponding to the hash.
			blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(j))
			// fmt.Printf("Block state root: 0x%x\n", blkHeader.Root)

			// Retrieve state root and construct the trie accordingly
			Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
			if err != nil {
				panic(err)
			}
			_, err = Trie.TryGetAccount(common.HexToAddress(account).Bytes())
			// acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
			if err != nil {
				fmt.Println(err)
			}
			// else {
			// 	fmt.Printf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), acc.Balance, i)
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
		} else if roundElapsed > shortestime && roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			fmt.Printf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	fmt.Printf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration((endNum-upNum)/rangeint+1))
	fmt.Printf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
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
	fmt.Printf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	fmt.Printf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	// Read bloom filter
	var prunedAddresses []map[common.Address]bool
	// prunedAddresses = append(prunedAddresses, map[common.Address]bool{})
	// cpBlockNum := upNum - upNum%200

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
	// fmt.Printf("prunedAddress length = %d\n", len(prunedAddresses))

	bloomTime := time.Now()

	for i := upNum; i <= endNum; i++ {
		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))
		if blkHash == (common.Hash{}) {
			fmt.Printf("Err: Internal Block not found: %v\n", i)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))

		// Retrieve transactions and perform rebuilding
		for _, tx := range blkBody.Transactions {
			// fmt.Printf("tx Hash: %v\n", tx.Hash())
			txFrom := getFromAddr(tx, big.NewInt(int64(i)))
			prunedAddresses[i-upNum][txFrom] = true
			if tx.To() != nil {
				txTo := *(tx.To())
				prunedAddresses[i-upNum][txTo] = true
			}
		}
	}
	bloomElapsed := time.Since(bloomTime) / time.Microsecond
	fmt.Printf("Bloom time: %d us.\n", bloomElapsed)

	fmt.Println("----------------------Pruned Point Query----------------------")
	startTime := time.Now()

	var longestime = time.Duration(0)
	var long2ndtime = time.Duration(0)
	var shortestime = time.Duration(10000000)
	var short2ndtime = time.Duration(10000000)

	var internalTime = time.Duration(0)
	for i := upNum; i < endNum; i += inter { // i: cpBlk
		for j := i; j < i+inter; j++ { // j: iterate queried blk
			roundTime := time.Now()

			// ReadCanonicalHash retrieves the hash assigned to a canonical1 block number.
			blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))

			if blkHash == (common.Hash{}) {
				fmt.Printf("Err: Checkpoint Block not found: %v\n", i)
			}

			// ReadHeader retrieves the block header corresponding to the hash.
			blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(i))
			// fmt.Printf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

			// Retrieve state root and construct the trie accordingly
			Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
			if err != nil {
				panic(err)
			}
			acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
			if err != nil {
				fmt.Println(err)
			}
			var nowBalance = acc.Balance

			var internalStart = time.Now()
			for k := i + 1; k <= j; k++ {
				// check bloom filter
				//if prunedAddresses[k-upNum][common.HexToAddress(account)] {
				// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
				blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(k))

				if blkHash == (common.Hash{}) {
					fmt.Printf("Err: Internal Block not found: %v\n", k)
				}

				// ReadBody retrieves the block body corresponding to the hash.
				blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(k))
				// fmt.Printf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

				// Retrieve transactions and perform rebuilding
				for _, tx := range blkBody.Transactions {
					// fmt.Printf("tx Hash: %v\n", tx.Hash())
					txFrom := getFromAddr(tx, big.NewInt(int64(k)))
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
				//}
			}

			internalTime += time.Since(internalStart)
			// fmt.Printf("Account 0x%x had balance %d in block %d.\n", common.HexToAddress(account), nowBalance, i)

			roundElapsed := time.Since(roundTime) / time.Microsecond
			//fmt.Printf("Block %d time: %d. sub: %d, inter: %d\n", i, roundElapsed, i-localCpBlockNum, inter)
			if roundElapsed > longestime {
				long2ndtime = longestime
				longestime = roundElapsed
			} else if roundElapsed > long2ndtime {
				long2ndtime = roundElapsed
			}
			if roundElapsed < shortestime {
				short2ndtime = shortestime
				shortestime = roundElapsed
			} else if roundElapsed > shortestime && roundElapsed < short2ndtime {
				short2ndtime = roundElapsed
			}
			if j%10000 == 0 {
				fmt.Printf("Block %d passed.\n", j)
			}
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	fmt.Printf("Total query time: %d us, internal time: %d us, average %d us.\n", elapsedTime, internalTime/time.Microsecond, elapsedTime/time.Duration(endNum-upNum+1))
	fmt.Printf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
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
	fmt.Printf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	fmt.Printf("currHeight: %d\n", *currHeight)

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
	// fmt.Printf("prunedAddress length = %d\n", len(prunedAddresses))

	for i := cpBlockNum + 1; i <= endNum; i++ {
		prunedAddresses = append(prunedAddresses, map[common.Address]bool{})

		// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
		blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(i))
		if blkHash == (common.Hash{}) {
			fmt.Printf("Err: Internal Block not found: %v\n", i)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))

		// Retrieve transactions and perform rebuilding
		for _, tx := range blkBody.Transactions {
			// fmt.Printf("tx Hash: %v\n", tx.Hash())
			txFrom := getFromAddr(tx, big.NewInt(int64(i)))
			prunedAddresses[i-cpBlockNum-1][txFrom] = true
			if tx.To() != nil {
				txTo := *(tx.To())
				prunedAddresses[i-cpBlockNum-1][txTo] = true
			}
		}
	}

	fmt.Println("----------------------Pruned Range Query----------------------")
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
			fmt.Printf("Err: Checkpoint Block not found: %v\n", localCpBlockNum)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(localCpBlockNum))
		// fmt.Printf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}
		acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
		if err != nil {
			fmt.Println(err)
		}

		var nowBalance = acc.Balance
		for j := localCpBlockNum + 1; j <= i+rangeint && j <= endNum; j += inter {
			var k = j
			for ; k < j+inter-1 && k <= endNum; k++ {
				if prunedAddresses[k-cpBlockNum-1][common.HexToAddress(account)] {
					// ReadCanonicalHash retrieves the hash assigned to a canonical block number.
					blkHash := rawdb.ReadCanonicalHash(ancientDb, uint64(k))

					if blkHash == (common.Hash{}) {
						fmt.Printf("Err: Internal Block not found: %v\n", k)
					}

					// ReadBody retrieves the block body corresponding to the hash.
					blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(k))
					// fmt.Printf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

					// Retrieve transactions and perform rebuilding
					for _, tx := range blkBody.Transactions {
						// fmt.Printf("tx Hash: %v\n", tx.Hash())
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
					fmt.Printf("Err: Checkpoint Block not found: %v\n", k)
				}

				// ReadHeader retrieves the block header corresponding to the hash.
				blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(k))
				// fmt.Printf("Checkpoint Block state root: 0x%x\n", blkHeader.Root)

				// Retrieve state root and construct the trie accordingly
				Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
				if err != nil {
					panic(err)
				}
				acc, err := Trie.TryGetAccount(common.HexToAddress(account).Bytes())
				if err != nil {
					fmt.Println(err)
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
		} else if roundElapsed > shortestime && roundElapsed < short2ndtime {
			short2ndtime = roundElapsed
		}
		if i%10000 == 0 {
			fmt.Printf("Block %d passed.\n", i)
		}
	}

	elapsedTime := time.Since(startTime) / time.Microsecond
	fmt.Printf("Total query time: %d us, average %d us.\n", elapsedTime, elapsedTime/time.Duration((endNum-upNum)/rangeint+1))
	fmt.Printf("Longest query time: %d us, shortest %d us.\n", long2ndtime, short2ndtime)
}
