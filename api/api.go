package api

import (
	"log"
	"net/http"

	"github.com/FishDontExist/TONindexer/controllers"
	"github.com/gorilla/mux"
)

func SetApi() {
	r := mux.NewRouter()
	r.HandleFunc("/ping", controllers.Ping).Methods("GET")
	r.HandleFunc("/height", controllers.GetHeight).Methods("GET")
	r.HandleFunc("/wallet", controllers.GenerateNewWallet).Methods("GET")

	http.Handle("/", r)
	log.Println("Listening on port 8000")
	log.Fatal(http.ListenAndServe(":8000", r))
}
