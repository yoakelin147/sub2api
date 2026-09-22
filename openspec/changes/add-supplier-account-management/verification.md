# 供应商账号管理交付验证记录

验证日期：2026-09-22
分支：`codex/supplier-module`

## 1. 交付结论

供应商模块已按本变更目录中的 README、能力地图、设计、凭据契约和五份能力规格完成。当前实现包括：

- 独立供应商租户和多个供应商成员。
- supplier JWT 与单供应商系统令牌两种认证方式。
- 数据库查询层的账号租户隔离。
- 全部约定 `platform + type` 的严格凭据白名单验证。
- 单个账号维护、测试、JSON/CSV/逐行文本批量导入和强制幂等。
- 管理员供应商 CRUD、成员、令牌、账号统计和原子审核治理。
- 供应商独立路由、菜单、账号页、令牌页、安全页及管理员供应商页面。
- 审计动作、认证方式、资源标识和递归凭据脱敏。
- 供应商维度共享限流；JWT 与机器令牌使用同一个 supplier_id 桶。
- 中文供应商 API 接入文档和可执行示例。

## 2. 自动化验证

### 2.1 后端

通过：

```text
go test ./internal/service -run Supplier...
go test ./internal/repository -run Supplier...
go test ./internal/handler ./internal/handler/admin -run Supplier...
go test ./internal/server/middleware -run Supplier...
go test ./migrations -run TestSuppliersMigration
go test -tags=unit ./...       # Linux golang:1.27.0-alpine 容器
golangci-lint run ./...        # 官方 v2.13.0 容器，0 issues
```

Linux 全量 unit 运行覆盖了 Windows 本机因没有 `sh` 而无法运行的三个 `pg_dump` 测试，最终全部通过。

Windows 首轮 `go test -tags=unit ./...` 仅有以下三个环境型失败，均在 Linux 容器复跑通过：

- `TestPgDumperHoldsMigrationLockThroughReaderClose`
- `TestPgDumperReleasesMigrationLockWhenProcessFails`
- `TestPgDumperReportsUnlockFailureAndDiscardsConnection`

失败原因为 Windows `%PATH%` 中不存在 `sh`，没有修改或跳过测试来绕过问题。

### 2.2 前端

通过：

```text
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build
```

最终全量 Vitest：

```text
Test Files  308 passed (308)
Tests       2294 passed (2294)
```

生产构建成功。Vite 仅报告仓库既有的大 chunk 和静态/动态混合导入提示，没有构建错误。

### 2.3 Docker Compose

执行：

```text
docker compose --env-file deploy/.env -f deploy/docker-compose.dev.yml up -d --build
```

结果：

```text
sub2api-dev            healthy   127.0.0.1:8080->8080/tcp
sub2api-postgres-dev   healthy
sub2api-redis-dev      healthy
```

镜像在 Linux 多阶段构建中完成前端 i18n、TypeScript 和 Vite 构建，并完成 Go `-tags embed` 编译。应用启动后 `suppliers` 表存在，迁移成功。

验收镜像 ID：`sha256:e19e10fe36a7b4a9ca82a1ad4dc1c9a376a7a4bdcc29881c4758ee4ef12a83d9`。

## 3. 双供应商真实 HTTP 验收

使用唯一 `e2e-*` 标识创建两家供应商和两个成员，分别签发 JWT 与系统令牌。测试结束后通过公开 API 删除账号、解绑成员并软删除供应商；数据库复核 `live_suppliers=0`、`live_accounts=0`。

| 场景 | 结果 |
|---|---|
| 两家供应商成员登录 | 200，角色均为 `supplier` |
| 新账号安全默认值 | `pending + disabled + schedulable=false` |
| 创建响应不回显凭据 | 通过 |
| A 读取 B 的账号 | 404 |
| A 修改 B 的账号 | 404 |
| A 删除 B 的账号 | 404 |
| A 测试 B 的账号 | 404 |
| 供应商令牌访问管理员接口 | 401 |
| 供应商令牌管理自身令牌 | 403 |
| 同键同正文批量重放 | 200，`X-Idempotency-Replayed=true`，无重复账号 |
| 管理员审核并绑定 Anthropic 分组 | 200 |
| 审核后进入调度 | `approved + active + schedulable=true` |
| 供应商更新凭据 | 立即恢复 `pending + disabled + schedulable=false` |
| 令牌轮换 | 旧令牌 401，新令牌 200 |
| 供应商统计 | 返回真实账号数量 |
| 审计动作 | 含 `supplier.account.*`、`supplier.token.*`、`admin.supplier.*` |

## 4. 浏览器验收

使用 Playwright CLI 对 Docker 服务执行真实浏览器验收：

