---
title: 'Configurable Release Image Repository'
type: brownfield
created: '2026-06-08T07:30:11Z'
status: done
route: one-shot
---

# Configurable Release Image Repository

## Intent

Problem: The release Docker workflow pushed images only to `calciumion/new-api`, so configuring the Kubernetes image repository alone could not publish images to the user's Alibaba Cloud ACR repository.

Approach: Parameterize the release workflow's build, push, cosign, and manifest steps with `IMAGE_REPOSITORY` and `IMAGE_REGISTRY`, keep the original Docker Hub repository as the default, and document the ACR configuration needed for `crpi-dgkl9khr1943eg60.cn-hangzhou.personal.cr.aliyuncs.com/hq-service/hth-newapi`.

## Suggested Review Order

1. [`.github/workflows/docker-build.yml`](../../.github/workflows/docker-build.yml) - Check image repository/env propagation across per-arch builds, manifest creation, and Kubernetes deploy fallback.
2. [`docs/contribution-guide.md`](../../docs/contribution-guide.md) - Check operator-facing variable and secret guidance for Docker Hub defaults and Alibaba Cloud ACR override.
