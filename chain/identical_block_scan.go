package chain

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

const (
	Limit          = 200
	NumberOfShards = 4
)

type LiteClient struct {
	api ton.APIClientWrapped
	ctx context.Context
}

func New() *LiteClient {
	client := liteclient.NewConnectionPool()

	// cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
	filepath := "../chain/globalconfig.json"
	cfg, err := liteclient.GetConfigFromFile(filepath)
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
		return nil
	}

	err = client.AddConnectionsFromConfig(context.Background(), cfg)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return nil
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(cfg)
	return &LiteClient{
		api: api,
		ctx: context.Background(),
	}
}

// TODO:
// GetParentBlocks()
func (l *LiteClient) GetHeight() (*ton.BlockIDExt, error) {

	masterchainInfo, err := l.api.GetMasterchainInfo(l.ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	shardInfoList, err := l.api.GetBlockShardsInfo(l.ctx, masterchainInfo)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	var wc0Shard *ton.BlockIDExt
	for _, shard := range shardInfoList {
		log.Println("shard+1")
		if shard.Workchain == 0 {
			wc0Shard = shard
			break
		}
	}

	if wc0Shard == nil {
		log.Println("No shard found for workchain 0")
		return nil, err
	}

	return wc0Shard, nil
}

func (l *LiteClient) GetBlockInfoByHeight(info *ton.BlockIDExt) (*[]BlockTransactions, error) {

	extract, _, err := l.api.GetBlockTransactionsV2(l.ctx, info, 300)
	if err != nil {
		return nil, err
	}
	// if !ok {
	// 	fmt.Println("No transactions found")
	// 	return nil, err
	// }
	transactoinList := logTransactionShortInfo(extract)
	return transactoinList, nil
}

func (l *LiteClient) GenerateWallet() (Wallet, error) {
	words := wallet.NewSeed()
	w, err := wallet.FromSeed(l.api, words, wallet.V3R2)
	if err != nil {
		log.Println(err)
	}
	return Wallet{Address: w.WalletAddress().String(), PrivateKey: words}, nil
}

func (l *LiteClient) Transfer(account string, pk []string, amount float64) (*tlb.Transaction, bool) {

	// privateKeyBytes, err := base64.StdEncoding.DecodeString(pk)
	// if err != nil {
	// 	log.Println("Failed to decode private key: ", err)
	// }
	// privateKey := ed25519.PrivateKey(privateKeyBytes)
	w, err := wallet.FromSeed(l.api, pk, wallet.ConfigV5R1Final{
		NetworkGlobalID: -239,
		Workchain:       0,
	})
	if err != nil {
		panic(err)
	}
	if w == nil {
		log.Println("wallet is nil")
		return nil, false
	}

	log.Println("wallet address:", w.WalletAddress())
	log.Println("fetching and checking proofs since config init block, it may take near a minute...")
	block, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return nil, false
	}
	log.Println("master proof checks are completed successfully, now communication is 100% safe!")

	balance, err := w.GetBalance(l.ctx, block)
	if err != nil {
		log.Fatalln("GetBalance err:", err.Error())
		return nil, false
	}
	log.Println("balance:", balance.String())
	addr := address.MustParseAddr(account)

	log.Println("sending transaction and waiting for confirmation...")

	// if destination wallet is not initialized (or you don't care)
	// you should set bounce to false to not get money back.
	// If bounce is true, money will be returned in case of not initialized destination wallet or smart-contract error
	bounce := true

	tonAmountsStr := fmt.Sprintf("%f", amount)
	transfer, err := w.BuildTransfer(addr, tlb.MustFromTON(tonAmountsStr), bounce, "Hello from tonutils-go!")
	if err != nil {
		log.Fatalln("Transfer err:", err.Error())
		return nil, false
	}

	tx, block, err := w.SendWaitTransaction(l.ctx, transfer)
	if err != nil {
		log.Fatalln("SendWaitTransaction err:", err.Error())
		return nil, false
	}

	balance, err = w.GetBalance(l.ctx, block)
	if err != nil {
		log.Fatalln("GetBalance err:", err.Error())
		return nil, false
	}

	log.Printf("transaction confirmed at block %d, hash: %s balance left: %s", block.SeqNo,
		base64.StdEncoding.EncodeToString(tx.Hash), balance.String())

	return tx, true

	// strAmount := fmt.Sprintf("%f", amount)
	// addr := address.MustParseAddr(account)
	// transfer, err := w.BuildTransfer(addr, tlb.MustFromTON(strAmount), true, "")
	// if err != nil {
	// 	log.Println(err)
	// }
	// tx, _, err := w.SendWaitTransaction(l.ctx, transfer)
	// if err != nil {
	// 	log.Println(err)
	// 	return nil, false
	// }

	// return tx, true
}
func (l *LiteClient) GetBalance(accountAddr string) (tlb.Coins, error) {

	b, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Println("get masterchain info err: ", err.Error())
		return tlb.Coins{}, err
	}
	addr := address.MustParseAddr(accountAddr)
	res, err := l.api.WaitForBlock(b.SeqNo).GetAccount(l.ctx, b, addr)
	if err != nil {
		log.Println("get account err: ", err.Error())
		return tlb.Coins{}, err
	}
	if !res.IsActive {
		return tlb.Coins{}, nil
	}

	return res.State.Balance, nil
}

