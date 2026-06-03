# 🎫 Simple Web App — 演唱會訂票系統

以 Go 實作的演唱會訂票平台，展示高流量場景下的後端架構設計：Redis 閘門、Kafka 非同步訂票、Elasticsearch 全文搜尋與 MySQL 原子更新。

## ✨ 功能

| 模組 | 功能 |
|------|------|
| User | 註冊、登入（session-based）、改名、搜尋 |
| Wallet | 建立錢包、存款、提款（原子 SQL 防止餘額為負） |
| Event | 建立活動、搜尋（MySQL 全文 / Elasticsearch）、更新、刪除 |
| Ticket | 批次建票、列表、狀態篩選 |
| Booking | 搶票（Redis 閘門 + Kafka 非同步）、取消（退款）、查詢 |

## 🏗️ 架構

```
Client
  │
  │  POST /bookings
  ▼
Redis SETNX(ticketID)
  │ 已被搶          │ 搶到
  ▼                 ▼
409 Conflict    預檢（票存在？錢夠？）
                    │
                    ▼
              INSERT booking (status=pending)
                    │
                    ▼
              Kafka WriteMessages
                    │
                    ▼
              202 Accepted ──────────────────────── Client 拿 ID 輪詢
                                                         │
                         (非同步)                        │  GET /bookings/{id}
                              │                          ▼
                    Kafka Consumer              status: pending → confirmed
                              │
                    ┌─────────┴──────────┐
                    │  BEGIN TRANSACTION  │
                    │  MarkSold(ticket)   │  WHERE status='available'
                    │  Withdraw(wallet)   │  WHERE balance >= price
                    │  status=confirmed   │
                    │  COMMIT             │
                    └────────────────────┘
```

**兩層防護**：
- **Redis SETNX**：快速閘門，阻擋 99% 的重複請求（毫秒級）
- **MySQL 原子 UPDATE**（`WHERE status='available'`）：最終正確性保證

**Kafka 序列化**：key = ticketID，相同票的請求進同一 partition，consumer 串行處理，確保不重複售出。

## 🛠️ 技術棧

- **語言**：Go 1.25
- **框架**：[simple-web-framework](https://github.com/xchwan/simple-web-framework)（自研輕量框架）
- **資料庫**：MySQL 8 + GORM
- **快取**：Redis 7
- **訊息佇列**：Apache Kafka（KRaft mode，無 Zookeeper）
- **搜尋**：Elasticsearch 8
- **Migration**：golang-migrate（embedded SQL）
- **容器**：Docker + Docker Compose

## 🚀 快速開始

### 前置需求

- Docker Desktop
- Make

### 啟動服務

```bash
# 建立 Docker image
make docker-build

# 啟動所有服務（MySQL, Redis, Kafka, Elasticsearch, App）
make up

# 查看 App log
make logs
```

App 啟動後監聽 `http://localhost:8080`。

API 文件：`http://localhost:8080/docs`

Kafka UI：`http://localhost:8081`

### 常用指令

```bash
make build       # 編譯
make test        # 跑 integration tests（需先 make up && make setup-test-db）
make staticcheck # 靜態分析
make format      # gofmt
make shell       # 進入開發容器互動 shell
```

> 所有 `make` 指令都在 Docker 容器內執行，不需要本機安裝 Go。

## 🧪 測試

```bash
# 建立測試用 DB（首次執行）
make setup-test-db

# 跑所有測試（83 個）
make test
```

### 測試類型

| 類型 | 數量 | 說明 |
|------|:----:|------|
| Integration | 74 | HTTP handler 層端對端測試 |
| Kafka 整合 | 4 | 真實 Kafka：async confirm、idempotency、Redis 閘門 |
| Elasticsearch 整合 | 5 | 真實 ES：keyword 搜尋、update re-index、delete remove |

**測試隔離**：
- MySQL：每個 test 前後 DELETE 清空
- Redis：每個 test 前後 FlushDB
- ES index：每個 test 前後刪除 + 自動重建
- Kafka topic：整套 suite 開始前 delete + recreate（各自用獨立 offset 隔離）

## 📁 專案結構

```
.
├── cmd/main/           # 進入點（main.go）
├── internal/
│   ├── booking/        # 訂票（service、handler、consumer、repo）
│   ├── event/          # 活動
│   ├── ticket/         # 票券
│   ├── user/           # 會員
│   ├── wallet/         # 錢包
│   └── infra/          # DB、Redis、Kafka、ES 連線
├── migrations/         # SQL migration 檔案（embedded）
├── test/               # Integration tests
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

## ⚙️ 環境變數

| 變數 | 預設值 | 說明 |
|------|--------|------|
| `DB_DSN` | — | MySQL DSN（必填） |
| `REDIS_ADDR` | `localhost:6379` | Redis 位址 |
| `KAFKA_BROKERS` | `localhost:9092` | Kafka broker 清單（逗號分隔） |
| `KAFKA_BOOKING_TOPIC` | `booking-requests` | Booking topic 名稱 |
| `ELASTIC_ADDR` | `http://localhost:9200` | Elasticsearch 位址 |
| `ES_EVENTS_INDEX` | `events` | ES index 名稱 |

> 測試環境自動使用 `-test` 後綴的 topic / index，與正式環境完全隔離。

## ⚡ 高流量設計重點

### 萬人搶票場景

1. **Redis SETNX**（TTL 30s）：同張票只允許一個請求進入 DB 流程，其餘立即返回 409
2. **Kafka 非同步**：前端拿到 202 Accepted，後台 consumer 序列處理，DB 不會被暴衝
3. **10 partitions**：Kafka topic 預設 10 個 partition，支援最多 10 個 consumer 並行消費不同票
4. **原子 UPDATE**：`WHERE status='available'` 作為最終防線，即使 Redis key 異常也不會重複售出

### 冪等性保護

Kafka consumer 在 commit offset 前 crash → 重啟後重複投遞同一 message。Consumer 首先檢查 booking 是否已是 `confirmed`，若是則直接跳過，不重複扣款。

## 📄 License

MIT
