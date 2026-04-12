# API Station 开发规范

> sub2api fork + 多源特性吸收 | 负载均衡代理网关

## 1. 仓库定位

基于 [Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api) fork，吸收 [AmazingAng/auth2api](https://github.com/AmazingAng/auth2api) 和 [router-for-me/CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的防封与调度特性，构建生产级 AI 编码套餐负载均衡代理。

## 2. 分支策略

```
upstream   ── 纯净上游镜像，只做 fetch + fast-forward ──→
                \                         \
main        └── merge upstream ───────merge──→  (生产分支)
                \
feature/*    └── 各吸收任务独立分支 ──→ PR into main
```

### 分支规则

| 分支 | 用途 | 保护规则 |
|------|------|----------|
| `upstream` | 追踪 Wei-Shaw/sub2api 原始仓库 | 禁止手动 commit，只允许 `git fetch upstream && git merge upstream/main` |
| `main` | 生产就绪分支 | PR only，需通过测试 |
| `feature/*` | 特性吸收开发分支 | 命名: `feature/<来源>-<特性>`，例 `feature/auth2api-billing-header` |
| `fix/*` | 修复分支 | 命名: `fix/<描述>` |

### 上游同步流程

```bash
# 1. 添加上游 remote (首次)
git remote add upstream https://github.com/Wei-Shaw/sub2api.git

# 2. 定期同步 (建议每周或上游有重要更新时)
git fetch upstream
git checkout upstream && git merge upstream/main --ff-only
git checkout main && git merge upstream

# 3. 解决冲突 → 跑测试 → commit
# 冲突优先保留自己的定制，但要检查上游改动意图
```

## 3. 定制代码隔离原则

**核心原则: 定制代码尽量写在新文件中，现有文件只加最小胶水代码。**

### 改动分类

| 改动类型 | 放置策略 | 合并冲突风险 | 示例 |
|----------|----------|:----------:|------|
| 新增 Service | 新文件 | 极低 | `auth2api_cloaking_service.go` |
| 新增 Middleware | 新文件 + router 一行注册 | 极低 | `rpm_throttle_middleware.go` |
| 扩展现有请求链路 | 抽接口到新文件，调用点加一行 | 低 | `RequestNormalizer` 接口 |
| 扩展 TLS 指纹 | profiles 表加记录 | 低 | 不改 dialer 代码 |
| 修改核心调度逻辑 | 最小 diff + 标记注释 | 中 | `// api-station: begin` |

### 标记规范

对现有文件的修改必须用标记包裹:

```go
// api-station: begin - billing header injection (from auth2api)
...定制代码...
// api-station: end
```

这样上游合并时可以快速定位所有定制点。

### 新文件命名规范

```
backend/internal/service/
  apistation_cloaking_service.go      # 来自 auth2api 的 cloaking 特性
  apistation_fingerprint_service.go   # 来自 auth2api 的指纹算法
  apistation_throttle_service.go      # 来自 sub2api/CLIProxyAPI 的节流
  apistation_monitor_service.go       # 自研: 封号监控与告警

backend/internal/middleware/
  apistation_rpm_throttle.go          # RPM 自适应节流中间件

backend/internal/pkg/
  apistation/                         # api-station 专属工具包
    billing.go                        # billing header 构造
    fingerprint.go                    # 指纹算法
    version_drift.go                  # CC 版本漂移检测
```

前缀 `apistation_` 确保新文件在目录中聚集，且与上游文件零命名冲突。

## 4. 特性吸收来源与优先级

### 从 auth2api 吸收

| 优先级 | 特性 | 源文件 | 目标 |
|--------|------|--------|------|
| P0 | Claude Code 指纹算法 SHA256 精确复刻 | `auth2api/src/upstream/cloaking.ts:17-37` | `apistation/fingerprint.go` |
| P0 | billing header 注入 (x-anthropic-billing-header) | `auth2api/src/upstream/cloaking.ts:39-52` | `apistation/billing.go` |
| P0 | 分级 Cooldown (rate_limit/auth/server 差异化退避) | `auth2api/src/accounts/manager.ts:19-25` | `apistation_cooldown_service.go` |
| P1 | metadata.user_id 三元组 (device_id+account_uuid+session_id) | `auth2api/src/upstream/cloaking.ts:61-71` | `apistation_cloaking_service.go` |
| P1 | Session ID 随机 TTL (30-300min) | `auth2api/src/upstream/anthropic-api.ts:68-93` | `apistation_cloaking_service.go` |
| P1 | 动态 anthropic-beta header | `auth2api/src/upstream/anthropic-api.ts:14-30` | `apistation_cloaking_service.go` |
| P2 | 真实 Claude Code 客户端透传模式 | `auth2api/src/upstream/anthropic-api.ts:154-175` | middleware |

### 从 CLIProxyAPI 吸收

| 优先级 | 特性 | 源文件 | 目标 |
|--------|------|--------|------|
| P1 | Thinking Block 签名缓存 (3h TTL, SHA256) | `CLIProxyAPI/internal/cache/signature_cache.go` | `apistation_signature_cache.go` |
| P1 | 敏感词零宽字符混淆 | `CLIProxyAPI/internal/config/config.go:323-344` | `apistation_cloaking_service.go` |
| P1 | StabilizeDeviceProfile (OS/Arch 锁定) | `CLIProxyAPI/internal/config/config.go:150` | `apistation_cloaking_service.go` |
| P2 | Sticky 选择器 (20-60min 随机窗口) | `CLIProxyAPI/sdk/cliproxy/auth/selector.go` | `apistation_sticky_selector.go` |
| P3 | 多存储后端抽象 (PG/Git/S3) | `CLIProxyAPI/cmd/server/main.go:135-259` | 评估后决定 |

### 自研

| 优先级 | 特性 | 目标 |
|--------|------|------|
| P0 | CC 版本漂移检测 + 飞书告警 | `apistation/version_drift.go` + webhook |
| P1 | 封号预警 (连续 401/403 → 告警) | `apistation_monitor_service.go` |
| P1 | 账号健康度仪表盘 | 前端新页面 |
| P2 | 指纹盐值变更监控 | cron + 告警 |

## 5. 运维监控要求

### 封号风险监控

| 监控项 | 触发条件 | 告警通道 |
|--------|----------|----------|
| 账号认证失败 | 连续 3 次 401/403 | 飞书 webhook |
| Rate Limit 触发 | 单账号 1h 内 5+ 次 429 | 飞书 webhook |
| CC 版本漂移 | Stainless SDK 版本不一致 | 飞书 webhook |
| TLS 指纹失效 | 新版 Node.js 发布 | 定期检查 |
| 指纹盐值变更 | CC 源码中 SALT 变化 | 定期检查 |

### 健康度指标

- 账号可用率 = 可调度账号数 / 总账号数
- 请求成功率 = 2xx / 总请求数 (按账号/模型维度)
- 平均首 token 延迟 (TTFT)
- 账号轮转均匀度 = std(requests_per_account) / mean(requests_per_account)

## 6. 开发流程

### 新特性开发

1. 从 main 创建 feature 分支
2. 开发 + 测试
3. PR → review → merge into main
4. 更新本文档的特性吸收状态

### 上游合并

1. `git fetch upstream`
2. 更新 upstream 分支
3. merge into main
4. 解决冲突 (优先检查 `// api-station:` 标记区域)
5. 跑全量测试
6. commit

### 发布

- main 分支始终保持可部署状态
- 使用 git tag 标记版本: `api-station-v{版本号}`
- Docker image tag 与 git tag 对齐

## 7. 本地分析资料

本工作区包含三个参考仓库的克隆:

```
workspace/api-station/
  CLIProxyAPI/          # Go, 多平台代理, 防封参考
  auth2api/             # TypeScript, Claude-only, 指纹精确度参考
  sub2api/              # Go, 企业级代理, fork 基座
  report-opus-analysis.md    # Opus 深度分析报告
  report-codex-analysis.md   # Codex/GPT 深度分析报告
  DEVELOPMENT.md             # 本文件
```
