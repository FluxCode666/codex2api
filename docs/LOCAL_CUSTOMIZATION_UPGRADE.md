# 本地定制与增量升级指南

本文档用于记录这份本地部署相对上游 `codex2api` 的**定制功能**，并提供后续从 GitHub 增量升级时的保留方案。

适用时间基线：**2026-04-25**（最后更新：**2026-04-25**，新增防封技术移植）

---

## 目标

这份本地版本有以下几类关键定制：

1. **大文件账号导入增强**
2. **按账号套餐类型分池的 API Key 路由**
3. **防封技术移植（Node.js TLS 指纹 + Header Wire Casing）**

后续无论你是：

- 直接从 GitHub 拉取上游更新
- 重新下载新版本源码
- 切换到新的干净仓库再迁移本地改动

都应该以本文档为“保留清单”。

---

## 当前部署画像

当前这台机器的实际运行画像是：

- 部署模式：`docker-compose.sqlite.yml`
- 数据库：`SQLite`
- 缓存：`memory`
- 容器名：`codex2api-sqlite`

这意味着：

- 当前实例**没有启用 Redis**
- 运行风险更偏向于**内存、goroutine、SQLite 文件大小、导入峰值负载**
- 升级后要优先验证导入能力、账号调度、API Key 池路由，而不是 Redis 兼容性

---

## 本地定制概览

### 1. 大文件导入增强

目标是解决以下问题：

- `32m` 上传限制不够
- `失败：请上传文件（字段名：file）`
- `multipart: message too large`
- 一次导入 6000+ 账号时刷新任务过猛

当前行为如下：

- 单文件导入上限提升到 **256MB**
- `multipart/form-data` 不再强制要求字段名必须是 `file`
- 只要 multipart 中带文件 part，就会被当作导入文件处理
- 非 multipart 请求也支持直接用 **raw body** 上传
- 导入后的账号刷新不再为每个账号直接起无限 goroutine
- 当前改为 **8 个 worker** 串行消费刷新队列，避免导入后瞬间把进程打爆

### 2. 按套餐类型分池的 API Key

目标是实现：

- `free` 一套 API Key
- `team` 一套 API Key
- `plus` 一套 API Key
- `pro` 一套 API Key

当前行为如下：

- `api_keys` 表新增字段：`pool_plan_type`
- 支持值：`"" | all | free | team | plus | pro`
- 其中 `""` 和 `all` 表示不限池
- 代理层会在选账号时把“模型限制”和“API Key 池限制”一起叠加
- 以下入口都已经纳入池过滤：
  - `/v1/responses`
  - `/v1/responses/compact`
  - `/v1/chat/completions`
  - `/v1/messages`
  - `/v1/images/generations`
  - `/v1/images/edits`
- 老的 `allowed_api_key_ids` 机制**没有删除**，仍然可用于少量特例精细绑定

### 3. API Key 热刷新体验优化

- 动态 API Key 缓存 TTL 调整为 **10 秒**
- 如果请求命中了未知 key，会强制再刷新一次数据库缓存

### 3. API Key 热刷新体验优化

为避免"刚创建 API Key 后要等几分钟才生效"：

- 动态 API Key 缓存 TTL 调整为 **10 秒**
- 如果请求命中了未知 key，会强制再刷新一次数据库缓存

这个优化很重要，升级时不要漏掉。

### 4. 防封技术移植（来自 FluxCode-main，2026-04-25）

**背景**：上游 codex2api 的 `utls_transport.go` 使用 `HelloChrome_Auto` 模拟 Chrome 浏览器的 TLS 指纹。FluxCode 项目通过对真实 Codex CLI（`codex_cli_rs`，基于 Node.js）流量抓包，获得了更精确的 Node.js 24.x TLS 指纹，防封效果更好。本次将其移植到本地版本。

**移植内容 A：Node.js 24.x TLS 指纹**

原来：`utls.HelloChrome_Auto`（Chrome 浏览器指纹）

现在：精确的 Node.js 24.x ClientHello 规格，包括：
- **JA3 Hash**: `44f88fca027f27bab4bb08d4af15f23e`
- **JA4**: `t13d1714h1_5b57614c22b0_7baf387fc6ff`
- 17 个 cipher suites（顺序精确）
- 3 个 supported groups（X25519、P256、P384）
- 9 个 signature algorithms
- 14 个 TLS 扩展（含 GREASE ECH，顺序精确）
- 使用 `HelloCustom` + `ApplyPreset` 方式应用

**移植内容 B：Header Wire Casing 基础设施**

