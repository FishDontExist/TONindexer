package main

import (
	"fmt"

	"github.com/FishDontExist/TONindexer/chain"
)

func main() {
	pk := []string{"essay", "route", "raise", "title", "field", "dumb", "torch", "desert", "vocal", "seminar", "sketch", "soda", "burger", "daughter", "clog", "cup", "best", "helmet", "another", "federal", "cause", "long", "bullet", "grape"}
	ln := chain.New()
	res, _ := ln.GetBalance(pk)
	fmt.Println(res.String())

}
