package middleware

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/authz"
)

type accessTokenRuleKind int

const (
	// accessTokenRuleScope requires the token to carry scope.
	accessTokenRuleScope accessTokenRuleKind = iota + 1
	// accessTokenRuleAny accepts every valid scoped token; the handler applies
	// its own checks.
	accessTokenRuleAny
	// accessTokenRuleSession rejects every access token, legacy ones included.
	accessTokenRuleSession
)

type accessTokenRouteRule struct {
	kind  accessTokenRuleKind
	scope string
}

var (
	accessTokenAnyRule     = accessTokenRouteRule{kind: accessTokenRuleAny}
	accessTokenSessionRule = accessTokenRouteRule{kind: accessTokenRuleSession}
)

func accessTokenScopeRule(scope string) accessTokenRouteRule {
	return accessTokenRouteRule{kind: accessTokenRuleScope, scope: scope}
}

// accessTokenRouteRules declares the token rule of every dashboard route that
// is not guarded by RequirePermission, keyed by method and gin full path.
// Casbin-guarded routes declare themselves through
// DeclareAccessTokenPermissionRoute when they are registered.
var accessTokenRouteRules = map[string]accessTokenRouteRule{
	// router/api-router.go: top level
	"GET /api/models":                     accessTokenScopeRule("profile:read"),
	"GET /api/status/test":                accessTokenScopeRule("log:read"),
	"GET /api/pricing":                    accessTokenAnyRule,
	"GET /api/perf-metrics/summary":       accessTokenAnyRule,
	"GET /api/perf-metrics":               accessTokenAnyRule,
	"GET /api/rankings":                   accessTokenAnyRule,
	"POST /api/oauth/state":               accessTokenSessionRule,
	"POST /api/oauth/email/bind/start":    accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/email/bind/resend":   accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/email/bind":          accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/wechat/bind":         accessTokenScopeRule("account_security:write"),
	"POST /api/oauth/telegram/bind/start": accessTokenSessionRule,
	"GET /api/oauth/:provider":            accessTokenSessionRule,
	"GET /api/verify/methods":             accessTokenAnyRule,
	"POST /api/verify":                    accessTokenAnyRule,

	// router/api-router.go: /api/user (self)
	"GET /api/user/sessions":                       accessTokenSessionRule,
	"DELETE /api/user/sessions/:sid":               accessTokenSessionRule,
	"POST /api/user/sessions/revoke-others":        accessTokenSessionRule,
	"GET /api/user/self/groups":                    accessTokenScopeRule("profile:read"),
	"GET /api/user/self":                           accessTokenScopeRule("profile:read"),
	"GET /api/user/models":                         accessTokenScopeRule("profile:read"),
	"PUT /api/user/self":                           accessTokenScopeRule("profile:write"),
	"DELETE /api/user/self":                        accessTokenScopeRule("account_security:write"),
	"GET /api/user/access_tokens":                  accessTokenSessionRule,
	"GET /api/user/access_tokens/catalog":          accessTokenSessionRule,
	"GET /api/user/access_tokens/scopes":           accessTokenSessionRule,
	"POST /api/user/access_tokens":                 accessTokenSessionRule,
	"PATCH /api/user/access_tokens/:id":            accessTokenSessionRule,
	"DELETE /api/user/access_tokens/:id":           accessTokenSessionRule,
	"DELETE /api/user/access_tokens/legacy":        accessTokenSessionRule,
	"GET /api/user/passkey":                        accessTokenScopeRule("account_security:read"),
	"POST /api/user/passkey/register/begin":        accessTokenScopeRule("account_security:write"),
	"POST /api/user/passkey/register/finish":       accessTokenScopeRule("account_security:write"),
	"POST /api/user/passkey/verify/begin":          accessTokenAnyRule,
	"POST /api/user/passkey/verify/finish":         accessTokenAnyRule,
	"DELETE /api/user/passkey":                     accessTokenScopeRule("account_security:write"),
	"GET /api/user/aff":                            accessTokenScopeRule("wallet:read"),
	"GET /api/user/topup/info":                     accessTokenScopeRule("wallet:read"),
	"GET /api/user/topup/self":                     accessTokenScopeRule("wallet:read"),
	"POST /api/user/topup":                         accessTokenScopeRule("wallet:write"),
	"POST /api/user/pay":                           accessTokenScopeRule("wallet:write"),
	"POST /api/user/amount":                        accessTokenScopeRule("wallet:read"),
	"POST /api/user/stripe/pay":                    accessTokenScopeRule("wallet:write"),
	"POST /api/user/stripe/amount":                 accessTokenScopeRule("wallet:read"),
	"POST /api/user/creem/pay":                     accessTokenScopeRule("wallet:write"),
	"POST /api/user/waffo/amount":                  accessTokenScopeRule("wallet:read"),
	"POST /api/user/waffo/pay":                     accessTokenScopeRule("wallet:write"),
	"POST /api/user/waffo-pancake/amount":          accessTokenScopeRule("wallet:read"),
	"POST /api/user/waffo-pancake/pay":             accessTokenScopeRule("wallet:write"),
	"POST /api/user/aff_transfer":                  accessTokenScopeRule("wallet:write"),
	"PUT /api/user/setting":                        accessTokenScopeRule("profile:write"),
	"GET /api/user/2fa/status":                     accessTokenScopeRule("account_security:read"),
	"POST /api/user/2fa/setup":                     accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/enable":                    accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/disable":                   accessTokenScopeRule("account_security:write"),
	"POST /api/user/2fa/backup_codes":              accessTokenScopeRule("account_security:write"),
	"GET /api/user/checkin":                        accessTokenScopeRule("wallet:read"),
	"POST /api/user/checkin":                       accessTokenScopeRule("wallet:write"),
	"GET /api/user/oauth/bindings":                 accessTokenScopeRule("account_security:read"),
	"DELETE /api/user/oauth/bindings/:provider_id": accessTokenScopeRule("account_security:write"),

	// router/api-router.go: /api/user (admin)
	"GET /api/user/":                                   accessTokenScopeRule("user:read"),
	"GET /api/user/topup":                              accessTokenScopeRule("billing:read"),
	"POST /api/user/topup/complete":                    accessTokenScopeRule("billing:write"),
	"GET /api/user/search":                             accessTokenScopeRule("user:read"),
	"GET /api/user/:id/oauth/bindings":                 accessTokenScopeRule("user:read"),
	"DELETE /api/user/:id/oauth/bindings/:provider_id": accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id/bindings/:binding_type":      accessTokenScopeRule("user:write"),
	"GET /api/user/:id":                                accessTokenScopeRule("user:read"),
	"POST /api/user/":                                  accessTokenScopeRule("user:write"),
	"POST /api/user/manage":                            accessTokenScopeRule("user:write"),
	"PUT /api/user/":                                   accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id":                             accessTokenScopeRule("user:write"),
	"DELETE /api/user/:id/reset_passkey":               accessTokenScopeRule("user:write"),
	"GET /api/user/2fa/stats":                          accessTokenScopeRule("user:read"),
	"DELETE /api/user/:id/2fa":                         accessTokenScopeRule("user:write"),

	// router/api-router.go: /api/subscription
	"GET /api/subscription/plans":                                    accessTokenScopeRule("wallet:read"),
	"GET /api/subscription/self":                                     accessTokenScopeRule("wallet:read"),
	"PUT /api/subscription/self/preference":                          accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/balance/pay":                             accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/epay/pay":                                accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/stripe/pay":                              accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/creem/pay":                               accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/waffo-pancake/pay":                       accessTokenScopeRule("wallet:write"),
	"GET /api/subscription/admin/plans":                              accessTokenScopeRule("billing:read"),
	"POST /api/subscription/admin/plans":                             accessTokenScopeRule("billing:write"),
	"PUT /api/subscription/admin/plans/:id":                          accessTokenScopeRule("billing:write"),
	"PATCH /api/subscription/admin/plans/:id":                        accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/bind":                              accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/plans/:id/subscriptions/reset":     accessTokenScopeRule("billing:write"),
	"GET /api/subscription/admin/users/:id/subscriptions":            accessTokenScopeRule("billing:read"),
	"POST /api/subscription/admin/users/:id/subscriptions":           accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/users/:id/subscriptions/reset":     accessTokenScopeRule("billing:write"),
	"POST /api/subscription/admin/user_subscriptions/:id/invalidate": accessTokenScopeRule("billing:write"),
	"DELETE /api/subscription/admin/user_subscriptions/:id":          accessTokenScopeRule("billing:write"),

	// router/api-router.go: /api/option
	"GET /api/option/":                                           accessTokenScopeRule("option:read"),
	"GET /api/option/request_policy":                             accessTokenScopeRule("option:read"),
	"PATCH /api/option/request_policy":                           accessTokenScopeRule("option:write"),
	"PUT /api/option/":                                           accessTokenScopeRule("option:write"),
	"PUT /api/option/passkey/domains":                            accessTokenScopeRule("option:write"),
	"GET /api/option/model_pricing":                              accessTokenScopeRule("option:read"),
	"PATCH /api/option/model_pricing":                            accessTokenScopeRule("option:write"),
	"POST /api/option/model_pricing/convert":                     accessTokenScopeRule("option:read"),
	"POST /api/option/model_pricing/preview":                     accessTokenScopeRule("option:read"),
	"POST /api/option/payment_compliance":                        accessTokenSessionRule,
	"GET /api/option/channel_affinity_cache":                     accessTokenScopeRule("option:read"),
	"DELETE /api/option/channel_affinity_cache":                  accessTokenScopeRule("option:write"),
	"POST /api/option/rest_model_ratio":                          accessTokenScopeRule("option:write"),
	"GET /api/option/waffo-pancake/catalog":                      accessTokenScopeRule("option:read"),
	"POST /api/option/waffo-pancake/pair":                        accessTokenScopeRule("option:write"),
	"POST /api/option/waffo-pancake/save":                        accessTokenScopeRule("option:write"),
	"POST /api/option/waffo-pancake/subscription-product":        accessTokenScopeRule("option:write"),
	"GET /api/option/waffo-pancake/subscription-product-options": accessTokenScopeRule("option:read"),

	// router/api-router.go: /api/custom-oauth-provider, /api/ratio_sync
	"POST /api/custom-oauth-provider/discovery": accessTokenScopeRule("option:write"),
	"GET /api/custom-oauth-provider/":           accessTokenScopeRule("option:read"),
	"GET /api/custom-oauth-provider/:id":        accessTokenScopeRule("option:read"),
	"POST /api/custom-oauth-provider/":          accessTokenScopeRule("option:write"),
	"PUT /api/custom-oauth-provider/:id":        accessTokenScopeRule("option:write"),
	"DELETE /api/custom-oauth-provider/:id":     accessTokenScopeRule("option:write"),
	"GET /api/ratio_sync/channels":              accessTokenScopeRule("option:read"),
	"POST /api/ratio_sync/fetch":                accessTokenScopeRule("option:write"),

	// router/api-router.go: /api/performance, /api/system-task, /api/system-info
	"GET /api/performance/stats":                   accessTokenScopeRule("ops:read"),
	"DELETE /api/performance/disk_cache":           accessTokenScopeRule("ops:write"),
	"POST /api/performance/reset_stats":            accessTokenScopeRule("ops:write"),
	"POST /api/performance/gc":                     accessTokenScopeRule("ops:write"),
	"GET /api/performance/logs":                    accessTokenScopeRule("ops:read"),
	"DELETE /api/performance/logs":                 accessTokenScopeRule("ops:write"),
	"POST /api/system-task/log-cleanup":            accessTokenScopeRule("ops:write"),
	"GET /api/system-task/list":                    accessTokenScopeRule("ops:read"),
	"DELETE /api/system-task/history":              accessTokenScopeRule("ops:write"),
	"GET /api/system-task/current":                 accessTokenScopeRule("ops:read"),
	"GET /api/system-task/:task_id":                accessTokenScopeRule("ops:read"),
	"GET /api/system-info/instances":               accessTokenScopeRule("ops:read"),
	"DELETE /api/system-info/stale-instances":      accessTokenScopeRule("ops:write"),
	"DELETE /api/system-info/instances/:node_name": accessTokenScopeRule("ops:write"),

	// router/api-router.go: /api/plugin/task
	"GET /api/plugin/task":                           accessTokenScopeRule("plugin:read"),
	"POST /api/plugin/task":                          accessTokenScopeRule("plugin:write"),
	"PUT /api/plugin/task":                           accessTokenScopeRule("plugin:write"),
	"GET /api/plugin/task/runtime/status":            accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/marketplace/sources":       accessTokenScopeRule("plugin:read"),
	"PUT /api/plugin/task/marketplace/sources":       accessTokenScopeRule("plugin:write"),
	"GET /api/plugin/task/:key":                      accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/:key/icon":                 accessTokenScopeRule("plugin:read"),
	"GET /api/plugin/task/:key/versions":             accessTokenScopeRule("plugin:read"),
	"POST /api/plugin/task/:key/activate":            accessTokenScopeRule("plugin:write"),
	"POST /api/plugin/task/:key/status":              accessTokenScopeRule("plugin:write"),
	"POST /api/plugin/task/:key/dryrun":              accessTokenScopeRule("plugin:write"),
	"DELETE /api/plugin/task/:key/versions/:version": accessTokenScopeRule("plugin:write"),

	// router/channel-router.go: the key route is RootAuth, not Casbin-guarded.
	"POST /api/channel/:id/key": accessTokenScopeRule(service.AccessTokenScopeOf(authz.ChannelSecretView)),

	// router/authz-router.go
	"GET /api/authz/catalog": accessTokenScopeRule("user:read"),

	// router/api-router.go: /api/token
	"GET /api/token/":            accessTokenScopeRule("api_key:read"),
	"GET /api/token/search":      accessTokenScopeRule("api_key:read"),
	"GET /api/token/auto-groups": accessTokenScopeRule("api_key:read"),
	"GET /api/token/:id":         accessTokenScopeRule("api_key:read"),
	"POST /api/token/:id/key":    accessTokenScopeRule("api_key:reveal"),
	"POST /api/token/":           accessTokenScopeRule("api_key:write"),
	"PUT /api/token/":            accessTokenScopeRule("api_key:write"),
	"DELETE /api/token/:id":      accessTokenScopeRule("api_key:write"),
	"POST /api/token/batch":      accessTokenScopeRule("api_key:write"),
	"POST /api/token/batch/keys": accessTokenScopeRule("api_key:reveal"),

	// router/api-router.go: /api/redemption
	"GET /api/redemption/":           accessTokenScopeRule("redemption:read"),
	"GET /api/redemption/search":     accessTokenScopeRule("redemption:read"),
	"GET /api/redemption/:id":        accessTokenScopeRule("redemption:read"),
	"POST /api/redemption/":          accessTokenScopeRule("redemption:write"),
	"POST /api/redemption/batch":     accessTokenScopeRule("redemption:write"),
	"PUT /api/redemption/":           accessTokenScopeRule("redemption:write"),
	"DELETE /api/redemption/invalid": accessTokenScopeRule("redemption:write"),
	"DELETE /api/redemption/:id":     accessTokenScopeRule("redemption:write"),

	// router/api-router.go: /api/audit, /api/log, /api/data
	"GET /api/audit/self":                       accessTokenScopeRule("usage:read"),
	"GET /api/log/":                             accessTokenScopeRule("log:read"),
	"GET /api/log/stat":                         accessTokenScopeRule("log:read"),
	"GET /api/log/self/stat":                    accessTokenScopeRule("usage:read"),
	"GET /api/log/channel_affinity_usage_cache": accessTokenScopeRule("log:read"),
	"GET /api/log/search":                       accessTokenScopeRule("log:read"),
	"GET /api/log/self":                         accessTokenScopeRule("usage:read"),
	"GET /api/log/self/search":                  accessTokenScopeRule("usage:read"),
	"GET /api/log/export":                       accessTokenScopeRule("log:read"),
	"GET /api/log/self/export":                  accessTokenScopeRule("usage:read"),
	"GET /api/data/":                            accessTokenScopeRule("log:read"),
	"GET /api/data/users":                       accessTokenScopeRule("log:read"),
	"GET /api/data/self":                        accessTokenScopeRule("usage:read"),
	"GET /api/data/flow":                        accessTokenScopeRule("log:read"),
	"GET /api/data/flow/self":                   accessTokenScopeRule("usage:read"),

	// router/api-router.go: /api/group, /api/prefill_group
	"GET /api/group/":               accessTokenScopeRule("group:read"),
	"GET /api/prefill_group/":       accessTokenScopeRule("group:read"),
	"POST /api/prefill_group/":      accessTokenScopeRule("group:write"),
	"PUT /api/prefill_group/":       accessTokenScopeRule("group:write"),
	"DELETE /api/prefill_group/:id": accessTokenScopeRule("group:write"),

	// router/api-router.go: /api/mj, /api/task
	"GET /api/mj/self":                 accessTokenScopeRule("usage:read"),
	"GET /api/mj/":                     accessTokenScopeRule("log:read"),
	"GET /api/task/self":               accessTokenScopeRule("usage:read"),
	"GET /api/task":                    accessTokenScopeRule("log:read"),
	"GET /api/task/:task_id/artifacts": accessTokenScopeRule("usage:read"),

	// router/api-router.go: /api/vendors, /api/models (admin)
	"POST /api/vendors/operations/preview":  accessTokenScopeRule("model:read"),
	"POST /api/vendors/operations":          accessTokenScopeRule("model:write"),
	"GET /api/vendors/":                     accessTokenScopeRule("model:read"),
	"GET /api/vendors/search":               accessTokenScopeRule("model:read"),
	"GET /api/vendors/:id":                  accessTokenScopeRule("model:read"),
	"POST /api/vendors/":                    accessTokenScopeRule("model:write"),
	"PUT /api/vendors/":                     accessTokenScopeRule("model:write"),
	"DELETE /api/vendors/:id":               accessTokenScopeRule("model:write"),
	"GET /api/models/sync_upstream/preview": accessTokenScopeRule("model:read"),
	"POST /api/models/sync_upstream":        accessTokenScopeRule("model:write"),
	"POST /api/models/delete":               accessTokenScopeRule("model:write"),
	"GET /api/models/missing":               accessTokenScopeRule("model:read"),
	"GET /api/models/":                      accessTokenScopeRule("model:read"),
	"GET /api/models/search":                accessTokenScopeRule("model:read"),
	"GET /api/models/:id":                   accessTokenScopeRule("model:read"),
	"POST /api/models/":                     accessTokenScopeRule("model:write"),
	"PUT /api/models/":                      accessTokenScopeRule("model:write"),
	"DELETE /api/models/:id":                accessTokenScopeRule("model:write"),

	// router/api-router.go: /api/deployments
	"GET /api/deployments/settings":                     accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/settings/test-connection":    accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/":                             accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/search":                       accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/test-connection":             accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/hardware-types":               accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/locations":                    accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/available-replicas":           accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/price-estimation":            accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/check-name":                   accessTokenScopeRule("deployment:read"),
	"POST /api/deployments/":                            accessTokenScopeRule("deployment:write"),
	"GET /api/deployments/:id":                          accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/logs":                     accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/containers":               accessTokenScopeRule("deployment:read"),
	"GET /api/deployments/:id/containers/:container_id": accessTokenScopeRule("deployment:read"),
	"PUT /api/deployments/:id":                          accessTokenScopeRule("deployment:write"),
	"PUT /api/deployments/:id/name":                     accessTokenScopeRule("deployment:write"),
	"POST /api/deployments/:id/extend":                  accessTokenScopeRule("deployment:write"),
	"DELETE /api/deployments/:id":                       accessTokenScopeRule("deployment:write"),

	// Local /api/enterprise routes.
	"GET /api/enterprise/departments/tree":                            accessTokenScopeRule("enterprise_organization:read"),
	"GET /api/enterprise/users/:id/departments":                       accessTokenScopeRule("enterprise_organization:read"),
	"PUT /api/enterprise/users/:id/departments":                       accessTokenSessionRule,
	"GET /api/enterprise/departments/:id/members":                     accessTokenScopeRule("enterprise_organization:read"),
	"GET /api/enterprise/departments/:id/budget":                      accessTokenScopeRule("enterprise_budget:read"),
	"GET /api/enterprise/departments/:id/budgets":                     accessTokenScopeRule("enterprise_budget:read"),
	"GET /api/enterprise/departments/:id/budgets/:budget_id":          accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/departments/:id/budget":                     accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/departments/:id/budgets/:budget_id/pause":   accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/departments/:id/budgets/:budget_id/resume":  accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/departments/:id/budgets/:budget_id/resize":  accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/public-budget-pools":                         accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/public-budget-pools":                        accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/public-budget-pools/:id":                     accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/public-budget-pools/:id/pause":              accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/public-budget-pools/:id/resume":             accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/public-budget-pools/:id/resize":             accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/quota-allocations":                           accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/quota-allocations":                          accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/quota-allocations/:id/supersede":            accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/quota-allocations/:id/cancel":               accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/quota-allocations/:id/reclaim":              accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/quota-allocations/:id/revoke":               accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/quota-requests":                              accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/quota-requests":                             accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/quota-requests/capability/:department_id":    accessTokenScopeRule("enterprise_budget:read"),
	"GET /api/enterprise/quota-requests/:id":                          accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/quota-requests/:id/decision":                accessTokenScopeRule("enterprise_budget:write"),
	"GET /api/enterprise/governance/timeline":                         accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/governance/notifications":                    accessTokenScopeRule("enterprise_governance:read"),
	"POST /api/enterprise/governance/notifications/:id/resend":        accessTokenScopeRule("enterprise_governance:write"),
	"GET /api/enterprise/budget-delegations":                          accessTokenScopeRule("enterprise_budget:read"),
	"POST /api/enterprise/budget-delegations":                         accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/budget-delegations/:id/supersede":           accessTokenScopeRule("enterprise_budget:write"),
	"POST /api/enterprise/departments/:id/members":                    accessTokenSessionRule,
	"PUT /api/enterprise/departments/:id/members/:user_id/username":   accessTokenSessionRule,
	"DELETE /api/enterprise/departments/:id/members/:user_id":         accessTokenSessionRule,
	"POST /api/enterprise/departments/:id/members/:user_id/restore":   accessTokenSessionRule,
	"POST /api/enterprise/departments/:id/admins":                     accessTokenSessionRule,
	"DELETE /api/enterprise/departments/:id/admins/:user_id":          accessTokenSessionRule,
	"GET /api/enterprise/departments/:id/owners":                      accessTokenScopeRule("enterprise_organization:read"),
	"POST /api/enterprise/departments/:id/owners/grants":              accessTokenSessionRule,
	"POST /api/enterprise/departments/:id/owners/denies":              accessTokenSessionRule,
	"DELETE /api/enterprise/departments/:id/owners/grants/:user_id":   accessTokenSessionRule,
	"DELETE /api/enterprise/departments/:id/owners/denies/:user_id":   accessTokenSessionRule,
	"GET /api/enterprise/admin-actions":                               accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/admin-actions/:id":                           accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/alerts/events":                               accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/alerts/department-summary":                   accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/alerts/deliveries":                           accessTokenScopeRule("enterprise_governance:read"),
	"POST /api/enterprise/alerts/deliveries/:id/resend":               accessTokenScopeRule("enterprise_governance:write"),
	"GET /api/enterprise/alerts/rules":                                accessTokenScopeRule("enterprise_governance:read"),
	"GET /api/enterprise/alerts/rules/:id":                            accessTokenScopeRule("enterprise_governance:read"),
	"PUT /api/enterprise/alerts/rules":                                accessTokenScopeRule("enterprise_governance:write"),
	"DELETE /api/enterprise/alerts/rules/:id":                         accessTokenScopeRule("enterprise_governance:write"),
	"GET /api/enterprise/usage/department-summary":                    accessTokenScopeRule("enterprise_usage:read"),
	"GET /api/enterprise/usage/department-overview":                   accessTokenScopeRule("enterprise_usage:read"),
	"GET /api/enterprise/usage/department-peers":                      accessTokenScopeRule("enterprise_usage:read"),
	"GET /api/enterprise/usage/export":                                accessTokenScopeRule("enterprise_usage:read"),
	"GET /api/enterprise/usage/department-detail":                     accessTokenScopeRule("enterprise_usage:read"),
	"GET /api/enterprise/usage/reports":                               accessTokenScopeRule("enterprise_usage:read"),
	"PUT /api/enterprise/usage/reports":                               accessTokenScopeRule("enterprise_usage:write"),
	"POST /api/enterprise/usage/reports/send":                         accessTokenScopeRule("enterprise_usage:write"),
	"GET /api/enterprise/dingtalk/config":                             accessTokenSessionRule,
	"PUT /api/enterprise/dingtalk/config":                             accessTokenSessionRule,
	"POST /api/enterprise/dingtalk/connectivity-test":                 accessTokenSessionRule,
	"POST /api/enterprise/dingtalk/sync/full":                         accessTokenSessionRule,
	"GET /api/enterprise/dingtalk/sync/tasks/:id":                     accessTokenSessionRule,
	"GET /api/enterprise/dingtalk/sync/logs":                          accessTokenSessionRule,
	"GET /api/enterprise/dingtalk/sync/conflicts":                     accessTokenSessionRule,
	"POST /api/enterprise/dingtalk/sync/conflicts/:id/bind-candidate": accessTokenSessionRule,

	// Local /api/agent-platform routes.
	"GET /api/agent-platform/model-filter":                              accessTokenScopeRule("option:read"),
	"PUT /api/agent-platform/model-filter":                              accessTokenScopeRule("option:write"),
	"GET /api/agent-platform/resources":                                 accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/assets/avatars":                           accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources":                                accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/resources/:id":                             accessTokenScopeRule("agent_platform:read"),
	"GET /api/agent-platform/mcps":                                      accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/mcps":                                     accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/mcps/:id":                                  accessTokenScopeRule("agent_platform:read"),
	"PUT /api/agent-platform/mcps/:id":                                  accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/mcps/:id/disable":                         accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/mcps/:id/enable":                          accessTokenScopeRule("agent_platform:write"),
	"DELETE /api/agent-platform/mcps/:id":                               accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/skills":                                    accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/skills":                                   accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/skills/:id":                                accessTokenScopeRule("agent_platform:read"),
	"PUT /api/agent-platform/skills/:id":                                accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/skills/:id/package":                       accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/skills/:id/package/download":               accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/skills/:id/disable":                       accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/skills/:id/enable":                        accessTokenScopeRule("agent_platform:write"),
	"DELETE /api/agent-platform/skills/:id":                             accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/knowledge-bases":                           accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/knowledge-bases":                          accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/knowledge-bases/:id":                       accessTokenScopeRule("agent_platform:read"),
	"PUT /api/agent-platform/knowledge-bases/:id":                       accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/knowledge-bases/:id/disable":              accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/knowledge-bases/:id/enable":               accessTokenScopeRule("agent_platform:write"),
	"DELETE /api/agent-platform/knowledge-bases/:id":                    accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/agents":                                    accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/agents":                                   accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/agents/:id":                                accessTokenScopeRule("agent_platform:read"),
	"PUT /api/agent-platform/agents/:id":                                accessTokenScopeRule("agent_platform:write"),
	"DELETE /api/agent-platform/agents/:id":                             accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/agents/:id/versions":                       accessTokenScopeRule("agent_platform:read"),
	"GET /api/agent-platform/agents/:id/versions/:version/grants":       accessTokenScopeRule("agent_platform:read"),
	"GET /api/agent-platform/agents/:id/publish-defaults":               accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/agents/:id/publish":                       accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources/:id/versions":                   accessTokenScopeRule("agent_platform:write"),
	"GET /api/agent-platform/resources/:id/versions/:version":           accessTokenScopeRule("agent_platform:read"),
	"POST /api/agent-platform/resources/:id/publish":                    accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources/:id/disable":                    accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources/:id/revoke":                     accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources/:id/offline":                    accessTokenScopeRule("agent_platform:write"),
	"POST /api/agent-platform/resources/:id/versions/:version/rollback": accessTokenScopeRule("agent_platform:write"),

	// Local /api/aionui/client-packages routes.
	"GET /api/aionui/client-packages":                   accessTokenScopeRule("client_release:read"),
	"GET /api/aionui/client-packages/:id/download-url":  accessTokenScopeRule("client_release:read"),
	"POST /api/aionui/client-packages":                  accessTokenScopeRule("client_release:write"),
	"POST /api/aionui/client-packages/uploads/init":     accessTokenScopeRule("client_release:write"),
	"POST /api/aionui/client-packages/uploads/complete": accessTokenScopeRule("client_release:write"),
	"PATCH /api/aionui/client-packages/:id/status":      accessTokenScopeRule("client_release:write"),
	"GET /api/aionui/client-packages/:id/rollout":       accessTokenScopeRule("client_release:read"),
	"PUT /api/aionui/client-packages/:id/rollout":       accessTokenScopeRule("client_release:write"),
	"DELETE /api/aionui/client-packages/:id":            accessTokenScopeRule("client_release:write"),

	// Local /api/aionui/clients routes.
	"GET /api/aionui/clients": accessTokenScopeRule("client_release:read"),

	// Local subscription wallet ordering.
	"POST /api/subscription/self/reorder":                     accessTokenScopeRule("wallet:write"),
	"POST /api/subscription/admin/user_subscriptions/reorder": accessTokenScopeRule("billing:write"),

	// router/relay-router.go
	"POST /pg/chat/completions": accessTokenSessionRule,
}

