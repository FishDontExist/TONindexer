package chain

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"log"
	"sort"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type LiteClient struct {
	api ton.APIClientWrapped
	ctx context.Context
}

func New() *LiteClient {
	client := liteclient.NewConnectionPool()

	cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
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

func LogTransactionShortInfo(tx ton.TransactionShortInfo) {
	accountHex := hex.EncodeToString(tx.Account)
	hashHex := hex.EncodeToString(tx.Hash)

	fmt.Printf("Transaction Info:\n")
	fmt.Printf("Account: %s\n", accountHex)
	fmt.Printf("LT: %d\n", tx.LT)
	fmt.Printf("Hash: %s\n", hashHex)
}

func (l *LiteClient) Transfer(account string, pk ed25519.PrivateKey, amount string, message string) bool {

	w, err := wallet.FromPrivateKey(l.api, pk, wallet.V3)
	if err != nil {
		panic(err)
	}

	addr := address.MustParseAddr(account)
	err = w.Transfer(l.ctx, addr, tlb.MustFromTON(amount), message)
	if err != nil {
		panic(err)
	}

	return true
}

func (l *LiteClient) GetBalance(pk ed25519.PrivateKey) (tlb.Coins, error) {
	w, err := wallet.FromPrivateKey(l.api, pk, wallet.V4R2)
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

func (l *LiteClient) GetTransactions(accountAddress string, pk ed25519.PrivateKey) {
	b, err := l.api.CurrentMasterchainInfo(l.ctx)
	if err != nil {
		log.Println("get masterchain info err: ", err.Error())
		return
	}

	addr := address.MustParseAddr(accountAddress)
	res, err := l.api.WaitForBlock(b.SeqNo).GetAccount(l.ctx, b, addr)
	if err != nil {
		log.Println("get account err: ", err.Error())
		return
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
	for {
		// last transaction has 0 prev lt
		if lastLt == 0 {
			break
		}

		// load transactions in batches with size 15
		list, err := l.api.ListTransactions(l.ctx, addr, 15, lastLt, lastHash)
		if err != nil {
			log.Printf("send err: %s", err.Error())
			return
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
		}
	}

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
