# 供应商账号管理实施任务

## 2026-09-22 追加：供应商接入操作与 API 指南

- [x] 对照管理员 Antigravity 接入及服务端凭据契约核对正确接入方式
- [x] 账号添加默认普通表单，高级 JSON 保留，修复类型切换、缺失地址模板及必填校验
- [x] 服务端凭据错误改为表单中文反馈，编辑过期时间复用本地时区格式化
- [x] 系统令牌页增加网页操作、令牌验证、单个/批量添加、列表查询和错误处理指南
- [x] 修复连接测试将 HTTP 200 误当作测试成功的问题
- [x] 完成定向测试、PC 实际组件预览与文档更新
- [x] 完成最终 Docker 构建与应用重建，容器 healthy

## 2026-09-22 追加：管理员页面布局与分页

- [x] 供应商全宽列表与详情分开，详情使用三个标签页
- [x] 接入成员服务端分页，保留账号分页，支持条数调整并显示真实总数
- [x] 表格限高、固定表头，审核配置使用弹窗
- [x] 验证筛选/翻页清空选择、末页回退及乱序响应保护
- [x] 更新治理规格和验收记录，完成 PC 浏览器验收并部署本地 Docker

## 0. 需求与接口冻结

- [x] 0.1 建立能力地图、需求总览、提案、设计和分能力规格
  - 验收：覆盖租户、账号归属、系统令牌、供应商操作、管理员治理五个模块
  - 验收：明确完整交付范围、非目标、默认决策和完成定义
  - 验证：逐项检查 README、proposal、design 与 specs 之间无冲突

- [x] 0.2 从现有账号实现冻结完整供应商凭据契约
  - 验收：覆盖 `apikey`、`upstream`、`setup-token`、`oauth`、`bedrock` 和 `service_account` 的有效 platform/type 组合
  - 验收：从现有表单、服务校验和测试夹具整理每种类型的必填字段、可选字段、Base URL 和敏感字段
  - 验收：Composite、影子账号和交互式 OAuth/SSO/密码/Cookie 换票接口保持管理员专属
  - 依赖：0.1
  - 预计：0.5–1 天

## 1. 数据模型基础

- [x] 1.1 新增 Supplier Ent schema 和领域模型
  - 描述：实现供应商 code、name、status、notes、allowed_account_kinds、令牌摘要元数据和软删除
  - 验收：code 对未删除供应商唯一；默认允许类型为空；状态只接受 active/disabled
  - 验证：Supplier schema/service/repository 单元测试
  - 可能文件：`backend/ent/schema/supplier.go`、`backend/internal/service/supplier.go`、`backend/internal/repository/supplier_repo.go`
  - 依赖：0.2
  - 预计：1–1.5 天

- [x] 1.2 增加 users.supplier_id 与 supplier 角色
  - 描述：扩展领域常量、User schema、service/DTO 映射和角色校验
  - 验收：supplier 必须绑定供应商；admin/user 不能绑定供应商；管理员不能把自己改成 supplier
  - 验证：用户创建/更新、角色约束和现有 admin/user 回归测试
  - 可能文件：`backend/internal/domain/constants.go`、`backend/ent/schema/user.go`、`backend/internal/service/user.go`、`backend/internal/handler/admin/user_handler.go`
  - 依赖：1.1
  - 预计：1–1.5 天

- [x] 1.3 增加 accounts 供应商归属与审核字段
  - 描述：新增 supplier_id、external_id、review 字段、索引和 Ent edge
  - 验收：历史账号默认 approved 且 supplier_id 为空；租户内 external_id 唯一；不同供应商可重复
  - 验证：schema、repository 和迁移集成测试
  - 可能文件：`backend/ent/schema/account.go`、`backend/internal/service/account.go`、`backend/internal/repository/account_repo.go`
  - 依赖：1.1
  - 预计：1.5–2 天

