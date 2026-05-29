package infra

import (
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

// ConnectElastic 建立 Elasticsearch client。
// 讀取環境變數 ELASTIC_ADDR（預設 http://localhost:9200）。
// 若連線失敗則 fatal，確保啟動時即發現問題。
func ConnectElastic() *elasticsearch.Client {
	addr := os.Getenv("ELASTIC_ADDR")
	if addr == "" {
		addr = "http://localhost:9200"
	}

	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{addr},
	})
	if err != nil {
		log.Fatalf("Elasticsearch client 建立失敗: %v", err)
	}

	log.Printf("Elasticsearch: %s", addr)
	return es
}
