## Purpose

为供应商提供一个可撤销、可轮换且最小权限的长期系统令牌，用于机器对机器账号同步。

## ADDED Requirements

### Requirement: 每个供应商只有一个有效系统令牌

系统 SHALL 允许 supplier JWT 或管理员为供应商生成、重新生成和撤销一个有效系统令牌。系统令牌 MUST 使用独立 supplier 前缀，MUST NOT 与管理员 API Key 或模型调用 Key 混用。

#### Scenario: 首次生成
- **WHEN** 供应商成员通过 JWT 或管理员请求生成令牌
- **THEN** 系统 MUST 使用密码学安全随机数生成完整令牌
- **THEN** 完整令牌 MUST 只在本次响应显示一次

#### Scenario: 重新生成
- **WHEN** 有权人员重新生成令牌
- **THEN** 旧令牌 MUST 立即失效
- **THEN** 新令牌 MUST 只显示一次

#### Scenario: 撤销令牌
- **WHEN** 有权人员撤销令牌
- **THEN** 后续使用该令牌的请求 MUST 返回 401

### Requirement: 数据库不得保存完整令牌

系统 SHALL 将令牌拆分为公开 selector 与秘密 secret，只保存 selector、secret 摘要和脱敏前缀。secret MUST 至少具有 256 bit 随机熵。

#### Scenario: 查询令牌状态
- **WHEN** 管理员或供应商成员查询令牌状态
- **THEN** 系统 MUST 只返回 exists、masked_key、created_at 和 last_used_at
- **THEN** 系统 MUST NOT 从数据库恢复完整令牌

#### Scenario: 验证令牌
- **WHEN** 请求携带格式合法的供应商令牌
- **THEN** 系统 MUST 通过 selector 定位供应商
- **THEN** 系统 MUST 使用常量时间比较验证 secret 摘要

### Requirement: 令牌权限固定为供应账号操作

供应商系统令牌 MUST 只能访问 `/api/v1/supplier/me` 和供应商账号操作接口，MUST NOT 管理令牌自身、成员、普通用户、支付或管理员资源。

#### Scenario: 令牌请求管理员路由
- **WHEN** 供应商令牌请求 `/api/v1/admin/*`
- **THEN** 系统 MUST 返回 401 或 403

#### Scenario: 令牌轮换自身
- **WHEN** 供应商令牌请求 access-token regenerate 或 delete
- **THEN** 系统 MUST 返回 403

### Requirement: 供应商状态控制所有认证方式

供应商状态 SHALL 在每次 supplier JWT 和系统令牌认证时检查。

#### Scenario: 禁用供应商后使用旧 JWT
- **WHEN** disabled 供应商成员使用尚未过期 JWT 请求供应商接口
- **THEN** 系统 MUST 返回 403 `SUPPLIER_DISABLED`

#### Scenario: 禁用供应商后使用令牌
- **WHEN** disabled 供应商使用正确系统令牌请求供应商接口
- **THEN** 系统 MUST 返回 403 `SUPPLIER_DISABLED`

### Requirement: 令牌不得进入日志或审计正文

系统 SHALL 对 supplier token 和上游 credentials 执行日志、错误响应和审计脱敏。

#### Scenario: 认证成功或失败
- **WHEN** 任意供应商令牌认证请求完成
- **THEN** 审计仅可记录 masked credential
- **THEN** 普通日志和错误响应 MUST NOT 包含完整令牌
