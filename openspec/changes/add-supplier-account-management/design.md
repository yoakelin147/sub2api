# Design: 供应商账号管理与系统令牌

## 1. 现有架构约束

### 1.1 身份与路由

- 后端角色常量目前只有 `admin` 和 `user`。
- 登录 JWT 认证后，管理员路由另行检查管理员角色；普通用户路由目前只检查 JWT，隐含假设所有非管理员登录者都是普通用户。
- 前端登录跳转和布局也按“管理员/非管理员”二分。
- 管理员 API Key 使用 `x-api-key`，拥有完整管理员权限，不能作为供应商令牌复用。

### 1.2 账号模型

- `accounts` 是平台级资源，没有所有者字段。
- `credentials` 是按平台/类型变化的 JSONB；管理员创建接口接受开放结构。
- 账号可绑定多个分组，调度状态由 `status`、`schedulable`、过期、限流等字段共同决定。
- 现有账号服务已经处理凭据脱敏、敏感字段合并、分组校验、调度缓存和 outbox 更新，应继续复用。

### 1.3 关键设计结论

1. 在现有 `users` 表增加 supplier 角色和归属，复用现有登录、JWT、刷新令牌和 TOTP。
2. 给现有普通用户路由增加显式 `admin/user` 角色守卫，不能继续依赖“非管理员即用户”。
3. 新建供应商专用路由和服务，不把管理员 handler 暴露给供应商。
4. `supplier_id` 必须由认证上下文提供，任何供应商输入 DTO 均不包含该字段。
5. 供应商 API 的开放凭据结构必须按平台/类型使用服务端允许列表，不直接透传任意 JSON。

## 2. 数据模型

### 2.1 Supplier

新增 Ent schema `Supplier`，表名 `suppliers`：

```text
id                         bigint PK
code                       varchar(64), live rows unique
name                       varchar(120)
status                     varchar(20): active|disabled
notes                      text nullable
allowed_account_kinds      jsonb NOT NULL DEFAULT '[]'
token_selector             varchar(64) nullable unique
token_hash                 varchar(64) nullable
token_prefix               varchar(32) nullable
token_created_at           timestamptz nullable
token_last_used_at         timestamptz nullable
created_at                 timestamptz
updated_at                 timestamptz
deleted_at                 timestamptz nullable
```

`allowed_account_kinds` 结构：

```json
[
  { "platform": "openai", "type": "apikey" },
  { "platform": "anthropic", "type": "apikey" },
  { "platform": "openai", "type": "upstream" },
  { "platform": "anthropic", "type": "setup-token" },
  { "platform": "openai", "type": "oauth" },
  { "platform": "anthropic", "type": "bedrock" },
  { "platform": "gemini", "type": "service_account" }
]
```

规则：

- 空数组表示不允许提交任何账号，保持 fail closed。
- 保存时按 `platform + type` 去重。
- 平台和类型必须来自服务端已支持常量。
- 可配置类型覆盖系统现有可独立供应的 `apikey`、`upstream`、`setup-token`、`oauth`、`bedrock` 和 `service_account`，但每种组合必须先在服务端注册表中实现严格校验。
- Composite 和凭据影子账号不允许供应商创建。
- `status=disabled` 时成员 JWT 和系统令牌均不能访问供应商路由。

### 2.2 User

现有 `users` 增加：

```text
supplier_id bigint NULL REFERENCES suppliers(id)
```

角色增加 `supplier`，并增加数据库 CHECK：

```sql
(role = 'supplier' AND supplier_id IS NOT NULL)
OR
(role IN ('admin', 'user') AND supplier_id IS NULL)
```

不允许供应商自助注册；供应商成员只能由管理员创建或绑定。管理员不能把自己直接改成 supplier。禁用供应商不修改成员记录，但认证时必须同时检查供应商状态。

### 2.3 Account

现有 `accounts` 增加：

```text
supplier_id                bigint NULL REFERENCES suppliers(id)
supplier_external_id       varchar(191) NULL
review_status              varchar(20) NOT NULL DEFAULT 'approved'
reviewed_at                timestamptz NULL
reviewed_by                bigint NULL REFERENCES users(id)
review_note                text NULL
```

约束及索引：

