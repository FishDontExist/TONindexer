package controllers

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/FishDontExist/TONindexer/chain"
	"github.com/xssnick/tonutils-go/ton"
)

type LiteNode struct {
	ln *chain.LiteClient
}
type Height struct {
	Height string `json:"height"`
}

type Transaction struct {
	PrivateKey []string `json:"private_key"`
	Sender     string   `json:"sender"`
	Reciever   string   `json:"receiver"`
	Amount     int      `json:"amount"`
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

	latestBlockInfo, err := l.ln.GetHeight()
	if err != nil {
		log.Println("get height err: ", err.Error())
	}

	shardBigInt := big.NewInt(latestBlockInfo.Shard)
	seqnoBigInt := big.NewInt(int64(latestBlockInfo.SeqNo))
	scaleFactor := big.NewInt(1e16)

	height := new(big.Int).Mul(shardBigInt, scaleFactor)
	height.Add(height, seqnoBigInt)

	rootHashHex := hex.EncodeToString(latestBlockInfo.RootHash)
	fileHashHex := hex.EncodeToString(latestBlockInfo.FileHash)

	// Concatenate height, root_hash, and file_hash into a single string
	combinedHeight := fmt.Sprintf("%s|%s|%s", height.String(), rootHashHex, fileHashHex)
	response := Height{
		Height: combinedHeight,
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

type HeightReq struct {
	Height string `json:"height"`
}

func (l *LiteNode) GetBlockData(w http.ResponseWriter, r *http.Request) {
	var height HeightReq
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewDecoder(r.Body).Decode(&height); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]error{"err": err})
	}
	heightStr, _ := decomposeHeight(height.Height)
	heightBigInt := heightStr.Height
	scaleFactor := big.NewInt(1e16)
	shardBigInt := new(big.Int).Div(heightBigInt, scaleFactor)

	// Calculate seqno = height % 10^16
	seqnoBigInt := new(big.Int).Mod(heightBigInt, scaleFactor)

	// Convert shard to int64 and seqno to uint32
	shard := shardBigInt.Int64()
	seqno := uint32(seqnoBigInt.Uint64())
	requestedBlock := &ton.BlockIDExt{
		Workchain: 0,
		Shard:     shard,
		SeqNo:     seqno,
		RootHash:  heightStr.RootHash,
		FileHash:  heightStr.FileHash,
	}
	result, err := l.ln.GetBlockInfoByHeight(requestedBlock)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]error{"err": err})
	}
	response, _ := encodeTransactionsToJSON(result)
	json.NewEncoder(w).Encode(response)

}

type DecomposeHeightT struct {
	Height   *big.Int
	RootHash []byte
	FileHash []byte
}

func decomposeHeight(combinedHeight string) (*DecomposeHeightT, error) {
	// Split the combined string by "|"
	parts := strings.Split(combinedHeight, "|")
	if len(parts) != 3 {
		return &DecomposeHeightT{}, fmt.Errorf("invalid combined height format")
	}

	// Parse height as big.Int
	height := new(big.Int)
	height.SetString(parts[0], 10)

	// Decode root hash from hex string
	rootHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return &DecomposeHeightT{}, fmt.Errorf("failed to decode root hash: %v", err)
	}

	// Decode file hash from hex string
	fileHash, err := hex.DecodeString(parts[2])
	if err != nil {
		return &DecomposeHeightT{}, fmt.Errorf("failed to decode file hash: %v", err)
	}

	return &DecomposeHeightT{Height: height, RootHash: rootHash, FileHash: fileHash}, nil
}
func encodeTransactionsToJSON(txs []ton.TransactionShortInfo) (string, error) {
	var transactions []chain.BlockTransactions

	for _, tx := range txs {
		transaction := chain.LogTransactionShortInfo(tx)
		transactions = append(transactions, *transaction)
	}

	// Encode the slice of BlockTransactions to JSON
	jsonData, err := json.MarshalIndent(transactions, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
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
	_ = json.NewDecoder(r.Body).Decode(&address)
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
	SeqNo uint32 `json:"seqNo"`
}

// func (l *LiteNode) GetBlockTransactions(w http.ResponseWriter, r *http.Request) {
// 	w.Header().Set("Content-Type", "application/json")
// 	var blockId BlockExt
// 	if err := json.NewDecoder(r.Body).Decode(&blockId); err != nil {
// 		http.Error(w, "Invalid request payload", http.StatusBadRequest)
// 		return
// 	}
// 	transactions, err := l.ln.GetBlockInfoByHeight(blockId.SeqNo)

// 	if err != nil {
// 		http.Error(w, "cannot retrieve transactions", http.StatusInternalServerError)
// 		return
// 	}
// 	var response []map[string]chain.BlockTransactions
// 	for transaction := range transactions {
// 		transactionInfo := chain.LogTransactionShortInfo(transactions[transaction])
// 		response = append(response, map[string]chain.BlockTransactions{"transaction": *transactionInfo})
// 	}

// 	w.WriteHeader(http.StatusOK)
// 	json.NewEncoder(w).Encode(response)

// }

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
	transactions, err := l.ln.GetTransactions(address.Addr)
	if err != nil {
		http.Error(w, "cannot retrieve transactions", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(transactions)

}
