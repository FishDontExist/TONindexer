package controllers

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/FishDontExist/TONindexer/chain"
	"github.com/xssnick/tonutils-go/ton"
)

type LiteNode struct {
	ln *chain.LiteClient
}
type Height struct {
	Height uint32
}

type Transaction struct {
	PrivateKey string `json:"private_key"`
	Sender     string `json:"sender"`
	Reciever   string `json:"receiver"`
	Amount     int    `json:"amount"`
}

func New() *LiteNode {
	return &LiteNode{
		ln: chain.New(),
	}
}

func Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": "pong"})
}
func (l *LiteNode) GetHeight(w http.ResponseWriter, r *http.Request) {

	height, err := l.ln.GetHeight()
	if err != nil {
		log.Println("get height err: ", err.Error())
	}
	response := Height{
		Height: height.SeqNo,
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

type HeightReq struct{
	Height int `json:"height"`
}
func (l *LiteNode)GetBlockData(w http.ResponseWriter, r *http.Request) {
	var height HeightReq
	w.Header().Set("Content-Type", "application/json")
	if err:= json.NewDecoder(r.Body).Decode(&height); err!=nil{
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]error{"err": err})
	}
	l.ln.GetBlockInfoByHeight(height.Height)
}

func (l *LiteNode) GenerateNewWallet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	wallet, err := l.ln.GenerateWallet()
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(wallet)
}

func (l *LiteNode) SendTransactionV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var transaction Transaction
	if err := json.NewDecoder(r.Body).Decode(&transaction); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	tx, ok := l.ln.Transfer(transaction.Reciever, transaction.PrivateKey, float64(transaction.Amount))
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Transaction failed"})
		return
	}
	hash := hex.EncodeToString(tx.Hash)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"tx": hash})
}

type Balance struct {
	Address string `json:"address"`
}

func (l *LiteNode) GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var address Balance
	_ = json.NewDecoder(r.Body).Decode(address)
	coins, err := l.ln.GetBalance(address.Address)
	if err != nil {
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
	}
	json.NewEncoder(w).Encode(map[string]int64{"balance": coins.Nano().Int64()})
}

func (l *LiteNode) GetSimpleBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	blockInfo, err := l.ln.GetHeight()
	if err != nil {
		log.Println(err)
	}
	response := map[string]any{"block": int(blockInfo.SeqNo), "timestamp": time.Now()}
	json.NewEncoder(w).Encode(response)
}

type BlockExt struct {
	seqNo uint32 `json:"seqNo"`
}

func (l *LiteNode) GetBlockTransactions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var BlockId BlockExt
	if err := json.NewDecoder(r.Body).Decode(&BlockId); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	blockIdReplicate, _ := l.ln.GetHeight()

	blockIDExt := ton.BlockIDExt{
		Workchain: blockIdReplicate.Workchain,
		Shard:     blockIdReplicate.Shard,
		SeqNo:     BlockId.seqNo,
		RootHash:  blockIdReplicate.RootHash,
		FileHash:  blockIdReplicate.FileHash,
	}
	transactions, err := l.ln.GetBlockInfoByHeight(blockIDExt)

	if err != nil {
		http.Error(w, "cannot retrieve transactions", http.StatusInternalServerError)
		return
	}
	var response []map[string]chain.BlockTransactions
	for transaction := range transactions {
		transactionInfo := chain.LogTransactionShortInfo(transactions[transaction])
		response = append(response, map[string]chain.BlockTransactions{"transaction": *transactionInfo})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

}

type TransactionForAddr struct {
	Addr string `json:"address"`
}

func (l *LiteNode) GetTransactionForAddr(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	var address TransactionForAddr
	if err := json.NewDecoder(r.Body).Decode(&address); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	transactions, err :=l.ln.GetTransactions(address.Addr)
	if err != nil {
		http.Error(w, "cannot retrieve transactions", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)

}
