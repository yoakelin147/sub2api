# Capability Map: Supplier Account Management

| Module ID | Responsibility | Depends on |
|---|---|---|
| `supplier-tenant` | 供应商实体、成员归属、角色与生命周期 | — |
| `supplier-account-ownership` | 账号归属、审核状态、租户级查询与写入隔离 | `supplier-tenant` |
| `supplier-access-token` | 单供应商系统令牌的签发、验证、轮换与撤销 | `supplier-tenant` |
| `supplier-account-portal` | 供应商网页和 API 的账号提交、维护、测试与批量导入 | `supplier-account-ownership`, `supplier-access-token` |
| `supplier-admin-governance` | 管理员创建供应商、配置准入、管理成员及审核账号 | `supplier-tenant`, `supplier-account-ownership` |

构建顺序：

```text
supplier-tenant
    ├── supplier-account-ownership
    │       ├── supplier-account-portal
    │       └── supplier-admin-governance
    └── supplier-access-token
            └── supplier-account-portal
```

接口契约必须在 `supplier-account-portal` 编码前冻结；数据库迁移、角色守卫和租户隔离测试必须先于任何供应商前端开发。