- `review_status IN ('pending', 'approved', 'rejected')`。
- 历史和管理员创建账号使用 `supplier_id=NULL, review_status='approved'`。
- `supplier_external_id` 去除首尾空白，空字符串转 NULL。
- 活跃行建立 `(supplier_id, supplier_external_id)` 部分唯一索引，条件为 `supplier_id IS NOT NULL AND supplier_external_id IS NOT NULL AND deleted_at IS NULL`。
- 建立 `(supplier_id, review_status)` 和 `(supplier_id, status)` 索引。
- 供应商提交账号必须写入 `pending + inactive + schedulable=false`，且不绑定分组。

### 2.4 删除语义

- Supplier、User、Account 均沿用软删除。
- 不允许硬删除仍拥有账号或成员的供应商。
- 供应商软删除等价于认证禁用，但不清空账号归属和审计记录。
- 供应商删除账号调用现有账号软删除和调度 outbox 流程。

## 3. 身份认证与授权

### 3.1 网页 JWT

- 现有登录接口允许 `role=supplier` 的启用用户登录。
- JWT 仍以 user ID 为主体；每次受保护请求从数据库读取最新用户和供应商状态。
- 成功登录后前端按角色跳转：
  - admin → `/admin/dashboard`
  - user → `/dashboard`
  - supplier → `/supplier/accounts`
- `/api/v1/user/*`、`/keys`、`/usage`、`/subscriptions`、支付等普通用户接口必须增加 `admin/user` 角色守卫。
- `/api/v1/supplier/*` 只接受 supplier JWT 或供应商系统令牌。

### 3.2 供应商系统令牌

令牌格式：

```text
supplier_<selector>_<secret>
```

- `selector` 和 `secret` 均使用 `crypto/rand` 生成。
- `selector` 用于直接定位供应商，不使用供应商数据库 ID。
- 数据库保存 `selector`、`SHA-256(secret)` 和可显示前缀，不保存完整令牌。
- 摘要验证使用常量时间比较。
- 完整令牌仅在生成/重新生成接口响应中出现一次。
- 供应商令牌通过 `x-api-key` 传递，不与 JWT Bearer 或模型调用 `sk-` Key 混用。
- 系统令牌只能调用账号操作和 `/supplier/me`，不能管理令牌自身。
- 令牌使用成功后异步或限频更新 `token_last_used_at`，避免每个请求都写数据库。

### 3.3 统一供应商认证上下文

认证中间件向 Gin context 写入：

```text
supplier_id
actor_user_id（令牌请求为空）
actor_email（令牌请求使用 supplier:<code>）
actor_role=supplier
auth_method=supplier_jwt|supplier_api_key
credential_masked
```

业务 service 的所有方法必须显式接收 `supplierID`。禁止从请求正文或 query 读取租户归属。

## 4. 账号所有权边界

新增窄接口 `SupplierAccountRepository`，避免供应商 service 依赖拥有全局 List/Get/Update 权限的管理员仓储接口：

```go
type SupplierAccountRepository interface {
    CreateOwned(ctx context.Context, supplierID int64, account *Account) error
    GetOwnedByID(ctx context.Context, supplierID, accountID int64) (*Account, error)
    ListOwned(ctx context.Context, supplierID int64, filter SupplierAccountFilter) (...)
    UpdateOwned(ctx context.Context, supplierID int64, account *Account) error
    DeleteOwned(ctx context.Context, supplierID, accountID int64) error
}
```

实现规则：

- 查询条件必须在同一条 SQL/Ent query 中同时包含 `ID`、`SupplierIDEQ` 和未删除条件。
- 不采用“先 GetByID，再在 handler 比较 supplier_id”的模式，避免漏检和 TOCTOU。
- 批量操作先按 `supplier_id` 加载目标；返回数量与输入去重后的数量不一致时整批拒绝。
- 跨租户目标统一映射为 `SUPPLIER_ACCOUNT_NOT_FOUND`。
- 管理员现有 repository 和 API 保持全局权限。

## 5. 供应商可写字段

### 5.1 创建字段

允许：

```text
external_id
name
notes
platform
type
credentials
expires_at
```

禁止：

```text
supplier_id
group_ids
proxy_id
priority
concurrency
load_factor
rate_multiplier
status
schedulable
review_status
reviewed_by
extra
```

禁止字段出现在请求中时返回 422；不能静默忽略，以免供应商误以为配置已生效。

### 5.2 更新字段

