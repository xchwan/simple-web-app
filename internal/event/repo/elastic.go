package eventrepo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

const indexName = "events"

// indexMapping 定義 events 索引的欄位型別。
const indexMapping = `{
	"mappings": {
		"properties": {
			"id":           { "type": "integer" },
			"organizer_id": { "type": "integer" },
			"name":         { "type": "text"    },
			"description":  { "type": "text"    },
			"start_at":     { "type": "date"    }
		}
	}
}`

// eventDoc 是存入 ES 的文件格式。
type eventDoc struct {
	ID          int    `json:"id"`
	OrganizerID int    `json:"organizer_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	StartAt     string `json:"start_at"` // RFC3339
}

// ElasticEventSearchRepository 實作 EventSearchRepository。
// 只負責搜尋索引的讀寫，不碰 source of truth（MySQL）。
type ElasticEventSearchRepository struct {
	es *elasticsearch.Client
}

// NewElasticSearchRepository 建立 ElasticEventSearchRepository。
// es 為 nil 時回傳 nil（不啟用 ES）。
func NewElasticSearchRepository(es *elasticsearch.Client) *ElasticEventSearchRepository {
	if es == nil {
		return nil
	}
	r := &ElasticEventSearchRepository{es: es}
	r.ensureIndex()
	return r
}

// Index 將 Event 寫入（或覆蓋）ES 索引文件。
func (r *ElasticEventSearchRepository) Index(e *Event) error {
	doc := eventDoc{
		ID:          e.ID,
		OrganizerID: e.OrganizerID,
		Name:        e.Name,
		Description: e.Description,
		StartAt:     e.StartAt.Format(time.RFC3339),
	}
	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	res, err := r.es.Index(
		indexName,
		bytes.NewReader(body),
		r.es.Index.WithDocumentID(fmt.Sprintf("%d", e.ID)),
		r.es.Index.WithRefresh("true"),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("ES index 回應錯誤: %s", res.String())
	}
	return nil
}

// Remove 從 ES 索引刪除指定文件。
func (r *ElasticEventSearchRepository) Remove(id int) {
	res, err := r.es.Delete(
		indexName,
		fmt.Sprintf("%d", id),
		r.es.Delete.WithRefresh("true"),
	)
	if err != nil {
		log.Printf("[ES] delete 失敗 (id=%d): %v", id, err)
		return
	}
	defer res.Body.Close()
	if res.IsError() {
		log.Printf("[ES] delete 回應錯誤 (id=%d): %s", id, res.String())
	}
}

// Search 使用 ES bool query 做全文搜尋與日期範圍過濾。
func (r *ElasticEventSearchRepository) Search(q EventQuery) ([]*Event, error) {
	body, err := json.Marshal(buildSearchBody(q))
	if err != nil {
		return nil, err
	}
	res, err := r.es.Search(
		r.es.Search.WithIndex(indexName),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.IsError() {
		return nil, fmt.Errorf("ES search 回應錯誤: %s", res.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source eventDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	events := make([]*Event, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		startAt, _ := time.Parse(time.RFC3339, h.Source.StartAt)
		events = append(events, &Event{
			ID:          h.Source.ID,
			OrganizerID: h.Source.OrganizerID,
			Name:        h.Source.Name,
			Description: h.Source.Description,
			StartAt:     startAt,
		})
	}
	return events, nil
}

// ensureIndex 若 events 索引不存在則建立，含欄位 mapping。
func (r *ElasticEventSearchRepository) ensureIndex() {
	res, err := r.es.Indices.Exists([]string{indexName})
	if err != nil {
		log.Printf("[ES] 檢查索引失敗: %v", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return
	}
	res2, err := r.es.Indices.Create(
		indexName,
		r.es.Indices.Create.WithBody(strings.NewReader(indexMapping)),
	)
	if err != nil {
		log.Printf("[ES] 建立索引失敗: %v", err)
		return
	}
	defer res2.Body.Close()
	if res2.IsError() {
		log.Printf("[ES] 建立索引回應錯誤: %s", res2.String())
		return
	}
	log.Printf("[ES] 索引 %q 建立完成", indexName)
}

// buildSearchBody 組出 ES bool query。
func buildSearchBody(q EventQuery) map[string]any {
	var mustClause any
	if strings.TrimSpace(q.Keyword) != "" {
		mustClause = map[string]any{
			"multi_match": map[string]any{
				"query":  q.Keyword,
				"fields": []string{"name^2", "description"},
			},
		}
	} else {
		mustClause = map[string]any{"match_all": map[string]any{}}
	}

	filterClause := []any{}
	if q.StartFrom != nil || q.StartTo != nil {
		rangeClause := map[string]any{}
		if q.StartFrom != nil {
			rangeClause["gte"] = q.StartFrom.Format(time.RFC3339)
		}
		if q.StartTo != nil {
			rangeClause["lte"] = q.StartTo.Format(time.RFC3339)
		}
		filterClause = append(filterClause, map[string]any{
			"range": map[string]any{"start_at": rangeClause},
		})
	}

	return map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must":   mustClause,
				"filter": filterClause,
			},
		},
	}
}
