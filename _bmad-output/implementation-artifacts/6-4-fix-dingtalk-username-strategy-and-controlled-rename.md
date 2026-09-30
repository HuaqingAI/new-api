---
baseline_commit: 8460396d9
create_story_refreshed_at: 2026-06-03 02:18 +0800
---

# Story 6.4: 钉钉账号 username 策略与受控修改

Status: done

<!-- Note: 本次按 create-story workflow 刷新上下文。由于 sprint-status 中 6.4 已为 done，未将状态回退到 ready-for-dev。 -->

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

- [x] 收敛钉钉用户默认 username 生成策略，优先产出可读且唯一的账号名 (AC: 1)
  - [x] 在 `service/enterprise/username_policy.go` 建立统一 username 策略：归一化为小写字母、数字和单下划线；长度遵守 `model.UserNameMaxLength`；冲突时追加稳定后缀。[Source: service/enterprise/username_policy.go]
  - [x] DingTalk OAuth 新用户创建通过 `DingTalkOAuthService.availableDingTalkUsername` 使用同一策略，优先从姓名、邮箱、手机号、外部 ID、union/open ID 生成可读名。[Source: service/enterprise/dingtalk_oauth.go]
  - [x] DingTalk 通讯录同步新用户创建通过 `DingTalkSyncService.availableSyncUsername` 使用同一策略，不再把 `dt_编号` 作为主要结果。[Source: service/enterprise/dingtalk_sync_apply.go; service/enterprise/dingtalk_sync_helpers.go]
  - [x] 保持现有 DingTalk identity binding、OAuth session、同步任务和用户创建主干，不新建第二套账号体系。[Source: _bmad-output/planning-artifacts/architecture.md#DingTalk Integration Boundary]

- [x] 实现受控 username 修改能力，并接入企业管理路径 (AC: 2)
  - [x] 后端暴露 `PUT /api/enterprise/departments/:id/members/:user_id/username`，由 `middleware.EnterpriseDepartmentAdmin("id")` 保护，修改入口绑定当前部门成员上下文。[Source: router/enterprise-router.go]
  - [x] `DepartmentMembershipService.RenameDepartmentMember` 校验部门、活跃成员关系、用户存在、username 格式和唯一性；成功后只更新 `users.username` 并调用 `model.InvalidateUserCache(userId)`。[Source: service/enterprise/department_membership.go]
  - [x] Controller 在同一事务内写入 `enterprise.organization.membership.rename` 管理动作审计，payload 包含 `previous_username` 与 `new_username`。[Source: controller/enterprise/department_membership.go; service/enterprise/admin_action.go]
  - [x] 错误语义映射为企业域错误：非法 username 返回 `enterprise.organization.username_invalid`，重复 username 返回 `enterprise.organization.username_exists`。[Source: service/enterprise/errors.go; controller/enterprise/department_membership.go]

- [x] 明确 username 变更后的展示、查询、缓存与历史快照规则 (AC: 2, 3)
  - [x] 当前治理视图刷新后读取最新 `users.username`；组织成员、预算池、预算详情和 allocation 查询按精确 query key 失效，避免全量刷新或跨部门缓存串用。[Source: web/default/src/features/enterprise-organization/index.tsx; web/default/src/features/enterprise-organization/api.ts]
  - [x] 历史 `logs.username`、风险事件 username snapshot 和历史统计快照保留原始写入值；6.4 禁止回写 `logs` 或批量改历史快照。[Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
  - [x] 用量聚合、告警筛选和治理时间线继续按现有快照/用户 join 语义工作；本故事只修当前用户可读名和当前治理视图，不改 relay、billing expression 或日志写入主链路。[Source: service/enterprise/usage_aggregation.go; service/enterprise/alert.go; service/enterprise/governance_timeline.go]

- [x] 在 Default 企业治理视图中提供最小可用 username 修改入口与回馈 (AC: 2, 3)
  - [x] 在当前部门的当前成员治理卡片中提供 `Rename Username` 表单，复用 6.3 的“当前部门 + 当前成员”上下文，不要求管理员回到全局用户页或输入裸 user ID。[Source: web/default/src/features/enterprise-organization/index.tsx]
  - [x] 前端 schema 与后端保持一致：3-20 字符、小写字母/数字/单下划线，错误时展示唯一性或格式提示，成功后提示 `Username updated`。[Source: web/default/src/features/enterprise-organization/index.tsx]
  - [x] 表单旁明确展示历史快照规则：“历史日志和风险事件保留原始 username snapshot；当前治理视图刷新后使用新 username”。[Source: web/default/src/features/enterprise-organization/index.tsx]
  - [x] Classic 不做深度治理台重构；既有 `web/classic/src/services/enterprise.js` 保持必要企业 API 服务入口，不复制 Default 的上下文交互。[Source: _bmad-output/planning-artifacts/architecture.md#Classic Theme]

- [x] 补齐测试与 i18n，锁定 username 可读化与受控修改链路 (AC: 1, 2, 3)
  - [x] 后端服务测试覆盖可读 username 生成、rename 成功、非法 username 拒绝、重复 username 拒绝。[Source: service/enterprise/dingtalk_oauth_test.go; service/enterprise/department_membership_test.go]
  - [x] Controller 测试覆盖 rename API 成功、审计动作写入，以及非法请求错误映射。[Source: controller/enterprise/department_membership_test.go]
  - [x] Default 前端测试覆盖 rename schema、错误消息映射和当前成员治理入口渲染。[Source: web/default/src/features/enterprise-organization/enterprise-organization.test.tsx]
  - [x] 新增文案已同步到 `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`。[Source: web/default/src/i18n/locales/en.json; web/default/src/i18n/locales/zh.json]

## Dev Notes

- Story 6.4 是对现有 DingTalk OAuth / 通讯录同步用户创建路径的收敛，不是新账号系统。DingTalk 仍是专用 enterprise integration module，不复用 `custom_oauth_providers`，登录成功后继续复用现有 session 和 `setupLogin`。[Source: _bmad-output/planning-artifacts/architecture.md#DingTalk Integration Boundary]
- Username 生成的真实策略在 `service/enterprise/username_policy.go`：先归一化可读输入，再尝试稳定身份后缀；只有极端失败才有 `dingtalk_<number>` 兜底，因此验收重点应确认主要路径不再产生 `dt_编号`。[Source: service/enterprise/username_policy.go]
- Rename API 的权限边界来自路由中 `EnterpriseDepartmentAdmin("id")`，并要求目标用户是当前部门活跃成员。不要绕过该路径在全局 user controller 中提供企业 rename 快捷口。[Source: router/enterprise-router.go; service/enterprise/department_membership.go]
- 历史日志语义是硬边界：`logs.username` 表示消费发生时刻写入的快照，企业模块只读消费，不新增字段、不修改 `LogConsume`、不批量回写历史 username。[Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
- 6.5 会继续统一“可读名称优先，username / ID 兜底”的跨页面展示规则；6.4 只负责 username 本身的可读化、受控修改和当前治理视图刷新基础。[Source: _bmad-output/planning-artifacts/epics.md#Story 6.5: 企业页面统一用户标识展示规则]

### Current State Of Updated Files

- `service/enterprise/username_policy.go`：封装 username 归一化、格式校验、候选生成和唯一可用名解析；不要在 OAuth/sync 里复制一份字符串处理逻辑。
- `service/enterprise/dingtalk_oauth.go`：`findOrCreateUserForIdentity` 使用 `availableDingTalkUsername` 创建新用户；已绑定用户只更新 identity 与登录时间，不因 DingTalk 姓名变化自动 rename。
- `service/enterprise/dingtalk_sync_apply.go` / `dingtalk_sync_helpers.go`：通讯录同步创建新用户时使用 `availableSyncUsername`；已存在 binding 的用户只更新 identity 和 membership，不静默改现有 username。
- `service/enterprise/department_membership.go`：rename 只允许当前部门活跃成员；同名 no-op 允许通过；重复其他用户 username 会拒绝。
- `controller/enterprise/department_membership.go`：rename 事务同时写用户更新和 admin action，避免出现已改名但无审计的状态。
- `web/default/src/features/enterprise-organization/index.tsx`：rename 入口在 `Current Member Governance` 中，成功后失效成员、用户部门、预算和 allocation 相关 query key。

### Project Structure Notes

- 主要实现文件：
  - `service/enterprise/username_policy.go`
  - `service/enterprise/dingtalk_oauth.go`
  - `service/enterprise/dingtalk_sync_apply.go`
  - `service/enterprise/dingtalk_sync_helpers.go`
  - `service/enterprise/department_membership.go`
  - `controller/enterprise/department_membership.go`
  - `dto/enterprise/department_membership.go`
  - `router/enterprise-router.go`
  - `web/default/src/features/enterprise-organization/api.ts`
  - `web/default/src/features/enterprise-organization/index.tsx`
  - `web/default/src/features/enterprise-organization/types.ts`
  - `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `web/default/src/i18n/locales/{en,zh,fr,ru,ja,vi}.json`
- 保持不改或仅只读依赖：
  - `relay/**`
  - `pkg/billingexpr/**`
  - `model/log.go`
  - `controller/log.go`
  - `docs/openapi/relay.json`
  - `custom_oauth_providers` 相关模型与 controller

### Previous Story Intelligence

- Story 6.3 已把组织页收敛为“当前部门 + 当前成员”的治理工作区，并修复了裸 `User ID` 主路径。6.4 的 rename 入口应继续锚定这个上下文，避免把管理员带回全局用户页。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]
- 6.3 review 要求精确 query key，不允许用 `invalidateQueries({ queryKey: ['enterprise'] })` 粗暴全刷。6.4 rename 成功后的失效范围应继续保持部门、成员、预算、allocation 粒度。[Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md#Senior Developer Review (AI)]

### Validation Notes

- 已按 create-story checklist 复核：
  - 未发明新账号、OAuth、日志或 relay 路径。
  - 文件位置符合 Router -> Controller -> Service -> Model 分层。
  - 明确了历史日志/风险事件 username snapshot 保留规则。
  - 明确了权限、唯一性、审计、缓存失效和 i18n/测试覆盖。
  - 未把 6.5 的统一展示规则提前扩大为本故事范围。
- 本次 create-story 刷新未运行测试；依据来自现有测试文件和代码扫描。后续若要重新验证，建议运行：
  - `GOCACHE=/private/tmp/new-api-go-build-cache go test ./service/enterprise ./controller/enterprise`
  - `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
  - `cd web/default && bun run typecheck`
  - `cd web/default && bun run i18n:sync`

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.4: 钉钉账号 username 策略与受控修改]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-6: 支持钉钉 OAuth 登录]
- [Source: _bmad-output/planning-artifacts/prds/prd-new-api-2026-05-27/prd.md#FR-14: 查看部门用量详情]
- [Source: _bmad-output/planning-artifacts/architecture.md#DingTalk Integration Boundary]
- [Source: _bmad-output/planning-artifacts/architecture.md#企业 ↔ 现有 log]
- [Source: _bmad-output/implementation-artifacts/2-3-support-dingtalk-oauth-login-and-account-binding.md]
- [Source: _bmad-output/implementation-artifacts/6-3-move-membership-and-budget-operations-into-department-context.md]

## Dev Agent Record

### Agent Model Used

GPT-5 Codex

### Debug Log References

- 2026-06-03 02:18 +0800：执行 `bmad-create-story` workflow for Story 6.4。已读取 `.claude/skills/bmad-create-story/SKILL.md`、`discover-inputs.md`、`template.md`、`checklist.md`。
- 2026-06-03 02:18 +0800：`_bmad/scripts/resolve_customization.py` 因当前默认 Python 缺少 `tomllib` 失败，已按 skill fallback 手工读取 `.claude/skills/bmad-create-story/customize.toml`；无 team/user override。
- 2026-06-03 02:18 +0800：已读取 BMAD config、sprint-status、Epic 6、PRD FR-6/FR-14、architecture DingTalk/log 边界、Story 6.3 以及相关代码实现。
- 2026-06-03 02:18 +0800：发现 Story 6.4 artifact 与 sprint-status 均已是 `done`，因此本次未按 create-story 默认流程把状态回退为 `ready-for-dev`，也未修改 `sprint-status.yaml`。
- 2026-06-03 02:18 +0800：未找到 `project-context.md` persistent fact 文件；本次以 AGENTS.md、BMAD planning artifacts 和代码扫描作为上下文来源。
- 2026-06-03 02:12 +0800：执行 `bmad-dev-story` workflow for Story 6.4。已读取 `.claude/skills/bmad-dev-story/SKILL.md` 与 checklist；resolver 因默认 Python 缺少 `tomllib` 失败，已按 fallback 手工读取 base customization，无 team/user override。
- 2026-06-03 02:12 +0800：复核 6.4 代码路径与 AC，确认 username 生成、受控 rename、审计、缓存刷新、历史快照规则、Default 入口、i18n 与测试覆盖均已落地。
- 2026-06-03 02:12 +0800：完整后端回归初次暴露非 6.4 阻塞：usage report 失败 job 被计入 processed，以及 DingTalk 测试依赖 `httptest.NewServer` 在当前 sandbox 无法绑定端口。已修复失败统计语义并将相关 DingTalk 测试改为注入式 mock HTTP client。
- 2026-06-03 02:12 +0800：最终验证通过：`GOCACHE=/private/tmp/new-api-go-build-cache go test ./service/enterprise ./controller/enterprise`、`bun test src/features/enterprise-organization/enterprise-organization.test.tsx`、`bun run typecheck`、`bun run i18n:sync`。

### Completion Notes List

- 刷新 Story 6.4 故事文档，补齐已实现代码路径、任务完成状态、架构边界、上一故事约束、测试证据和 checklist 复核结论。
- 明确可读 username 生成与受控 rename 均已落在现有企业模块内，不新增 OAuth/session/relay/billing/log 主链路。
- 明确历史日志和风险事件继续保留原 username snapshot，当前治理视图通过 query 失效读取最新 username。
- 完成 dev-story 验证并修复回归阻塞：usage report 失败 job 现在会在写入失败状态后继续向调度层返回失败；DingTalk service 测试不再依赖本地端口监听。

### Change Log

- 2026-06-03 02:12 +0800：执行 Story 6.4 dev-story 验证；确认全部任务与验收条件完成，状态更新为 `review`。
- 2026-06-03 02:12 +0800：修复完整 enterprise 回归中的失败统计与 DingTalk 测试环境适配问题，确保后端 enterprise 包、Default 组织页测试、typecheck 与 i18n sync 全部通过。
- 2026-06-03 02:27 +0800：执行 Story 6.4 story-automator review；修复 username 生成事务上下文、rename 嵌套事务问题，补齐 API 测试文件记录，状态更新为 `done`。

### Senior Developer Review (AI)

#### Review Summary

- 结论：Approve after auto-fix。已验证 AC 1/2/3 和已勾选任务，剩余 CRITICAL 问题为 0，Story 状态更新为 `done`。
- Git vs Story File List：发现 1 个文档差异：`tests/api/enterprise_department_member_username_test.go` 已作为 rename API / 历史日志快照验证出现，但未列入 story File List；已补入 File List。

#### Findings Fixed

- [HIGH][Fixed] `service/enterprise/username_policy.go` 的 username 可用性解析通过全局 `model.DB` / `model.GetMaxUserId()` 判断唯一性，而 DingTalk OAuth / sync 创建路径实际运行在 service DB 或事务句柄上。测试库、事务内新建用户、以及非全局 DB 上下文可能得到错误候选名。已改为 `resolveAvailableEnterpriseUsername(db, ...)`，并由 DingTalk OAuth / sync caller 传入当前 GORM 句柄。
- [HIGH][Fixed] `service/enterprise/department_membership.go` 的 `RenameDepartmentMember` 在 controller 已开启事务后再次开启 service 内部事务。SQLite/GORM nested transaction 会增加 savepoint 复杂度，且不利于保证“用户改名 + admin action 审计”由 controller 同一事务统一提交。已移除 service 内层事务，所有 rename 校验、更新和缓存失效都使用传入的 DB/tx 句柄执行。
- [MEDIUM][Fixed] story File List 未记录 `tests/api/enterprise_department_member_username_test.go`，但该测试覆盖 rename API 当前视图刷新、历史 `logs.username` snapshot 保留、审计 payload 和错误映射。已补齐 File List，避免验收证据与 story 文档不一致。

#### Validation

- `GOCACHE=/private/tmp/new-api-go-build-cache go test ./service/enterprise ./controller/enterprise ./tests/api`
- `cd web/default && bun test src/features/enterprise-organization/enterprise-organization.test.tsx`
- `cd web/default && bun run typecheck`
- `cd web/default && bun run i18n:sync`

### File List

- `_bmad-output/implementation-artifacts/6-4-fix-dingtalk-username-strategy-and-controlled-rename.md`
- `_bmad-output/implementation-artifacts/sprint-status.yaml`
- `service/enterprise/username_policy.go`
- `service/enterprise/usage_report_task.go`
- `service/enterprise/dingtalk_client_test.go`
- `service/enterprise/dingtalk_connectivity_test.go`
- `service/enterprise/dingtalk_oauth.go`
- `service/enterprise/dingtalk_oauth_test.go`
- `service/enterprise/dingtalk_sync_apply.go`
- `service/enterprise/dingtalk_sync_helpers.go`
- `service/enterprise/department_membership.go`
- `tests/api/enterprise_department_member_username_test.go`
- `controller/enterprise/department_membership.go`
- `dto/enterprise/department_membership.go`
- `router/enterprise-router.go`
- `web/default/src/features/enterprise-organization/api.ts`
- `web/default/src/features/enterprise-organization/index.tsx`
- `web/default/src/features/enterprise-organization/types.ts`
- `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/vi.json`
