package eventdb

// ElasticEventRepository 以 Elasticsearch 實作 EventRepository。
// 採用 Decorator 模式包住下一個 repo（通常是 MySQLEventRepository）：
//   - Search  → 由 ES 處理，利用全文索引提供更好的搜尋體驗
//   - Save / Update / Delete → 同步更新 ES 索引，再委派給 next
//   - FindByID → ES 不擅長 point query，直接委派給 next
//
// TODO: 接入真實 ES client（github.com/elastic/go-elasticsearch）
type ElasticEventRepository struct {
	next EventRepository // 責任鏈的下一個節點，通常是 MySQLEventRepository
}

// NewElasticRepository 建立一個包住 next 的 ElasticEventRepository。
func NewElasticRepository(next EventRepository) EventRepository {
	return &ElasticEventRepository{next: next}
}

func (r *ElasticEventRepository) Save(e *Event) error {
	// TODO: 寫入 ES 索引
	return r.next.Save(e)
}

func (r *ElasticEventRepository) FindByID(id int) (*Event, bool) {
	return r.next.FindByID(id)
}

func (r *ElasticEventRepository) Search(q EventQuery) []*Event {
	// TODO: 使用 ES query DSL 實作全文搜尋
	// 目前 fallback 到 next（MySQL LIKE）
	return r.next.Search(q)
}

func (r *ElasticEventRepository) Update(e *Event) error {
	// TODO: 更新 ES 索引
	return r.next.Update(e)
}

func (r *ElasticEventRepository) Delete(id int) {
	// TODO: 從 ES 刪除索引
	r.next.Delete(id)
}
