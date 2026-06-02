# 测试自动化总结

## 已生成/补强的测试

### API 测试
- [x] `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 以现有 bundled `node:test` 模式补强钱包侧额度申请 API contract 覆盖，验证员工入口继续复用 `/api/enterprise/users/:user_id/departments`、`/api/enterprise/quota-requests/capability/:department_id` 与 `/api/enterprise/quota-requests`，并复用 `enterprise.organization.quota-request` query scope。

### E2E / UI 风格测试
- [x] `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 覆盖钱包页“申请额度”入口可见文案、显式目标部门/预算池选择、申请额度输入与提交按钮，且不依赖 `Enterprise Organization` 导航。
- [x] `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 覆盖仅保留 active membership 的部门过滤、预算池必须来自当前部门 capability、无可申请部门时的员工侧空态引导。
- [x] `web/default/src/features/wallet/components/employee-quota-request-card.test.tsx` - 覆盖钱包提交成功后失效治理时间线与通知投递 query cache，避免负责人工作台闭环视图滞后。
- [x] `controller/enterprise/department_membership_test.go` - 覆盖普通员工可读取自身部门成员关系、不可读取他人部门成员关系，确保钱包入口不依赖管理员菜单且不扩大权限面。
- [x] 既有 `web/default/src/features/enterprise-organization/enterprise-organization.test.tsx` - 回归负责人/管理员治理工作台审批、治理时间线与通知投递视图，确保入口迁移不破坏治理闭环。

## 覆盖范围

- API endpoints: 3/3（本故事涉及的既有 quota request workflow 端点均已通过前端 API wrapper 测试覆盖）。
- UI features: 3/3 个验收标准已覆盖：
  - AC1：普通员工在钱包区域可见“申请额度”入口，断言不依赖 `Enterprise Organization` 管理菜单。
  - AC2：提交前必须显式选择目标部门与目标预算池，并继续复用现有 quota request API / workflow。
  - AC3：企业组织治理工作台的审批表格、治理时间线和通知投递视图保持在既有测试中回归覆盖。

## Checklist 校验

- [x] API tests generated（适用；覆盖 3 个既有 quota request workflow endpoint wrapper）。
- [x] E2E tests generated（UI 存在，已补强钱包侧渲染/流程测试）。
- [x] Tests use standard test framework APIs（沿用项目现有 `node:test`、`assert`、Rsbuild node bundle 与 React server rendering）。
- [x] Tests cover happy path（钱包侧入口文案、部门/预算池显式选择、API workflow 复用）。
- [x] Tests cover 1-2 critical error cases（非 active membership 过滤、预算池不属于 capability 时返回 `null`、无可申请部门空态）。
- [x] All generated tests run successfully。
- [x] Tests use proper locators/assertions（基于语义可见文本与明确 API payload/query key 断言）。
- [x] Tests have clear descriptions。
- [x] No hardcoded waits or sleeps。
- [x] Tests are independent（每个测试使用独立 QueryClient；API monkey patch 在 finally 中恢复）。
- [x] Test summary created。
- [x] Tests saved to appropriate directories。
- [x] Summary includes coverage metrics。

## 验证结果

- `cd web/default && bun run test:e2e` ✅ 通过。Rsbuild 输出既有 optional dependency warning：`supports-color` 未解析；随后全部前端 bundled tests 通过，其中 `employee-quota-request-card` 为 6/6 通过，整体 99/99 通过。
- `cd web/default && bun run typecheck` ✅ 通过。
- `cd /Users/hq-it/repository/github/huaqingai/new-api && go test ./controller/enterprise` ✅ 通过。

## 后续建议

- 合并前继续在 CI 中运行 `bun run test:e2e`，锁定钱包侧员工额度申请入口与企业治理闭环回归覆盖。
