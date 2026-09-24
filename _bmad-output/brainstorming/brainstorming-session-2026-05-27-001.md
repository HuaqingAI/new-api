---
stepsCompleted: [1, 2, 3, 4]
inputDocuments: []
session_topic: 'new-api AI API 网关企业功能扩展'
session_goals: '基于 BMAD 流程，系统性探索和规划企业级功能方向，识别适合企业客户的功能——涵盖管控、安全、合规、运营效率等维度'
selected_approach: 'ai-recommended'
techniques_used: ['基于现有代码的方向引导']
ideas_generated: 12
session_status: 'completed'
next_step: 'bmad-product-brief'
context_file: ''
---

# Brainstorming Session Results

**Facilitator:** hth
**Date:** 2026-05-27

## Session Overview

**Topic:** new-api AI API 网关企业功能扩展
**Goals:** 基于 BMAD 流程，系统性探索和规划方向。棕地项目，已有扎实基础能力（用户管理、计费、限速、40+ AI 供应商接入）。

## 方向决策（用户最终确认）

### 🏢 企业管控主题（确定要做）

| 编号 | 方向 | 说明 |
|---|---|---|
| B | 组织/部门层级 | 真正的组织树，支持配额组织级分配和用量聚合 |
| C | 钉钉登录（一期） | 替代通用 LDAP，后续扩展飞书/企微 |
| E | 钉钉通讯录同步 | 自动同步部门/成员，作为 B 的数据源 |
| F | 组织用量看板 | 按部门维度的用量聚合、CSV 导出、定期邮件报告 |
| G | 内容监控告警 | 现有过滤基础上，补充监控（违规率、告警通知） |

**关键洞察：** B + C + E 可合并为「钉钉企业集成」连贯 Epic，钉钉组织树直接作为 B 的数据来源。

### 🧠 平台能力扩展主题（新增方向）

| 方向 | 说明 |
|---|---|
| 企业知识库 + DeepWiki | 新增能力域 |
| Playground WebChat 增强 | 接入知识库 + 多模态（图片生成等） |
| 图片/视频生成 | 接入开源项目，复用用户 API Key |

### 💼 商业模式主题（长远）

| 方向 | 说明 |
|---|---|
| SaaS 多租户 | 针对小型团队不想自部署，提供租户化平台 |
| 上游服务商模式 | 提供 Token + 运维 + 培训等增值服务 |

### 🗂 方向池（暂不做，未来评估）

- **D API Key 管控策略**：现有功能已较完整，扩展（强制过期、轮换提醒、代管）优先级低
- **H 私有化授权 License 机制**：与 SaaS 方向有冲突，待整体定位明确后再评估
- **A 专用审计日志**：现有 `Log` 表 `Manage` 类型已覆盖基本需求

## 现状能力评估摘要

| 功能方向 | 评估 |
|---|---|
| 用户管理 / 角色 | 已有（3 级角色 + 分组标签） |
| 组织 / 多团队 | **缺失**（无 tenant_id） |
| 额度与计费 | 已有（完整，含订阅套餐、4 种支付） |
| API Key 管理 | 已有（完整，含模型/IP 白名单） |
| 请求日志 | 已有（较完整） |
| 渠道/供应商管理 | 已有（完整，含多 Key、亲和性） |
| 限速/限流 | 已有（多层、分组可配置） |
| OIDC/OAuth | 已有（含自定义提供商 + 访问策略） |
| WebAuthn / 2FA | 已有 |
| SAML / LDAP / SCIM | 缺失 |
| 多租户隔离 | **缺失** |

## 战略性观察

1. **范围已发生质变**：从"做几个企业功能"扩展到"企业版 + 平台 + SaaS"，需要先定整体战略
2. **部署模型冲突**：钉钉集成（企业自部署）vs SaaS 多租户（公共注册）是两种模式，须在简报中决策"自部署 / SaaS / 双轨"
3. **棕地项目大扩展规则**：必须先做产品简报，对齐边界后再 PRD

## Session 结论

**下一步流程**：`bmad-product-brief` → 产品简报 → PRD → 架构 → Epics/Stories

**简报需重点决策**：
- 项目演化定位：自部署 / SaaS / 双轨
- 各主题的阶段排序与依赖
- 商业模式（开源 + 商业版？SaaS 收费？）

---

_Session Status: Completed and ready to proceed to product brief._
