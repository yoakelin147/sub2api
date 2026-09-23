# 供应商账号 API 接入指南

本文档面向已由平台管理员开通的号池供应商。供应商 API 只允许管理当前供应商自己提交的上游账号，不具备用户、余额、订阅、支付、分组、倍率或管理员配置权限；可以选择库存代理并设置并发、优先级等账号参数。

网页中的“系统令牌”页面已内置操作指南，可选择账号类型并复制 Bash/cURL 或 PowerShell 示例。示例中的系统令牌和账号凭据都是占位符，必须替换后自行执行；脱敏令牌不能直接调用接口。

## 0. 正确的供应商操作流程

1. 管理员开通供应商并允许所需平台/类型。
2. 供应商先通过 `GET /supplier/proxies` 获取库存代理，在“供应账号 → 添加账号”绑定代理并填写真实认证信息；OAuth / Setup Token 还要提供网页登录邮箱和密码。批量操作也可使用 API。
3. 默认提交后待审核；长期可信供应商由管理员配置免审及平台分组后可自动审核启用。提交不代表上游连接测试成功。
4. 管理员查看提交的配置（秘密默认隐藏），按需要分配分组并审核。
5. 连接测试只有收到明确成功结果才算通过；账号是否进入调度还取决于审核及启用状态。

**Antigravity 注意事项**：`apikey` 对应上游中转服务，必须同时提供该服务的 `api_key` 和 `base_url`；真实 Antigravity 账号 Token 应选择 `oauth` 并提供 `access_token`，可附带 `refresh_token` 等字段。仅填写 API Key 缺少地址会被拒绝，不应把任意字符串当作可用凭据。

**三种地址/权限要分清**：平台 API 根地址用于调用本系统；`credentials.base_url` 是账号指向的上游服务；`proxy_id` 只能引用平台已配置且启用、未过期的库存代理。自定义上游地址必须是 HTTPS 且满足管理员白名单。

代理为必填项，库存为空时无法提交；其余运营默认值为无分组、并发 1、优先级 50、倍率 1、到期自动暂停。分组及计费倍率仍由管理员管理。

## 1. 基础约定

- Base URL：`https://你的域名/api/v1`
- Content-Type：`application/json`
- 机器认证：`x-api-key: supplier_<selector>_<secret>`
- 网页成员认证：`Authorization: Bearer <supplier-jwt>`
- 单个创建和批量创建必须带 `Idempotency-Key`，长度不超过 128 个可打印 ASCII 字符。
- 每个供应商只保留一个有效系统令牌；轮换后旧令牌立即失效。
- 完整系统令牌仅在生成/轮换响应中显示一次，平台数据库只保存摘要。
- 成功响应统一为 `{"code":0,"message":"success","data":...}`；失败响应包含 HTTP 状态、`message` 和稳定的 `reason`。
- 时间字段为 RFC 3339 字符串；创建/更新请求中的 `expires_at` 为 Unix 秒。

## 2. 获取系统令牌

供应商成员先通过网页正常登录，再在“系统令牌”页面生成令牌。也可以使用供应商成员 JWT 调用：

```bash
curl -X POST "https://example.com/api/v1/supplier/access-token/regenerate" \
  -H "Authorization: Bearer SUPPLIER_MEMBER_JWT"
```