- [x] 1.4 编写增量 SQL 迁移并生成 Ent/Wire 代码
  - 描述：创建 suppliers 表，扩展 users/accounts，回填历史审核状态并增加约束/索引
  - 验收：空库和已有数据均可升级；重复执行安全；不修改历史账号行为
  - 验证：`cd backend && go generate ./ent && go generate ./cmd/server`；迁移测试通过
  - 可能文件：`backend/migrations/239_add_suppliers.sql`、`backend/migrations/*supplier*_test.go`、Ent/Wire 生成文件
  - 依赖：1.1、1.2、1.3
  - 预计：1–1.5 天

### Checkpoint A：数据基础

- [x] 新旧数据库迁移通过
- [x] 现有管理员、用户和历史账号查询无行为变化
- [x] Ent/Wire 生成结果已提交且无非预期 diff

## 2. 角色与认证隔离

- [x] 2.1 给现有普通用户路由增加显式角色守卫
  - 描述：盘点所有 JWT-only 路由，用户业务只允许 admin/user，supplier 只保留个人安全接口
  - 验收：supplier 无法访问用户 Key、余额、订阅、支付和使用记录；admin/user 行为不变
  - 验证：路由矩阵单元测试覆盖 admin/user/supplier 三种角色
  - 可能文件：`backend/internal/server/middleware/role_guard.go`、`backend/internal/server/routes/user.go`、`backend/internal/server/routes/payment.go`、对应测试
  - 依赖：1.2
  - 预计：1.5–2 天

- [x] 2.2 实现供应商 JWT 认证上下文
  - 描述：为 supplier JWT 加载用户和供应商，校验两者状态并写入 supplier_id/actor/auth_method
  - 验收：未绑定、已禁用或已删除供应商均被拒；active 成员可访问 supplier 路由
  - 验证：中间件成功、禁用、错误角色和缺失供应商测试
  - 可能文件：`backend/internal/server/middleware/supplier_auth.go`、`backend/internal/server/middleware/context.go`、对应测试
  - 依赖：1.1、1.2
  - 预计：1–1.5 天

- [x] 2.3 实现供应商系统令牌生命周期
  - 描述：生成 selector/secret、只保存摘要、查询掩码、重新生成、撤销和最后使用时间
  - 验收：完整令牌只返回一次；数据库没有明文；旧令牌轮换后立即失效
  - 验证：生成熵、解析、摘要、常量时间验证、轮换和撤销测试
  - 可能文件：`backend/internal/service/supplier_token.go`、`backend/internal/repository/supplier_repo.go`、对应测试
  - 依赖：1.1
  - 预计：1–1.5 天

- [x] 2.4 把供应商令牌接入统一认证中间件
  - 描述：`x-api-key` 识别 supplier token，与 supplier JWT 归一为同一认证上下文
  - 验收：令牌只能访问 supplier 账号路由；不能访问管理员/普通用户/令牌管理接口
  - 验证：完整权限矩阵和禁用供应商即时失效测试
  - 可能文件：`backend/internal/server/middleware/supplier_auth.go`、`backend/internal/server/routes/supplier.go`、对应测试
  - 依赖：2.2、2.3
  - 预计：1–1.5 天

### Checkpoint B：认证边界

- [x] admin/user/supplier 三角色路由矩阵全部通过
- [x] supplier JWT 与 supplier token 生成相同 supplier_id 上下文
- [x] 管理员 API Key、用户模型 Key 和供应商令牌不能互相替代

## 3. 账号租户服务

- [x] 3.1 实现 SupplierAccountRepository 作用域查询
  - 描述：增加 CreateOwned/GetOwned/ListOwned/UpdateOwned/DeleteOwned，不暴露全局账号读取
  - 验收：每条资源 SQL/Ent 查询均绑定 supplier_id；跨租户统一返回 not found
  - 验证：两供应商集成测试覆盖读、写、删、测和批量混入越权 ID
  - 可能文件：`backend/internal/service/supplier_account.go`、`backend/internal/repository/account_repo.go`、对应测试
  - 依赖：1.3、1.4
  - 预计：2–2.5 天

