package chain

type Wallet struct {
	Address    string   `json:"address"`
	PrivateKey []string `json:"private_key"`
}

type BlockTransactions struct {
	Account string `json:"account"`
	Hash    string `json:"hash"`
	LT      uint64 `json:"lt"`
}