允许更新：名称、备注、供应商外部编号、凭据、过期时间和供应商主动暂停状态。

- 修改任意敏感凭据、Base URL、平台或类型，必须重新进入 pending 并停止调度。
- 首版不允许原地修改平台或类型；应删除后重建，减少跨类型残留凭据风险。
- 供应商可以把已审核账号设为 inactive。
- 只有 `review_status=approved` 的账号才允许恢复 active；pending/rejected 不能由供应商启用。
- `error` 由系统维护，供应商不能直接设置。

### 5.3 响应脱敏

供应商账号响应仅返回：

```text
id, external_id, name, notes, platform, type
credential_masked / has_credentials
status, schedulable, review_status, review_note
expires_at, last_used_at, created_at, updated_at
```

不得返回完整 `credentials`、`extra`、代理配置、分组内部配置或调度评分。可复用现有账号凭据脱敏规则，但供应商 DTO 应采用字段白名单而非复用完整管理员 DTO。

## 6. 凭据验证

### 6.1 类型注册表

建立后端静态注册表，以 `platform + type` 为键描述：

- 是否允许供应商提交。
- 必填 credential 字段。
- 可选 credential 字段。
- 最大长度。
- Base URL 是否可编辑。
- Base URL 允许的 scheme/host。
- 对应现有账号创建规范化函数。

该注册表是服务端唯一权威；前端可以镜像用于表单展示，但服务端必须再次校验。

### 6.2 完整支持范围

- 各已支持平台的静态 `apikey`。
- 管理员允许的 `upstream`，Base URL 只允许 HTTPS 并按显式域名白名单验证；拒绝 localhost、环回、链路本地、RFC1918、IPv6 ULA、云元数据地址和重定向到私网。
- 已取得的 `setup-token` 和 `oauth` 凭据包；允许 access/refresh token 等该平台必要字段，但不允许供应商调用交互式 OAuth、SSO、密码或 Cookie 换票接口。
- `bedrock` 的 API Key 或 SigV4 凭据，以及 `service_account` JSON；只接受现有账号模型已支持且校验通过的字段。
- 所有类型均由 supplier 的 `allowed_account_kinds` 显式启用；默认全部拒绝。
- 不接受任意请求头覆写和 Composite/影子账号内部字段；模型映射仅在平台现有校验规则允许时开放。

## 7. 审核与调度

### 7.1 审核状态独立于运行状态

`review_status` 表示平台是否批准这份凭据配置；现有 `status/schedulable` 表示运行态。两者不得复用：

- pending：等待管理员确认，必须 inactive 且不可调度。
- rejected：管理员拒绝，必须 inactive 且不可调度。
- approved：允许管理员按现有规则启用或暂停。

### 7.2 审核操作

管理员批量审核请求包含账号 ID、目标分组和可选调度配置。服务端必须：

1. 确认所有账号属于目标供应商。
2. 确认全部为 pending/rejected，而不是其他供应商或历史账号。
3. 执行现有分组和混合渠道校验。
4. 在事务内写入审核字段、账号分组和运行状态。
5. 写入调度 outbox。
6. 返回逐项结果或原子失败；同一审核批次采用原子失败，避免半批进入生产。

## 8. API 契约

### 8.1 Supplier API

| Method | Path | Authentication | Description |
|---|---|---|---|
| GET | `/api/v1/supplier/me` | JWT/token | 当前供应商精简信息 |
| GET | `/api/v1/supplier/accounts` | JWT/token | 分页查询自己的账号 |
| GET | `/api/v1/supplier/accounts/:id` | JWT/token | 获取自己的账号 |
| POST | `/api/v1/supplier/accounts` | JWT/token | 创建一个待审核账号 |
| POST | `/api/v1/supplier/accounts/batch` | JWT/token | 批量创建待审核账号 |
| PUT | `/api/v1/supplier/accounts/:id` | JWT/token | 更新允许字段 |
| DELETE | `/api/v1/supplier/accounts/:id` | JWT/token | 软删除自己的账号 |
| POST | `/api/v1/supplier/accounts/:id/test` | JWT/token | 测试自己的账号 |
| GET | `/api/v1/supplier/access-token` | JWT only | 查询脱敏令牌状态 |
| POST | `/api/v1/supplier/access-token/regenerate` | JWT only | 创建或轮换令牌 |
| DELETE | `/api/v1/supplier/access-token` | JWT only | 撤销令牌 |

