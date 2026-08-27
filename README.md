# LightGo

LightGo 是一个只使用 Go 标准库实现的轻量 Web 框架与博客演示站，支持：

- 静态、路径参数和通配符路由，路由分组及 404/405；
- 全局、分组和路由级洋葱中间件；
- JSON、Form、Query、Path、Header 请求绑定和声明式校验；
- HTML 模板、JSON/XML/Text/Blob 渲染及带协商缓存的静态文件；
- 注册登录、Token 鉴权、文章 CRUD、筛选分页和统计页面；
- 分类目录与按分类筛选、登录用户评论与评论删除权限控制。

## 环境

- Go 1.22 或更高版本
- 无第三方依赖

## 启动

```bash
go run ./cmd/server
# 自定义端口
go run ./cmd/server -port 9090
# 或 PORT=9090 go run ./cmd/server
```

打开 `http://localhost:8080`。演示账号：`admin / secret123`、`alice / secret123`。

## 验证

```bash
make check       # gofmt 检查、go vet、go test -race、go build
make smoke       # 启动临时服务并检查页面和 API
make package     # 生成 dist/lightgo_<os>_<arch>.tar.gz
```

直接构建：

```bash
go build -o bin/lightgo-server ./cmd/server
```

二进制运行时默认从当前目录读取 `web/`，也可以用 `-web /path/to/web` 指定资源目录。

## API 摘要

- `POST /api/auth/register`、`POST /api/auth/login`
- `GET /api/blog/posts`、`GET /api/blog/posts/:id`
- `POST/PUT/PATCH/DELETE /api/blog/posts[/:id]`
- `GET /api/blog/categories`
- `GET/POST /api/blog/posts/:id/comments`、`DELETE /api/blog/posts/:id/comments/:commentID`
- `GET /api/users`、`GET /api/stats`

成功响应格式为 `{"code":0,"message":"ok","data":...}`。
