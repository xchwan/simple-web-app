module github.com/xchwan/simple-web-app

go 1.25.10

require (
	github.com/elastic/go-elasticsearch/v8 v8.19.6
	github.com/go-sql-driver/mysql v1.8.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/redis/go-redis/v9 v9.19.0
	github.com/segmentio/kafka-go v0.4.51
	github.com/xchwan/simple-web-framework v0.0.0-20260528054552-dbcf9f0d7878 // replaced by local
	gorm.io/driver/mysql v1.6.0
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.9.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/pierrec/lz4/v4 v4.1.16 // indirect
	github.com/shamaton/msgpack/v2 v2.4.1 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/xchwan/simple-web-framework => /Users/xch1/Documents/waterball/simple_web_framework
