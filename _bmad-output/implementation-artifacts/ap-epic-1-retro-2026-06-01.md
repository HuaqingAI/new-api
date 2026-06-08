# Agent Platform Epic AP-1 Retrospective: 资源治理基线

日期：2026-06-01

## 参与角色

- Amelia（Developer / Facilitator）
- Alice（Product Owner）
- Charlie（Senior Dev）
- Dana（QA Engineer）
- hth（Project Lead）

## Epic Review

本次回顾针对 **Agent Platform Epic AP-1**，不是旧的 enterprise Epic 1。

Epic 1 已完成 6/6 个 story：

- Story 1.1 建立共享资源注册表与稳定资源身份：新增 `agentplatform` bounded context 骨架、共享 registry、稳定 `resource_id`、管理面路由和 OpenAPI 基线。
- Story 1.2 落地 typed detail 持久化：新增 `resource_version` 与 `skill_def` / `knowledge_def` / `agent_def` 三类 detail 表，保持 shared registry + typed detail 结构。
- Story 1.3 实现统一生命周期、版本与回滚动作：补齐 publish / disable / revoke / offline / rollback 以及 Agent Platform 自有审计事实源。
- Story 1.4 建立与客户端注册解耦的 projection baseline：先把 projection/exposure 建成独立对象，不前向依赖 Epic 2 的真实 client registration。
- Story 1.5 建立 per-target exposure 模型：让同一资源对不同 target 拥有独立 visible / callable 状态和独立 revoke 语义。
- Story 1.6 交付 `web/default` 管理控制面骨架：接入专用 route、sidebar 导航、Overview 到 Audit & Diagnostics 的壳层结构与前端测试。

对应提交为：

- `3768bd82`
- `9bc7c4cc`
- `fc3d5f20`
- `bdde663b`
- `91822996`
- `c2a21134`

Sprint 状态中这六个 `ap-1-*` story 都已是 `done`。本次 retrospective 额外发现并修正了 source-of-truth 漂移：`ap-epic-1` 主键仍停留在 `backlog`，与 story 完成事实不一致；现已同步修正为 `done`，并将 `ap-epic-1-retrospective` 标记为 `done`。

## Quality Evidence

Epic 1 的交付证据比较完整，但分散在 story artifact 与 orchestration 记录里：

- Story 1.1-1.5 均明确记录并验证了定向 Go 测试：`go test ./model/agentplatform ./service/agentplatform ./controller/agentplatform`。
- Story 1.1 的 review 自动修复了稳定身份相关关键问题，包括 `resource_id` 生成时序、`resource_id` / `resource_type` 不可变约束、非法 `resource_type` 过滤值校验等。
- Story 1.2-1.5 的实现记录显示，shared registry、typed detail、lifecycle、projection、per-target exposure 都沿同一个 bounded context 持续演进，没有回流到 `enterprise` 或 `relay/**`。
- Story 1.6 的 story artifact 记录了 `web/default` 的 typecheck、`test:e2e` 与 `i18n:sync` 验证，说明前端壳层不是只写静态页面，而是完成了最小集成闭环。
- story-automator 的编排日志 `orchestration-1-20260531-145351.md` 记录了 Agent Platform Epic AP-1 的真实过程：create-story 与 dev-story 多次漂移，最终依靠 manual takeover + source-of-truth closure 收口。

已知限制：

- 本次 retrospective 没有重新跑仓库级全量 `go test ./...` 或全站前端回归，主要依据 story 级定向验证和已落库的实现证据。
- 前端 Story 1.6 的验证证据来自 story artifact 记录，本次 retro 未重复执行 Bun 命令。

## What Went Well

- **bounded context 边界从第一步就定住了。** `model/service/controller/dto/agentplatform` 和 `/api/agent-platform/**` 在 1.1 即建立完成，后续 1.2-1.5 都是在这个壳体里累加能力，而不是把逻辑散落到 enterprise、relay 或临时 util 中。
- **shared registry + typed detail 的抽象成立了。** 1.1 先锁定稳定资源身份，1.2 再引入 typed detail，没有把三类资源塞进一个巨型多态 payload，后续 lifecycle 与 exposure 也因此有清晰落点。
- **projection/exposure 的演进顺序是对的。** 1.4 先把 projection 从 definition 状态中解耦，1.5 再扩成 per-target exposure。这个顺序避免了 Epic 2 客户端接入反向推翻前序模型。
- **审计和治理动作没有借旧域偷跑。** 1.3 把 lifecycle action、rollback 与 `agent_platform_*` 自有审计事实源一起落下，避免后续诊断再依赖 generic log 或 enterprise 审计表。
- **前端壳层交付时机合适。** 1.6 没有在后端模型还不稳定时抢跑 UI，也没有拖到后续 epic 才补入口；它建立了信息架构、专用 route 与导航，为 Epic 2 继续接 Clients/Publishing/Audit 页面提供了真实承载面。

