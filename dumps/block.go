package ton_service

import (
	"context"
	"encoding/json"

	"log"
	"os"

	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)

type SimpleBlockInfo struct {
	Workchain int32 
	Shard     int64 
	SeqNo     uint32
}
func ConnectToLiteNode() (*ton.APIClient, error) {
    client := liteclient.NewConnectionPool()

    content, err := os.ReadFile("./global-config.json")
    if err != nil {
        log.Fatal("Error when opening file: ", err)
		return nil, err
    }

    config := liteclient.GlobalConfig{}
    err = json.Unmarshal(content, &config)
    if err != nil {
        log.Fatal("Error during Unmarshal(): ", err)
		return nil, err
    }

    err = client.AddConnectionsFromConfig(context.Background(), &config)
    if err != nil {
        log.Fatalln("connection err: ", err.Error())
        return nil, err
    }
    // initialize ton API lite connection
    api := ton.NewAPIClient(client)
    api = api.WithRetry().(*ton.APIClient)
	return api, nil
}

func getMasterchainInfo(api *ton.APIClient) (*ton.BlockIDExt, error) {
	blockInfo, err := api.GetMasterchainInfo(context.Background())
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return blockInfo, nil
}

func GetLatestBlockInfo(api *ton.APIClient) {
    masterchainInfo, err := getMasterchainInfo(api)
	if err != nil {
		log.Fatal("BlockIDExt erro: ", err)
	}
    // Lookup the latest block
    blockInfo, err := api.LookupBlock(context.Background(), masterchainInfo.Workchain, masterchainInfo.Shard, masterchainInfo.SeqNo)
    if err != nil {
        log.Fatal("Something wrong in looking up the latest block: ", err)
        return
    }

    // Get transactions in the latest block
    transactions, getBlockBool, err := api.GetBlockTransactionsV2(context.Background(), blockInfo, 100)
    if err != nil {
        log.Fatal("Error getting block transactions: ", err)
        return
    }
	log.Println("getBlockBool: ", getBlockBool)
	

    // Print the transactions
    log.Println("Transactions in the latest block:")
    for _, tx := range transactions {
        id3 := tx.ID3()
        log.Printf("Transaction ID3: %+v\n", id3)
    }
}