func (l *LiteClient) GetTransactions(accountAddress string) ([]*tlb.Transaction, error) {
	b, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Println("get masterchain info err: ", err.Error())
		return nil, err
	}

	addr := address.MustParseAddr(accountAddress)
	res, err := l.api.WaitForBlock(b.SeqNo).GetAccount(l.ctx, b, addr)
	if err != nil {
		log.Println("get account err: ", err.Error())
		return nil, err
	}

	fmt.Printf("Is active: %v\n", res.IsActive)
	if res.IsActive {
		fmt.Printf("Status: %s\n", res.State.Status)
		fmt.Printf("Balance: %s TON\n", res.State.Balance.String())
		if res.Data != nil {
			fmt.Printf("Data: %s\n", res.Data.Dump())
		}
	}

	lastHash := res.LastTxHash
	lastLt := res.LastTxLT

	fmt.Printf("\nTransactions:\n")
	var transactions []*tlb.Transaction
	for {
		// last transaction has 0 prev lt
		if lastLt == 0 {
			break
		}

		// load transactions in batches with size 15
		list, err := l.api.ListTransactions(l.ctx, addr, 15, lastLt, lastHash)
		if err != nil {
			log.Printf("send err: %s", err.Error())
			return nil, err
		}
		// set previous info from the oldest transaction in list
		lastHash = list[0].PrevTxHash
		lastLt = list[0].PrevTxLT

		// reverse list to show the newest first
		sort.Slice(list, func(i, j int) bool {
			return list[i].LT > list[j].LT
		})

		for _, t := range list {
			fmt.Println(t.String())
			transactions = append(transactions, t)
		}

	}
	return transactions, nil
}

/*
func (l *LiteClient) GetTransactionByHash(hash string) (ton.TransactionShortInfo, error) {

	b, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Println("get masterchain info err: ", err.Error())
		return ton.TransactionShortInfo{}, err
	}

	addr := address.MustParseAddr("EQAYqo4u7VF0fa4DPAebk4g9lBytj2VFny7pzXR0trjtXQaO")
	res, err := l.api.WaitForBlock(b.SeqNo).GetTransactionByHash(l.ctx, b, addr, hash)
	if err != nil {
		log.Println("get account err: ", err.Error())
		return ton.TransactionShortInfo{}, err
	}

	return res, nil
}
*/

// type SimpleBlock struct {
// 	block *ton.BlockIDExt
// 	time  time.Time
// }

// func (l *LiteClient) GetSimpleBlock() *SimpleBlock {
// 	block, err := l.GetHeight()
// 	if err != nil {
// 		return nil
// 	}
// 	return &SimpleBlock{block: block, time: time.Now()}
// }

func (l *LiteClient) GetFee(pk ed25519.PrivateKey, accountAddr string) (float64, error) {
	// w, err := wallet.FromPrivateKey(l.api, pk, wallet.V3)
	// if err != nil {
	// 	log.Println(err)
	// }
	// addr := address.MustParseAddr(accountAddr)
	// recieverAddr := address.MustParseAddr(accountAddr)
	// amount, comment := "0.1", "test"
	// block, _ := l.api.CurrentMasterchainInfo(l.ctx)
	fee := 0.07
	return fee, nil
}

