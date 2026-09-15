---
title: 'Fix Kubernetes deploy auth diagnostics'
type: 'bugfix'
created: '2026-06-18'
status: 'done'
route: 'one-shot'
---

# Fix Kubernetes deploy auth diagnostics

## Intent

**Problem:** The Docker release workflow failed during Kubernetes deployment with `the server has asked for the client to provide credentials`, but the newly added Helm dry-run guard reported it as a chart/values pre-scale failure.

**Approach:** Validate Kubernetes auth/permission before deployment, keep Helm pre-scale rendering independent from the misleading `helm upgrade --dry-run=client` path, and document that `KUBE_CONFIG` must contain CI-usable non-interactive credentials.

## Suggested Review Order

1. `.github/workflows/docker-build.yml` -- Confirm deployment auth validation now fails with a precise kubeconfig/permission error before Helm work starts.
2. `.github/workflows/docker-build.yml` -- Confirm Helm template validation preserves install/upgrade semantics closely enough for the RWO pre-scale guard.
3. `docs/contribution-guide.md` -- Confirm deployment configuration docs now explain CI kubeconfig and namespace permission requirements.
