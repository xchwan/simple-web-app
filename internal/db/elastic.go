package db

import (
	"log"
	"os"
)

// ElasticClient 是 Elasticsearch client 的佔位型別。
// TODO: 換成 github.com/elastic/go-elasticsearch/v8 的 *elasticsearch.Client
type ElasticClient struct {
	Addr string
}

// ConnectElastic 建立 Elasticsearch 連線。
// 讀取環境變數 ELASTIC_ADDR（預設 http://localhost:9200）。
func ConnectElastic() *ElasticClient {
	addr := os.Getenv("ELASTIC_ADDR")
	if addr == "" {
		addr = "http://localhost:9200"
	}
	log.Printf("Elasticsearch: %s", addr)
	return &ElasticClient{Addr: addr}
}