响应中的 `data.key` 是唯一一次可见的完整令牌。以下接口只能使用供应商成员 JWT，系统令牌不能管理或撤销自身：

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/supplier/access-token` | 查询是否存在、掩码、创建时间、最后使用时间 |
| `POST` | `/supplier/access-token/regenerate` | 首次生成或轮换 |
| `DELETE` | `/supplier/access-token` | 撤销 |

## 3. 账号接口

| 方法 | 路径 | 说明 |
|---|---|---|
| `GET` | `/supplier/me` | 当前供应商、允许类型、令牌状态和账号统计 |
| `GET` | `/supplier/proxies` | 可选的库存代理 ID 和名称（不返回代理密钥） |
| `POST` | `/supplier/oauth/:platform/auth-url` | 以 `proxy_id`、`type` 发起本人 OAuth 授权；Gemini 可加 `oauth_type`、`project_id`、`tier_id` |
| `POST` | `/supplier/oauth/:platform/exchange-code` | 以 `type`、`session_id`、`code`、`state` 换取本人 Token，不接收代理和重定向覆盖 |
| `POST` | `/supplier/oauth/:platform/credential-exchange` | Claude Cookie、Grok SSO 或平台启用的 Grok 邮箱密码登录换票 |
| `GET` | `/supplier/oauth/grok/capabilities` | 查询 Grok 密码授权是否启用 |
| `GET` | `/supplier/accounts` | 分页查询自己的账号 |
| `POST` | `/supplier/accounts` | 创建一个待审核账号，要求幂等键 |
| `POST` | `/supplier/accounts/batch` | 批量创建，要求幂等键 |
| `GET` | `/supplier/accounts/:id` | 获取自己的账号详情 |
| `PUT` | `/supplier/accounts/:id` | 更新允许字段或暂停/恢复 |
| `DELETE` | `/supplier/accounts/:id` | 软删除自己的账号 |
| `POST` | `/supplier/accounts/:id/test` | 测试已保存的账号；请求体至少为 `{}` |

列表参数：

- `page`、`page_size`（默认 1/20）
- `platform`、`type`、`status`、`review_status`
- `search`
- `sort_by`、`sort_order=asc|desc`

跨供应商读取、更新、测试和删除统一返回 `404 SUPPLIER_ACCOUNT_NOT_FOUND`，不会透露目标账号是否存在。

## 4. 创建一个账号

`external_id` 是可选的供应商自有系统编号，只用于与自己的系统映射和租户内防重；不填时数据库自动生成平台账号 `id`，创建响应中的 `data.id` 即该编号。不要为没有自有系统编号的账号额外编造编号。

```bash
curl -X POST "https://example.com/api/v1/supplier/accounts" \
  -H "x-api-key: supplier_REPLACE_ME" \
  -H "Idempotency-Key: vendor-create-ext-10001-v1" \
  -H "Content-Type: application/json" \
  -d '{
    "external_id": "ext-10001",
    "name": "OpenAI Key 10001",
    "notes": "供应商内部批次 2026-09",
    "platform": "openai",
    "type": "apikey",
    "proxy_id": 123,
    "credentials": {"api_key": "sk-REPLACE_ME"},
    "expires_at": 1790000000
  }'
```

PowerShell：

```powershell
$headers = @{
  'x-api-key' = 'supplier_REPLACE_ME'
  'Idempotency-Key' = 'vendor-create-ext-10001-v1'
}
$body = @{
  external_id = 'ext-10001'
  name = 'OpenAI Key 10001'
  platform = 'openai'
  type = 'apikey'
  proxy_id = 123
  credentials = @{ api_key = 'sk-REPLACE_ME' }
} | ConvertTo-Json -Depth 8

Invoke-RestMethod -Method Post `
  -Uri 'https://example.com/api/v1/supplier/accounts' `
  -Headers $headers -ContentType 'application/json' -Body $body
```

默认审核供应商的新账号创建为：

- `review_status=pending`
- `status=disabled`
- `schedulable=false`
- 无分组

免审供应商必须由管理员预先指定同平台启用分组，创建时才自动审核并启用；否则拒绝提交。供应商必须提供库存 `proxy_id`，可选 `concurrency`、`priority`、`load_factor`、`auto_pause_on_expired`；不能提交 `supplier_id`、`group_ids`、`rate_multiplier`、`schedulable`、`review_status` 或 `extra`。

OAuth / Setup Token 的 `credentials` 还必须包含 `access_token`、`email`、`password`（网页登录密码）。例如：`{"access_token":"TOKEN", "email":"name@example.com", "password":"WEB_LOGIN_PASSWORD"}`。服务端加密保存密码；只允许管理员按需查看，供应商的列表、详情、创建和更新响应均不回显密码或其他敏感凭据。请通过 HTTPS 提交，且服务器须配置稳定的 `totp.encryption_key`，否则这类账号会拒绝保存。

## 5. 批量创建

- 最大 500 条。
- 请求正文最大 2 MiB。
- 每项凭据正文最大 1 MiB。
- 合法项可以成功，格式/业务错误项返回逐项错误。
- 数据库、Redis 或其他基础设施错误会使整批失败。

```bash
curl -X POST "https://example.com/api/v1/supplier/accounts/batch" \
  -H "x-api-key: supplier_REPLACE_ME" \
  -H "Idempotency-Key: vendor-batch-20260922-001" \
  -H "Content-Type: application/json" \
  -d '{
    "accounts": [
      {
        "external_id": "ext-10002",
        "name": "OpenAI Key 10002",
        "platform": "openai",
        "type": "apikey",
        "proxy_id": 123,
        "credentials": {"api_key": "sk-REPLACE_ME"}
      },
      {
        "external_id": "ext-10003",
        "name": "Anthropic Key 10003",
        "platform": "anthropic",
        "type": "apikey",
        "proxy_id": 123,
        "credentials": {"api_key": "sk-ant-REPLACE_ME"}
      }
    ]
  }'
