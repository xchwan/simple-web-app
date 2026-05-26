# ===== 專案變數 =====
BINARY_NAME=myapp
MAIN_PATH=./cmd/main/main.go
IMAGE_NAME=my-go-app-dev

# Docker 執行命令共用參數
DOCKER_RUN=docker run --rm -v $(PWD):/app -w /app $(IMAGE_NAME)
DOCKER_RUN_TTY=docker run --rm -it -v $(PWD):/app -w /app $(IMAGE_NAME)

# ===== 進入點 =====
all: staticcheck format test build

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

test:
	@echo "透過 Docker 執行測試..."
	$(DOCKER_RUN) go test ./test/... -v
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

# 執行資料庫 migration（需先 make up）
migrate:
	docker compose run --rm app ./migrate

# 停止並移除所有容器
down:
	docker compose down

# 停止並移除容器＋volumes（清除 DB 資料）
down-v:
	docker compose down -v

# 即時查看 app 的 log
logs:
	docker compose logs -f app

.PHONY: all build run test tidy clean staticcheck shell docker-build docker-clean format up down down-v logs migrate
