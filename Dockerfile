# ============================================================
# Stage 1: dev
# 供 make 指令使用（staticcheck、test、build、shell 等）
# ============================================================
FROM golang:1.25-bookworm AS dev

WORKDIR /app

# 安裝基礎工具
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    && rm -rf /var/lib/apt/lists/*

# 安裝 staticcheck
RUN go install honnef.co/go/tools/cmd/staticcheck@latest

# 安裝 go-delve（偵錯器）
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 先複製 go.mod / go.sum 以利用 Docker Layer Cache
COPY go.mod go.sum* ./
RUN go mod download

# 複製原始碼
COPY . .

CMD ["make", "run"]

# ============================================================
# Stage 2: builder
# 在 dev 環境中編譯出靜態二進位檔
# ============================================================
FROM dev AS builder

RUN go build -o /app/myapp ./cmd/main/main.go
RUN go build -o /app/migrate ./cmd/migrate/main.go

# ============================================================
# Stage 3: runtime
# 最終執行映像，只含二進位檔，體積最小
# ============================================================
FROM debian:bookworm-slim AS runtime

WORKDIR /app

# 安裝 ca-certificates（發送 HTTPS 請求時需要）
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/myapp .
COPY --from=builder /app/migrate .

EXPOSE 8080

CMD ["./myapp"]
