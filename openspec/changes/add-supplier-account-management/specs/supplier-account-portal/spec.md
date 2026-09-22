## Purpose

提供供应商网页和机器 API，使供应商只能以受控字段提交和维护自己的上游账号。

## ADDED Requirements

### Requirement: 网页和系统令牌共享同一账号服务

供应商 JWT 与供应商系统令牌 SHALL 进入同一 SupplierAccountService；不同认证方式 MUST 具有相同的数据范围和账号字段权限。

#### Scenario: 两种认证创建账号
- **WHEN** 同一供应商分别通过 JWT 和系统令牌提交合法账号
- **THEN** 两个账号 MUST 写入相同 supplier_id
- **THEN** 两个账号 MUST 遵循相同审核和字段限制

### Requirement: 供应商只能提交管理员允许的账号类型

系统 SHALL 根据 supplier.allowed_account_kinds 和服务端静态类型注册表验证 platform、type 和 credentials。

#### Scenario: 允许类型
- **WHEN** 供应商提交已启用的 platform/type 且 credentials 合法
- **THEN** 系统 MUST 创建待审核账号

#### Scenario: 未允许类型
- **WHEN** 供应商提交未在 allowed_account_kinds 中的 platform/type
- **THEN** 系统 MUST 返回 403 `ACCOUNT_KIND_NOT_ALLOWED`

#### Scenario: 未知凭据字段
- **WHEN** credentials 包含该 platform/type 未声明的字段
- **THEN** 系统 MUST 返回 422 `INVALID_CREDENTIALS`
- **THEN** 系统 MUST NOT 静默保存未知字段

### Requirement: 供应商不能设置运营字段

供应商创建和更新 DTO MUST NOT 包含 supplier_id、group_ids、proxy_id、priority、concurrency、load_factor、rate_multiplier、schedulable、review_status 或 extra。

#### Scenario: 请求包含禁止字段
- **WHEN** 供应商请求包含任一禁止字段
- **THEN** 系统 MUST 返回 422
- **THEN** 禁止字段 MUST NOT 被写入

### Requirement: 账号凭据永不回显

供应商列表和详情 SHALL 使用独立白名单 DTO，不返回完整 credentials 或 extra。

#### Scenario: 查询刚创建的账号
- **WHEN** 供应商查询刚提交账号
- **THEN** 响应 MUST 只包含凭据掩码或 has_credentials
- **THEN** 响应 MUST NOT 包含完整 api_key、token、cookie 或 secret

### Requirement: 批量创建有明确边界和逐项结果

批量创建 SHALL 最多接受 500 条且正文不超过 2 MiB。每项 SHALL 返回输入索引、external_id 和成功结果或结构化错误。

#### Scenario: 部分数据无效
- **WHEN** 500 条以内的批次包含合法和非法项
- **THEN** 合法项 MUST 创建成功
- **THEN** 非法项 MUST 返回逐项错误且不得落库

#### Scenario: 超过数量限制
- **WHEN** 批次包含超过 500 条
- **THEN** 系统 MUST 返回 413 `BATCH_TOO_LARGE`
- **THEN** 不得创建任何账号

### Requirement: 创建操作支持安全重试

单个和批量创建 MUST 要求 Idempotency-Key，并将请求正文哈希绑定到该键。

#### Scenario: 相同请求重试
- **WHEN** 同一供应商以相同幂等键重试相同正文
- **THEN** 系统 MUST 返回首次结果并标记 replayed
- **THEN** 账号数量 MUST 不增加

#### Scenario: 幂等键正文冲突
- **WHEN** 同一供应商以相同幂等键提交不同正文
- **THEN** 系统 MUST 返回 422

### Requirement: 自定义上游地址必须通过 SSRF 校验

供应商提供的 Base URL SHALL 默认只允许 HTTPS 和管理员显式允许的域名，并拒绝解析到私网或保留地址的目标及危险重定向。

#### Scenario: 云元数据地址
- **WHEN** 供应商提交指向 `169.254.169.254` 或等价解析结果的 Base URL
- **THEN** 系统 MUST 返回 422

#### Scenario: 白名单公网域名
- **WHEN** Base URL 使用 HTTPS、域名在允许列表且解析结果均为公网地址
- **THEN** 系统 MUST 允许保存

### Requirement: 供应商控制台仅展示供应商能力

supplier 角色 SHALL 使用独立路由和菜单，只显示供应账号、系统令牌及必要个人安全功能。

#### Scenario: supplier 登录跳转
- **WHEN** supplier 用户登录成功
- **THEN** 前端 MUST 跳转到 `/supplier/accounts`
- **THEN** 管理员和普通用户菜单 MUST 不显示

