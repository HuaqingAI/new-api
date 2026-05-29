# Test Automation Summary

## Generated Tests

### API Tests
- [x] `tests/api/enterprise_usage_test.go` - 管理员权限、非法时间范围、空快照列表，以及多部门重复计入口径在跨窗口聚合后仍保留
- [x] `controller/enterprise/usage_test.go` - 控制器参数校验、未归属部门归一化、`model_distribution` 空数组契约

### E2E Tests
- [x] `web/default/src/features/enterprise-usage/enterprise-usage.test.tsx` - 时间预设解析、自定义范围解析、管理员路由门禁、固定 disclaimer、未归属行、模型分布摘要、loading/empty/error 状态

## Coverage

- API endpoint: `GET /api/enterprise/usage/department-summary` 4/4 critical scenarios covered
- UI feature: `enterprise-usage` 8/8 targeted story behaviors covered

## Validation

- [x] `bun run typecheck`
- [x] `bun run test:e2e`
- [x] `env GOCACHE=/private/tmp/new-api-go-build go test ./controller/enterprise ./tests/api -run 'Test(UsageSummary|EnterpriseUsageSummary)' -count=1`

## Notes

- 项目当前没有 Playwright/Cypress；Story 4.2 的“E2E”沿用现有 `rsbuild` + `node:test` 静态渲染测试流水。
- `bun run test:e2e` 存在既有 `supports-color` 构建 warning，但不影响测试通过。
