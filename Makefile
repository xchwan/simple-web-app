# ===== 專案變數 =====
BINARY_NAME=myapp
MAIN_PATH=./cmd/main/main.go
IMAGE_NAME=my-go-app-dev

# Docker 執行命令共用參數
FRAMEWORK_DIR=/Users/xch1/Documents/waterball/simple_web_framework
DOCKER_RUN=docker run --rm -v $(PWD):/app -v $(FRAMEWORK_DIR):$(FRAMEWORK_DIR) -w /app $(IMAGE_NAME)
DOCKER_RUN_TTY=docker run --rm -it -v $(PWD):/app -v $(FRAMEWORK_DIR):$(FRAMEWORK_DIR) -w /app $(IMAGE_NAME)

# ===== 進入點 =====
all: staticcheck format build

# ===== 開發環境管理 =====

# 建立 dev Docker 映像檔（供 make 指令使用）
docker-build:
	docker build -t $(IMAGE_NAME) --target dev .

# 刪除 Docker image
docker-clean:
	@echo "刪除 Docker image $(IMAGE_NAME)..."
	docker rmi -f $(IMAGE_NAME) || true

# 進入容器互動介面
shell:
	$(DOCKER_RUN_TTY) /bin/bash

# ===== 編譯與執行 =====

build:
	@echo "透過 Docker 編譯..."
	$(DOCKER_RUN) go build -o $(BINARY_NAME) $(MAIN_PATH)

run:
	@echo "透過 Docker 執行程式..."
	$(DOCKER_RUN) ./$(BINARY_NAME)

# 建立測試用 DB（appdb_test），首次使用時執行（migration 由測試自動處理）
setup-test-db:
	docker compose exec mysql mysql -uroot -proot_secret -e \
		"CREATE DATABASE IF NOT EXISTS appdb_test; \
		 GRANT ALL PRIVILEGES ON appdb_test.* TO 'app'@'%'; \
		 FLUSH PRIVILEGES;"

# 執行測試（需先 make up && make setup-test-db）
test:
	docker run --rm \
		--network simple-web-app_default \
		-v $(PWD):/app -v $(FRAMEWORK_DIR):$(FRAMEWORK_DIR) -w /app \
		-e DB_DSN="app:secret@tcp(mysql:3306)/appdb_test?parseTime=true" \
		-e REDIS_ADDR="redis:6379" \
		$(IMAGE_NAME) go test ./test/... -v
# ===== 檢查與測試 (Check & Testing) =====

staticcheck:
	@echo "透過 Docker 執行 staticcheck..."
	$(DOCKER_RUN) staticcheck ./...

tidy:
	@echo "透過 Docker 執行 go mod tidy..."
	$(DOCKER_RUN) go mod tidy

# ===== 格式化程式碼 =====
format:
	@echo "透過 Docker 執行 gofmt..."
	$(DOCKER_RUN) gofmt -w .

# ===== 清理 =====
clean:
	@echo "透過 Docker 清理編譯檔案..."
	$(DOCKER_RUN) go clean
	rm -f $(BINARY_NAME)

# ===== Docker Compose =====

# 啟動所有服務（背景執行，重新 build app）
up:
	docker compose up --build -d

# 停止並移除所有容器
down:
	docker compose down

# 停止並移除容器＋volumes（清除 DB 資料）
down-v:
	docker compose down -v

# 即時查看 app 的 log
logs:
	docker compose logs -f app

.PHONY: all build run test tidy clean staticcheck shell docker-build docker-clean format up down down-v logs setup-test-db
