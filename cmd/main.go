package main

import "github.com/FishDontExist/TONindexer/chain"

func main() {
	// pk := []string{"essay", "route", "raise", "title", "field", "dumb", "torch", "desert", "vocal", "seminar", "sketch", "soda", "burger", "daughter", "clog", "cup", "best", "helmet", "another", "federal", "cause", "long", "bullet", "grape"}
	ln := chain.New()
	ln.GetPrevBlocks()
	// api.SetApi()
}
