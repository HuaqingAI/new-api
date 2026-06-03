# 测试自动化总结

## Story

- Story 6.7: 提供 Mock、Fixture 与 Contract Conformance 套件
- Workflow: `.claude/skills/bmad-qa-generate-e2e-tests`
- 时间: 2026-06-03 +0800

## 已生成/补齐的测试

### API / Contract Conformance 测试

- [x] `tests/agentplatform/conformance/fixtures.go` - 补齐 `open_capability_quota_or_rate_limited` fixture，覆盖 `quotaOrRateLimited` 稳定错误码与 retryable envelope。
- [x] `tests/agentplatform/conformance/fixtures_test.go` - 增加 fixture payload round-trip 测试，按真实 DTO / service response 类型反序列化，锁定字段解析和必填 JSON 字段。
- [x] `tests/agentplatform/conformance/fixtures_test.go` - 强化错误 envelope 断言，覆盖 `permissionDenied`、`resourceRevoked`、`resourceOffline`、`quotaOrRateLimited`、`timeout`、`upstreamFailed`、`contractInvalid`。
- [x] `tests/agentplatform/conformance/fixtures_test.go` - 强化 synthetic data / provider-native 泄漏扫描，禁止 provider secret、tenant secret、provider config、provider-native 字段和 endpoint 泄漏进入 fixture。
- [x] `controller/agentplatform/open_capabilities_test.go` - 复跑现有 runtime handler conformance，覆盖 discovery/detail/refresh、TTL/ETag/version diagnostics、revoked/offline、Skill invoke timeout/upstream failure、Knowledge provider failure 和 missing scope。
- [x] `controller/agentplatform/open_capabilities_test.go` - 新增 expired access token 与 revoked grant token 的 bearer API 回归，锁定 OAuth 失败 payload 映射为 `permissionDenied`。
- [x] `service/agentplatform/open_capability_errors.go` - 自动应用测试发现的 gap：JWT expired / malformed / signature / not-valid-yet 错误现在归一到 open-capability stable error envelope 的 `permissionDenied`。

### E2E / UI 测试

- [x] 本 story 不涉及 UI surface；无需新增 Playwright/Cypress/browser E2E。
- [x] 项目当前未在 AP-6 fixture/conformance 中使用前端测试框架；本次沿用现有 Go API 集成测试和 conformance package。

## 覆盖范围

- API surfaces: `/api/agent-platform/oauth/authorize`、`/api/agent-platform/oauth/token`、`/api/agent-platform/oauth/revoke`、`/api/open-capabilities/discovery`、`/api/open-capabilities/resources/{id}`、`/api/open-capabilities/refresh`、`/api/open-capabilities/skills/{id}/invoke`、`/api/open-capabilities/knowledge-bases/{id}/query`、`/api/open-capabilities/agents/{id}`。
- Fixture and runtime cases: OAuth success / expired token / revoked grant-token / missing scope，discovery empty / success / contract mismatch，detail visible-callable / visible-not-callable / invalid / revoked / offline，refresh fresh / stale / revoked / offline / ETag-version mismatch / TTL non-compliance，Skill invoke success / invalid / timeout / upstream failure，Knowledge query success / offline / upstream failure。
- Pending prerequisite cases: AP-6.5 model discovery default/no default/multiple default/default disabled/provider offline/unavailable/account-tenant mismatch；AP-6.6 error client state matrix。
- Coverage metrics: 36 fixture cases total, including 28 executable OAuth/open-capabilities/error cases and 8 explicit pending prerequisite cases.

## 验证结果

- `GOCACHE=/private/tmp/go-build-cache go test ./tests/agentplatform/conformance` 通过。
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform` 通过。
- `GOCACHE=/private/tmp/go-build-cache go test ./tests/...` 通过。
- `GOCACHE=/private/tmp/go-build-cache go test ./service/agentplatform ./controller/agentplatform ./tests/agentplatform/conformance ./tests/...` 通过。

## Checklist

- [x] API tests generated where applicable
- [x] E2E/UI tests generated for UI, or marked not applicable because this story has no UI surface
- [x] Tests use standard project framework APIs (`go test`)
- [x] Tests cover happy path
- [x] Tests cover critical error cases
- [x] All generated tests run successfully
- [x] Tests use proper API/contract assertions instead of brittle waits
- [x] Tests have clear descriptions and fixture names
- [x] No hardcoded waits or sleeps
- [x] Tests are independent
- [x] Test summary created
- [x] Tests saved to appropriate directories
- [x] Summary includes coverage metrics

## 备注

- `resolve_customization.py` 因当前 Python 缺少 `tomllib` 未能自动解析；已按 skill fallback 手工读取 `.claude/skills/bmad-qa-generate-e2e-tests/customize.toml`，无 team/user override，activation prepend/append 为空。
- `on_complete` 在 base customize 中为空；resolver 同样受 `tomllib` 限制，已按 fallback 确认为无后置动作。
- 额外复跑 `GOCACHE=/private/tmp/go-build-cache go test ./...` 未通过；失败集中在既有无关包：`model` 测试表未初始化、`relay/channel/claude` 文件内容转换断言、`relay/helper` stream status 断言。Story 6.7 目标包与 conformance 测试均通过。
