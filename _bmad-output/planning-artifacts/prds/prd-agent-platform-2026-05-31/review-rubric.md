# PRD Quality Review — new-api 通用 Agent 能力平台 PRD

## Overall verdict
这份 PRD 现在已经达到可以更稳地进入 architecture freeze 的水平。它的核心产品论点、范围边界、角色责任、Glossary 和下游可抽取性都比较扎实，而且前一轮最关键的 Done-ness 问题已经被收敛：`Knowledge` 的 MVP 形态被锁死为 `retrieval`，缓存/撤销收敛规则也被量化到了统一窗口。

剩余问题主要是轻量级的文档打磨，而不是会阻塞架构或 Epic 拆分的结构性空洞。最值得继续留意的是：如果后续真要引入第二个消费者，`namespaced` 扩展边界和 `Agent` 的 detail 表达要继续按 DG-1 验证，避免再次回滑成“看起来通用、实际上偏某个客户端”的协议。

## Decision-readiness — strong
这份 PRD 已经做了足够多的实质决策。它不仅明确拒绝 Cherry 专属私有协议、复杂 `Agent` runtime、多租户 SaaS 运营和本地副本镜像，还把 freeze 前必须拍板的事项压缩成三个真正的 decision gates：有限客户端扩展边界、第二个验证消费者优先级、审计/诊断的上线门槛。

更重要的是，前一轮会削弱决策效力的点已经被收敛。`Knowledge` 的 MVP 调用形态不再是“单形态还是多形态集合”的开放题，而是被锁定为 `retrieval`；MVP 闭环也已经明确要求 `Skill`、`Knowledge` 和轻量 `Agent` 定义一起进入验证。

## Substance over theater — strong
这份 PRD 依旧没有明显的“模板家具”。角色矩阵、命名主角的用户旅程、Glossary 和 NFR 都在承担真正的约束作用，而不是为了显得完整。尤其是 `Published Projection`、`Visible`、`Callable`、`Freshness`、`Knowledge Mode` 这些术语，确实在后文被反复稳定复用。

## Strategic coherence — strong
PRD 的主线是清楚且自洽的：把 `Skill` / `Knowledge` / `Agent` 纳入统一治理与统一开放能力层，并用首个消费者验证其可复用性。§1 的痛点、§4 的 FR 分组、§7 的闭环定义、§8 的成功指标和 counter-metrics 都围绕这个论点服务。

上一轮的一个弱点是 `Agent` 在范围里、但不在指标里。现在新增的 `SM-6` 和 `SM-C4` 已经把 `Agent` 的 discover/detail/reference 路径纳入明确验证，同时继续防止它被偷渡成复杂 runtime，主线因此更完整。

## Done-ness clarity — adequate
Done-ness 维度已经明显提升。`FR-10` 现在把 `Knowledge` MVP 固定为 `retrieval`，并明确了结构化引用结果的语义；`FR-6` 也给出了 5 分钟 `TTL` 上限、refresh 触发和撤销/回滚的收敛窗口。这些补强足以让 architecture、API 和测试团队提取稳定的完成定义。

仍然可以继续加强的地方不在“缺要求”，而在“要求是否需要更示例化”。例如 `retrieval` 结果中的最小字段集合、排序信息的粒度、以及 `Agent` detail 中依赖边界的表现形式，后续在 architecture 文档里最好用 schema 草图继续钉死，但它们已经不属于 PRD 层面的结构缺口。

## Scope honesty — strong
范围控制依然是这份文档的优点。它明确排除了多租户运营、本地副本、知识库底层 ingestion/embedding、`Knowledge` 扩展形态、复杂 `Agent` orchestration 和客户端间双向同步冲突。用户如果后续要把这些塞回 MVP，将会直接与 PRD 正文冲突，而不是留下模糊空间。

假设索引也更干净了。那些实际上已经是产品结论的内容，例如“首个消费者不是唯一标准定义者”“AP-6 基于既有 OpenAPI / open 能力扩展”“Agent 在 MVP 中只是定义与引用对象”，已经从 Assumption 降噪为正文决策。

## Downstream usability — strong
作为后续 architecture / epics / stories 的上游输入，这份 PRD 现在相当顺手。Glossary 稳定，编号连续，UJs 有命名主角，SM 与 Counter-metrics 一一对冲，闭环定义、FR 和成功指标之间的口径也已经对齐。

下游最需要的几个抓手都已经有了：`Knowledge` 模式、客户端收敛窗口、`Agent` 的轻量验证边界，以及不应引入复杂 runtime 的反向约束。

## Shape fit — strong
对一个多角色、强治理、强契约的平台产品来说，这份 PRD 的形状是合适的。它既没有误写成 consumer UX PRD，也没有退化成只罗列后台模块的 capability list。`Agent` 被控制在“定义与引用对象”的轻量形态，也让这份文档避免了过早承诺超出 MVP 的能力。

## Mechanical notes
- Glossary 术语整体一致；`Resource` / `Definition` / `Published Projection` / `Visible` / `Callable` / `Freshness` / `Knowledge Mode` 没有发现正文级漂移。
- UJ-1 到 UJ-4 均有命名主角，且主角携带场景上下文，没有 floating journey。
- FR、UJ、SM、SM-C、DG 编号都唯一且连续；FR 的叙述顺序仍是 `FR-1..FR-6` 后插入 `FR-15` 再进入 `FR-7..FR-14`，合法但略影响顺序扫描。
- Inline `[ASSUMPTION]` 与 Assumptions Index 现在一致，只保留仍未被正文吸收的两条假设。
