# Zen-Codes Production Runbook

## 部署架构
- VPS: racknerd1g (192.3.127.79, 1c1g Ubuntu 24.04, Santa Clara CA)
- 域名: zen-codes.com (Cloudflare DNS, grey cloud)
- 三容器: sub2api (384M) + postgres (256M) + redis (96M)
- 反代: Caddy 2.x + Let's Encrypt 自动证书
- Go binary 内嵌前端（-tags embed），前端 dist 由 dev host 预 build 后 rsync 到 VPS

## 常规升级（场景 A）
```
bash deploy/prod/update.sh
```

## Upstream 同步（场景 B）
```
git fetch upstream
git log main..upstream/main --oneline | head -20   # 预览新 commits
git merge upstream/main                             # 可能有冲突，优先保留 apistation 侵入点
(cd frontend && pnpm install && pnpm run build)
(cd backend && go test ./...)
# 本地验证 OK 后：
bash deploy/prod/update.sh
```

## 破坏性变更（场景 C）
- PG schema 迁移（migrations/*.sql）会自动跑，但应**先备份后升级**
- 新 SettingKey 冲突：检查 upstream 是否重名我们的 apistation_* key
- Go/Alpine 版本升级：Dockerfile.prebuilt 的 ARG 跟着改

## 回滚
```
bash deploy/prod/rollback.sh                 # 交互式选 tag
bash deploy/prod/rollback.sh --with-db       # 带 DB restore
```

## 日常运维
- 查日志: `ssh racknerd1g "docker compose -f /opt/api-station/deploy/docker-compose.local.yml -f /opt/api-station/deploy/docker-compose.override.yml logs --tail 100 sub2api"`
- 资源: `ssh racknerd1g "docker stats --no-stream && free -h && df -h /"`
- 手动触发备份: `ssh racknerd1g "bash /etc/cron.daily/pg-backup-zen-codes"`
- 手动跑水位告警测试: `ssh racknerd1g "bash /etc/cron.hourly/disk-alert-zen-codes"`

## 已知约束（1c1g）
- 前端 build 必须在 dev host，VPS 上 V8 heap OOM (exit 134)
- Swap 2G，容器内存硬限 736M 总量
- 磁盘 23G 总，业务稳态 ~5G，留 5-6G buffer
- migration 时表锁可能短暂影响请求延迟，低流量时段升级

## 凭证位置
- `/opt/api-station/deploy/.admin-password-remember-to-rotate` (mode 600, VPS only)
- 飞书 webhook URL 在 `.env` (未入 git) 和 DB settings 表

## 下一步候选工单
- CSP 扩白名单（Stripe 支付才需要）
- 监控 audit_logs 表行数变化（验证 retention 实际生效）
- 账号池压测（50 并发请求调度均匀性）
