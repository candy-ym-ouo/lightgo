# 官方 Go 镜像，自带完整 Go 工具链并支持 amd64/arm64。
FROM golang:1.22

WORKDIR /app

# 本项目无第三方依赖，因此不需要 go.sum 或额外依赖下载步骤。
COPY go.mod ./
RUN go mod download

# 复制完整项目源码、模板和脚本。
COPY . .

# 构建时验证项目可以完整编译。
RUN go build ./...

# 保留完整 Go 工具链，方便在容器内继续测试和修改。
CMD ["bash"]
