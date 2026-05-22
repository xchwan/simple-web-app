# 使用官方 Go 1.25 鏡像
FROM golang:1.25-bookworm

# 設定工作目錄
WORKDIR /app

# 安裝基礎工具 (如 git, make)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    make \
    && rm -rf /var/lib/apt/lists/*

# --- 安裝程式碼檢查工具 ---

# 1. 安裝 staticcheck (專注於 Bug 偵測與效能優化)
RUN go install honnef.co/go/tools/cmd/staticcheck@latest

# 2. 安裝 go-delve (Go 的偵錯器，相當於 C++ 的 LLDB)
RUN go install github.com/go-delve/delve/cmd/dlv@latest

# 先複製 go.mod 和 go.sum 以利用 Docker Layer Cache
COPY go.mod go.sum* ./
RUN go mod download

# 複製其餘原始碼
COPY . .

# 預設執行指令 (可改為你的啟動指令)
CMD ["make", "run"]
