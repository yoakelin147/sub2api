## Purpose

为上游账号建立供应商归属和审核状态，并在持久化查询层保证跨供应商数据隔离。

## ADDED Requirements

### Requirement: 账号可归属于一个供应商

系统 SHALL 为账号提供 nullable supplier_id、supplier_external_id 和 review_status。历史账号和管理员直接创建账号 MUST 保持 supplier_id 为空且 review_status 为 approved。

#### Scenario: 供应商创建账号
- **WHEN** active 供应商提交合法账号
- **THEN** 系统 MUST 从认证上下文写入 supplier_id
- **THEN** 默认 MUST 创建为 pending、inactive、schedulable=false 且无分组；管理员配置可信免审和同平台分组时，创建为 approved、active、schedulable=true 且绑定预设分组

#### Scenario: 客户端伪造 supplier_id
- **WHEN** 供应商请求正文、query 或 header 中提供其他 supplier_id
- **THEN** 系统 MUST 拒绝该字段或完全不允许其进入输入模型
- **THEN** 数据归属 MUST 仍为认证上下文中的供应商

#### Scenario: 历史账号兼容
- **WHEN** 升级前存在平台账号
- **THEN** 迁移后该账号 MUST 保持 supplier_id 为空并继续按现有规则调度

### Requirement: 供应商外部编号在租户内唯一

非空 supplier_external_id SHALL 在同一供应商的未删除账号中唯一，但不同供应商可使用相同外部编号。

#### Scenario: 同一供应商重复外部编号
- **WHEN** 同一供应商创建第二个相同 external_id 的未删除账号
- **THEN** 系统 MUST 返回 409 `SUPPLIER_EXTERNAL_ID_EXISTS`

#### Scenario: 不同供应商使用相同外部编号
- **WHEN** 两个供应商分别创建相同 external_id 的账号
- **THEN** 两次创建 MUST 均可成功

### Requirement: 所有供应商资源访问均按租户查询

供应商账号 Get、List、Update、Delete、Test 和批量操作 MUST 在同一数据库查询中绑定 supplier_id，MUST NOT 先全局读取再由 handler 判断所有权。

#### Scenario: 跨租户读取
- **WHEN** 供应商 A 请求供应商 B 的账号 ID
- **THEN** 系统 MUST 返回 404 `SUPPLIER_ACCOUNT_NOT_FOUND`
- **THEN** 响应 MUST NOT 表明该 ID 是否真实存在

#### Scenario: 跨租户写入
- **WHEN** 供应商 A 尝试修改、测试、暂停或删除供应商 B 的账号
- **THEN** 系统 MUST 返回 404
- **THEN** 目标账号 MUST 保持不变

#### Scenario: 批量请求混入越权 ID
- **WHEN** 批量写请求包含不属于当前供应商的账号 ID
- **THEN** 系统 MUST 整批拒绝
- **THEN** 当前供应商自己的账号也 MUST NOT 被部分更新

### Requirement: 审核状态与运行状态相互独立

review_status SHALL 只表示平台对凭据配置的审核结论；status 和 schedulable SHALL 保持现有运行时语义。

#### Scenario: 待审核账号
- **WHEN** 账号 review_status 为 pending
- **THEN** 账号 MUST 为 inactive 且 schedulable=false
- **THEN** 调度器 MUST NOT 选择该账号

#### Scenario: 驳回账号
- **WHEN** 管理员拒绝账号并填写 review_note
- **THEN** review_status MUST 变为 rejected
- **THEN** 账号 MUST 保持不可调度
- **THEN** 供应商 MUST 能看到经过安全处理的驳回原因

#### Scenario: 审核通过
- **WHEN** 管理员通过账号并配置合法分组
- **THEN** 系统 MUST 在事务内写入 approved、分组和目标运行状态
- **THEN** 系统 MUST 发布现有调度 outbox 更新

### Requirement: 凭据身份变化触发重新审核

默认需要审核的供应商修改上游 Key、Base URL、代理或其他连接凭据字段时，系统 MUST 自动将账号置为 pending、disabled、schedulable=false。免审供应商的已审核账号修改后保持原运行状态；单独修改名称 MUST NOT 触发重新审核。需审核时供应商不得修改备注等非连接配置。

#### Scenario: 修改 API Key
- **WHEN** 供应商更新已审核账号的 api_key
- **THEN** 系统 MUST 保存新凭据并立即停止该账号调度
- **THEN** review_status MUST 变为 pending

#### Scenario: 仅修改名称
- **WHEN** 供应商只修改 name
- **THEN** 原审核状态和调度状态 MUST 保持不变
