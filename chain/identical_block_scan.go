package chain

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
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
func (l *LiteClient) GetHeight() (*ton.BlockIDExt, error) {

	info, err := l.api.GetMasterchainInfo(l.ctx)
	if err != nil {
		return nil, err
	}
	return info, nil
}

func (l *LiteClient) GetBlockInfoByHeight(info ton.BlockIDExt) ([]ton.TransactionShortInfo, error) {
	extract, _, err := l.api.GetBlockTransactionsV2(l.ctx, &info, 100)
	if err != nil {
		return nil, err
	}
	// if !ok {
	// 	fmt.Println("No transactions found")
	// 	return nil, err
	// }
	return extract, nil
}

type Wallet struct {
	Address    string   `json:"address"`
	PrivateKey []string `json:"private_key"`
}

func (l *LiteClient) GenerateWallet() (Wallet, error) {
	words, err := GenerateSeedPhrase(12)
	if err != nil {
		return Wallet{}, err
	}
	w, err := wallet.FromSeed(l.api, words, wallet.V3)
	if err != nil {
		log.Println(err)
	}
	return Wallet{Address: w.WalletAddress().String(), PrivateKey: words}, nil
}

type BlockTransactions struct {
	Account string `json:"account"`
	Hash    string `json:"hash"`
	LT      uint64 `json:"lt"`
}

func LogTransactionShortInfo(tx ton.TransactionShortInfo) *BlockTransactions {
	accountHex := hex.EncodeToString(tx.Account)
	hashHex := hex.EncodeToString(tx.Hash)

	fmt.Printf("Transaction Info:\n")
	fmt.Printf("Account: %s\n", accountHex)
	fmt.Printf("LT: %d\n", tx.LT)
	fmt.Printf("Hash: %s\n", hashHex)
	return &BlockTransactions{Account: accountHex, Hash: hashHex, LT: tx.LT}
}

func (l *LiteClient) Transfer(account string, pk string, amount float64) (*tlb.Transaction, bool) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(pk)
	if err != nil {
		log.Println("Failed to decode private key: ", err)
	}
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	w, err := wallet.FromPrivateKey(l.api, privateKey, wallet.V2R1)
	if err != nil {
		panic(err)
	}
	strAmount := fmt.Sprintf("%f", amount)
	addr := address.MustParseAddr(account)
	transfer, err := w.BuildTransfer(addr, tlb.MustFromTON(strAmount), true, "")
	if err != nil {
		log.Println(err)
	}
	tx, _, err := w.SendWaitTransaction(l.ctx, transfer)
	if err != nil {
		log.Println(err)
		return nil, false
	}

	return tx, true
}

func (l *LiteClient) GetBalance(pk string) (tlb.Coins, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(pk)
	if err != nil {
		log.Println("Failed to decode private key: ", err)
	}
	privateKey := ed25519.PrivateKey(privateKeyBytes)
	w, err := wallet.FromPrivateKey(l.api, privateKey, wallet.V4R2)
	if err != nil {
		panic(err)
	}
	block, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Println("get masterchain info err: ", err.Error())
		return tlb.Coins{}, err
	}
	coins, err := w.GetBalance(l.ctx, block)
	if err != nil {
		log.Println(err)
		return tlb.Coins{}, err
	}
	return coins, nil
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

type SimpleBlock struct {
	block *ton.BlockIDExt
	time  time.Time
}

func (l *LiteClient) GetSimpleBlock() *SimpleBlock {
	block, err := l.GetHeight()
	if err != nil {
		return nil
	}
	return &SimpleBlock{block: block, time: time.Now()}
}

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

func (l *LiteClient) SendJetton(privateKey ed25519.PrivateKey, amount string, reciever string) (string, bool) {
	w, err := wallet.FromPrivateKey(l.api, privateKey, wallet.V3)
	if err != nil {
		log.Println(err)
		return "", false
	}
	token := jetton.NewJettonMasterClient(l.api, address.MustParseAddr("EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs"))
	tokenWallet, err := token.GetJettonWallet(l.ctx, w.WalletAddress())
	if err != nil {
		log.Println(err)
		return "", false
	}
	// tokenBalance, err := tokenWallet.GetBalance(l.ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }
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