列表支持：`page`、`page_size`、`platform`、`type`、`status`、`review_status`、`search`、`sort_by`、`sort_order`。

### 8.2 Admin API

| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/admin/suppliers` | 供应商分页列表及账号统计 |
| POST | `/api/v1/admin/suppliers` | 创建供应商 |
| GET | `/api/v1/admin/suppliers/:id` | 供应商详情 |
| PUT | `/api/v1/admin/suppliers/:id` | 编辑供应商和允许类型 |
| DELETE | `/api/v1/admin/suppliers/:id` | 软删除供应商 |
| GET | `/api/v1/admin/suppliers/:id/members` | 成员列表 |
| POST | `/api/v1/admin/suppliers/:id/members` | 创建或绑定成员 |
| DELETE | `/api/v1/admin/suppliers/:id/members/:user_id` | 解绑/禁用成员 |
| GET | `/api/v1/admin/suppliers/:id/access-token` | 查询脱敏令牌状态 |
| POST | `/api/v1/admin/suppliers/:id/access-token/regenerate` | 生成或轮换令牌 |
| DELETE | `/api/v1/admin/suppliers/:id/access-token` | 撤销令牌 |
| GET | `/api/v1/admin/suppliers/:id/accounts` | 查看供应商账号 |
| POST | `/api/v1/admin/suppliers/:id/accounts/approve` | 批量审核通过 |
| POST | `/api/v1/admin/suppliers/:id/accounts/reject` | 批量拒绝 |
| POST | `/api/v1/admin/suppliers/:id/accounts/pause` | 批量暂停供应商账号 |

### 8.3 写请求幂等

- `POST /supplier/accounts` 和 `/batch` 强制要求 `Idempotency-Key`。
- 幂等作用域包含 `supplier_id + auth principal + route + key`。
- 请求正文使用稳定 JSON 计算哈希。
- 相同键、相同正文返回首次结果并带 `X-Idempotency-Replayed: true`。
- 相同键、不同正文返回 409 `IDEMPOTENCY_KEY_CONFLICT`（复用现有幂等服务）。
- 处理中重复请求返回 409。
- 幂等记录保留时间不得短于供应商客户端最大重试窗口；沿用项目现有写请求默认 TTL，并在 API 文档中声明。

### 8.4 批量边界

- 最大 500 条。
- 最大请求正文 2 MiB。
- 账号名称最大 100 字符，外部编号最大 191 字符，备注按现有账号限制。
- 每项独立校验并返回索引、external_id、account_id 或结构化错误。
- 单项格式错误不阻止其他合法项创建。
- 基础设施错误或幂等存储不可用时整批失败，不能继续无幂等写入。

## 9. 前端设计

### 9.1 路由与布局

- 增加 `requiresSupplier` route meta。
- 增加 role-based redirect helper，替换当前 admin/非 admin 二分。
- 供应商使用精简布局和独立菜单。
- 管理员和普通用户页面均明确拒绝 supplier 角色。

### 9.2 管理员页面

新增 `SuppliersView.vue` 和供应商详情组件：

- 列表：名称、code、状态、成员数、账号总数、待审核、可调度、异常、令牌状态、最后使用时间。
- 新建/编辑：名称、code、备注、允许平台与账号类型。
- 成员：创建、绑定、禁用和解绑。
- 令牌：状态、创建、重新生成、复制一次、撤销。
- 账号：筛选、批量审核、批量拒绝、批量暂停。

现有管理员账号页只增加供应商来源标签和详情跳转，不把完整供应商管理逻辑塞进现有大页面。

### 9.3 供应商页面

新增：

- `SupplierAccountsView.vue`
- `SupplierAccountForm.vue`
- `SupplierBatchImportDialog.vue`
- `SupplierAccessTokenView.vue`

供应商账号表仅展示允许字段。批量导入完整支持 JSON、CSV 和逐行文本；三种格式最终转换为同一个经过验证的 JSON 请求模型，避免形成三套业务逻辑。

## 10. 审计与可观测性

扩展现有审计系统：

```text
auth_method: supplier_jwt | supplier_api_key
actions:
  supplier.account.create
  supplier.account.batch_create
  supplier.account.update
  supplier.account.delete
  supplier.account.test
  supplier.token.regenerate
  supplier.token.revoke
  admin.supplier.create
  admin.supplier.update
  admin.supplier.disable
  admin.supplier.account.approve
  admin.supplier.account.reject
