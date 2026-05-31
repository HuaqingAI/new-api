---
baseline_commit: 8560774d7ca607ce43cfa50fe75958fdb93ff61b
---

# Story 6.4: 钉钉账号 username 策略与受控修改

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a 管理员,
I want 企业账号默认 username 更可读，并支持受控修改,
so that 登录、查询、统计、预算分配和运维识别不会长期暴露不可读账号名。

## Acceptance Criteria

1. **Given** 钉钉同步或 OAuth 创建新用户  
   **When** 系统生成默认 username  
   **Then** 系统优先生成可读且唯一的 username  
   **And** 不再以 `dt_编号` 作为主要结果。
2. **Given** 企业账号已创建  
   **When** 有权限的管理员执行 username 修改  
   **Then** 系统支持受控修改 username  
   **And** 明确校验唯一性、保留审计记录，并处理相关缓存刷新与后续查询联动。
3. **Given** username 已被修改  
   **When** 页面、查询、日志入口和企业视图再次读取该用户  
   **Then** 系统按照更新后的规则展示与筛选该用户  
   **And** 历史日志与历史统计是否保留原 username 快照必须有明确规则并可验证。

## Tasks / Subtasks

- [ ] 收敛钉钉用户默认 username 生成策略，优先产出可读且唯一的账号名 (AC: 1)
  - [ ] 追踪当前钉钉同步与 OAuth 创建用户的 username 生成路径，定位仍会产出 `dt_编号` 或等价不可读默认值的逻辑，并在现有用户创建/绑定链路内修正；不要为 6.4 新建第二套用户创建通道。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#fr-6; _bmad-output/implementation-artifacts/2-3-support-dingtalk-oauth-login-and-account-binding.md]
  - [ ] 默认 username 生成策略至少要同时满足“可读 + 唯一”两条约束，并定义明确冲突回退规则（例如名称归一化后追加短后缀或 ID）；不得简单继续暴露 `dt_编号`，也不能因为可读化而丢失唯一性。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#V12-Follow-up-2026-05-31; _bmad-output/planning-artifacts/architecture.md#Authentication & Security]
  - [ ] 保持现有登录/绑定/同步主干和三库兼容，不在 6.4 顺手重构整个用户模型、session 或 relay 协议表面。[Source: AGENTS.md#Rule 2; controller/user.go; router/enterprise-router.go]

- [ ] 实现受控 username 修改能力，并把它接入企业管理路径 (AC: 2)
  - [ ] 在后端新增/扩展企业用户 rename API，要求显式校验唯一性、拒绝非法/空 username，并对修改动作写审计记录；继续遵守 Router -> Controller -> Service -> Model 分层，不在 controller 里写重逻辑。[Source: _bmad-output/planning-artifacts/architecture.md#Architectural Boundaries; service/enterprise/admin_action.go]
  - [ ] 修改能力必须是“受控”的：只允许有权限的企业管理员/部门管理员从企业治理视图触发，不要退回到全局用户管理页随意改名，也不要把能力散落到 DingTalk 设置页或登录流程中。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.4: 钉钉账号 username 策略与受控修改; middleware/enterprise_dept_admin.go]
  - [ ] 修改成功后，要处理与 username 强相关的缓存/查询链路刷新，例如组织页成员表、预算分配成员上下文、用量页日志入口筛选和后续企业页面展示，而不仅仅是 DB 落库成功。[Source: _bmad-output/implementation-artifacts/4-3-view-department-usage-details.md; _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]

- [ ] 明确 username 变更后的展示与筛选联动规则，不让旧快照语义变得模糊 (AC: 2, 3)
  - [ ] 对“历史日志与历史统计是否保留原 username 快照”给出明确处理方式并落实到当前系统真实语义：例如 `logs.username` / 风险事件快照继续保留历史写入值，而当前管理视图优先展示最新可读名并用 username / ID 消歧。不得在 6.4 随意回写历史日志表或批量重写快照。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#V12-Follow-up-2026-05-31; _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
  - [ ] 对企业页面中的 username 查询与展示链路做最小必要调整，让修改后的 username 在组织页、预算/分配相关区域、用量页日志入口和风险事件筛选语义上保持可验证一致；为 6.5 的统一展示规则预留清晰基础，但不提前把 6.5 的整套 UI 也一起做掉。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#V12-Follow-up-2026-05-31; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#302]

- [ ] 在 Default 企业治理视图中提供最小可用的 username 修改入口与回馈 (AC: 2, 3)
  - [ ] 继续在 `web/default/src/features/enterprise-organization/`（以及必要时 `enterprise-dingtalk/`）里落地最小入口，优先复用 6.3 刚完成的当前成员上下文，不让管理员再跳回全局用户页输入 ID 进行重命名。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md; web/default/src/features/enterprise-organization/index.tsx]
  - [ ] 修改入口要提供明确的当前值、唯一性错误提示、成功回馈，以及修改后相关列表/上下文自动刷新；不要只给一个隐藏 API 钩子或只在 debug 面板里暴露能力。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [ ] 如需最小 Classic 兼容，仅保持必要入口或展示语义不崩，不把 6.4 扩成 Classic 深度改造。[Source: web/classic/src/services/enterprise.js; _bmad-output/planning-artifacts/architecture.md#Important Decisions]

- [ ] 补齐测试与 i18n，锁定 username 可读化与受控修改链路 (AC: 1, 2, 3)
  - [ ] 为后端 username 生成/修改逻辑补测试：覆盖默认可读名生成、重复冲突回退、非法 username 拒绝、权限控制、修改审计、缓存/查询刷新触发点，以及历史日志快照不被回写的语义。[Source: AGENTS.md; controller/enterprise/department_membership_test.go; tests/api]
  - [ ] 扩展 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`（以及必要时企业日志/风险相关测试），覆盖当前成员上下文下的 rename 入口、成功/失败提示、修改后显示刷新和与 username 相关的筛选参数更新。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
  - [ ] 所有新增文案同步到 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`，并运行 `bun run i18n:sync`；确保“可读 username”“已更新 username”“用户名已存在”等消息多语言齐备。[Source: AGENTS.md#Internationalization i18n]
  - [ ] 运行最小回归：`cd web/default && bun run typecheck`、`bun run test:e2e`、`bun run i18n:sync`；如动到后端用户创建/绑定逻辑，补最小 API/服务回归，确认现有 DingTalk OAuth 与同步链路不回归。[Source: _bmad-output/implementation-artifacts/2-3-support-dingtalk-oauth-login-and-account-binding.md]

## Dev Notes

- Story 6.4 的关键不是“给用户表多加一个字段”，而是重新定义企业账号默认 username 的可读化策略，并把受控修改能力接入现有企业治理链路。[Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#V12-Follow-up-2026-05-31]
- 这条故事会直接影响 DingTalk OAuth / 同步时的用户创建，以及后续企业页面的用户展示和筛选，因此一定要明确“历史日志快照保留旧 username，当前管理视图展示最新可读名”之类的边界，而不能在 6.4 粗暴重写历史数据。[Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log; _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#302]
- 6.4 最好复用 6.3 已经引入的“当前成员上下文”工作区来承载 rename 入口，这样能避免管理员再次回退到全局用户页/ID 输入流程，也能自然驱动预算分配、成员展示与日志入口刷新。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
- 后端权限与日志核心语义仍是硬边界：不要修改 relay、billingexpr、`model/log.go`、`controller/log.go` 或通过批量 SQL 回写历史 `logs.username` 伪完成需求。[Source: AGENTS.md; _bmad-output/planning-artifacts/architecture.md#Architectural Boundaries]

### Project Structure Notes

- 重点修改文件预计包括：
  - `service/enterprise/dingtalk_oauth.go`
  - `service/enterprise/dingtalk_sync.go`
  - `controller/enterprise/dingtalk.go` 或新增企业 rename controller
  - `service/enterprise/*user*` / `service/enterprise/errors.go`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - 必要时 `web/default/src/features/enterprise-dingtalk/*`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
- 保持不改或仅只读依赖：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `docs/openapi/relay.json`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.4: 钉钉账号 username 策略与受控修改]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#V12-Follow-up-2026-05-31]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#173]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#302]
- [Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
- [Source: _bmad-output/implementation-artifacts/2-3-support-dingtalk-oauth-login-and-account-binding.md]
- [Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 手工接管 `create-story`，因为 story-automator 的 `6.4` create 会话尚未落地 story artifact，也还没有把 `sprint-status` 从 `backlog` 推到 `ready-for-dev`。
- 已读取 Epic 6 中 Story 6.4 的 AC、V1.2 PRD follow-up 对 username 可读化与受控修改的说明、architecture 对历史日志语义的边界，以及 Story 2.3 / 6.3 的已完成功能。
- 已确认 6.4 的主要实现面应围绕现有 DingTalk 用户创建/绑定路径和企业治理视图，不扩张到 relay、billing 或历史日志回写。

### Completion Notes List

- 手工补齐 Story 6.4 故事文档，明确默认 username 可读化、受控修改、唯一性校验、审计、以及与日志/筛选联动的边界。
- 明确 6.4 是对现有用户创建与企业治理链路的收敛性增强，不是新建独立账号体系。

### File List

- `_bmad-output/implementation-artifacts/6-4-fix-dingtalk-username-strategy-and-controlled-rename.md`