var (
	permissionRouteRulesMu sync.RWMutex
	permissionRouteRules   = map[string]accessTokenRouteRule{}
)

// DeclareAccessTokenPermissionRoute records that the Casbin permission guarding
// a route is also the token scope it requires. It runs while routes are
// registered and panics on conflicting declarations.
func DeclareAccessTokenPermissionRoute(method, fullPath string, permission authz.Permission) {
	key := method + " " + fullPath
	rule := accessTokenScopeRule(service.AccessTokenScopeOf(permission))
	if _, ok := accessTokenRouteRules[key]; ok {
		panic(fmt.Sprintf("access token route %s is declared twice", key))
	}
	permissionRouteRulesMu.Lock()
	defer permissionRouteRulesMu.Unlock()
	if existing, ok := permissionRouteRules[key]; ok && existing != rule {
		panic(fmt.Sprintf("access token route %s is declared with %s and %s", key, existing.scope, rule.scope))
	}
	permissionRouteRules[key] = rule
}

// AccessTokenRouteRule returns the rule for "METHOD /full/path".
func AccessTokenRouteRule(key string) (accessTokenRouteRule, bool) {
	if rule, ok := accessTokenRouteRules[key]; ok {
		return rule, true
	}
	permissionRouteRulesMu.RLock()
	defer permissionRouteRulesMu.RUnlock()
	rule, ok := permissionRouteRules[key]
	return rule, ok
}

// AccessTokenRouteRuleKeys lists every declared route key, sorted.
func AccessTokenRouteRuleKeys() []string {
	permissionRouteRulesMu.RLock()
	keys := slices.Collect(maps.Keys(permissionRouteRules))
	permissionRouteRulesMu.RUnlock()
	keys = slices.AppendSeq(keys, maps.Keys(accessTokenRouteRules))
	slices.Sort(keys)
	return keys
}

// Scope returns the token scope the rule requires, or "" for any/session rules.
func (rule accessTokenRouteRule) Scope() string {
	return rule.scope
}

// Kind names the rule category: "scope", "any" or "session".
func (rule accessTokenRouteRule) Kind() string {
	switch rule.kind {
	case accessTokenRuleScope:
		return "scope"
	case accessTokenRuleAny:
		return "any"
	case accessTokenRuleSession:
		return "session"
	}
	return ""
}
