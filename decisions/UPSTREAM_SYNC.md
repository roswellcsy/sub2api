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
| YYYY-MM-DD | N | N | +X/-Y | Nm | gateway_service.go: Forward() beta header | — |

## 阈值告警规则

超过任一阈值，则重启 Phase 4 结构拆分讨论：

- 单次冲突行数 > 100 行
- 单次解决耗时 > 60 分钟
- fork 在 `gateway_service.go` 的累计改动 > 500 行

## 首次演练 TODO

- [ ] 记录 baseline merge-base: `97f14b7a086bf75c72b3549e0d546907a720eb8e`
- [ ] 记录 baseline fork divergence (`git rev-list --left-right --count main...upstream/main`): `14 0`
- [ ] 记录 baseline upstream ahead count (`git log main..upstream/main --oneline | wc -l`): `0`
- [ ] 记录 baseline `gateway_service.go` 差异规模，并在首次演练时补充主要冲突点说明