```

审计记录包含 supplier ID、账号 ID、actor user/token 掩码、IP、request ID、结果和失败码；不得包含上游 Key、完整 token、Cookie、密码或 service account 内容。

增加基础指标：

- 供应商认证成功/失败次数。
- 单个/批量提交数量及失败数。
- 待审核账号数量。
- 跨租户拒绝次数。
- 供应商账号测试成功率。

## 11. 威胁模型

| 威胁 | 风险 | 控制 |
|---|---|---|
| 供应商伪造 supplier_id | 跨租户操作 | supplier_id 仅来自认证上下文，DTO 不含该字段 |
| 修改 URL 中账号 ID | IDOR | repository 查询同时绑定 account_id 与 supplier_id，跨租户统一 404 |
| 泄漏系统令牌 | 冒充供应商 | 256-bit secret、只存摘要、可撤销、限流、最后使用记录 |
| 批量重试 | 重复建号 | Idempotency-Key + external_id 唯一索引 |
| 恶意 Base URL | SSRF/用户数据外泄 | HTTPS、域名白名单、DNS/IP 检查、禁止重定向到私网 |
| 凭据进入日志 | Key 泄漏 | 复用递归审计脱敏，供应商 DTO 白名单，错误不回显上游正文 |
| 供应商直接进入调度 | 服务污染 | 默认 pending/inactive/unschedulable，管理员审核后启用 |
| 供应商修改已审核 Key | 绕过审核 | 凭据身份变更自动 pending 并写调度 outbox |
| 大批量请求 | DoS | 500 条、2 MiB、Redis 限流、处理超时 |
| supplier 被当成普通 user | 权限扩散 | 普通用户路由显式 admin/user role guard + 回归测试 |

## 12. 凭据静态存储风险

当前账号凭据使用 JSONB 存储，应用运行时需要读取明文。此变更必须做到：

- 供应商系统令牌只存摘要。
- 供应商 API 和 UI 不回显已存凭据。
- 数据库、备份和 Docker volume 的访问权限按生产密钥库级别管理。

把全部账号敏感子字段改造成应用层加密会影响大量 JSON 查询、调度快照和凭据刷新逻辑，不纳入本次供应商模块。若合规要求数据库管理员也不能看到上游 Key，应单独建立凭据加密迁移变更，不在本功能中做半套加密。

## 13. 迁移与兼容

迁移全部为增量：

1. 新增 suppliers 表。
2. users 新增 nullable supplier_id；扩展 role CHECK（如现有数据库有约束）。
3. accounts 新增 nullable supplier_id、external_id 和审核字段。
4. 回填全部历史账号 review_status=approved。
5. 增加索引和部分唯一约束。
6. 生成 Ent 代码和 Wire 依赖。

供应商表为空时，新代码不改变现有业务。回滚应用时保留新增表和列，先禁用供应商和撤销令牌，不执行破坏性降级迁移。

## 14. 验证命令

```bash
cd backend
go generate ./ent
go generate ./cmd/server
go test -tags=unit ./...
go test -tags=integration ./internal/repository/...
go test ./...
golangci-lint run ./...

cd ../frontend
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build

cd ..
docker compose --env-file deploy/.env -f deploy/docker-compose.dev.yml up -d --build
```

## 15. 设计决策记录

| 决策 | 选择 | 原因 |
|---|---|---|
| 供应商身份 | users.role + supplier_id | 复用登录/JWT/TOTP，避免第二套认证系统 |
| 数据隔离 | account.supplier_id + scoped repository | 数据库查询层强制所有权，避免 handler 漏检 |
| API 路由 | 独立 `/supplier` | 不向供应商暴露管理员 handler |
| 系统令牌 | 每供应商一个、只存摘要 | 满足当前需求并保持最小实现 |
| 新账号状态 | 默认待审核 | 防止恶意/错误凭据直接进入生产调度 |
| 批量格式 | JSON API，最大 500 | 先覆盖系统对接，CSV 按真实样例后加 |
| OAuth 等复杂凭据 | 支持已取得的凭据包，不开放交互式登录 | 供应商负责取得凭据，平台负责严格校验和安全存储 |
| 自动审核 | 不提供 | 供应商始终不能绕过平台审核和调度配置 |
