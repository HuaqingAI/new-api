---
title: 'Release pipeline builds Docker image and updates Kubernetes deployment'
type: 'feature'
created: '2026-06-05T00:00:00+08:00'
status: 'done'
baseline_commit: '75fd752dda885c49b18f175967fc1b8a3f4b7d7e'
context:
  - '{project-root}/CLAUDE.md'
  - '{project-root}/docs/contribution-guide.md'
---

<frozen-after-approval reason="human-owned intent — do not modify unless human renegotiates">

## Intent

**Problem:** 当前仓库已有 tag 触发的二进制发布和 Docker Hub 多架构镜像构建，但缺少“GitHub Release 发布后自动构建镜像并更新 Kubernetes 镜像版本”的完整发布链路。发布完成后仍需要人工登录集群改镜像，容易延迟或改错版本。

**Approach:** 在现有 Docker 镜像发布 workflow 上增加 GitHub Release `published` 触发，并在多架构 manifest 推送成功后追加 Kubernetes 部署更新 job。部署 job 通过 GitHub Secrets/Variables 注入 kubeconfig 与 Helm 或裸 Deployment 参数，Helm 部署执行 `helm upgrade --reuse-values --set-string ... --wait`，裸 Deployment 部署执行 `kubectl set image` 与 `kubectl rollout status`，保证“我们的版本号 + 对应 new-api 官方版本号”的组合镜像版本被滚动到目标集群。

## Boundaries & Constraints

**Always:** 保留手动补跑能力；复用现有 Dockerfile、多架构矩阵、Docker Hub 登录、manifest 创建、cosign 签名路径；Release 事件下我们的版本号必须来自 `github.event.release.tag_name`；对应的 new-api 官方版本号必须来自 release tag 对应代码中的 `UPSTREAM_VERSION` 文件，示例 `v1.0.0-rc.10`；构建进应用的 `VERSION` 必须同时包含两个版本号；Docker/k8s 镜像标签必须包含两个版本号并保持 Docker tag 兼容；k8s 更新必须发生在组合版本 manifest 推送成功之后；k8s 凭据只能来自 GitHub Secrets，不能写入仓库；Helm release/chart、Helm image values key、namespace、image repository 必须可配置；裸 Deployment fallback 必须支持多个 Deployment；部署摘要必须展示我们的版本号、new-api 官方版本号、组合镜像 tag、image、deploy strategy、namespace、Helm release/chart 或 deployments 与 rollout 结果。

**Ask First:** 如果需要删除或替换现有 `release.yml`、改默认 Docker Hub 仓库名、改受保护项目/组织标识、引入 Helm/Argo CD/Flux 等新的部署体系，必须先询问用户。

**Never:** 不提交 kubeconfig、token、registry 密码或集群地址等敏感值；不把生产集群命名空间或 Deployment 名称硬编码为唯一值；不移除二进制 Release 发布；不改变应用运行时代码、数据库逻辑、前端 UI 或默认部署镜像名；不绕过 Docker Hub manifest 成功状态直接更新 k8s。

## I/O & Edge-Case Matrix

| Scenario | Input / State | Expected Output / Behavior | Error Handling |
|----------|--------------|---------------------------|----------------|
| Release published | GitHub Release `v2.0.0` published，tag 对应代码中的 `UPSTREAM_VERSION` 为 `v1.0.0-rc.10` | workflow checkout 对应 tag，写入 `VERSION=v2.0.0+new-api.v1.0.0-rc.10`，构建并推送 `calciumion/new-api:v2.0.0-newapi-v1.0.0-rc.10-{amd64,arm64}`、`v2.0.0-{amd64,arm64}`、`latest-{amd64,arm64}`，创建组合版本、我们的版本和 `latest` 多架构 manifest，随后通过 Helm 或 kubectl 更新 k8s 到组合版本镜像 | 任一构建、推送、manifest 或 rollout 失败时 job 失败，摘要保留失败前上下文 |
| Manual deploy/build | Maintainer 手动输入 tag，tag 对应代码包含 `UPSTREAM_VERSION` | workflow 校验 tag 存在，按双版本号构建镜像并更新 k8s 到组合版本 tag | tag 不存在、`UPSTREAM_VERSION` 缺失或为空时明确报错并停止，不更新 k8s |
| Missing k8s configuration | 缺少 kubeconfig、Helm release/chart 或 kubectl deployments/container 配置 | 镜像构建和 manifest 可完成；部署 job 在配置检查阶段失败并说明缺少的变量/Secret | 不执行 `helm upgrade` 或 `kubectl set image`，避免更新到未知目标 |

</frozen-after-approval>

## Code Map

- `.github/workflows/docker-build.yml` -- 现有多架构 Docker Hub 发布 workflow；需要由 GitHub Release/手动触发驱动，并新增 k8s rollout job。
- `Dockerfile` -- 现有容器构建入口；本次只复用，不修改。
- `UPSTREAM_VERSION` -- 记录当前合并的上游 new-api 官方版本；发布 workflow 从 release tag 中读取。
- `docs/contribution-guide.md` -- 发布流程说明；需要同步 Release 触发、双版本号来源和 k8s 配置要求。

