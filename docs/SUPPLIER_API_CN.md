# 供应商账号 API 接入指南

本文档面向已由平台管理员开通的号池供应商。供应商 API 只允许管理当前供应商自己提交的上游账号，不具备用户、余额、订阅、支付、分组、代理、优先级、倍率、并发或管理员配置权限。

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
  credentials = @{ api_key = 'sk-REPLACE_ME' }
} | ConvertTo-Json -Depth 8

Invoke-RestMethod -Method Post `
  -Uri 'https://example.com/api/v1/supplier/accounts' `
  -Headers $headers -ContentType 'application/json' -Body $body
```

新账号固定创建为：

- `review_status=pending`
- `status=disabled`
- `schedulable=false`
- 无分组

供应商不能在请求中提供 `supplier_id`、`group_ids`、`proxy_id`、`priority`、`concurrency`、`load_factor`、`rate_multiplier`、`schedulable`、`review_status` 或 `extra`。出现未知或管理员字段时整项返回 422，不会静默忽略。

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
        "credentials": {"api_key": "sk-REPLACE_ME"}
      },
      {
        "external_id": "ext-10003",
        "name": "Anthropic Key 10003",
        "platform": "anthropic",
        "type": "apikey",
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

只有已审核通过账号可以设置为 `active`。更新 `credentials` 会立即把账号重置为 `pending + disabled + schedulable=false`，必须重新审核；仅修改名称、备注或外部编号不会重置审核。

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
| `openai` | `oauth` / `setup-token` | `access_token` |
| `anthropic` | `apikey` | `api_key` |
| `anthropic` | `upstream` | `api_key`, `base_url` |
| `anthropic` | `oauth` / `setup-token` | `access_token` |
| `anthropic` | `bedrock` | `auth_mode`, `aws_region`，并按模式提供 SigV4 或 API Key |
| `anthropic` | `service_account` | `service_account_json`, `location` |
| `gemini` | `apikey` | `api_key` |
| `gemini` | `oauth` | `access_token` |
| `gemini` | `service_account` | `service_account_json`, `location` |
| `antigravity` | `oauth` | `access_token` |
| `antigravity` | `apikey` | `api_key`, `base_url` |
| `grok` | `apikey` | `api_key` |
| `grok` | `oauth` | `access_token` |
| `kimi` / `zhipu` / `deepseek` / `minimax` / `opencode_go` | `apikey` | `api_key`, `account_mode`, `api_protocol` |

OAuth 只接受已经取得的 token bundle，不开放授权码、SSO、Cookie 或密码换票。Composite、影子账号、任意请求头覆写和管理员调度字段均不支持。

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
