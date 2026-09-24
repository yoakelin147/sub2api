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
- **THEN** 系统 MUST 按供应商审核策略创建账号；默认待审核

#### Scenario: 未允许类型
- **WHEN** 供应商提交未在 allowed_account_kinds 中的 platform/type
- **THEN** 系统 MUST 返回 403 `ACCOUNT_KIND_NOT_ALLOWED`

#### Scenario: 未知凭据字段
- **WHEN** credentials 包含该 platform/type 未声明的字段
- **THEN** 系统 MUST 返回 422 `INVALID_CREDENTIALS`
- **THEN** 系统 MUST NOT 静默保存未知字段

### Requirement: 供应商只能设置受控账号参数

默认需审核时，供应商仅能提交账号名称、类型、自有 `external_id`、库存 `proxy_id`、连接必需凭据和可选主动停用；备注、运行参数、分组、模型映射及 `extra` MUST 被拒绝。免审时 MAY 提交备注、过期时间、并发、优先级、负载因子、过期停调、合法模型映射及逐项校验的 `extra`，创建或更新时 `group_ids` 仅能包含管理员预设的当前平台分组（可多选）；未提供时使用首个合法预设分组。任何模式均 MUST NOT 写 supplier_id、rate_multiplier、schedulable、review_status。倍率固定为 1。`proxy_id` 必须引用启用且未过期的库存代理；仅当平台不存在可用库存代理时允许无代理直连。
供应商可主动停用；需审核供应商不能自行恢复管理员暂停的账号，免审供应商可恢复自身已批准账号，任何供应商都不得启用待审核账号。

#### Scenario: 请求包含禁止字段
- **WHEN** 供应商请求包含任一禁止字段
- **THEN** 系统 MUST 返回 422
- **THEN** 禁止字段 MUST NOT 被写入

#### Scenario: 未配置有效代理
- **WHEN** 库存中存在有效代理但供应商新建或编辑账号时未提供该代理
- **THEN** 系统 MUST 返回 422 `SUPPLIER_PROXY_REQUIRED`
- **THEN** 账号 MUST NOT 被保存或启用

#### Scenario: 库存无有效代理
- **WHEN** 平台无有效库存代理且供应商不指定代理
- **THEN** 系统 MAY 使用默认无代理连接，不得接受自定义代理地址

#### Scenario: 免审供应商提交授权分组和模型映射
- **WHEN** 免审供应商创建账号并携带该平台预设分组、合法模型映射和经过校验的高级选项
- **THEN** 系统 MUST 绑定授权分组、固定倍率 1 并允许账号按配置运行
- **THEN** 不匹配分组、越权字段及不合法的模型映射 MUST 返回 422

免审供应商网页表单 MUST 按当前平台/账号类型提供有用途说明的可选参数控件，不得要求手写 `extra` JSON；系统令牌教程 MUST 列出适用范围、`extra` 字段名和允许的取值。仅 OpenAI 可设置 `openai_compact_mode`，仅 Anthropic API Key 可设置 `web_search_emulation`，`upstream_request_id_header` 可用于所有类型。默认设置无需显式传递。

#### Scenario: 更新非敏感模型配置
- **WHEN** 免审供应商仅更新自身账号的模型映射或协议规则
- **THEN** 系统 MUST 保留原始隐藏 Key、Token 与密码，不要求重新提交敏感连接凭据
- **THEN** 规则 MUST 按账号类型校验，其他凭据修改仍须提交完整连接凭据

### Requirement: 账号凭据永不回显

OAuth / Setup Token 类型的创建及凭据更新 MUST 接收 token、网页登录 `email` 与 `password`。密码 MUST 使用稳定密钥加密保存；缺少固定密钥时拒绝保存。供应商不得读取已保存密码，且其他账号类型不得提交密码。

供应商后台 SHALL 复用管理员 OAuth 授权界面，为已获准的 OpenAI、Claude、Gemini、Antigravity、Grok OAuth 及 Claude Setup Token 提供授权码换票；Claude Cookie、Grok SSO 和平台启用的 Grok 密码登录也可换票。必须选择平台有效库存代理；授权码会话 MUST 绑定当前供应商和代理，并校验适用的 OAuth state，不得接受换票时覆盖代理或重定向地址。换得的 Token SHALL 仅用于填充供应商本人的账号表单，仍需供应商提交账号并经过适用的审核；授权码、密码、Cookie、SSO 和 Token 不得写入审计正文。

供应商列表和详情 SHALL 使用独立白名单 DTO，不返回完整 credentials 或未审核的 extra；仅返回允许供应商自行编辑的非敏感高级选项。

#### Scenario: 查询刚创建的账号
- **WHEN** 供应商查询刚提交账号
- **THEN** 响应 MUST 只包含凭据掩码或 has_credentials
- **THEN** 响应 MUST NOT 包含完整 api_key、token、password、cookie 或 secret

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
- **THEN** 系统 MUST 返回 409 `IDEMPOTENCY_KEY_CONFLICT`

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

### Requirement: 供应商账号表单明确凭据要求和操作结果

账号表单 SHALL 默认按当前 platform/type 显示普通凭据输入项，保留高级 JSON 入口，复用统一凭据模板。必填字段和上游地址格式 SHALL 在提交前检查，服务端继续承担完整凭据、类型授权和 SSRF 校验。

#### Scenario: 切换账号类型
- **WHEN** 供应商切换平台或认证类型
- **THEN** 表单 MUST 显示新类型的字段与模板
- **THEN** 已填写凭据被替换前 MUST 提示确认，取消不得清除原值

#### Scenario: Antigravity API Key 缺少地址
- **WHEN** 供应商仅填写 api_key，未填写完整 base_url
- **THEN** 表单 MUST 指明缺失字段或地址格式问题并阻止提交
- **THEN** 系统 MUST 明确上游地址不等于平台 API 地址或管理员网络代理

#### Scenario: 上游测试通过 HTTP 200 返回失败事件
- **WHEN** 账号测试返回 error 事件，或未返回完整的 test_complete/success=true
- **THEN** 界面 MUST NOT 提示连接测试通过
- **THEN** 提交成功、测试通过、审核通过 SHALL 分别表达，不互相代替
