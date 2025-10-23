package rag

import (
	"context"
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/everfid-ever/ThinkForge/core"
	"github.com/everfid-ever/ThinkForge/core/config"
)

var ragSvr = &core.Rag{}

func init() {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		log.Printf("NewClient of es8 failed, err=%v", err)
		return
	}
	ragSvr, err = core.New(context.Background(), &config.Config{
		Client:    client,
		IndexName: "rag-test",
		APIKey:    os.Getenv("OPENAI_API_KEY"),
		BaseURL:   os.Getenv("OPENAI_BASE_URL"),
		ChatModel: "text-embedding-3-large",
	})
	if err != nil {
		log.Printf("New of rag failed, err=%v", err)
		return
	}
}

func GetRagSvr() *core.Rag {
	return ragSvr
}