- [x] 3.2 实现供应商账号类型注册表和严格凭据校验
  - 描述：按 platform/type 声明允许字段、必填字段、长度、规范化和 Base URL 策略
  - 验收：未知字段、未授权类型、非法 URL 和敏感高级配置均被拒绝
  - 验证：每个已支持类型的表驱动测试和 SSRF 边界测试
  - 可能文件：`backend/internal/service/supplier_account_kinds.go`、`backend/internal/service/supplier_account_validation.go`、对应测试
  - 依赖：0.2
  - 预计：2–3 天

- [x] 3.3 实现供应商账号单个 CRUD
  - 描述：创建、列表、详情、有限字段更新、暂停、启用和软删除，复用现有账号规范化及 outbox
  - 验收：新账号 pending/inactive；响应不含 credentials；凭据变化自动重新审核
  - 验证：handler/service 单元测试及数据库集成测试
  - 可能文件：`backend/internal/handler/supplier/account_handler.go`、`backend/internal/service/supplier_account_service.go`、`backend/internal/server/routes/supplier.go`、对应测试
  - 依赖：2.4、3.1、3.2
  - 预计：2–3 天

- [x] 3.4 实现供应商账号测试接口
  - 描述：只测试已归属账号，复用现有 AccountTestService，不允许客户端传入临时任意 URL
  - 验收：跨租户无法测试；测试结果脱敏；测试不能隐式批准或启用账号
  - 验证：成功、上游失败、SSRF、跨租户和超时测试
  - 可能文件：`backend/internal/handler/supplier/account_handler.go`、`backend/internal/service/supplier_account_service.go`、对应测试
  - 依赖：3.3
  - 预计：1–1.5 天

- [x] 3.5 实现批量提交和幂等
  - 描述：最多 500 条/2 MiB，要求 Idempotency-Key，逐项校验结果，external_id 唯一
  - 验收：相同请求安全重放；同键异体拒绝；部分格式错误不影响合法项；基础设施错误整批失败
  - 验证：并发重试、正文冲突、重复 external_id、超限和部分失败测试
  - 可能文件：`backend/internal/handler/supplier/account_batch_handler.go`、`backend/internal/service/supplier_account_batch.go`、对应测试
  - 依赖：3.3
  - 预计：2–3 天

### Checkpoint C：供应商账号 API

- [x] 使用两家供应商数据跑完全部跨租户场景
- [x] 凭据未出现在响应、普通日志和审计正文
- [x] 新账号审核前无法进入任一调度查询
- [x] API 契约和错误码冻结

## 4. 管理员治理

- [x] 4.1 实现管理员供应商 CRUD 与成员管理 API
  - 描述：创建/编辑/禁用供应商，配置允许类型，创建/绑定/解绑成员
  - 验收：角色与 supplier_id 原子更新；供应商默认无提交权限；禁用即时阻断认证
  - 验证：admin handler/service 集成测试和审计断言
  - 可能文件：`backend/internal/handler/admin/supplier_handler.go`、`backend/internal/service/admin_supplier.go`、`backend/internal/server/routes/admin.go`、对应测试
  - 依赖：1.4、2.1
  - 预计：2–2.5 天

- [x] 4.2 实现管理员供应商令牌管理 API
  - 描述：管理员查询脱敏状态、生成/轮换和紧急撤销供应商令牌
  - 验收：管理员操作进入审计；完整令牌只在生成响应显示一次
  - 验证：admin JWT 和 admin API Key 行为按现有敏感操作策略测试
  - 可能文件：`backend/internal/handler/admin/supplier_token_handler.go`、`backend/internal/service/admin_supplier.go`、对应测试
  - 依赖：2.3、4.1
  - 预计：1 天

- [x] 4.3 实现账号审核、拒绝和批量暂停 API
  - 描述：在事务内处理 review、分组、运行状态和调度 outbox
  - 验收：批次必须全部属于目标供应商；混合渠道/分组校验复用现有逻辑；失败不部分提交
  - 验证：事务回滚、跨供应商混入、非法分组、批准后调度可见测试
  - 可能文件：`backend/internal/handler/admin/supplier_account_handler.go`、`backend/internal/service/admin_supplier_account.go`、对应测试
  - 依赖：3.1、4.1
  - 预计：2–3 天

