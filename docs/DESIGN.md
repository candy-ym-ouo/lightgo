# LightGo 轻量 Web 框架 —— 项目设计文档

> 文档版本：v1.1（分类与评论模块扩展后）
> 状态：已实现并通过验收（2026-08-27）
> 需求来源（项目提示词）：
>
> 轻量 Web 框架：用 Go 实现 Web 框架，支持路由与参数匹配、中间件链、请求绑定校验、静态文件与模板渲染。

---

## 目录

1. [项目概述](#1-项目概述)
2. [功能设计与业务逻辑（扩写）](#2-功能设计与业务逻辑扩写)
3. [架构设计](#3-架构设计)
4. [目录结构与代码规模预算](#4-目录结构与代码规模预算)
5. [前端界面设计](#5-前端界面设计)
6. [API 设计](#6-api-设计)
7. [测试计划](#7-测试计划)
8. [开发里程碑与验收标准](#8-开发里程碑与验收标准)
9. [风险与对策](#9-风险与对策)

---

## 1. 项目概述

### 1.1 项目定位

**LightGo** 是一个用 Go 标准库（零第三方依赖）实现的轻量 Web 框架，聚焦框架四大核心能力：

1. **路由与参数匹配**：方法路由、路径参数、通配符、查询参数、路由分组；
2. **中间件链**：洋葱模型、全局 / 分组 / 路由三级中间件、短路终止；
3. **请求绑定校验**：JSON / Form / Query / Path / Header 多来源绑定 + 声明式校验规则；
4. **静态文件与模板渲染**：带缓存与 304 协商的静态文件服务、支持布局与自定义函数的模板引擎。

在框架之上，内置一个**博客演示站**（文章、用户、登录鉴权、统计），用于完整展示框架能力，同时搭配一套**简单的前端界面**（原生 HTML/CSS/JS，无构建步骤，由框架的静态服务与模板渲染承载）。

### 1.2 技术选型与硬性约束

| 项目 | 说明 |
| --- | --- |
| 语言 / 版本 | Go 1.22+ |
| 依赖 | **仅 Go 标准库**（net/http、html/template、encoding/json、reflect、crypto 等），不引入任何第三方库 |
| 数据存储 | 内存存储（sync.RWMutex + map），进程内演示级别 |
| 前端 | 原生 HTML5 + CSS3 + JavaScript（ES6），服务端模板渲染首屏 + fetch 增强交互 |
| 服务端口 | 默认 `:8080`，支持 `-port` 标志与 `PORT` 环境变量覆盖 |
| 二进制名 | `lightgo-server`（`cmd/server`） |

### 1.3 非目标（明确不做）

- 不做 HTTPS / HTTP/2 专属特性、集群与水平扩展、磁盘持久化；
- 不做数据库访问层、用户密码明文存储（演示用哈希即可）；
- 不做生产级并发压测优化（演示规模，反射绑定性能可接受）；
- 前端不做构建工具链、框架（Vue/React）、打包压缩。

---

## 2. 功能设计与业务逻辑（扩写）

### 2.1 框架核心功能

#### 2.1.1 路由与参数匹配（router / tree）

- **方法路由**：支持 `GET / POST / PUT / DELETE / PATCH / HEAD / OPTIONS` 以及 `ANY`（任意方法）注册。
- **路径模式**三类节点，按优先级匹配：
  - 静态段：`/api/users`；
  - 参数段：`/api/users/:id`，`c.Param("id")` 取值；
  - 通配段：`/static/*path`，`c.Param("path")` 取剩余路径（含子路径）。
- **匹配优先级**：静态 > 参数 > 通配；同段内注册冲突（如 `/a/:x` 与 `/a/b` 并存、重复参数名）在**注册时报错**，避免运行时歧义。
- **查询参数**：`c.Query("key")` / `c.QueryDefault("key", "default")`。
- **路由分组（Group）**：`g := engine.Group("/api", authMW)` 支持前缀分组与**分组级中间件**，可嵌套。
- **404 / 405**：未匹配路径返回 404；路径匹配但方法不匹配返回 405 并附带 `Allow` 响应头。
- **路由表打印**：启动时输出全部注册路由，便于排查。
- **实现要点**：基于**基数树（Radix Tree）**按 `/` 分段组织节点，压缩公共前缀，匹配复杂度与路由数量弱相关。

#### 2.1.2 中间件链（middleware）

- 中间件签名：`type Middleware func(c *Context, next NextFunc)`，`NextFunc` 为后续处理（洋葱模型）。
- **三级注册**：`Engine.Use`（全局）、`Group.Use`（分组）、`Route.Use`（单路由），顺序执行、后注册在外层。
- **短路控制**：中间件不调用 `next` 即终止链路（如鉴权失败直接 401）。
- **内置中间件**（`middlewares` 包，全部可选装配）：

| 中间件 | 职责 | 关键行为 |
| --- | --- | --- |
| `RequestID` | 请求追踪 | 无 `X-Request-ID` 则生成 UUID，注入响应头与 `c` 上下文 |
| `Logger` | 访问日志 | 记录方法、路径、状态码、耗时、请求 ID，输出到 stdout |
| `Recovery` | 崩溃兜底 | `recover` 捕获 panic，打印堆栈，统一返回 500 并记录请求 ID |
| `CORS` | 跨域 | 可配置 `AllowOrigin` / `AllowMethods` / `AllowHeaders`，处理 OPTIONS 预检 |
| `Gzip` | 响应压缩 | 按 Accept-Encoding 协商，压缩 JSON/HTML 等文本响应，设置 `Vary` 头 |

#### 2.1.3 请求绑定与校验（binding）

- **多来源绑定**：依据 `Content-Type` 自动选择 JSON / Form（`x-www-form-urlencoded`、`multipart/form-data`），另提供显式 `BindJSON` / `BindForm` / `BindQuery` / `BindParam`（路径参数）/ `BindHeader`。
- **结构体 tag 约定**：
  - 字段来源：`json` / `form` / `query` / `param` / `header`；
  - 默认值：`default:"x"`；
  - 校验规则：`validate:"required|min=3|max=50|email|numeric|len=11|oneof=draft published|regexp=^[a-z]+$"`。
- **校验规则集**：`required`、`min`、`max`、`len`、`email`、`numeric`、`alpha`、`oneof`、`regexp`、`datetime`（按需实现前 9 项即可）。
- **错误处理**：绑定失败返回 **400**；校验失败返回 **422**，响应体携带字段级错误数组：
  ```json
  { "code": 422, "message": "参数校验失败", "data": { "errors": [ { "field": "Title", "rule": "required" } ] } }
  ```
- **实现要点**：基于 `reflect` 遍历结构体字段，解析 tag、按来源取值、类型转换、执行校验规则；嵌套结构体递归处理。

#### 2.1.4 静态文件与模板渲染（static / render）

- **静态文件服务** `static.FileServer(dir, opts)`：
  - 路径清洗防穿越（清理 `..` 后再校验前缀）；
  - 自动识别 MIME 类型，设置 `Content-Type`；
  - 缓存协商：`ETag` + `Last-Modified`，支持 `If-Modified-Since` / `If-None-Match` → **304**；
  - `Cache-Control` 可配置（默认 `public, max-age=3600`）；目录请求回退 `index.html`。
- **模板引擎** `render.TemplateEngine`：
  - `Load(dir)` 递归加载 `web/templates/*.html`；
  - **布局机制**：`layout.html` 定义骨架与 `{{block "content" .}}`，页面模板 `{{define "content"}}...{{end}}`；
  - **自定义模板函数**：`date`（时间格式化）、`truncate`（摘要截断）、`upper`/`lower`、`add`/`sub`、`default`、`safeHTML`；
  - 渲染失败时返回 500 并在日志中暴露具体模板名与错误。
- **响应渲染器**（`render` 包）：`JSON` / `XML` / `Text` / `HTML`（模板）/ `Redirect` / `Blob`（文件下载）/ `Status`（纯状态码），统一设置 Content-Type 与状态码。

#### 2.1.5 服务器生命周期与错误处理（server / errors）

- **生命周期**：`engine.Run(addr)` 内部构造 `http.Server`，配置 `ReadTimeout` / `WriteTimeout` / `IdleTimeout`；监听 `SIGINT` / `SIGTERM` 后执行 **优雅关闭**（`Shutdown` + 超时兜底 `Close`）。
- **错误模型**：`HTTPError{ Code int, Message string, Err error }`；`c.Error(err)` 统一流转到**全局错误处理器**（可 `SetErrorHandler` 覆盖）；API 响应统一包装为 `{code, message, data}`，业务成功 `code=0`。
- **panic 兜底**：任何未捕获 panic 由 `Recovery` 中间件转 500，页面渲染 500 模板，API 返回统一错误 JSON。

### 2.2 演示业务：LightGo 博客站

#### 2.2.1 业务对象

| 对象 | 字段 | 说明 |
| --- | --- | --- |
| User | ID、Username（唯一）、PasswordHash、Role（`admin`/`author`）、CreatedAt | 注册用户 |
| Post | ID、Title、Summary、Content、AuthorID、Category、Tags[]、Status（`draft`/`published`）、Views、CommentCount、CreatedAt、UpdatedAt | 文章 |
| Category | Name、PostCount | 从已发布文章实时聚合的分类统计 |
| Comment | ID、PostID、AuthorID、Author、Content、CreatedAt | 文章评论 |
| Token | Token、UserID、ExpiresAt | 登录凭证（内存） |

#### 2.2.2 业务规则

1. **注册**：用户名唯一（冲突返回 409）；密码存储为 `SHA-256(盐 + 明文)`（演示级哈希，仅标准库）。
2. **登录**：校验用户名密码，签发 HMAC-SHA256 签名 token（含用户 ID 与过期时间），有效期 24 小时。
3. **鉴权中间件**：`Auth` 解析 `Authorization: Bearer <token>`，失败 401；成功将用户注入 `c.Set("user", u)`。
4. **文章列表**：分页（`page`/`pageSize`，默认 1/10）、按 `keyword` 对标题+摘要模糊搜索、按 `category` 与 `status` 过滤、按发布时间或浏览量排序。
5. **文章详情**：访问即浏览量 +1（防刷不做，演示级）；草稿仅作者本人可见。
6. **分类目录**：从已发布文章实时聚合分类与文章数，点击分类复用既有文章筛选逻辑，不额外维护冗余分类数据。
7. **评论互动**：登录用户可对已发布文章发表评论；评论作者或管理员可删除评论，删除文章时同步清理其评论。
8. **新建 / 编辑**：需登录；绑定校验：标题必填 3–100 字、摘要必填、正文必填、分类必填、状态 `oneof=draft published`；仅**作者本人或 admin** 可编辑/删除（403）。
9. **统计概览**：聚合文章总数、用户总数、总浏览量、最近 5 篇文章。

#### 2.2.3 存储层

- `store.Store`：内部四类数据（users / posts / comments / tokens），`sync.RWMutex` 保证并发安全；
- ID 自增（原子计数）；分页、排序、过滤在存储层实现（演示规模，遍历过滤即可）；
- 提供 `Reset()` 与种子数据（预置 2 个用户、8 篇文章）便于演示与测试。

---

## 3. 架构设计

### 3.1 分层架构

```
┌──────────────────────────────────────────────────────────┐
│                    客户端：浏览器 / curl                     │
└───────────────────────────┬──────────────────────────────┘
                            │ HTTP
┌───────────────────────────▼──────────────────────────────┐
│             http.Server（标准库，含超时与优雅关闭）            │
│  ┌────────────────────────────────────────────────────┐  │
│  │               Engine（框架核心）                      │  │
│  │  ┌─────────┐  ┌──────────┐  ┌───────────────────┐  │  │
│  │  │ 中间件链 │→│ 路由器(Radix) │→│ Handler 执行区      │  │  │
│  │  └─────────┘  └──────────┘  └────────┬──────────┘  │  │
│  │    Context（请求/响应门面）             │              │  │
│  │    ├─ binding 绑定校验  ──────────────┤              │  │
│  │    ├─ render  渲染(JSON/模板/静态) ────┤              │  │
│  │    └─ errors 统一错误处理 ─────────────┘              │  │
│  └────────────────────────────────────────────────────┘  │
└───────────────────────────┬──────────────────────────────┘
                            │ 调用
┌───────────────────────────▼──────────────────────────────┐
│              演示业务层（internal/）                       │
│   api（HTTP 处理器）→ store（内存存储）→ model（模型/DTO）   │
└──────────────────────────────────────────────────────────┘
```

### 3.2 请求生命周期

```
1. 请求进入 http.Server → Engine.ServeHTTP
2. 创建 Context（含 ResponseWriter、Request、空 Params）
3. 执行中间件链（洋葱模型：RequestID → Logger → Recovery → CORS → Auth(分组)）
4. 路由器匹配（方法 + 路径）：
   - 未匹配路径        → 404
   - 路径匹配方法不符  → 405 + Allow 头
   - 命中             → 解析出路径参数，注入 Context
5. 执行业务 Handler：
   a. 绑定校验（c.Bind → 400/422）
   b. 业务逻辑（调用 store）
   c. 渲染响应（JSON / 模板 / 重定向）
6. 中间件链收尾（响应后处理：访问日志、gzip 压缩、请求 ID 落盘）
7. 任意 panic 被 Recovery 捕获 → 统一 500
8. 响应返回客户端
```

### 3.3 关键流程说明

- **路由匹配**：路径按 `/` 分段，从根节点逐段下钻；优先静态子节点，其次参数节点，最后通配节点；参数段在匹配时写入 `params` 切片，`c.Param(name)` 按注册名取值。
- **中间件链**：注册顺序即外层顺序；`next` 调用即进入下一层，返回后继续本层后续逻辑；短路 = 不调用 `next`。
- **绑定校验**：`c.Bind` 先按 Content-Type 选绑定器，绑定成功后自动调用校验器；校验错误聚合成字段级列表。
- **模板渲染**：`TemplateEngine.Render(w, name, data)` 以 layout 为骨架、页面模板为内容块执行 `ExecuteTemplate`；模板函数表全局注册。
- **静态文件**：URL 清洗 → 前缀校验 → 磁盘映射 → 文件信息（大小/修改时间）→ 协商缓存（304）→ 流式输出。

---

## 4. 目录结构与代码规模预算

### 4.1 目录结构

```
LightGo/                        # 项目根（模块名：lightgo）
├── go.mod
├── README.md                   # 快速开始（实现阶段补充）
├── docs/
│   └── DESIGN.md               # 本文档
├── lightgo/                    # 框架核心包
│   ├── lightgo.go              # Engine、Group、选项、默认实例、Use/Group 入口
│   ├── context.go              # Context：参数/查询/渲染/绑定/响应头/错误
│   ├── router.go               # 路由器：注册、匹配、分组、404/405、路由表打印
│   ├── tree.go                 # 基数树：插入、搜索、冲突检测
│   ├── middleware.go           # 中间件类型与链执行
│   ├── errors.go               # HTTPError 与全局错误处理器
│   ├── server.go               # Run/优雅关闭/http.Server 装配
│   ├── binding/
│   │   ├── binding.go          # 绑定器注册与反射取值（JSON/Form/Query/Param/Header）
│   │   └── validate.go         # 校验规则引擎与错误收集
│   ├── render/
│   │   ├── render.go           # 渲染器：JSON/XML/Text/Redirect/Blob/Status
│   │   └── template.go         # 模板引擎：加载、布局、自定义函数
│   ├── static/
│   │   └── static.go           # 静态文件服务（缓存协商、防穿越）
│   └── middlewares/
│       ├── logger.go           # 访问日志
│       ├── recovery.go         # panic 恢复
│       ├── cors.go             # 跨域
│       ├── requestid.go        # 请求 ID
│       └── gzip.go             # gzip 压缩
├── cmd/
│   └── server/
│       ├── main.go             # 入口：解析参数、装配引擎、启动/优雅退出
│       └── routes.go           # 全部路由与中间件注册（框架演示 + 业务）
├── internal/
│   ├── api/
│   │   ├── blog.go             # 文章处理器（列表/详情/新建/编辑/删除）
│   │   ├── community.go        # 分类与评论处理器
│   │   └── user.go             # 用户处理器（注册/登录/列表/详情/统计）
│   ├── model/
│   │   └── model.go            # User/Post/Token 模型与请求/响应 DTO
│   └── store/
│       ├── community.go        # 分类聚合、评论 CRUD 与权限
│       └── store.go            # 内存存储（并发安全、种子数据）
├── web/
│   ├── templates/
│   │   ├── layout.html         # 布局骨架
│   │   ├── index.html          # 首页
│   │   ├── blog_list.html      # 文章列表
│   │   ├── category_list.html  # 分类目录
│   │   ├── blog_detail.html    # 文章详情（含评论区）
│   │   ├── blog_form.html      # 新建/编辑共用表单
│   │   ├── user_list.html      # 用户列表
│   │   ├── login.html          # 登录/注册
│   │   └── error.html          # 404/500 通用错误页
│   └── static/
│       ├── css/style.css       # 全局样式
│       └── js/app.js           # fetch 封装与表单交互
└── （各包 *_test.go 测试文件，不计入规模）
```

### 4.2 扩展后代码规模统计

分类与评论模块已在原有博客业务上完成增量扩展。以下为 **2026-08-27** 的实际统计结果，不再沿用设计阶段的行数上限。

| 指标 | 实测结果 |
| --- | --- |
| Go 代码总行数（不含测试） | **2430 行** |
| 非测试 Go 源文件数 | **25 个** |

**计数约定**：

- 行数 = `wc -l` 物理行数（含空行与注释），统计所有 `*.go` 文件，**排除 `*_test.go`**；
- 文件数 = 非测试 `*.go` 文件数量；`web/` 下 HTML/CSS/JS 与 `go.mod`、`docs/` 均不计入。

**本次扩展涉及的 Go 文件**：

| 文件 | 行数 | 主要职责 |
| --- | ---: | --- |
| `internal/api/community.go` | 91 | 分类列表、评论查询/创建/删除 HTTP 处理器 |
| `internal/store/community.go` | 84 | 已发布文章分类聚合、评论 CRUD、评论删除权限 |
| `internal/model/model.go` | 84 | Category、Comment、CommentRequest 与 Post.CommentCount |
| `internal/store/store.go` | 303 | 评论内存仓库、种子数据、文章删除级联清理、评论数统计 |
| `cmd/server/routes.go` | 120 | 分类页面/API、评论 API 路由注册 |

### 4.3 规模校验方法

完成代码变更后执行以下命令复核：

```bash
# 总行数（排除测试文件）
find . -name '*.go' ! -name '*_test.go' -type f -print0 | xargs -0 wc -l | tail -1

# 非测试 Go 文件数
find . -name '*.go' ! -name '*_test.go' -type f | wc -l
```

---

## 5. 前端界面设计

### 5.1 设计原则

- **零构建**：纯原生 HTML/CSS/JS，由框架的静态服务（`/static/*`）与模板渲染（`/templates`）直接承载；
- **服务端渲染首屏 + 客户端增强**：页面骨架与首屏数据由模板渲染，交互（表单提交、删除、登录态）由 `app.js` 用 `fetch` 调用 REST API 完成；
- **简洁卡片式 UI**：单一 `style.css`，浅色主题、响应式栅格（文章卡片 / 表格 / 表单）。

### 5.2 页面清单

| 路由 | 模板 | 说明 |
| --- | --- | --- |
| `/` | index.html | 首页：框架能力介绍 + 站点概览统计卡片（文章/用户/浏览量/最新文章） |
| `/blog` | blog_list.html | 文章列表：分页、关键词搜索、分类筛选 |
| `/blog/:id` | blog_detail.html | 文章详情：正文、作者、标签、浏览量；作者可编辑/删除 |
| `/blog/new` | blog_form.html | 新建文章表单 |
| `/blog/:id/edit` | blog_form.html | 编辑文章表单（回填数据） |
| `/users` | user_list.html | 用户列表（表格：用户名/角色/注册时间） |
| `/categories` | category_list.html | 分类目录：展示已发布文章分类和文章数，点击后跳转到对应筛选结果 |
| `/login` | login.html | 登录 / 注册双 Tab 表单 |
| 任意未匹配 | error.html | 404/500 通用错误页（区分状态码文案） |

### 5.3 模板与静态资源

- **模板**：`layout.html` 提供导航栏（首页/文章/用户/登录）、页脚与内容块；页面模板定义 `content` 块；模板函数用于 `date` 格式化、`truncate` 摘要、`default` 空值兜底。
- **静态资源**：
  - `css/style.css`：变量驱动的配色、卡片、表格、表单、分页、标签、提示条样式；
  - `js/app.js`：`api(path, opts)` fetch 封装（自动附带 `Authorization` token、统一错误提示）、登录/注册提交、文章表单提交（新建/编辑）、删除确认、列表搜索防抖。

### 5.4 交互流程

1. **登录**：`/login` 提交 → `POST /api/auth/login` → 成功后将 token 存入 `localStorage`，跳转首页；导航栏显示当前用户名与退出按钮。
2. **文章 CRUD**：列表页「新建」跳转表单页；表单校验提示字段错误（来自 422 响应）；编辑页回填；详情页作者可见「编辑/删除」按钮。
3. **统计刷新**：首页加载时调用 `GET /api/stats` 渲染统计卡片。

---

## 6. API 设计

### 6.1 REST API 一览

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/health` | 无 | 健康检查 |
| POST | `/api/auth/register` | 无 | 注册（用户名/密码） |
| POST | `/api/auth/login` | 无 | 登录，返回 token |
| GET | `/api/users` | 无 | 用户列表（分页） |
| GET | `/api/users/:id` | 无 | 用户详情 |
| GET | `/api/blog/posts` | 无 | 文章列表：`page`、`pageSize`、`keyword`、`category`、`status`、`sort` |
| GET | `/api/blog/posts/:id` | 无 | 文章详情（浏览量 +1） |
| POST | `/api/blog/posts` | Bearer | 新建文章 |
| PUT | `/api/blog/posts/:id` | Bearer | 更新文章（作者/admin） |
| DELETE | `/api/blog/posts/:id` | Bearer | 删除文章（作者/admin） |
| GET | `/api/blog/categories` | 无 | 已发布文章分类及数量 |
| GET | `/api/blog/posts/:id/comments` | 无 | 文章评论列表 |
| POST | `/api/blog/posts/:id/comments` | Bearer | 发布评论（仅已发布文章） |
| DELETE | `/api/blog/posts/:id/comments/:commentID` | Bearer | 删除本人评论或管理员删除 |
| GET | `/api/stats` | 无 | 站点概览 |

### 6.2 统一响应格式

- API 一律返回：`{ "code": 0, "message": "ok", "data": {...} }`，`code=0` 成功，非 0 为业务/HTTP 错误码。
- 页面路由（模板渲染）不走统一包装，直接传数据。

**示例：登录**

```
POST /api/auth/login
Content-Type: application/json
{"username": "alice", "password": "secret123"}

→ 200
{ "code": 0, "message": "ok", "data": { "token": "eyJ...", "user": { "id": 1, "username": "alice", "role": "author" } } }

→ 401（密码错误）
{ "code": 401, "message": "用户名或密码错误", "data": null }
```

**示例：新建文章（校验失败）**

```
POST /api/blog/posts
Authorization: Bearer <token>
{"title": "", "content": "x", "category": ""}

→ 422
{ "code": 422, "message": "参数校验失败",
  "data": { "errors": [ { "field": "Title", "rule": "required" },
                        { "field": "Category", "rule": "required" } ] } }
```

### 6.3 错误码约定

| 状态码 | 业务 code | 场景 |
| --- | --- | --- |
| 400 | 400 | 请求体解析/绑定失败、参数非法 |
| 401 | 401 | 未登录、token 缺失/失效/过期 |
| 403 | 403 | 无权限（非作者/非 admin 编辑他人文章） |
| 404 | 404 | 资源不存在、路径未匹配 |
| 405 | 405 | 路径匹配但方法不允许（附 Allow 头） |
| 409 | 409 | 用户名已存在 |
| 422 | 422 | 校验规则不通过（附字段级 errors） |
| 500 | 500 | 服务器内部错误 / panic 兜底 |

---

## 7. 测试计划

> 测试代码不计入行数约束。目标：核心能力全覆盖、业务关键路径覆盖。

| 测试文件 | 覆盖点 |
| --- | --- |
| `lightgo/tree_test.go` | 基数树插入/搜索/优先级/冲突报错/通配符 |
| `lightgo/router_test.go` | 路由注册、参数匹配、404/405、分组、ANY |
| `lightgo/context_test.go` | JSON/文本渲染、查询参数、路径参数、错误流转 |
| `lightgo/middleware_test.go` | 链顺序、短路、嵌套分组中间件 |
| `lightgo/binding/binding_test.go` | JSON/Form/Query/Param/Header 绑定、默认值、类型转换失败 |
| `lightgo/binding/validate_test.go` | 各规则（required/min/max/email/oneof/regexp…）与错误收集 |
| `lightgo/render/template_test.go` | 模板加载、布局、自定义函数、缺失模板报错 |
| `lightgo/static/static_test.go` | MIME、304 协商、防穿越、index.html |
| `lightgo/middlewares/middlewares_test.go` | gzip 压缩、recovery 兜底、requestid 注入、CORS 头 |
| `internal/store/store_test.go` | 并发安全（-race）、文章 CRUD/分页过滤、分类聚合、评论创建与权限 |
| `cmd/server/routes_test.go` | httptest 集成：注册/登录/鉴权/文章 CRUD/权限/统计、分类与评论 API |
| `scripts/smoke.sh` | 冒烟脚本：启动服务，curl 关键接口、分类页面与评论接口断言 |

运行方式：`go test ./... -race` + `go vet ./...`。

---

## 8. 开发里程碑与验收标准

### 8.1 里程碑

| 阶段 | 内容 | 累计行数目标 |
| --- | --- | --- |
| M1 | 工程初始化（go.mod）+ 框架核心：Engine / Context / 路由树 / 中间件链 | ~715 |
| M2 | 绑定校验 + 渲染 / 模板 / 静态文件 | ~1205 |
| M3 | 内置中间件 ×5 + 服务器生命周期 + 错误处理 | ~1595 |
| M4 | 演示业务（model / store / api / routes / main） | ~2075 |
| M5 | 前端页面（templates + css + js，不计行数） | — |
| M6 | 测试补齐、规模复核微调、冒烟验证 | — |

### 8.2 验收标准

1. `go build ./...`、`go vet ./...`、`go test ./... -race` 全部通过，仅使用标准库；
2. 非测试 Go 文件数 = **25**；
3. 非测试 Go 代码行数 = **2430**；
4. 框架四能力（路由参数、中间件链、绑定校验、静态+模板）均有测试与冒烟验证；
5. `lightgo-server` 启动后：文章 CRUD、分类目录/筛选、评论发布/删除全流程（含登录鉴权、校验失败提示、404/500 页）可用。

---

## 9. 风险与对策

| 风险 | 影响 | 对策 |
| --- | --- | --- |
| 分类数据重复维护 | 分类数与文章状态不一致 | 分类仅从已发布文章实时聚合，不新增独立分类持久化状态 |
| 删除文章后遗留评论 | 出现无法访问的悬挂评论 | `DeletePost` 同步清理该文章下所有评论，并由测试覆盖 |
| 评论越权删除 | 破坏评论归属 | 存储层校验评论作者或管理员角色，API 使用鉴权中间件 |
| 反射绑定实现复杂度高 | 拖期 | 先实现 JSON/Query/Form 三类来源 + 核心 9 条规则，Header/Param 绑定后置 |
| 模板与静态目录路径错误 | 运行时报错 | 启动时校验目录存在并输出加载清单；缺失时给出明确错误并退出 |
| 并发数据竞争 | 数据错乱 | store 全量 RWMutex 保护，测试开启 `-race` 验证 |

---

*本文档记录当前已实现版本；后续增量模块变更后，应同步回写实际规模、路由与测试覆盖。*