## Challenges

- **story-automator 在 Agent Platform 流程上明显漂移。** 1.1 的 create-story 两次失败，dev-story 也多次停留在 enterprise 参考阅读；1.2-1.5 则几乎都要靠 manual create-story takeover 或 direct manual dev takeover 才能继续。
- **Agent Platform 的 epic/source 映射曾经不稳定。** 1.1 的 story artifact 已明确记下 `1.1 -> ap-1-1-*` key 映射与 epic source 解析是修正过的，这说明 automator 对“新产品线 + 新 story key 体系”的识别此前并不稳。
- **source-of-truth 分裂依然存在。** orchestration log 显示 Epic 1 六个 story 实际都完成了，但 `sprint-status.yaml` 中 `ap-epic-1` 仍是 `backlog`。这说明 automator 已经学会盯 story key，却还没有把 epic 聚合状态当成同等重要的同步对象。
- **bounded context 脚手架阶段对提示词容错要求很低。** 一旦 create/dev 会话回到旧的 enterprise 代码上下文，就容易长时间阅读已有结构而不真正写出新的 `agentplatform` 文件树。
- **前端 Story 1.6 的证据粒度偏薄。** 目前 artifact 只摘要说明壳层、测试和 i18n 已完成，没有像 1.1-1.5 那样列出更细的交互合同或 review 修复记录；这会让后续 Epic 2 继续接页面时，需要再次回读代码本身才能建立完整上下文。

## Story Pattern Synthesis

跨 6 个 story，Agent Platform Epic AP-1 呈现出很清楚的模式：

1. 真正的成功点不是“六个故事都做完了”，而是 **实现顺序高度尊重架构分层**：identity -> detail -> lifecycle -> projection baseline -> per-target exposure -> frontend shell。
2. 真正的失败模式也很一致：**automator 对新 bounded context 的上下文切换能力弱于对成熟域的工作能力**。它能在 source-of-truth 校验时收口，但在 create/dev 的前半程很容易漂移回旧代码或长时间阅读。
3. Epic 1 的每个后续 story 都显式以上一个 story 为前序上下文，这说明 Agent Platform 的故事链是强依赖的；这也意味着 **story artifact 的质量比 tmux/session 活性更重要**。
4. `web/default` 壳层不是独立支线，而是控制面主线的一部分。Epic 1 证明，前端壳层只要建立在稳定的 bounded context 与固定的信息架构上，就能在不透支后续 epic 的前提下提前落地。

## Significant Discoveries

本次 retrospective 识别出 3 项会直接影响后续 Epic 2 的显著发现：

1. **story-automator 的主要脆弱点不是 review，而是 create-story 与早期 dev 对上下文的抓取。**
   影响：后续继续跑 Agent Platform epic 时，应把“连续两次未产出 story artifact 或未写入目标目录树”视为明确切换到 manual takeover 的触发条件。

2. **Agent Platform 的 epic 聚合状态需要单独校验。**
   影响：不能只验证 `ap-*` stories 是否完成，还要验证 `ap-epic-*` 与 `ap-epic-*-retrospective` 主键是否同步，否则 sprint-status 会持续对外撒谎。

3. **frontend shell 已经把信息架构冻结成真实入口。**
   影响：Epic 2 及以后应直接沿用 `Overview -> Clients -> Skills -> Knowledge -> Agents -> Publishing -> Audit & Diagnostics` 这条导航主线推进，不要再重新讨论 Agent Platform 控制面放在哪、如何进 sidebar。

## First-Retro Notes

这是 Agent Platform 产品线的第一次 retrospective，没有更早的 Agent Platform retro 可供回看。旧的 `epic-1-retro-2026-05-28.md` 属于 enterprise 线，不应被视为本产品线的前序复盘。

这意味着 Epic 1 的很多结论会直接成为 Epic 2 的默认团队约定，而不是“可以以后再看”的建议。

## Key Learnings

1. 新产品线的第一批 story，最该优先交付的是 **bounded context 可生长性**，不是 UI 漂亮度，也不是先跑 open capability 协议。Epic 1 这样做是对的。
2. shared registry、typed detail、lifecycle、projection、exposure 这些层次一旦顺序错了，后续每一步都会返工；Epic 1 的顺序值得保持。
3. 对 Agent Platform 这种全新域，automator 的默认“先广泛读上下文再动手”策略不够好。更有效的是“先写 story artifact / 先锁目标文件树 / 先验证 sprint-status key 是否正确”。
4. source-of-truth 不只是 story artifact 和 story status，还包括 **epic 聚合状态**。这次 retro 能发现 `ap-epic-1: backlog`，说明当前收尾流程仍有漏口。
5. 前端壳层应在信息架构冻结后尽早落地。它能提前暴露导航、命名、i18n、测试入口等问题，但不会像“先做完整交互”那样把后端域模型拖偏。

