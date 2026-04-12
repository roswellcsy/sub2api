# 上游合并演练追踪 (Upstream Merge Practice Log)

## 目的

每月执行一次 upstream 合并演练，量化冲突趋势，并用可重复的数据决定是否重启 Phase 4 `gateway_service.go` 结构拆分讨论。

## 执行流程

1. `git fetch upstream && git log main..upstream/main --oneline | wc -l`
   - 统计 upstream 相对本地 `main` 的新增 commit 数。
2. `git merge upstream/main --no-commit`
   - 执行 dry-run 合并，观察是否出现冲突以及冲突集中区域。
3. 记录本次演练数据
   - 冲突文件数
   - 冲突行数
   - 解决耗时
   - 主要冲突点
4. `git merge --abort` 或 `git commit`
   - 根据评估结论放弃本次演练，或在确认可接受时继续完成合并。
5. 更新本文档
   - 将本次演练结果追加到追踪表，并补充任何需要触发重评估的观察。

## 追踪表

| 日期 | upstream 新 commits | 冲突文件数 | 冲突行数 | 解决耗时 | 主要冲突点 | 备注 |
|---|---|---|---|---|---|---|
| 2026-04-12 | 0 | 0 | 0 | 0m | 无冲突（upstream tip 未前进） | Baseline：fork 14 commits 全为 api-station Phase 1-3+ADR，merge-base 97f14b7a；gateway_service.go +50/-9（2 个 hook 点 Forward() line 4074/4476） |

## 阈值告警规则

超过任一阈值，则重启 Phase 4 结构拆分讨论：

- 单次冲突行数 > 100 行
- 单次解决耗时 > 60 分钟
- fork 在 `gateway_service.go` 的累计改动 > 500 行

## 首次演练 Baseline (2026-04-12)

- [x] merge-base: `97f14b7a086bf75c72b3549e0d546907a720eb8e`
- [x] Fork divergence (`git rev-list --left-right --count main...upstream/main`): `14 0`
- [x] upstream ahead count: `0`（演练时 upstream 尚无新 commit，fork 在 tip 上开发）
- [x] gateway_service.go 差异: +50/-9 (59 行)，hook 点 2 个（Forward() line 4074 thinking signature / line 4476 request audit）
- [x] 总 divergence: 49 文件 / +3341/-44 行
- [x] merge-tree dry-run 冲突标记: 0（因 upstream 无新 commit）

Baseline 含义: 当前 fork 处于最干净合并状态（upstream tip 上开发，无冲突）。这也意味着首次真实冲突演练需等 upstream 有新 commit（通常下次月度节点）。下次演练触发点: 观测到 `git log main..upstream/main --oneline | wc -l` > 0 后尽快执行。

## 下次演练计划

- 触发时机: 2026-05 月初 OR upstream 新增 commit 数 ≥ 10 时
- 执行脚本: `bash scripts/upstream_sync_check.sh`
- 记录项: 追加一行到上方追踪表
- 若触发阈值（冲突 > 100 行 / 耗时 > 60m / 累计 > 500 行），引用 ADR-001 重启 Phase 4 讨论

## 观察事项 (Observations)

### 2026-04-12: upstream 存量 TypeScript 错误

- **文件**: `frontend/src/api/client.ts:95`
- **错误**: `TS2352: Conversion of type 'ApiResponse<unknown>' to type 'Record<string, unknown>' may be a mistake`
- **引入 commit**: upstream `faee59ee` (touwaeriol, `fix(payment): propagate reason/metadata in API error responses`)
- **状态**: 已在 fork merge-base (`97f14b7a`) 之前合入，属于**继承的历史遗留**，非本 fork 引入
- **影响**: `pnpm run typecheck` 退出码 2；但 `build` (Vite) 不阻塞运行时
- **处置**: 未来作为 upstream PR 素材（类型断言改 `as unknown as Record<string, unknown>` 或重新定义类型），不在 fork 内修复（避免 divergence）