新增 `codexWireHeaders` map，记录 Codex CLI 上游请求头的精确大小写（基于真实抓包）。新增 `setWireHeader()` 工具函数，可绕过 Go 的 canonical 规范化直接写入指定 key。

> 注：HTTP/2 协议本身要求 header 名称小写，`golang.org/x/net/http2` 会自动处理，所以 wire casing 在当前 HTTP/2 路径下由协议层保证。`setWireHeader` 主要为未来 HTTP/1.1 场景或精细调试保留。

---

## 关键实现文件

后续升级时，以下文件是**最需要重点保留和人工比对**的文件。

### 导入增强相关

- `admin/handler.go`
  - `maxImportFileSizeBytes = 256 * 1024 * 1024`
  - `parseImportRequest`
  - `parseMultipartImportRequest`
  - `enqueueImportedAccountRefresh`
  - `importRefreshWorkers = 8`

### API Key 分池相关

- `database/postgres.go`
  - `api_keys.pool_plan_type` 列
  - `APIKeyRow.PoolPlanType`
  - `ListAPIKeys`
  - `InsertAPIKey`
- `database/sqlite.go`
  - `api_keys` 建表字段
  - `ensureSQLiteColumn` 补列逻辑
- `admin/handler.go`
  - `createKeyReq.PoolPlanType`
  - `normalizeAPIKeyPoolPlanType`
  - `CreateAPIKey`
- `admin/responses.go`
  - `MaskedAPIKeyRow.PoolPlanType`
  - `createAPIKeyResponse.PoolPlanType`
- `proxy/handler.go`
  - `contextAPIKeyPoolPlanType`
  - `requestAPIKeyPoolPlanType`
  - `normalizeAPIKeyPoolPlanType`
  - `accountFilterForAPIKeyPool`
  - `combineAccountFilters`
  - `authMiddleware` 中对池类型的上下文注入
  - `dbKeyCacheTTL`
  - `refreshDBKeys(force bool)`
- `proxy/handler_anthropic.go`
  - `/v1/messages` 的账号池过滤
- `proxy/images.go`
  - 图片接口的账号池过滤

### 前端管理台相关

- `frontend/src/types.ts`
- `frontend/src/api.ts`
- `frontend/src/pages/APIKeys.tsx`
- `frontend/src/locales/zh.json`
- `frontend/src/locales/en.json`

### 防封技术相关

- `proxy/utls_transport.go`
  - `nodejs24CipherSuites`：17 个 cipher suites（顺序关键）
  - `nodejs24Curves`：3 个 supported groups
  - `nodejs24SignatureAlgorithms`：9 个签名算法
  - `nodejs24ExtensionOrder`：14 个 TLS 扩展顺序（含 GREASE ECH）
  - `buildNodeJS24ClientHelloSpec()`：构建精确 ClientHello 规格
  - `createConnection()`：改用 `HelloCustom` + `ApplyPreset`（原为 `HelloChrome_Auto`）
- `proxy/executor.go`
  - `codexWireHeaders`：Codex CLI 请求头精确大小写 map
  - `setWireHeader()`：绕过 Go canonical 规范化写入 header
  - `resolveWireKey()`：key 映射工具函数
  - `applyCodexRequestHeaders()`：重构为先清空再重建，顺序受控

### 测试文件

这些不是运行必需，但升级时如果上游测试布局变化，建议对照保留：

- `admin/handler_test.go`
- `admin/responses_test.go`
- `proxy/handler_test.go`

---

## 数据结构与接口变化

### 数据库变化

新增数据库字段：

- 表：`api_keys`
- 字段：`pool_plan_type`
- 默认值：`''`

升级时如果上游重写了建表或迁移逻辑，必须确认这个字段仍然存在。

### 管理接口变化

#### `POST /api/admin/keys`

新增请求字段：

```json
{
  "name": "Pro Pool",
  "pool_plan_type": "pro"
}
```

支持值：

- `all`
- `free`
- `team`
- `plus`
- `pro`
- 或省略 / 空字符串

#### `GET /api/admin/keys`

返回新增字段：

- `pool_plan_type`
- `raw_key`

### 导入接口变化

#### `POST /api/admin/accounts/import`

当前本地行为比上游更宽松：

- multipart 文件字段名**不要求**必须叫 `file`
- raw body 也能直接导入
- 单文件最大 256MB

如果升级后重新出现：

- `请上传文件（字段名：file）`
- `multipart: message too large`

说明导入增强被上游覆盖掉了。

---

