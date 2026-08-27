基于 Go 实现的轻量 Web 框架与博客演示站项目，一款后端 Web 服务，提供路由、中间件、请求绑定、模板渲染、用户认证、文章管理、分类筛选与评论功能。

# LightGo 项目说明

## 项目简介

LightGo 是一个只使用 Go 标准库实现的轻量 Web 框架与博客演示站，包含可复用的 Web 框架能力和完整的博客示例业务。

## 环境要求

- Go 1.22 或更高版本
- 无第三方 Go 依赖
- Docker（可选，用于标准化镜像构建）

## 编译

```bash
go build ./...
```

## 启动

```bash
go run ./cmd/server
```

默认监听 `http://localhost:8080`，也可以指定端口：

```bash
go run ./cmd/server -port 9090
# 或
PORT=9090 go run ./cmd/server
```

二进制运行时默认从当前目录读取 `web/`，也可以通过 `-web` 指定资源目录：

```bash
go build -trimpath -o bin/lightgo-server ./cmd/server
./bin/lightgo-server -port 8080 -web ./web
```

演示账号：

- `admin / secret123`
- `alice / secret123`

## 测试与检查

```bash
go vet ./...
go test ./...
go test -race ./...
make check
make smoke
```

## 本地打包

```bash
make package
```

打包产物位于：

```text
dist/lightgo_<os>_<arch>.tar.gz
```

## Docker 标准打包

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh lightgo linux/amd64
./build_benzhi_docker.sh lightgo linux/arm64
docker run -it lightgo:latest
```

进入容器后可以执行：

```bash
go build ./...
go test ./...
```
