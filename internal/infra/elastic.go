package infra

import (
	"log"
	"os"

	"github.com/elastic/go-elasticsearch/v8"
)

// ConnectElastic 建立 Elasticsearch client。
// 讀取環境變數 ELASTIC_ADDR（預設 http://localhost:9200）。
// 若建立失敗則 fatal，確保啟動時即發現問題。
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

// TryConnectElastic 與 ConnectElastic 相同，但 ELASTIC_ADDR 未設定時回傳 nil（不啟用）。
// 供測試環境使用，允許 ES 為可選服務。
func TryConnectElastic() *elasticsearch.Client {
	addr := os.Getenv("ELASTIC_ADDR")
	if addr == "" {
		return nil
	}
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: []string{addr},
	})
	if err != nil {
		log.Printf("Elasticsearch client 建立失敗: %v", err)
		return nil
	}
	log.Printf("Elasticsearch (test): %s", addr)
	return es
}

// CleanESIndex 刪除指定 ES index（測試專用）。
// index 會在下次 NewElasticEventSearchRepository 呼叫時透過 ensureIndex() 自動重建。
// es 為 nil 或 index 為空時不執行。
func CleanESIndex(es *elasticsearch.Client, index string) {
	if es == nil || index == "" {
		return
	}
	res, err := es.Indices.Delete([]string{index})
	if err != nil {
		log.Printf("[ES] 清理 index %q 失敗: %v", index, err)
		return
	}
	defer res.Body.Close()
	// 404 = index 不存在，可忽略
}
