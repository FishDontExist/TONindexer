package controllers

type DecomposeHeightT struct {
	Shard    int64
	SeqNo    uint32
	RootHash []byte
	FileHash []byte
}

type HeightReq struct {
	Height string `json:"height"`
}

type Height struct {
	Height string `json:"height"`
}

type Transaction struct {
	PrivateKey []string `json:"private_key"`
	Sender     string   `json:"sender"`
	Reciever   string   `json:"receiver"`
	Amount     int      `json:"amount"`
}

type Balance struct {
	Address string `json:"address"`
}

type TransactionForAddr struct {
	Addr string `json:"address"`
}

type BlockExt struct {
	SeqNo uint32 `json:"seqNo"`
}

type Jetton struct {
	Reciever   string   `json:"reciever"`
	PrivateKey []string `json:"private_key"`
	Amount     string   `json:"amount"`
}