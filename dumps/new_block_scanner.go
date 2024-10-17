package dumps

import (
	"context"

	"log"


	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)


func InitializeTONapi(cfg *liteclient.GlobalConfig, ctx context.Context) ton.APIClientWrapped {
	client := liteclient.NewConnectionPool()

	err := client.AddConnectionsFromConfig(ctx, cfg)
	if err != nil {
		log.Fatalln("connection err: ", err.Error())
		return nil
	}
	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()
	api.SetTrustedBlockFromConfig(cfg)

	log.Println("checking proofs since config init block, it may take near a minute...")
	return api

}

