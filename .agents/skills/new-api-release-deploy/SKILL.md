---
name: new-api-release-deploy
description: Release and deploy the HuaqingAI/new-api project from the dev branch using the repository Docker build workflow. Use when the user asks to deploy, redeploy, publish a release, create a release version like v0.1.1-20260618125127+new-api.v1.0.0-rc.11, rerun the docker-build workflow, or update production through Helm/GitHub Actions for this repository.
---

# new-api Release Deploy

## Overview

Create a timestamped release from `dev`, derive the upstream suffix from `UPSTREAM_VERSION`, and trigger the `docker-build.yml` workflow. Use the bundled script for deterministic release naming, GitHub CLI calls, and workflow monitoring.

## Workflow

1. Confirm the requested base version.
   - If the user gives a full release name such as `v0.1.1-20260618125127+new-api.v1.0.0-rc.11`, extract only the base version before the first timestamp: `v0.1.1`.
   - If the user does not give a base version, ask one short question for it. In Plan mode, prefer `askUserQuestion` / `request_user_input`; otherwise ask directly.
   - Do not ask for the timestamp or upstream version. Generate the timestamp at execution time and read the upstream version from `UPSTREAM_VERSION`.

2. Commit and push first when the user asks for code submission or when local changes are part of the deployment.
   - Review `git status --short --branch`.
   - Never discard user changes.
   - Run focused validation for changed workflow files when present.
   - Commit to `dev` and push `origin dev` before creating the release.

3. Run the release script from the repository root.

```bash
python3 .agents/skills/new-api-release-deploy/scripts/release_deploy.py --base-version v0.1.1
```

Use `--dry-run` before a live deployment when checking the generated names:

```bash
python3 .agents/skills/new-api-release-deploy/scripts/release_deploy.py --base-version v0.1.1 --dry-run
```

4. Watch the workflow result.
   - The script creates tag `v0.1.1-YYYYMMDDHHMMSS`.
   - The release title/body version is `v0.1.1-YYYYMMDDHHMMSS+new-api.<UPSTREAM_VERSION>`.
   - The release event triggers `.github/workflows/docker-build.yml`.
   - Report the release URL, workflow URL, run conclusion, and any failed job diagnostics.

## Deployment Notes

- This project uses Helm deployment for production. Do not ask for or manually configure database passwords for normal Helm upgrades; the chart should reuse existing Kubernetes Secrets.
- The known required GitHub Actions repository variable for the split image configuration is `HELM_EXTRA_SET=image.repository=hq-service/hth-newapi`.
- `IMAGE_REPOSITORY` remains the full pushed image repository, currently `crpi-dgkl9khr1943eg60.cn-hangzhou.personal.cr.aliyuncs.com/hq-service/hth-newapi`.
- If deployment fails, inspect `gh run view <run-id> --log-failed` and the deployment job diagnostics. Distinguish build/push failures, kubeconfig auth failures, Helm template/dry-run failures, and Kubernetes rollout failures.

## Script

`scripts/release_deploy.py` performs the release workflow. It requires:

- GitHub CLI authenticated for `HuaqingAI/new-api`.
- Current branch `dev` unless `--allow-non-dev` is explicitly passed for testing.
- Clean worktree unless `--allow-dirty` is explicitly passed.
- `UPSTREAM_VERSION` present at the repository root.