## Next Epic Preparation

Epic 2 的成功，不取决于能不能快速写出 client registration 或 OAuth controller，而取决于是否把 Epic 1 的治理基线直接复用起来。

### 对 Epic 2 的准备结论

- 继续把 `agentplatform` 作为独立 bounded context 推进，不把 client/token/open capability 逻辑下沉到 enterprise、oauth 杂项目录或 relay 协议面。
- 客户端注册必须直接复用 1.4/1.5 已落下的 projection/exposure 语义，不要重新发明“client-specific publish table”。
- OAuth/token 与 open-capability discovery/detail 的实现要严格把 control plane、auth plane、open plane 分开，不要因为已有 admin shell 就把控制面会话混用成数据面鉴权。
- `web/default` 现有壳层应直接承接 Clients、Publishing、Audit 的真实页面，不要重做第二套 Agent Platform 入口。

## Readiness Assessment

- Story 完成度：6/6 stories `done`。
- 代码完成度：registry、typed detail、resource version、lifecycle、rollback、projection baseline、per-target exposure、admin shell 均已落地。
- 测试与质量：Go 侧已有持续的定向测试证据；前端壳层有 typecheck、E2E 与 i18n 同步记录。
- 流程健康：实现链路可交付，但 story-automator 对 Agent Platform 的 create/dev 收敛能力仍明显弱于人工文件优先流程。
- Epic 2 准备度：高，但前提是先把 automation 的 source-of-truth 修补和 epic 聚合状态校验纳入默认流程。

结论：Agent Platform Epic AP-1 可以标记为 `done`，`ap-epic-1-retrospective` 可以标记为 `done`，并可作为 Epic 2 的正式起点。

## Action Items

1. Owner: Developer / Automation
   Action: 在 story-automator 中为 Agent Platform 类新 story key 增加“连续两次未产出 artifact 即 manual takeover”的自愈分支。
   Success criteria: `ap-*` create-story 不再长时间停留在旧域上下文阅读，能更早切换到人工补齐模式。

2. Owner: Developer / Automation
   Action: 把 `ap-epic-*` 聚合状态与 retrospective 状态纳入 source-of-truth 校验，而不是只检查 story 级 key。
   Success criteria: 所有 Agent Platform epic 在故事完结后，其 epic 主键和 retrospective 主键都会自动与事实源同步。

3. Owner: Developer
   Action: 为 Agent Platform dev-story 默认加入“目标目录树白名单 + 前序 story artifact 摘要”约束，降低会话回流到 enterprise 上下文的概率。
   Success criteria: 后续 Epic 2 的 dev 会话能直接进入 `model/service/controller/dto/agentplatform` 与 `web/default/src/features/agent-platform` 目标文件树编辑。

4. Owner: Developer / Frontend
   Action: 在 Epic 2 开始前补一份更细的 Agent Platform shell 实现说明，覆盖现有 route、sidebar、页面占位和 i18n key 策略。
   Success criteria: 后续前端 story 不需要先反查 `index.tsx` 和 `use-sidebar-data.ts` 才知道壳层约束。

5. Owner: QA / Test Architect
   Action: 为 Agent Platform 建立一组持续存在的定向回归入口，至少覆盖 registry、resource_version、lifecycle、exposure 和前端 shell route 挂载。
   Success criteria: Epic 2 继续扩展时，可以复用同一组高价值回归入口，而不是每个 story 单独拼命令。

## Team Agreements

- Agent Platform 后续故事继续保持 `model/service/controller/dto/agentplatform` 和 `web/default` 为唯一主实现面。
- create-story、dev-story、review、retro 的收口优先信任 sprint-status、story artifact、目标文件树变化与定向测试证据，不优先信任 session 是否仍活着。
- 任何新的客户端接入实现都必须复用 `visible` / `callable`、projection / exposure、resource version 与 lifecycle 词汇，不允许另起一套近义模型。
- `web/classic` 不作为 Agent Platform MVP 的对等目标，直到架构或产品范围明确升级。

## Final Summary

Agent Platform Epic AP-1 最重要的成果，不是“又完成了六个 story”，而是把一个全新的产品线从零拉成了可持续演进的控制面骨架。shared registry、typed detail、lifecycle、projection、per-target exposure 和 `web/default` shell 已经形成了一条清晰主线，Epic 2 不需要再回头补地基。

这次回顾也把自动化编排的真实问题暴露得很清楚：对于新 bounded context，story-automator 的 create/dev 前半程比 review 更容易漂移，source-of-truth 同步也还没有覆盖 epic 聚合状态。把这两个问题修掉，Agent Platform 后续 epic 的推进速度和稳定性都会明显更好。
