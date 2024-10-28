package main

import (
	"github.com/FishDontExist/TONindexer/chain"
)

func main() {
	pk := []string{"essay", "route", "raise", "title", "field", "dumb", "torch", "desert", "vocal", "seminar", "sketch", "soda", "burger", "daughter", "clog", "cup", "best", "helmet", "another", "federal", "cause", "long", "bullet", "grape"}
	ln := chain.New()
	ln.SendJetton(pk, "0.1", "UQB03G_qlolqO67Q5nutMkJ4Yy84o8r5_b_Ijam8Lfj0t1sC")

}