## 强烈建议的升级方式

## 情况 A：以后使用 Git 管理（推荐）

如果你后续想“保留自己的功能，再跟 GitHub 增量更新”，**一定要切换成 git 工作流**，不要继续只靠下载 zip 包。

推荐流程：

1. 从 GitHub 重新 `git clone` 一个干净仓库
2. 新建本地分支，例如 `uq/local-custom`
3. 按本文档把本地定制迁入这条分支
4. 提交一个或多个清晰 commit
5. 以后每次升级只需要：
   - `git fetch upstream`
   - `git merge upstream/main` 或 `git rebase upstream/main`
   - 解决冲突
   - 重新构建验证

建议把定制拆成两个 commit：

1. `feat(import): support 256MB streaming imports and bounded refresh queue`
2. `feat(api-keys): add plan-scoped account pools`

这样以后冲突更容易处理。

## 情况 B：当前目录不是 Git 仓库

如果你当前这份代码是下载包，目录里没有 `.git`，那它**不适合直接做长期增量升级**。

建议做一次性迁移：

1. 保留当前目录作为“已验证运行版本”
2. 重新从 GitHub 克隆一份带历史的干净仓库
3. 在新仓库新建本地分支
4. 参照本文档，把本地功能人工迁过去
5. 迁完以后只在新的 git 仓库里继续维护

这样后面每次升级就不需要重新回忆改动点。

---

## 每次升级前的检查清单

升级前先做这几件事：

1. 备份 SQLite 数据文件或整个卷
2. 备份 `.env`
3. 记录当前镜像标签和容器状态
4. 保留本文档副本

当前 SQLite 部署常用命令：

```bash
docker compose -f docker-compose.sqlite.yml ps
docker compose -f docker-compose.sqlite.yml logs --tail=100 codex2api
docker exec codex2api-sqlite sh -lc 'cp /data/codex2api.db /data/codex2api.db.bak'
```

---

## 升级迁移步骤

### 方案 1：已经切到 Git 仓库

```bash
git fetch upstream
git checkout uq/local-custom
git merge upstream/main
```

冲突时按本文档重点检查以下区域：

1. `admin/handler.go`
2. `proxy/handler.go`
3. `database/postgres.go`
4. `database/sqlite.go`
5. `proxy/handler_anthropic.go`
6. `proxy/images.go`
7. `frontend/src/pages/APIKeys.tsx`
8. `proxy/utls_transport.go`（防封：TLS 指纹）
9. `proxy/executor.go`（防封：Header Wire Casing + applyCodexRequestHeaders）

### 方案 2：从当前非 Git 目录迁到新版本

```bash
# 1. 下载/克隆新的上游源码
# 2. 把本文档复制到新仓库 docs/ 目录
# 3. 对照本文档逐个合并关键文件
# 4. 完成后重新构建镜像并启动
docker build -t ghcr.io/james-6-23/codex2api:latest .
docker compose -f docker-compose.sqlite.yml up -d
```

不要直接无脑覆盖整文件，优先做**增量合并**，否则容易把上游新修复一起覆盖掉。

---

## 升级后验证清单

升级完必须做以下回归验证。

### 1. 后台健康检查

```bash
curl -sS -H "X-Admin-Key: <your-admin-key>" http://localhost:8080/api/admin/ops/overview
```

重点确认：

- 服务能正常启动
- `database_driver` 正确
- `cache_driver` 与当前部署模式一致

### 2. API Key 分池验证

创建一个测试 key：

```bash
curl -sS \
  -H "X-Admin-Key: <your-admin-key>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Pro Pool","pool_plan_type":"pro"}' \
  http://localhost:8080/api/admin/keys
```

然后检查：

```bash
curl -sS -H "X-Admin-Key: <your-admin-key>" http://localhost:8080/api/admin/keys
```

必须看到：

- 返回字段里有 `pool_plan_type`
- 值正确写入

### 3. 导入增强验证

验证 multipart 非 `file` 字段名：

```bash
curl -sS \
  -H "X-Admin-Key: <your-admin-key>" \
  -F "accounts=@/path/to/accounts.txt" \
  "http://localhost:8080/api/admin/accounts/import?format=txt"
```

验证 raw body 导入：

```bash
curl -sS \
  -H "X-Admin-Key: <your-admin-key>" \
  -H "Content-Type: text/plain" \
  --data-binary @/path/to/accounts.txt \
  "http://localhost:8080/api/admin/accounts/import?format=txt"
```

只要这两种都正常，说明导入增强还在。

### 4. 图片接口池路由验证

