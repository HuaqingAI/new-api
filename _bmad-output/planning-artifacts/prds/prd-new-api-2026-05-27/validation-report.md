# Validation Report — `{NEW_PRODUCT_NAME}` V1 企业管控基线 PRD

- **PRD:** `_bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md`
- **Rubric:** `.claude/skills/bmad-prd/assets/prd-validation-checklist.md`
- **Run at:** 2026-05-27T12:00:00+08:00
- **Grade:** Poor *（由单条 critical 触发；其余 7 维度中 5 项 strong、1 项 adequate、1 项 thin）*

## Overall verdict

整体定位、范围与下游可用性立得住：四条 UJ 都带命名主角与边界情况、FR 与 UJ 双向绑定、棕地代码引用经抽查全部与代码一致、FR-3 / FR-11 守住"部门 ≠ Group"与"部门只消费 quota"两条最易出错的边界。Substance / Strategic coherence / Scope honesty / Downstream usability / Shape fit 五项均 strong。

核心风险集中在 Done-ness。FR-10 部门配额并发安全是 PRD 自己最重的承诺（addendum A-4 已点名"验收风险最高"），但 PRD 主体只写"不能造成超扣或漏扣"，没有给量化命题——下游 story 据此写不出 acceptance。FR-6（主部门时机）、FR-19（重试策略）是同性质问题。把 FR-10 量化、FR-6 时机、FR-19 重试、OQ-3 扩旧表代价表四件事补完，本 PRD 即可上升到 Good。

## Dimension verdicts

- Decision-readiness — **adequate**
- Substance over theater — **strong**
- Strategic coherence — **strong**
- Done-ness clarity — **thin**
- Scope honesty — **strong**
- Downstream usability — **strong**
- Shape fit — **strong**

## Findings by severity

### Critical (1)

**[Done-ness clarity]** — FR-10 部门配额并发安全无可执行验证命题 (§4.3 FR-10)
addendum A-4 已说"FR-10 验收风险最高，应优先出技术设计和并发测试"，但 FR-10 仅写"不能造成超扣或漏扣"。这是 PRD 的核心治理承诺，下游 story 据此写不出 acceptance。
Fix: 增加"在 ≥N 并发、跨同部门 M 个用户、含 `tiered_expr` 模型的混合负载下，连续 X 分钟无部门剩余额度负数、无用户 quota 与部门 quota 终态偏差超过 ε"之类量化条件；明确预消费 vs 结算两条路径上的扣减时机（addendum A-4 已枚举候选）。

### High (3)

**[Decision-readiness]** — OQ-3 没有让取舍可比较 (§13 OQ-3 / §4.1 FR-4 / addendum A-2)
OQ 只问"是否扩到旧表"，PRD 主体没有把两条路径在 V1 工期、迁移成本、索引改造、SaaS 切换难度上的代价摆出来。
Fix: 在 §4.1 FR-4 或 §13 OQ-3 下增补两行表格："只新表预留：V1 +X 张表 schema，SaaS 切换需补 Y。扩到旧表：迁移影响 `users`/`logs`/`quota_data`，索引改造 Z 项。"

**[Done-ness clarity]** — FR-6 主部门一致性时机未定 (§4.2 FR-6)
"登录成功后用户的主部门与最近一次有效同步结果一致"对登录与同步并发场景不清晰；当登录回调早于同步任务完成时主部门来源是缓存、上次同步还是钉钉本次返回？
Fix: 写明读取顺序（如：登录时优先以钉钉返回 + 本地映射规则计算主部门；若同步进行中则使用上一次成功同步快照）。

**[Done-ness clarity]** — FR-19 告警重试策略空 (§4.5 FR-19 / §11 风险表"告警噪声"行)
"成功、失败、重试状态可查询"未规定重试次数、间隔、最终失败的处理。FR-18 与 §11"默认低频"并未给出阈值数。
Fix: 给出 V1 默认重试次数（如 3 次指数退避），最终失败必须可在后台一键重发并标记。

### Medium (4)

**[Decision-readiness]** — OQ-6 / OQ-7 决定 FR 验收口径却被推迟 (§13 OQ-6, OQ-7 / §4.3 FR-9 / §4.1 FR-2)
FR-9 周期 / FR-2 主部门来源未定，acceptance 写不实。
Fix: 进入 epic/story 之前由 PM 与 IT 林工对齐两条规则，回填 FR-9 / FR-2 可验证结果。

**[Done-ness clarity]** — FR-5 错误"可区分"未给可观测形态 (§4.2 FR-5)
Fix: 列出 V1 至少返回的 4 类错误码或后台标识。

**[Done-ness clarity]** — FR-15 CSV"一致"口径未定 (§4.4 FR-15)
排序、空值、未归属部门行未定。
Fix: 明确"按当前筛选条件聚合后逐行输出，包含未归属部门行"；给一份示例字段顺序。

**[Done-ness clarity]** — FR-12 默认阈值缺省 (§4.3 FR-12)
Fix: 给 V1 默认值（如 80% / 95% 两档）或转 OQ。

**[Done-ness clarity]** — FR-20 风险率口径定义缺失 (§4.5 FR-20)
Fix: 在 FR-20 写明"风险率 = 当周期内该部门触发风险事件请求数 / 部门请求总数；被拦截请求计入分子分母"。

### Low (3)

**[Decision-readiness]** — "6-8 周交付窗口"未在 PRD 内出现 (§11 风险表 / §6 MVP)
Fix: §6 MVP 或 §12 增一句"V1 目标交付窗口约 X 周（来自 brief）"。

**[Scope honesty]** — OQ 密度对绿灯 PRD 偏高 (§13)
Fix: 进入 epic/story 前由 PM 至少关掉 OQ-6 / OQ-7 / OQ-4，把 OQ-2 / OQ-3 显式标为"架构阶段决策，不阻塞 V1 起步"。

**[Done-ness clarity]** — §8 NFR 性能项未给数据量/响应时间阈值 (§8 NFR 性能)
Fix: 给一个上限假设（如"100 万条 log 下 30 天看板查询 < 3 s"），明确假设可由架构阶段调整。

## Mechanical notes

- **术语漂移：**未发现关键漂移。"主部门" / "部门" / "`Group`"全文一致；"内容风险事件"全文一致。
- **ID 连续性：**FR-1..FR-20、UJ-1..UJ-4、SM-1..SM-5、SM-C1..SM-C3、OQ-1..OQ-7 全连续，无重号无跳号。
- **Cross-reference 解析：**抽查 FR → UJ、SM → FR、§14 假设索引、§13 OQ → FR 双向引用全部能解析。
- **Assumptions Index roundtrip：**3 条 ASSUMPTION 内联与索引双向覆盖。
- **UJ 主角：**林工 / 王经理 / 陈总 / 赵安全均命名且场景写清。无 floating UJ。
- **必备段落：**愿景 / 目标用户 / UJ / 术语表 / FR / 非目标 / MVP / SM / NFR / 约束 / 集成 / 风险 / 发布 / OQ / 假设索引 全部存在；frontmatter 含 status / created / updated / source_brief / source_addendum，链接路径已修正。
- **占位符：**`{NEW_PRODUCT_NAME}` 在标题、§0、§1、§3、UJ-1、FR-6 多处出现，OQ-1 已提示其确认时机；占位符使用一致。

## Reviewer files

- `review-rubric.md`
