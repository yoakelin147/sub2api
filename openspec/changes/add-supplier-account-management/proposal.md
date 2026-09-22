## Why

平台需要接入多家号池供应商，并允许供应商自行维护其提供的大量上游 Key。现有系统只有管理员和普通用户角色，账号没有供应商归属，管理员 API Key 又拥有完整系统权限，无法安全地向供应商开放账号管理能力。

## What Changes

- 新增供应商租户和 `supplier` 登录角色，一个供应商可拥有多个登录成员。
- 为账号增加供应商归属、供应商外部编号和独立审核状态；历史账号保持管理员所有并默认已审核。
- 新增供应商专用认证中间件，支持供应商 JWT 和单供应商系统令牌。
- 新增供应商账号 API 和独立控制台，只允许管理当前供应商自己的账号及有限字段。
- 新增管理员供应商管理、成员管理、令牌撤销和账号批量审核能力。
- 新提交或凭据被供应商修改的账号默认退出调度并进入待审核。
- 为批量提交增加数量/正文限制、幂等、租户内外部编号去重、审计和限流。
- 覆盖现有可独立供应的 `apikey`、`upstream`、`setup-token`、`oauth` 凭据包、`bedrock` 和 `service_account`；Composite、影子账号及交互式 OAuth 登录保持管理员专属。
- 网页批量导入支持 JSON、CSV 和逐行文本，供应商 API 使用 JSON 稳定契约。

## Capabilities

### New Capabilities

- `supplier-tenant`：供应商组织、成员、状态与角色隔离。
- `supplier-account-ownership`：账号供应商归属、审核状态与数据隔离。
- `supplier-access-token`：供应商系统令牌生命周期及机器认证。
- `supplier-account-portal`：供应商网页/API 账号操作。
- `supplier-admin-governance`：管理员供应商治理和账号审核。

### Modified Capabilities

- 现有 JWT 角色判断从 `admin/user` 扩展为显式的 `admin/user/supplier`。
- 现有普通用户路由增加角色守卫，避免把所有非管理员角色当作普通用户。
- 现有管理员账号响应增加可选供应商来源信息，但管理员权限和历史接口语义保持不变。

## Impact

- **数据库**：新增 `suppliers` 表；`users`、`accounts` 增加供应商字段；增加部分唯一索引和角色/归属约束。
- **后端**：新增 supplier repository/service/handler/routes/middleware；扩展用户和账号领域模型；复用现有账号创建、脱敏、幂等、审计和调度失效逻辑。
- **前端**：新增管理员供应商页面、供应商专用布局和账号页面；扩展登录后角色路由。
- **公共 API**：新增 `/api/v1/supplier/*`，不修改现有 `/api/v1/admin/*` 和网关接口。
- **安全**：供应商令牌只保存摘要；账号凭据不回显；Base URL 受限；所有资源访问在数据库查询中绑定 `supplier_id`。
- **上线**：采用增量迁移；供应商表为空时功能天然不生效，不额外增加 feature flag。

## Non-goals

- 供应商结算和财务系统。
- 供应商内部细粒度 RBAC。
- 多令牌并存；按用户给出的“系统访问令牌”模式，每个供应商保留一个可重新生成的有效令牌。
- 供应商自动配置平台调度参数或免审上线。
- 全账号凭据应用层加密迁移。