```

批量响应中的每项包含 `index`、`external_id`、`success`，成功时包含 `account_id`，失败时包含 `error_code` 和安全错误信息。

## 6. 幂等重试

幂等记录默认保留 24 小时，作用域包含供应商、HTTP 方法、路由和请求正文：

- 同一供应商、同一键、相同正文：返回首次结果，并带 `X-Idempotency-Replayed: true`。
- 同一键、不同正文：返回 `409 IDEMPOTENCY_KEY_CONFLICT`。
- 首次请求仍在执行：返回 `409 IDEMPOTENCY_IN_PROGRESS`。
- 幂等存储不可用：返回 `503 IDEMPOTENCY_STORE_UNAVAILABLE`，不会绕过幂等继续写入。

客户端应为一次业务写入生成稳定键；网络超时后重试必须复用原键和原正文，不能每次重试都生成新键。

## 7. 更新、暂停和测试

更新非敏感信息：

```bash
curl -X PUT "https://example.com/api/v1/supplier/accounts/123" \
  -H "x-api-key: supplier_REPLACE_ME" \
  -H "Content-Type: application/json" \
  -d '{"name":"新名称","notes":"更新备注"}'
```

暂停或恢复：

```json
{"status":"disabled"}
```

只有已审核通过账号可以设置为 `active`。需要编辑 OAuth 登录信息时，必须完整重新提交 token、邮箱及密码，已有秘密不可读取。默认审核供应商更新 `credentials` 或变更代理会重置为 `pending + disabled + schedulable=false`；仅修改名称、备注或外部编号不重置审核。免审策略不允许自行启用被驳回账号。

测试账号：

```bash
curl -X POST "https://example.com/api/v1/supplier/accounts/123/test" \
  -H "x-api-key: supplier_REPLACE_ME" \
  -H "Content-Type: application/json" \
  -d '{}'
