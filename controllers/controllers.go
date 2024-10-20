package controllers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/FishDontExist/TONindexer/chain"
)

type Height struct {
	Height uint32
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
