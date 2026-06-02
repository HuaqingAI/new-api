# 测试自动化总结

## Story

- Story 6.5: 企业页面统一用户标识展示规则
- Workflow: `.claude/skills/bmad-qa-generate-e2e-tests`
- 时间: 2026-06-03 +0800

## 已生成/补齐的测试

### API 测试

- [x] `controller/enterprise/usage_test.go` - 新增 `TestUsageDetailAPIKeepsUsernameAndUserIDWhenDisplayNameMissing`，覆盖用量详情在用户缺少 `display_name` 时仍返回 `username`、`user_id` 和空 `display_name`，确保前端可按统一规则回退展示。
- [x] `controller/enterprise/usage_test.go` - 验证 recent logs 用户筛选选项仍保留真实 `username`，避免把可读名误用为日志筛选值。
- [x] 复跑现有 6.5 API 回归，覆盖用量详情、风险事件、风险投递 trace 与 quota allocation 的用户展示字段透传。

### E2E / UI 测试

- [x] `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - 新增共享企业用户展示 helper 断言，锁定“可读名称优先，username 次之，user ID 最后兜底”的统一规则。
- [x] `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - 补充缺少 `user_id` 时的辅助标签断言，避免输出 `User ID #-`。
- [x] `web/classic/src/pages/Enterprise/Department.smoke.test.js` - 新增 Classic 部门成员入口 smoke，验证成员表使用共享格式化器并按可读名优先展示，补齐 Classic 最小一致性缺口。
- [x] `web/classic/src/pages/Enterprise/Alerts.smoke.test.js` / `Department.smoke.test.js` - 补充 Classic 缺少 `user_id` 时的辅助标签断言，避免输出 `用户 ID #-`。
- [x] 复跑现有 Default 企业组织、用量和风险测试，覆盖成员、预算 wallet、用量排行、recent logs 用户 chips、风险事件与投递 trace 的展示规则。

## 覆盖范围

- API endpoints: `GET /api/enterprise/usage/department-detail`、`GET /api/enterprise/alerts/events`、`GET /api/enterprise/alerts/deliveries`、quota allocation workflow
- UI features: Default 企业组织、预算 wallet、用量排行、最近日志入口、风险事件、风险投递 trace、Classic 部门成员表、Classic 风险事件表
- Happy path: 有 `display_name` 时主展示可读名，并用 `username / user ID` 辅助消歧
- Critical fallback cases: 无 `display_name` 时回退 `username`；无 `username` 时回退 `user ID`
- Contract preservation: recent logs 筛选仍使用真实 `username`，风险历史 `username_snapshot` 不被当前可读名覆盖

## 验证结果

- `GOCACHE=/private/tmp/new-api-go-build-cache go test ./controller/enterprise -run 'TestUsageDetailAPIValidatesTimeRangeAndNormalizesArrays|TestUsageDetailAPIKeepsUsernameAndUserIDWhenDisplayNameMissing|TestAlertEventsAPIValidatesQueryAndReturnsPaginationEnvelope|TestAlertDeliveriesAPIValidatesQueryAndReturnsEnvelope|TestQuotaAllocationAPIWorkflow' -count=1` 通过
- `cd web/default && bun test ./src/features/enterprise-organization/enterprise-organization.test.tsx ./src/features/enterprise-usage/enterprise-usage.test.tsx ./src/features/enterprise-alerts/enterprise-alerts.test.tsx` 通过，93/93
- `cd web/default && bun run test:e2e` 通过
- `cd web/classic && bun test ./src/pages/Enterprise/Alerts.smoke.test.js ./src/pages/Enterprise/Department.smoke.test.js` 通过，6/6
- `cd web/classic && bun test ./src/pages/Enterprise/*.smoke.test.js` 通过，6/6
- 2026-06-03 review 复跑：`cd web/default && bun test ./src/features/enterprise-organization/enterprise-organization.test.tsx ./src/features/enterprise-usage/enterprise-usage.test.tsx ./src/features/enterprise-alerts/enterprise-alerts.test.tsx` 通过，93/93
- 2026-06-03 review 复跑：`cd web/classic && bun test ./src/pages/Enterprise/Alerts.smoke.test.js ./src/pages/Enterprise/Department.smoke.test.js` 通过，6/6

## Checklist

- [x] API tests generated where applicable
- [x] E2E/UI tests generated for UI
- [x] Tests use standard project framework APIs (`go test`, `bun test`, `rsbuild` + `node:test`)
- [x] Tests cover happy path
- [x] Tests cover critical error/fallback cases
- [x] All generated tests run successfully
- [x] Tests use existing semantic/static UI assertions available in the current framework
- [x] Tests have clear descriptions
- [x] No hardcoded waits or sleeps
- [x] Tests are independent
- [x] Test summary created
- [x] Tests saved to appropriate directories
- [x] Summary includes coverage metrics

## 备注

- 项目当前未使用 Playwright/Cypress；本次继续沿用现有 Go API 集成测试、Default `rsbuild`/`node:test` E2E 构建脚本和 Classic Bun smoke 测试模式。
- `bun run test:e2e` 构建阶段仍输出现有 `debug` 包可选依赖 `supports-color` warning；测试执行全部通过。
- customization resolver 因当前 Python 缺少 `tomllib` 未能自动解析；已按 skill fallback 手工读取 `.claude/skills/bmad-qa-generate-e2e-tests/customize.toml`，无 team/user override，`on_complete` 为空。
