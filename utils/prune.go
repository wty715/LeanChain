package utils

import (
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	dbPath      = "E:/ethereum/geth/chaindata"
	ancientPath = dbPath + "/ancient"

	bloomPath = "D:/deleted/Accounts_"
)

func DoPrune(cmdline []string) {
	if len(cmdline) < 3 {
		fmt.Println("Error! Must indicate checkpoint block interval, begin blknum, and end blknum.")
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
	fmt.Printf("currHeader: %x\n", currHeader)

	// ReadHeaderNumber returns the header number assigned to a hash.
	currHeight := rawdb.ReadHeaderNumber(ancientDb, currHeader)
	fmt.Printf("currHeight: %d\n", *currHeight)

	// Create in-memory trie database
	triedb := trie.NewDatabase(ancientDb)

	fmt.Println("----------------------------------------------------------------")

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
			fmt.Printf("Err: Block not found: %v\n", i)
		} else {
			fmt.Printf("Etherscan url: https://etherscan.io/block/%v\n", i)
			fmt.Printf("BlockHash: %x\n", blkHash)
		}

		// ReadHeader retrieves the block header corresponding to the hash.
		blkHeader := rawdb.ReadHeader(ancientDb, blkHash, uint64(i))
		fmt.Printf("Block state root: 0x%x\n", blkHeader.Root)

		// Retrieve state root and construct the trie accordingly
		Trie, err := trie.NewStateTrie(common.Hash{}, blkHeader.Root, triedb)
		if err != nil {
			panic(err)
		}

		// ReadBody retrieves the block body corresponding to the hash.
		blkBody := rawdb.ReadBody(ancientDb, blkHash, uint64(i))
		fmt.Printf("BlkBody Tx size: %d\n", len(blkBody.Transactions))

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
			fmt.Printf("Last sliding window we have %v unique accounts.\n", num_of_account)
			num_of_account = 0

			// Then check each tx to find influenced account
			for _, tx := range blkBody.Transactions {
				// fmt.Printf("tx Hash: %v\n", tx.Hash())
				txFrom := getFromAddr(tx, big.NewInt(int64(i)))
				if !influenced_account[txFrom] {
					influenced_account[txFrom] = true
					// fmt.Printf("[Adding] tx From: %v\n", txFrom)
					num_of_account++
				}
				if tx.To() != nil {
					txTo := *(tx.To())
					if !influenced_account[txTo] {
						influenced_account[txTo] = true
						// fmt.Printf("[Adding] tx To  : %v\n", txTo)
						num_of_account++
					}
				}
			}
		} else {
			// Perform pruning
			for _, tx := range blkBody.Transactions {
				// fmt.Printf("tx Hash: %v\n", tx.Hash())
				txFrom := getFromAddr(tx, big.NewInt(int64(i)))
				if !influenced_account[txFrom] {
					influenced_account[txFrom] = true
					// fmt.Printf("[Adding] tx From: %v\n", txFrom)
					num_of_account++
				} else {
					// Delete this account's state
					err = Trie.TryDeleteAccount(txFrom.Bytes())
					if err == nil {
						if !deleted_account[txFrom] {
							// fmt.Printf("[NewDel] tx From  : %v\n", txFrom)
							total_del_account++
							deleted_account[txFrom] = true
						}
					} else {
						fmt.Println(err)
						fmt.Printf("[ErrDel] tx From  : %v\n", txFrom)
					}
				}
				if tx.To() != nil {
					txTo := *(tx.To())
					if !influenced_account[txTo] {
						influenced_account[txTo] = true
						// fmt.Printf("[Adding] tx To  : %v\n", txTo)
						num_of_account++
					} else {
						// Delete this account's state
						acc, err := Trie.TryGetAccount(txTo.Bytes())
						if err == nil {
							if acc != nil && acc.CodeHash == nil && tx.Value().Cmp(big.NewInt(0)) == 1 {
								err := Trie.TryDeleteAccount(txTo.Bytes())
								if err == nil {
									if !deleted_account[txTo] {
										// fmt.Printf("[NewDel] tx To  : %v\n", txTo)
										total_del_account++
										deleted_account[txTo] = true
									}
								} else {
									fmt.Println(err)
									fmt.Printf("[ErrDel] tx To  : %v\n", txTo)
								}
							}
						} else {
							fmt.Println(err)
							fmt.Printf("[ErrGet] tx To  : %v\n", txTo)
						}
					}
				}
			}
		}
		fmt.Printf("Block %v deleted %v accounts.\n", i, total_del_account)
		for acc := range deleted_account {
			file.WriteString(acc.String() + " ")
			delete(deleted_account, acc)
		}
		file.WriteString("BLKEND\n")
		// ReadBlock retrieves an entire block corresponding to the hash
		if blkHash != rawdb.ReadBlock(ancientDb, blkHash, uint64(i)).Hash() {
			fmt.Println("Error! blkhash doesn't match the block")
			return
		}
		// root, nodeset, err := Trie.Commit(true)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Printf("Block %v now trie root = %v\n", i, root)
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
		fmt.Println("----------------------------------------------------------------")
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