- [x] 4.4 扩展账号和供应商审计
  - 描述：新增 supplier JWT/token auth_method 和供应商相关 audit action，复用递归敏感字段脱敏
  - 验收：审计可定位 supplier/actor/account/request；任何凭据字段均为 `***`
  - 验证：审计中间件和批量正文脱敏测试
  - 可能文件：`backend/internal/service/audit_log.go`、`backend/internal/server/middleware/audit_log.go`、对应测试
  - 依赖：2.4、3.3、4.3
  - 预计：1–1.5 天

## 5. 前端垂直切片

- [x] 5.1 扩展角色类型、登录跳转和路由守卫
  - 描述：增加 supplier role、requiresSupplier、三角色默认页和错误重定向
  - 验收：supplier 登录进入 `/supplier/accounts`；无法打开 admin/user 页面；其他角色行为不变
  - 验证：router guards、setup redirect、auth store 单元测试
  - 可能文件：`frontend/src/types/index.ts`、`frontend/src/router/index.ts`、`frontend/src/router/meta.d.ts`、`frontend/src/stores/auth.ts`
  - 依赖：2.1、2.2
  - 预计：1–1.5 天

- [x] 5.2 实现供应商专用布局和导航
  - 描述：只提供供应账号、系统令牌、个人安全和退出入口
  - 验收：无管理员或普通用户功能入口；桌面和移动端可用；键盘导航可达
  - 验证：组件测试、路由手动检查和基本无障碍检查
  - 可能文件：`frontend/src/components/supplier/SupplierLayout.vue`、`frontend/src/components/supplier/SupplierSidebar.vue`、i18n 文件、对应测试
  - 依赖：5.1
  - 预计：1.5–2 天

- [x] 5.3 实现供应商账号列表与单个维护
  - 描述：分页筛选、创建、编辑、暂停、测试和删除；只使用 supplier API client
  - 验收：展示审核/运行状态和脱敏错误；表单不出现管理员字段；凭据不回填
  - 验证：API mock 组件测试、类型检查和真实后端手动流程
  - 可能文件：`frontend/src/api/supplier/accounts.ts`、`frontend/src/views/supplier/SupplierAccountsView.vue`、`frontend/src/components/supplier/SupplierAccountForm.vue`、对应测试
  - 依赖：3.3、3.4、5.2
  - 预计：2.5–3.5 天

- [x] 5.4 实现供应商批量导入和系统令牌页面
  - 描述：JSON、CSV、逐行文本批量输入统一转换、逐项结果、令牌状态、生成、复制一次和撤销
  - 验收：批量错误可定位到输入项；令牌不写 localStorage；刷新后只显示掩码
  - 验证：批量边界、复制失败、令牌轮换和页面刷新测试
  - 可能文件：`frontend/src/components/supplier/SupplierBatchImportDialog.vue`、`frontend/src/views/supplier/SupplierAccessTokenView.vue`、API client、对应测试
  - 依赖：2.3、3.5、5.2
  - 预计：2–3 天

- [x] 5.5 实现管理员供应商列表和详情
  - 描述：供应商 CRUD、允许类型、成员、令牌状态、账号统计和待审核账号
  - 验收：管理员可以完成供应商开通到账号审核的完整流程；敏感值只显示一次
  - 验证：SuppliersView 组件测试和真实后端手动流程
  - 可能文件：`frontend/src/api/admin/suppliers.ts`、`frontend/src/views/admin/SuppliersView.vue`、`frontend/src/components/admin/supplier/SupplierDetail.vue`、对应测试
  - 依赖：4.1、4.2、4.3、5.1
  - 预计：3–4 天

