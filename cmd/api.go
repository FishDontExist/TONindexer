package main

import (
	"log"
	"net/http"

	"github.com/FishDontExist/TONindexer/controllers"
	"github.com/gorilla/mux"
)

func SetApi() {
	r := mux.NewRouter()
	r.HandleFunc("/v1/getheight", controllers.GetHeight).Methods("GET")
	r.HandleFunc("/v1/")
	http.Handle("/", r)
	log.Fatal(http.ListenAndServe(":8000", r))
}
