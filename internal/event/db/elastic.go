package eventdb

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
// name / description 使用 text（支援全文搜尋），start_at 使用 date。
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

// ElasticEventRepository 以 Elasticsearch 實作 EventRepository。
//
// Decorator 模式：包住下一層 repo（通常是 MySQLEventRepository）。
//   - Search  → ES 全文索引，提供比 LIKE 更好的搜尋體驗
//   - Save / Update / Delete → 先交給 next 處理（MySQL），再同步 ES 索引
//   - FindByID → ES 不擅長 point query，直接委派給 next
//
// 若 ES 操作失敗（如 ES 暫時不可用），只記 log 而不回傳錯誤，
// 保持 MySQL 為唯一 source of truth；Search 失敗時 fallback 到 next。
type ElasticEventRepository struct {
	es   *elasticsearch.Client
	next EventRepository
}

// NewElasticRepository 建立包住 next 的 ElasticEventRepository，
// 並確保 ES 索引已建立（index + mapping）。
func NewElasticRepository(es *elasticsearch.Client, next EventRepository) EventRepository {
	r := &ElasticEventRepository{es: es, next: next}
	r.ensureIndex()
	return r
}

// ensureIndex 若 events 索引不存在則建立，含欄位 mapping。
func (r *ElasticEventRepository) ensureIndex() {
	res, err := r.es.Indices.Exists([]string{indexName})
	if err != nil {
		log.Printf("[ES] 檢查索引失敗: %v", err)
		return
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		return // 已存在
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

// Save 先寫入 MySQL，成功後再寫入 ES 索引。
func (r *ElasticEventRepository) Save(e *Event) error {
	if err := r.next.Save(e); err != nil {
		return err
	}
	if err := r.upsert(e); err != nil {
		log.Printf("[ES] index 失敗 (id=%d): %v", e.ID, err)
	}
	return nil
}

// FindByID 直接委派給 next（MySQL point query 效率更好）。
func (r *ElasticEventRepository) FindByID(id int) (*Event, bool) {
	return r.next.FindByID(id)
}

// Search 使用 ES bool query 做全文搜尋與日期範圍過濾。
// 若 ES 失敗，fallback 到 next（MySQL LIKE）。
func (r *ElasticEventRepository) Search(q EventQuery) []*Event {
	body, err := json.Marshal(buildSearchBody(q))
	if err != nil {
		log.Printf("[ES] 序列化 query 失敗: %v", err)
		return r.next.Search(q)
	}

	res, err := r.es.Search(
		r.es.Search.WithIndex(indexName),
		r.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		log.Printf("[ES] search 失敗，fallback MySQL: %v", err)
		return r.next.Search(q)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Printf("[ES] search 回應錯誤，fallback MySQL: %s", res.String())
		return r.next.Search(q)
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Source eventDoc `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		log.Printf("[ES] 解析回應失敗，fallback MySQL: %v", err)
		return r.next.Search(q)
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
	return events
}

// Update 先更新 MySQL，再 re-index 到 ES。
func (r *ElasticEventRepository) Update(e *Event) error {
	if err := r.next.Update(e); err != nil {
		return err
	}
	if err := r.upsert(e); err != nil {
		log.Printf("[ES] re-index 失敗 (id=%d): %v", e.ID, err)
	}
	return nil
}

// Delete 先刪除 MySQL，再刪除 ES 文件。
func (r *ElasticEventRepository) Delete(id int) {
	r.next.Delete(id)
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

// upsert 將 Event 寫入（或覆蓋）ES 文件。
func (r *ElasticEventRepository) upsert(e *Event) error {
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

// buildSearchBody 依 EventQuery 組出 ES bool query。
//
// must 子句：
//   - 有 keyword → multi_match（name^2 + description）
//   - 無 keyword → match_all
//
// filter 子句：
//   - StartFrom / StartTo → range query on start_at
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
