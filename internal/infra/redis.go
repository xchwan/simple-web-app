package infra

import (
	"os"

	"github.com/redis/go-redis/v9"
)

// ConnectRedis 從環境變數 REDIS_ADDR 建立 Redis 連線。
func ConnectRedis() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	return redis.NewClient(&redis.Options{
		Addr: addr,
	})
}