```

测试接口只能使用数据库中已归属当前供应商的账号，不能临时传入 URL 或凭据，也不会隐式批准或启用账号。

## 8. 支持的凭据类型

每个供应商还必须由管理员在 `allowed_account_kinds` 中显式允许对应组合；默认空列表表示全部拒绝。

| Platform | Type | 必填凭据 |
|---|---|---|
| `openai` | `apikey` | `api_key` |
| `openai` | `upstream` | `api_key`, `base_url` |
| `openai` | `oauth` / `setup-token` | `access_token`, `email`, `password` |
| `anthropic` | `apikey` | `api_key` |
| `anthropic` | `upstream` | `api_key`, `base_url` |
| `anthropic` | `oauth` / `setup-token` | `access_token`, `email`, `password` |
| `anthropic` | `bedrock` | `auth_mode`, `aws_region`，并按模式提供 SigV4 或 API Key |
| `anthropic` | `service_account` | `service_account_json`, `location` |
| `gemini` | `apikey` | `api_key` |
| `gemini` | `oauth` | `access_token`, `email`, `password` |
| `gemini` | `service_account` | `service_account_json`, `location` |
| `antigravity` | `oauth` | `access_token`, `email`, `password` |
| `antigravity` | `apikey` | `api_key`, `base_url` |
| `grok` | `apikey` | `api_key` |
| `grok` | `oauth` | `access_token`, `email`, `password` |
| `kimi` / `zhipu` / `deepseek` / `minimax` / `opencode_go` | `apikey` | `api_key`, `account_mode`, `api_protocol` |

OAuth 账号创建接口仍接受已经取得的 token bundle，不接收授权码、SSO 或 Cookie。供应商后台复用管理员授权界面，支持 OpenAI、Claude、Gemini、Antigravity、Grok OAuth 和 Claude Setup Token。调用 `POST /supplier/oauth/:platform/auth-url`，提交 `{ "proxy_id": 123, "type": "oauth" }`（Claude Setup Token 填 `setup-token`；Gemini 可选 `oauth_type` 为 `code_assist`、`google_one` 或 `ai_studio`，并可填 `project_id`、`tier_id`），返回 `auth_url` 和 `session_id`；打开地址登录，再把 `code`、`session_id`、适用的 `state` 提交到 `POST /supplier/oauth/:platform/exchange-code`。会话绑定当前供应商和库存代理，30 分钟有效且成功后失效，不允许换票时覆盖代理或重定向地址。所有接口需 `X-API-Key` 或供应商后台登录态，换票不会自动创建或审核账号。

Claude 还可通过 `POST /supplier/oauth/anthropic/credential-exchange` 提交 `{ "type": "oauth", "proxy_id": 123, "method": "cookie", "session_key": "..." }`（Setup Token 改为 `type: "setup-token"`）；Grok 可通过 `/supplier/oauth/grok/credential-exchange` 使用 `method: "sso"` + `sso_token`，或平台启用密码授权时使用 `method: "password"` + `email`、`password`。先查询 `/supplier/oauth/grok/capabilities` 的 `password_auth_enabled`。换得 Token 后仍须通过账号创建/更新接口提交。Composite、影子账号、任意请求头覆写和管理员调度字段均不支持。

以上 OAuth / Setup Token 凭据中的 `refresh_token` 可选；网页登录 `email`、`password` 可能不会由换票接口提供，提交账号时仍需填写。供应商“系统令牌”页面可分别选“查询可用库存代理”“生成授权地址”“授权码换 Token”复制完整 cURL / PowerShell 示例；授权/换票接口不要求 `Idempotency-Key`，创建/批量创建账号仍要求。

自定义 `base_url`：

- 只允许 HTTPS。
- 必须命中平台内置域名或管理员配置的域名白名单。
- 私网、环回、链路本地、云元数据和保留地址会被拒绝。
- `service_account_json` 必须包含一致的 `project_id`、`client_email` 和合法 PEM 私钥。

完整字段契约以仓库中的 `openspec/changes/add-supplier-account-management/credential-contract.md` 为准。

## 9. 常见错误

| HTTP | reason | 说明 |
|---:|---|---|
| 400 | `IDEMPOTENCY_KEY_REQUIRED` | 创建请求缺少幂等键 |
| 401 | `INVALID_SUPPLIER_TOKEN` | 令牌格式、摘要或 selector 无效 |
| 403 | `SUPPLIER_DISABLED` | 供应商已禁用或删除 |
| 403 | `ACCOUNT_KIND_NOT_ALLOWED` | 管理员未允许该 platform/type |
| 404 | `SUPPLIER_ACCOUNT_NOT_FOUND` | 账号不存在或不属于当前供应商 |
| 409 | `SUPPLIER_EXTERNAL_ID_EXISTS` | 当前供应商内 external_id 重复 |
| 409 | `IDEMPOTENCY_KEY_CONFLICT` | 同一幂等键配了不同正文 |
| 413 | `BATCH_TOO_LARGE` | 超过批量数量或正文边界 |
| 422 | `INVALID_CREDENTIALS` / `SUPPLIER_ACCOUNT_INPUT_INVALID` | 凭据、字段、URL 或状态不合法 |
| 429 | `RATE_LIMITED` | 供应商租户达到面板 RPM 限制；按 `Retry-After` 重试 |
| 503 | `IDEMPOTENCY_STORE_UNAVAILABLE` | 幂等存储不可用，写请求 fail closed |

## 10. 安全要求

- 令牌只能放在 HTTPS 请求头，不能放进 URL、日志、工单截图或前端 localStorage。
- 上游账号凭据只在创建/更新请求中单向传入；列表和详情不会回显。
- 供应商系统应记录自己的 `external_id`、幂等键和平台返回的 `account_id`，但不得记录完整系统令牌或上游 Key。
- 发现令牌疑似泄漏时立即由供应商成员或管理员轮换/撤销。
- 不要把管理员 API Key、用户模型 Key 或 JWT 当作供应商系统令牌；三者不能互相替代。
