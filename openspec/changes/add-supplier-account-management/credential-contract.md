# Supplier Credential Contract

本文件冻结供应商模块允许提交的现有账号凭据形状。它来自当前账号创建表单、OAuth token DTO、服务端读取逻辑和测试夹具；供应商 API 必须按白名单校验，不能把任意 JSON 直接交给管理员账号服务。

## 通用规则

- 所有字符串去除首尾空白；必填秘密不得为空。
- 单个 credential 字符串最大 1 MiB；完整单账号 JSON 最大 1 MiB；批量请求最大 2 MiB。
- 禁止字段：`password`、`sso_token`、`cookie`、`session_key`、任意 header override、调度瞬态字段和影子账号字段。
- `model_mapping`、`compact_model_mapping`、pool mode、临时不可调度规则首版供应商接口不接受；它们由管理员审核时配置。
- OAuth 仅接收已经取得的 token bundle，不向供应商开放授权码、密码、Cookie 或 SSO 换票接口。
- `base_url` 只允许 HTTPS；非内置默认域名必须命中管理员维护的供应商域名白名单并通过 SSRF 校验。

## 允许组合

| Platform | Type | 必填凭据 | 可选凭据 |
|---|---|---|---|
| `openai` | `apikey` | `api_key` | `base_url` |
| `openai` | `upstream` | `api_key`, `base_url` | — |
| `openai` | `oauth` | `access_token` | `refresh_token`, `id_token`, `expires_at`, `email`, `chatgpt_account_id`, `chatgpt_user_id`, `organization_id`, `plan_type`, `subscription_expires_at`, `client_id` |
| `openai` | `setup-token` | `access_token` | `refresh_token`, `expires_at`, `email`, `chatgpt_account_id` |
| `anthropic` | `apikey` | `api_key` | `base_url`, `auth_scheme` |
| `anthropic` | `upstream` | `api_key`, `base_url` | `auth_scheme` |
| `anthropic` | `oauth` | `access_token` | `refresh_token`, `expires_at`, `token_type`, `scope` |
| `anthropic` | `setup-token` | `access_token` | `refresh_token`, `expires_at`, `token_type`, `scope` |
| `anthropic` | `bedrock` | `auth_mode`, `aws_region` | SigV4 模式：`aws_access_key_id`, `aws_secret_access_key`, `aws_session_token`, `aws_force_global`；API Key 模式：`api_key` |
| `anthropic` | `service_account` | `service_account_json`, `location` | `project_id`, `client_email`, `tier_id` |
| `gemini` | `apikey` | `api_key` | `base_url`, `tier_id` |
| `gemini` | `oauth` | `access_token` | `refresh_token`, `token_type`, `expires_at`, `scope`, `project_id`, `oauth_type`, `tier_id` |
| `gemini` | `service_account` | `service_account_json`, `location` | `project_id`, `client_email`, `tier_id` |
| `antigravity` | `oauth` | `access_token` | `refresh_token`, `token_type`, `expires_at`, `project_id`, `email`, `plan_type` |
| `antigravity` | `apikey` | `api_key`, `base_url` | — |
| `grok` | `apikey` | `api_key` | `base_url` |
| `grok` | `oauth` | `access_token` | `refresh_token`, `id_token`, `token_type`, `expires_at`, `client_id`, `scope`, `email`, `sub`, `team_id`, `subscription_tier`, `entitlement_status`, `base_url` |
| `kimi` | `apikey` | `api_key`, `account_mode`, `api_protocol` | `base_url`, `api_base_urls` |
| `zhipu` | `apikey` | `api_key`, `account_mode`, `api_protocol` | `base_url`, `api_base_urls`, `zhipu_organization`, `zhipu_project` |
| `deepseek` | `apikey` | `api_key`, `account_mode`, `api_protocol` | `base_url`, `api_base_urls` |
| `minimax` | `apikey` | `api_key`, `account_mode`, `api_protocol` | `base_url`, `api_base_urls` |
| `opencode_go` | `apikey` | `api_key`, `account_mode`, `api_protocol` | `base_url`, `api_base_urls`, `protocol_rules` |

## 条件约束

- Bedrock `auth_mode=sigv4` 时 access key ID 和 secret access key 必填；`auth_mode=api_key` 时 api_key 必填。
- Service Account JSON 必须可解析，且包含 `project_id`、`client_email` 和合法 PEM `private_key`；外层 project/client_email 与 JSON 不一致时拒绝。
- CN 平台 `account_mode` 只允许平台当前支持的 `payg/coding/zen/go`；`api_protocol` 只允许 `chat_completions/anthropic/responses/adaptive` 中该平台支持的组合。
- `api_protocol=adaptive` 时 `api_base_urls` 必须覆盖平台要求的协议入口，且每个 URL 单独执行 SSRF 校验。
- OAuth `expires_at` 接受当前系统支持的 Unix 秒或规范化字符串形式，保存前统一归一化。
- 未在本表中的 platform/type 组合返回 `ACCOUNT_KIND_NOT_SUPPORTED`；即使管理员错误配置为允许，也不能绕过服务端注册表。

## 响应规则

供应商响应不返回 credentials。可返回 `credential_masked`、`has_credentials` 和经过白名单筛选的非敏感配置摘要。完整凭据仅在创建/更新请求中单向进入系统。