// TODO
func (l *LiteClient) GetJettonInfo() bool {
	tokenContract := address.MustParseAddr("EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs")
	master := jetton.NewJettonMasterClient(l.api, tokenContract)
	data, err := master.GetJettonData(l.ctx)
	if err != nil {
		log.Fatal(err)
	}
	decimals := 9
	content := data.Content.(*nft.ContentOnchain)
	log.Println("total supply:", data.TotalSupply.Uint64())
	log.Println("mintable:", data.Mintable)
	log.Println("admin addr:", data.AdminAddr)
	log.Println("onchain content:")
	log.Println("	name:", content.GetAttribute("name"))
	log.Println("	symbol:", content.GetAttribute("symbol"))
	if content.GetAttribute("decimals") != "" {
		decimals, err = strconv.Atoi(content.GetAttribute("decimals"))
		if err != nil {
			log.Println("invalid decimals")
			return false
		}
	}
	log.Println("	decimals:", decimals)
	log.Println("	description:", content.GetAttribute("description"))
	log.Println()

	tokenWallet, err := master.GetJettonWallet(l.ctx, address.MustParseAddr("EQCWdteEWa4D3xoqLNV0zk4GROoptpM1-p66hmyBpxjvbbnn"))
	if err != nil {
		log.Fatal(err)
	}

	tokenBalance, err := tokenWallet.GetBalance(l.ctx)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("jetton balance:", tlb.MustFromNano(tokenBalance, decimals))
	return true
}

func (l *LiteClient) SendJetton(pk []string, amount string, reciever string) (string, bool) {

	w, err := wallet.FromSeed(l.api, pk, wallet.ConfigV5R1Final{
		NetworkGlobalID: -239,
		Workchain:       0,
	})

	if err != nil {
		log.Println(err)
		return "", false
	}
	token := jetton.NewJettonMasterClient(l.api, address.MustParseAddr("EQC7Vk6yHv-3Sc7sShVUo_kpO-LoCABRepLCjklU5DtQlHvx"))

	tokenWallet, err := token.GetJettonWallet(l.ctx, w.WalletAddress())

	if err != nil {
		log.Println(err)
		return "", false
	}
	tokenBalance, err := tokenWallet.GetBalance(l.ctx)

	if err != nil {
		log.Fatal(err)
		return "", false
	}
	fmt.Println("jetton balance:", tokenBalance.String())
	amountTokens := tlb.MustFromDecimal(amount, 9)

	// IF needed
	comment, err := wallet.CreateCommentCell("Hello from Zion!")
	if err != nil {
		log.Fatal(err)
	}

	to := address.MustParseAddr(reciever)
	transferPayload, err := tokenWallet.BuildTransferPayloadV2(to, to, amountTokens, tlb.ZeroCoins, comment, nil)
	if err != nil {
		log.Println(err)
		return "", false
	}

	fee := "0.05"
	msg := wallet.SimpleMessage(tokenWallet.Address(), tlb.MustFromTON(fee), transferPayload)
	log.Println("sending transaction...")

	tx, _, err := w.SendWaitTransaction(l.ctx, msg)
	if err != nil {
		log.Println(err)
		return "", false
	}
	log.Println("transaction confirmed, hash:", base64.StdEncoding.EncodeToString(tx.Hash))
	hash := base64.StdEncoding.EncodeToString(tx.Hash)
	return hash, true
}

// //////////////////////////////////////////////////////////

type TimedTransaction struct {
	TransactionID string
	// Add other relevant fields
}

// Transaction represents a single transaction in the API response.
type Transaction struct {
	InMsg   json.RawMessage `json:"in_msg"`
	OutMsgs json.RawMessage `json:"out_msgs"`
	// Add other relevant fields as needed
}

type APIResponse struct {
	Transactions []Transaction `json:"transactions"`
}