## Tasks & Acceptance

**Execution:**
- [x] `.github/workflows/docker-build.yml` -- 增加 `release: types: [published]` 触发，并统一解析 tag 来源为手动输入或 Release 事件 -- 满足“发布 Release 自动触发”且保留人工补跑能力。
- [x] `.github/workflows/docker-build.yml` -- 在 manifest job 后新增 `deploy_kubernetes` job，依赖 `create_manifests`，通过 GitHub Secret `KUBE_CONFIG` 与 Variables/Secrets 配置 Helm 或 kubectl 部署参数，执行 `helm upgrade --wait` 或 `kubectl set image` + `rollout status` -- 实现镜像版本滚动更新。
- [x] `.github/workflows/docker-build.yml` -- 为部署配置检查、kubectl 初始化、rollout 摘要增加清晰日志与 `$GITHUB_STEP_SUMMARY` 输出 -- 让维护者能快速定位缺失配置或 rollout 失败原因。
- [x] `UPSTREAM_VERSION` -- 新增并写入当前上游官方版本 `v1.0.0-rc.10` -- 让官方版本跟随合并代码进入 git 历史。
- [x] `docs/contribution-guide.md` -- 更新发布说明，记录 `UPSTREAM_VERSION` 双版本号来源、手动触发输入和 k8s 配置项 -- 保持维护文档与 workflow 行为一致。

**Acceptance Criteria:**
- Given maintainer publishes GitHub Release `v2.0.0` and that tag contains `UPSTREAM_VERSION=v1.0.0-rc.10`, when the workflow runs, then Docker image `calciumion/new-api:v2.0.0-newapi-v1.0.0-rc.10` is built as a multi-arch manifest before Kubernetes deployment starts.
- Given Helm secrets and variables are configured, when manifest creation succeeds, then the configured Helm release is upgraded with the combined dual-version image tag and Helm waits for rollout.
- Given kubectl fallback variables are configured with multiple deployments, when manifest creation succeeds, then each configured Deployment container image is updated to the combined dual-version image tag and rollout status is awaited.
- Given required k8s configuration is missing, when deployment job starts, then the job fails before invoking `helm upgrade` or `kubectl set image` and reports the missing configuration names.
- Given a maintainer manually dispatches the workflow with a valid tag whose code contains `UPSTREAM_VERSION`, when it runs, then it builds and deploys the same dual-version image tag as the release-triggered path.
- Given a tag is pushed before a GitHub Release is published, when only the tag push event occurs, then this Docker deploy workflow does not run and waits for the Release event or manual dispatch.

## Design Notes

Use repository-level configuration so the workflow is reusable across environments:

```text
Secret: KUBE_CONFIG
Variable/Secret: KUBE_DEPLOY_STRATEGY (helm or kubectl; auto-detects helm when Helm settings exist)
Variable/Secret: KUBE_NAMESPACE
Optional Variable/Secret: KUBE_IMAGE_REPOSITORY (default calciumion/new-api)
Variable/Secret: HELM_RELEASE and HELM_CHART for Helm deployments
Optional Variable/Secret: HELM_IMAGE_REPOSITORY_KEYS and HELM_IMAGE_TAG_KEYS
Variable/Secret: KUBE_DEPLOYMENTS and KUBE_CONTAINER for kubectl fallback
```

Prefer Helm for Helm-managed master/slave deployments so chart values remain the source of truth. The kubectl path is retained for simple non-Helm Deployment-based deployments.

## Verification

**Commands:**
- `ruby -e "require 'yaml'; Dir['.github/workflows/*.yml'].each { |f| YAML.load_file(f); puts f }"` -- expected: all workflow YAML files parse successfully.
- `git diff --check` -- expected: no trailing whitespace or whitespace errors.

## Suggested Review Order

**Workflow entry and version contract**

- Release/manual entry point defines when image publishing and deploy run.
  [docker-build.yml:3](../../.github/workflows/docker-build.yml#L3)

- Double-version resolver reads `UPSTREAM_VERSION` and derives image tag.
  [docker-build.yml:38](../../.github/workflows/docker-build.yml#L38)

- Build writes combined app version into Docker build context.
  [docker-build.yml:136](../../.github/workflows/docker-build.yml#L136)

**Image publishing and rollout**

- Multi-arch build publishes combined, release, and latest arch tags.
  [docker-build.yml:167](../../.github/workflows/docker-build.yml#L167)

- Manifest job promotes combined and release tags after arch builds.
  [docker-build.yml:200](../../.github/workflows/docker-build.yml#L200)

- Kubernetes job validates config before touching the cluster.
  [docker-build.yml:253](../../.github/workflows/docker-build.yml#L253)

- Helm rollout updates all chart-managed workloads to the combined image tag.
  [docker-build.yml:324](../../.github/workflows/docker-build.yml#L324)

**Operator docs**

- Contribution guide documents `UPSTREAM_VERSION` and required k8s settings.
  [contribution-guide.md:23](../../docs/contribution-guide.md#L23)