如果未来上游改了图片代理层，容易漏掉图片接口的池过滤。

因此要特别确认：

- `free/team/plus/pro` key 调图片接口时没有绕过分池逻辑

### 5. TLS 指纹验证

快速检查：确认 `proxy/utls_transport.go` 里仍然有 Node.js 24.x 指纹相关代码：

```bash
grep -n "nodejs24CipherSuites\|buildNodeJS24\|HelloCustom\|ApplyPreset" proxy/utls_transport.go
```

预期输出：应该能看到 `nodejs24CipherSuites`、`buildNodeJS24ClientHelloSpec`、`HelloCustom`、`ApplyPreset` 这几个关键词。

如果输出为空，说明防封代码被上游覆盖了，需要重新移植。

同样检查 executor.go：

```bash
grep -n "codexWireHeaders\|setWireHeader\|req.Header = make" proxy/executor.go
```

预期输出：应该能看到这三个关键词。

---

## 最容易被上游覆盖的点

以下逻辑最容易在升级时被覆盖掉：

### 1. 导入逻辑

风险原因：

- 上游可能重写上传解析
- 上游可能恢复对 `file` 字段名的强依赖
- 上游可能把上限改回较小值

识别信号：

- 再次出现 `请上传文件（字段名：file）`
- 再次出现 `multipart: message too large`

### 2. API Key 表结构

风险原因：

- 上游可能修改 `api_keys` 结构或查询字段

识别信号：

- `GET /api/admin/keys` 不再返回 `pool_plan_type`
- 创建 key 时 `pool_plan_type` 不生效

### 3. 代理入口过滤

风险原因：

- 上游可能重构 `Responses` / `ChatCompletions` / `Messages` / `Images` 入口

识别信号：

- 不同池 key 开始混用账号
- `pro` 池 key 能打到非 `pro` 账号
- 图片接口绕过池限制

### 4. 前端管理台

风险原因：

- 上游可能改 API key 页面结构或接口字段

识别信号：

- 创建 API key 时看不到”账号池”下拉框
- 列表不显示 `pool_plan_type`

### 5. TLS 指纹（防封）

风险原因：

- 上游可能重写 `proxy/utls_transport.go`（例如升级 utls 版本、改连接池逻辑）
- 上游可能把 `createConnection` 改回 `HelloChrome_Auto` 或其他预设

识别信号：

- `proxy/utls_transport.go` 里出现 `HelloChrome_Auto` 或 `HelloFirefox_Auto` 等预设
- `createConnection` 里没有 `ApplyPreset` 调用
- 文件顶部没有 `nodejs24CipherSuites` 等变量定义

恢复方法：

- 对照本文档”关键实现文件 → 防封技术相关”，把 `buildNodeJS24ClientHelloSpec()` 和相关变量重新加回去
- 把 `createConnection` 里的握手方式改回 `HelloCustom` + `ApplyPreset`

### 6. Header Wire Casing 基础设施

风险原因：

- 上游可能重写 `proxy/executor.go` 中的 `applyCodexRequestHeaders`

识别信号：

- `executor.go` 里没有 `codexWireHeaders` map
- `applyCodexRequestHeaders` 里没有 `req.Header = make(http.Header)` 的清空重建逻辑

恢复方法：

- 把 `codexWireHeaders`、`setWireHeader`、`resolveWireKey` 三个定义加回 `executor.go`
- 把 `applyCodexRequestHeaders` 改为先 `req.Header = make(http.Header)` 再逐项设置

---

## 建议长期维护方式

后续建议把本地定制长期维持为三层：

1. **上游主线**
2. **你的本地定制分支**
3. **本文档**

原则：

- 本文档负责记录“功能目标、影响文件、验证方式”
- Git commit 负责记录“具体代码差异”
- 每次升级后先跑验证，再替换线上容器

不要只记“我好像改过上传”和“我好像改过 API key 路由”，这样下一次冲突一定会漏。

---

## 最小验收标准

如果你以后升级后只想做最小确认，至少看这 5 条：

1. 大文件导入仍然支持 256MB
2. multipart 非 `file` 字段名仍然能导入
3. `/api/admin/keys` 仍然返回 `pool_plan_type`
4. `free/team/plus/pro` 四类 key 仍然各走各的账号池
5. `proxy/utls_transport.go` 里仍然有 `nodejs24CipherSuites` 和 `buildNodeJS24ClientHelloSpec`（TLS 防封指纹）

只要这 5 条不丢，你这次本地定制的核心价值就还在。