- supplier 登录自动进入 `/supplier/accounts`。
- 侧边栏只显示“供应账号 / 系统令牌 / 账号安全”。
- 供应账号页显示统计、筛选、单个新增和批量导入入口。
- 1280×900 桌面视口和 320×800 移动视口均可访问。
- 网页实际创建 Anthropic API Key 账号成功，列表显示“待审核 / 已禁用”，启用按钮不可用。
- 修复共享侧边栏曾触发普通用户 `/keys` 探测的问题后，供应商页面控制台为 `0 errors, 0 warnings`。
- 浏览器测试账号和账号数据已清理。

## 5. 安全审核

### 5.1 已验证控制

- `supplier_id` 仅来自认证上下文；供应商 DTO 不接受归属字段。
- Get/Update/Delete/Test 在同一仓储查询中绑定 `account_id + supplier_id`。
- 系统令牌使用 256-bit secret，只保存 selector、SHA-256 摘要和掩码。
- 令牌管理只接受 supplier JWT；机器令牌不能轮换或撤销自身。
- admin API Key、用户模型 Key、JWT 和 supplier token 格式/权限不可互换。
- 创建和批量创建强制幂等；幂等存储不可用时 fail closed。
- 单账号正文 1 MiB、批量正文 2 MiB、批量数量 500；合法 JSON 加超量尾部也会返回 413。
- 凭据按静态 allowlist 校验；未知字段、Cookie、密码、SSO、header override 和管理员字段被拒绝。
- Base URL 仅 HTTPS、受域名白名单和私网/保留地址检查约束。
- 供应商操作按 supplier_id 共用 Redis 限流桶，机器令牌不能绕过 user_id 限流。
- 审计请求体把整个 `credentials` 节点替换为 `***`，包括未知嵌套键。
- Docker 日志测试凭据命中数：0；审计请求体测试凭据命中数：0。
- 前端完整令牌只存在组件内存中，未写入 localStorage/sessionStorage。

### 5.2 明确保留的系统边界

- 现有 `accounts.credentials` 仍为数据库 JSONB 明文，沿用项目现有运行方式；全量应用层加密迁移不属于本变更范围。
- Composite、影子账号和交互式 OAuth/SSO/密码/Cookie 换票保持管理员专属。
- 不包含供应商结算、账单或供应商内部多级 RBAC。

## 6. 部署与回滚

- 数据库迁移为纯增量：新增表、nullable 列、约束和索引；历史账号回填为 `approved`，不删除历史数据。
- 应用回滚时保留新增表和列，不执行破坏性降级；先禁用供应商并撤销令牌，再切回上一应用镜像。
- 本次 Compose 重建已验证应用可在已有数据库上升级并健康启动。
- E2E 清理验证软删除不会破坏供应商、账号与审计关联。

## 7. 手工验收入口

- 管理员：`http://127.0.0.1:8080/admin/suppliers`
- 供应商账号：`http://127.0.0.1:8080/supplier/accounts`
- 供应商令牌：`http://127.0.0.1:8080/supplier/access-token`
- 供应商安全：`http://127.0.0.1:8080/supplier/security`

管理员可先创建供应商、选择允许类型并添加成员，再用成员邮箱登录。供应商新建账号后应先显示待审核；管理员在供应商详情选择账号和匹配平台的分组进行审核。

## 8. 2026-09-22 管理界面布局与分页修复

- 供应商列表改为全宽表格；打开详情后通过供应账号、成员、基本信息与令牌三个标签切换，保留返回列表和 URL 定位供应商。
- 修复成员只读取前 100 条且无翻页入口的问题：成员、账号、供应商列表均显式传递页码和每页条数，默认 20 条，并显示后端总数。
- 表格内容限高 384px、表头固定、分页位于滚动区域外；审核分组及备注移入批量操作弹窗，未勾选时不展示批量工具栏。
- 翻页清空账号勾选；筛选、每页条数变化返回第一页；删除成员或审核导致末页消失时回到有效页；序号校验阻止过期请求覆盖结果。
- 保持原有管理员接口、供应商归属和后端授权边界；成员仍固定供应商角色，令牌仅在内存保留一次性明文。未修改数据库、权限或依赖。

验证结果：

- 7 个相关测试文件通过，共 20 个测试（首轮 19 个通过，补充末页审核回退测试后对应文件 9 个测试通过）。覆盖 101 名成员分页、页容量变化、筛选清空勾选、末页回退、乱序响应、失败重试、审核分组和键盘标签切换。
- TypeScript 类型检查及修改文件 ESLint 通过；本机 ESLint 使用已安装的 vue-eslint-parser 所在 node_modules 作为 NODE_PATH，未添加依赖或修改检查规则。
- Docker 多阶段构建通过，包括 i18n、vue-tsc、Vite 和 Go 嵌入前端构建；仅重建应用容器，Compose 等待结果为 healthy。
- 真实浏览器验证供应商列表、条数选择、全宽详情、成员表格及分页、添加成员弹窗、设置分区和返回列表。PC 端布局正常，未观察到浏览器控制台 error。
- 用户确认仅需 PC 端布局，停止扩大验证范围。已恢复验收用视口及每页条数设置；未创建或删除实际业务成员、账号，也未轮换令牌。
