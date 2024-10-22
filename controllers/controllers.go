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
	PrivateKey string `json:"private_key"`
	Sender     string `json:"sender"`
	Reciever   string `json:"receiver"`
	Amount     int    `json:"amount"`
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
	tx, ok := ln.Transfer(transaction.Reciever, transaction.PrivateKey, float64(transaction.Amount))
	if !ok {
		json.NewEncoder(w).Encode(map[string]string{"error": "transaction failed"})
	}
	hash := hex.EncodeToString(tx.Hash)
	json.NewEncoder(w).Encode(map[string]string{"tx": hash})
}

type Balance struct {
	Address string `json:"address"`
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var address Balance
	_ = json.NewDecoder(r.Body).Decode(address)
	ln := chain.New()
	coins, err:=ln.GetBalance(address.Address)
	if err!=nil{
		log.Println(err)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
	}
	json.NewEncoder(w).Encode(map[string]int64{"balance": coins.Nano().Int64()})
}