- [x] 5.6 在现有管理员账号页增加供应商来源提示
  - 描述：仅增加来源标签和跳转，不把供应商治理逻辑塞入现有大页面
  - 验收：历史账号显示平台自有；供应商账号显示供应商名称；现有筛选和操作不回归
  - 验证：AccountsView 定向组件测试
  - 可能文件：`frontend/src/views/admin/AccountsView.vue`、`frontend/src/types/index.ts`、i18n/测试
  - 依赖：4.1、5.5
  - 预计：0.5–1 天

### Checkpoint D：端到端业务流

- [x] 管理员创建供应商和成员
- [x] 供应商登录并生成系统令牌
- [x] 网页和 API 分别提交账号
- [x] 管理员批量审核并绑定分组
- [x] 审核后账号进入调度，修改凭据后立即退出调度
- [x] 第二家供应商始终无法看到第一家数据

## 6. 发布准备

- [x] 6.1 编写供应商 API 文档和接入样例
  - 描述：认证、端点、请求/响应、错误码、幂等、批量限制、轮换和安全要求
  - 验收：供应商仅凭文档可完成 token 验证、单个提交、批量提交和查询
  - 验证：文档中的 curl/PowerShell 示例在 Docker 环境执行成功
  - 可能文件：`docs/SUPPLIER_API_CN.md`、`README_CN.md`
  - 依赖：Checkpoint C
  - 预计：1 天

- [x] 6.2 完成安全与回归测试
  - 描述：覆盖 IDOR、SSRF、令牌泄漏、角色扩散、批量 DoS、幂等并发和日志脱敏
  - 验收：所有规格场景有自动化测试或明确手动验收记录
  - 验证：后端 unit/integration/full test、golangci-lint、前端 lint/typecheck/test/build
  - 可能文件：各模块测试文件、`openspec/.../verification.md`
  - 依赖：Checkpoint D、4.4
  - 预计：2–3 天

- [x] 6.3 Docker 灰度与回滚演练
  - 描述：用现有 Compose 构建，创建两家测试供应商，验证迁移、权限、上传、审核、调度和撤销
  - 验收：三个容器 healthy；旧业务回归；供应商禁用/撤销立即生效；回滚不删除新增数据
  - 验证：记录镜像版本、数据库备份点、验收请求和回滚结果
  - 可能文件：`openspec/changes/add-supplier-account-management/verification.md`
  - 依赖：6.2
  - 预计：1–1.5 天

## 7. 总体估算与顺序

| 阶段 | 预计净开发日 | 必须顺序 |
|---|---:|---|
| 需求样例冻结 | 0.5–1 | 第一 |
| 数据模型 | 4.5–6.5 | 第二 |
| 认证与角色隔离 | 4.5–6.5 | 第三 |
| 供应商账号服务/API | 11–16 | 第四 |
| 管理员治理后端 | 6–8 | 第五，可与部分前端并行 |
| 前端 | 12–17 | API 契约冻结后 |
| 文档、测试、灰度 | 4–5.5 | 最后 |
| **合计** | **约 44–64 人日** | 单人约 9–13 周 |

估算包含全部现有可独立供应凭据类型、JSON/CSV/文本导入、安全和回归覆盖；不包含供应商结算、免审上线、Composite/影子账号和全账号凭据应用层加密。

## 8. 工作安排建议

按依赖顺序执行：

```text
0.2 真实凭据样例确认
  → 1.x 数据模型
  → Checkpoint A
  → 2.x 认证边界
  → Checkpoint B
  → 3.x 供应商账号 API
  → Checkpoint C / API 契约冻结
  → 4.x 管理员治理后端
  → 5.x 前端
  → Checkpoint D
  → 6.x 安全回归、文档和灰度
```

可并行项：

- 2.3 令牌服务可与 2.1 路由角色加固并行。
- Checkpoint C 后，5.3/5.4 供应商前端可与 4.1/4.2 管理员治理后端并行。
- 5.5 管理员前端在 4.1–4.3 接口冻结后开始。
- 文档样例可在 API 契约冻结后提前编写。
