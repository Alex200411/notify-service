# Lessons Learned

记录每次迭代中遇到的问题、踩过的坑、以及值得保留的好模式。

---

## Session 1 — MVP 实现 (2026-09-17)

### Bugs & Issues

#### 1. Go 空 map 字面量语法错误
- **问题**: 测试代码里写了 `payload: map[string]interface{}` 作为空 map 值，编译报错 `(type) is not an expression`
- **原因**: `map[string]interface{}` 是类型声明，空 map 字面量需要 `map[string]interface{}{}`（多一对花括号）
- **修复**: 加上 `{}`
- **教训**: Go 的 map 类型和 map 字面量长得很像，写的时候容易漏掉尾部的 `{}`

#### 2. Docker Compose 不可用
- **问题**: `docker compose up -d` 报 `unknown shorthand flag: 'd'`，`docker-compose` 二进制不存在
- **原因**: 机器上装的是 Colima + 原生 Docker CLI，没有 compose 插件
- **解决**: 回退到本地直接跑 PostgreSQL + Redis，不依赖 Docker
- **教训**: 不要假设环境里有 docker-compose，提前检查可用工具。Dockerfile 和 docker-compose.yml 保留给有完整 Docker 环境的场景

#### 3. Go module 下载超时
- **问题**: `go mod tidy` 连 `proxy.golang.org` 超时
- **解决**: 切换到 `GOPROXY=https://goproxy.cn,direct`
- **教训**: 国内网络环境下，Go proxy 需要备选方案

#### 4. golangci-lint 版本不兼容 Go 1.26
- **问题**: CI 里用 golangci-lint v1.59 → typecheck 报错；升级到 v2.1 → 报 `Go language version (go1.24) lower than targeted Go version (1.26.5)`
- **原因**: golangci-lint 的发布版本是用固定 Go 版本编译的，如果项目的 Go 版本比它新就会挂
- **解决**: 用 `go vet`（Go 内置，版本永远匹配）替代 golangci-lint
- **教训**: 第三方 lint 工具的 Go 版本兼容性是个隐藏坑，MVP 阶段 `go vet` 完全够用

### Good Patterns

#### 1. Persist-before-acknowledge 模式
- 先写数据库再推队列，确保消息不丢。即使 Redis 挂了，recovery sweep 能兜底
- 这个模式在面试讲解时很有说服力，因为它简单但覆盖了核心可靠性场景

#### 2. 接口解耦 + Mock 测试
- Repository 和 Queue 都定义了 interface，测试时用 mock 实现
- 不需要真实数据库和 Redis 就能跑完所有单元测试
- 4 个投递测试覆盖了：成功、重试后成功、dead letter、无模板 fallback

#### 3. 两个 sweep 兜底机制
- **Retry sweep**: 每 10s 扫 `failed` 且 `next_retry_at` 过期的记录
- **Recovery sweep**: 每 30s 扫 `pending` 超 1 分钟或 `processing` 超 2 分钟的记录
- 这两个 sweep 分别处理"正常重试"和"异常恢复"，职责清晰

#### 4. 供应商配置抽象
- 用 `body_template` (Go text/template) + `headers` (JSONB) 抽象不同供应商的 API 差异
- 新增供应商只需要加一条数据库记录，不用改代码
