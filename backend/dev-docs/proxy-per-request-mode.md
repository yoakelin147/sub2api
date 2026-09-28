# HTTP/HTTPS 代理每请求独立连接模式

管理员代理配置中的 `connection_mode` 默认为 `reuse`。设为 `per_request` 时，通用 HTTP 上游客户端对该代理的每次请求单独建连、禁用 HTTP/2 多路复用并在响应体关闭后释放连接。该模式仅允许 HTTP/HTTPS 正向代理；不是“无隧道”协议。旧代理和 SOCKS5 仍维持原有行为。

代理项目在该模式下需要记录两个大小写不敏感的请求头：

| 请求头 | 值 | 出现位置 |
| --- | --- | --- |
| `X-Client-Request-ID` | sub2api 平台入口请求 ID | HTTPS 目标的 CONNECT；HTTP 明文目标的代理请求 |
| `X-Sub2API-Attempt-ID` | 每次上游建连/尝试生成的 UUID | 同上 |

HTTPS 目标仍通过 CONNECT 建立端到端 TLS 隧道；代理仅能在 CONNECT 时看到这些头，不能解密后续业务请求。独立连接模式保证每次通用 HTTP 上游请求重新建 CONNECT；同一入口请求若重试，`X-Client-Request-ID` 可以相同，但 `X-Sub2API-Attempt-ID` 不同。代理应把两者与目标 host:port、入站/出口 IP、建连时间及状态一起记录，**不要记录 `Proxy-Authorization`**。

对于明文 HTTP 目标，请代理在转发给目标服务商前移除上述两个追踪头；标准代理不会自动移除自定义头。无平台请求 ID 的后台请求不会带这两个头。连接复用模式不保证向代理传递逐请求 ID。

当前独立连接和追踪头由通用 HTTP 上游转发客户端实现。使用专用 WebSocket、OAuth 或第三方客户端库的路径仍沿用各自连接策略，不应把代理 CONNECT 日志当作全部流量的逐请求账本。
