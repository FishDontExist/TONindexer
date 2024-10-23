package api

import (
	"log"
	"net/http"

	"github.com/FishDontExist/TONindexer/controllers"
	"github.com/gorilla/mux"
)

func SetApi() {
	r := mux.NewRouter()
	lt := controllers.New()
	r.HandleFunc("/ping", controllers.Ping).Methods("GET")
	r.HandleFunc("/height", lt.GetHeight).Methods("GET")
	r.HandleFunc("/wallet", lt.GenerateNewWallet).Methods("GET")
	r.HandleFunc("/sendtx", lt.SendTransactionV2).Methods("POST")
	r.HandleFunc("/getblock", lt.GetBlockTransactions).Methods("POST")

	http.Handle("/", r)
	log.Println("Listening on port 8000")
	log.Fatal(http.ListenAndServe(":8000", r))
}
