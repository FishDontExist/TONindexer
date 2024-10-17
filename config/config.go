package config

import (
	"context"
	"log"

	"github.com/xssnick/tonutils-go/liteclient"
)

func GetConfig() (*liteclient.GlobalConfig, error) {

	cfg, err := liteclient.GetConfigFromUrl(context.Background(), "https://ton.org/global.config.json")
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
		return nil, err
	}
	return cfg, nil
}
