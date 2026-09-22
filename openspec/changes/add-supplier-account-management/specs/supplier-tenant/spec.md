## Purpose

建立供应商组织和成员身份，使多个供应商能够共享现有登录基础设施，同时保持明确的数据租户边界。

## ADDED Requirements

### Requirement: 系统维护独立供应商实体

系统 SHALL 提供供应商实体，至少包含稳定唯一 code、名称、状态、备注、允许账号类型和软删除时间。供应商状态 MUST 仅为 active 或 disabled。

#### Scenario: 创建供应商
- **WHEN** 管理员提交唯一 code、有效名称和允许账号类型
- **THEN** 系统 MUST 创建 active 供应商
- **THEN** 系统 MUST 返回供应商信息且不生成隐式成员或账号

#### Scenario: 重复供应商 code
- **WHEN** 管理员创建与现有未删除供应商相同 code 的供应商
- **THEN** 系统 MUST 返回 409
- **THEN** 系统 MUST NOT 创建重复供应商

#### Scenario: 禁用供应商
- **WHEN** 管理员将供应商状态改为 disabled
- **THEN** 该供应商的成员 JWT 和系统令牌 MUST 无法继续访问供应商接口
- **THEN** 系统 MUST NOT 隐式删除或改变其历史账号

### Requirement: 供应商成员复用现有用户身份

系统 SHALL 新增 supplier 角色并允许一个供应商拥有多个成员。supplier 用户 MUST 绑定一个有效供应商；admin 和 user MUST NOT 绑定供应商。

#### Scenario: 创建供应商成员
- **WHEN** 管理员为 active 供应商创建 supplier 用户
- **THEN** 用户 MUST 能使用现有登录接口登录
- **THEN** 登录后 MUST 跳转到供应商控制台

#### Scenario: supplier 未绑定供应商
- **WHEN** 创建或更新用户为 supplier 但 supplier_id 为空或无效
- **THEN** 系统 MUST 返回 422
- **THEN** 用户记录 MUST NOT 进入不一致状态

#### Scenario: 普通角色携带供应商归属
- **WHEN** admin 或 user 用户携带非空 supplier_id
- **THEN** 系统 MUST 拒绝该写入

### Requirement: 新角色不得继承普通用户或管理员权限

系统 SHALL 对现有 JWT 路由使用显式角色允许列表，MUST NOT 把 supplier 当作普通 user。

#### Scenario: supplier 访问管理员接口
- **WHEN** supplier JWT 请求 `/api/v1/admin/*`
- **THEN** 系统 MUST 返回 403

#### Scenario: supplier 访问普通用户业务
- **WHEN** supplier JWT 请求用户 API Key、余额、订阅、支付或使用记录接口
- **THEN** 系统 MUST 返回 403

#### Scenario: supplier 访问账户安全资料
- **WHEN** supplier JWT 请求允许的个人资料、修改密码或 TOTP 接口
- **THEN** 系统 MUST 按现有身份安全规则允许操作

### Requirement: 供应商生命周期保留审计关联

供应商和成员 SHALL 使用软删除或禁用语义，历史账号和审计记录 MUST 保持原 supplier_id 和 actor 信息。

#### Scenario: 删除有历史账号的供应商
- **WHEN** 管理员删除已有供应账号的供应商
- **THEN** 系统 MUST 软删除供应商并阻止其认证
- **THEN** 历史账号与审计记录 MUST 保留供应商归属

