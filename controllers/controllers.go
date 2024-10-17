package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/FishDontExist/TONindexer/chain"
)

type Height struct {
	WorkChain int32  "json:work_chain"
	ShardNo   int64  "json:shard_no"
	SeqNo     uint32 "json:seq_no"
}

func GetHeight(w http.ResponseWriter, r *http.Request) {
	liteCient := chain.New()
	height, _ := liteCient.GetHeight()
	response := Height{
		WorkChain: height.Workchain,
		ShardNo:   height.Shard,
		SeqNo:     height.SeqNo,
	}
	w.Header().Set("Content-Type", "application/json")

	// Write the JSON response
	json.NewEncoder(w).Encode(response)
}

func GetBlockData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
}