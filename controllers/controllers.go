package controllers

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/FishDontExist/TONindexer/chain"
)

type Height struct {
	Height uint32
}

type Transaction struct {
	PrivateKey string "json:private_key"
	Sender     string "json:sender"
	reciever   string "json:receiver"
	amount     int    "json:amount"
}

func Ping(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": "pong"})
}
func GetHeight(w http.ResponseWriter, r *http.Request) {

	liteCient := chain.New()
	height, err := liteCient.GetHeight()
	if err != nil {
		log.Println("get height err: ", err.Error())
	}
	response := Height{
		Height: height.SeqNo,
	}
	w.Header().Set("Content-Type", "application/json")

	json.NewEncoder(w).Encode(response)
}

func GetBlockData(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
}

func GenerateNewWallet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}

func SendTransactionV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var transaction Transaction
	_ = json.NewDecoder(r.Body).Decode(transaction)
	ln := chain.New()
	tx, ok := ln.Transfer(transaction.reciever, transaction.PrivateKey, float64(transaction.amount))
	if !ok {
		json.NewEncoder(w).Encode(map[string]string{"error": "transaction failed"})
	}
	hash := hex.EncodeToString(tx.Hash)
	json.NewEncoder(w).Encode(map[string]string{"tx": hash})
}