func GetTransactionWithHash(txHash string) ([]TimedTransaction, error) {
	baseURL := "https://toncenter.com/api/v3/transactions"

	// Prepare query parameters
	params := url.Values{}
	params.Add("workchain", "0")
	params.Add("hash", txHash)
	params.Add("limit", "100")
	params.Add("offset", "0")
	params.Add("sort", "desc")

	// Construct the full URL
	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	var tx *Transaction = nil
	var apiResp APIResponse
	var transactions []TimedTransaction

	maxRetries := 4
	for attempt := 1; attempt <= maxRetries; attempt++ {
		resp, err := http.Get(fullURL)
		if err != nil {
			log.Printf("Attempt %d: Error making GET request: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}

		// Read and parse the response
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Attempt %d: Error reading response body: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("Attempt %d: Non-OK HTTP status: %s", attempt, resp.Status)
			log.Printf("Response Body: %s", string(body))
			time.Sleep(5 * time.Second)
			continue
		}

		// Parse JSON response
		err = json.Unmarshal(body, &apiResp)
		if err != nil {
			log.Printf("Attempt %d: Error parsing JSON response: %v", attempt, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(apiResp.Transactions) > 0 {
			tx = &apiResp.Transactions[0]
			break
		}

		// If no transactions found, wait and retry
		log.Printf("Attempt %d: No transactions found for hash %s. Retrying...", attempt, txHash)
		time.Sleep(5 * time.Second)
	}

	if tx == nil {
		return nil, errors.New("no transaction found after retries")
	}

	// Check for the presence of "in_msg" and "out_msgs"
	if len(tx.InMsg) == 0 || len(tx.OutMsgs) == 0 {
		return nil, errors.New("transaction missing 'in_msg' or 'out_msgs'")
	}

	// Parse the transaction
	parsedTx, err := parseTx(*tx)
	if err != nil {
		return nil, fmt.Errorf("error parsing transaction: %v", err)
	}

	if len(parsedTx) != 0 {
		transactions = append(transactions, parsedTx...)
	}

	return transactions, nil
}

func parseTx(tx Transaction) ([]TimedTransaction, error) {
	var timedTxs []TimedTransaction

	var inMsg string
	var outMsgs []string

	err := json.Unmarshal(tx.InMsg, &inMsg)
	if err != nil {
		log.Printf("Error unmarshaling in_msg: %v", err)
		return timedTxs, err
	}

	err = json.Unmarshal(tx.OutMsgs, &outMsgs)
	if err != nil {
		log.Printf("Error unmarshaling out_msgs: %v", err)
		return timedTxs, err
	}

	timedTx := TimedTransaction{
		TransactionID: inMsg,
	}

	timedTxs = append(timedTxs, timedTx)
	return timedTxs, nil
}

func logTransactionShortInfo(t []ton.TransactionShortInfo) *[]BlockTransactions {

	var blockTransactions []BlockTransactions
	var accountHex, hashHex string

	for _, transaction := range t {
		accountHex = hex.EncodeToString(transaction.Account)
		hashHex = hex.EncodeToString(transaction.Hash)

		fmt.Printf("Transaction Info:\n")
		fmt.Printf("Account: %s\n", accountHex)
		fmt.Printf("LT: %d\n", transaction.LT)
		fmt.Printf("Hash: %s\n", hashHex)

		blockTransactions = append(blockTransactions, BlockTransactions{Account: accountHex, Hash: hashHex, LT: transaction.LT})
	}

	return &blockTransactions
}

///////////////////////////////////////////////////////////////
////      get shards ///////////
//////////////////////////////////

func (l *LiteClient) GetPrevBlocks() {
	ctx := l.ctx
	api := l.api

	masterchainInfo, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		log.Fatalf("Failed to get masterchain info: %v", err)
	}

	blocksMap := make(map[string]*ton.BlockIDExt)
	var blocks []*ton.BlockIDExt

	masterBlock := masterchainInfo

	var mu sync.Mutex

	for len(blocks) < Limit {
		shardBlocks, err := api.GetBlockShardsInfo(ctx, masterBlock)
		if err != nil {
			log.Fatalf("Failed to get shard blocks: %v", err)
		}
		for _, shard := range shardBlocks {
			log.Println(shard.Workchain, shard.Shard)
		}

		var workchain0Shards []*ton.BlockIDExt
		for _, shard := range shardBlocks {
			if shard.Workchain == 0 {
				workchain0Shards = append(workchain0Shards, shard)
			}
		}

		if len(workchain0Shards) == 0 {
			log.Fatalf("No workchain 0 shard blocks found at masterchain seqno %d", masterBlock.SeqNo)
		}

		type blockResult struct {
			blocks []*ton.BlockIDExt
			err    error
		}
		resultCh := make(chan blockResult, len(workchain0Shards))

		var wg sync.WaitGroup

		for _, shardBlock := range workchain0Shards {
			wg.Add(1)
			go func(shardBlock *ton.BlockIDExt) {

				defer wg.Done()

				shardBlocksCollected, err := collectShardBlocks(ctx, api, shardBlock, Limit/NumberOfShards)
				if err != nil {
					resultCh <- blockResult{nil, err}
					return
				}
				log.Println(len(shardBlocksCollected))
				resultCh <- blockResult{shardBlocksCollected, nil}
			}(shardBlock)
		}

		wg.Wait()
		close(resultCh)

		for res := range resultCh {
			if res.err != nil {
				log.Fatalf("Error collecting shard blocks: %v", res.err)
			}
			mu.Lock()
			for _, blk := range res.blocks {
				blockKey := fmt.Sprintf("%d:%d:%d", blk.Workchain, blk.Shard, blk.SeqNo)
				if _, exists := blocksMap[blockKey]; !exists {
					blocksMap[blockKey] = blk
					blocks = append(blocks, blk)
					if len(blocks) >= Limit {
						break
					}
				}
			}
			mu.Unlock()
			if len(blocks) >= Limit {
				break
			}
		}

		if len(blocks) >= Limit {
			break
		}

		prevBlockData, err := api.GetBlockData(ctx, masterBlock)
		if err != nil {
			log.Fatalf("Failed to get previous masterchain block data: %v", err)
		}

		prevBlocks, _ := getPrevBlocks(&prevBlockData.BlockInfo)

		if len(prevBlocks) == 0 {
			break
		}

		masterBlock = prevBlocks[0]
	}

	// if len(blocks) > Limit {
	// 	blocks = blocks[:Limit]
	// }

	sort.Slice(blocks, func(i, j int) bool {
		if blocks[i].SeqNo == blocks[j].SeqNo {
			return blocks[i].Shard > blocks[j].Shard
		}
		return blocks[i].SeqNo > blocks[j].SeqNo
	})

	for _, blk := range blocks {
		// fmt.Printf("Block: Workchain %d, Shard %d, SeqNo %d\n", blk.Workchain, blk.Shard, blk.SeqNo)
		fmt.Printf(" Shard %d, SeqNo %d\n", blk.Shard, blk.SeqNo)

	}
}

// collectShardBlocks traverses backward through a shardchain collecting blocks
func collectShardBlocks(ctx context.Context, api ton.APIClientWrapped, startBlock *ton.BlockIDExt, limit int) ([]*ton.BlockIDExt, error) {
	var blocks []*ton.BlockIDExt
	currentBlock := startBlock

	for len(blocks) < limit {
		blocks = append(blocks, currentBlock)

		blockData, err := api.GetBlockData(ctx, currentBlock)
		if err != nil {
			return nil, fmt.Errorf("failed to get block data for block %d: %w", currentBlock.SeqNo, err)
		}

		prevBlocks, err := blockData.BlockInfo.GetParentBlocks()
		if err != nil {
			return nil, fmt.Errorf("failed to get previous blocks for block %d: %w", currentBlock.SeqNo, err)
		}
		if len(prevBlocks) > 1 {
			log.Println(len(prevBlocks))
		}

		if len(prevBlocks) == 0 {
			break
		}

		currentBlock = prevBlocks[0]
	}

	return blocks, nil
}

func getPrevBlocks(blockInfo *tlb.BlockHeader) ([]*ton.BlockIDExt, error) {
	prevBlocks, err := blockInfo.GetParentBlocks()
	if err != nil {
		return nil, err
	}
	return prevBlocks, nil
}
