---
name: cherry-knowledge-search
description: Cherry Studio 知识库检索。用于用户要求检索、查询、引用 Cherry Studio 知识库内容，或提出“检索知识库”“查询知识库”“知识库搜索”“从知识库查资料”等请求时。
---

# Cherry Studio 知识库检索

使用此 skill 通过本地 Cherry Studio API 从知识库中检索信息。

## 工作流程

1. 读取当前 skill 目录下的 `config.json`。
2. 如果 `auth.api_key` 为空，先向用户询问 Cherry Studio API key，再执行查询。
3. 如果 `knowledge_base_ids` 为空，检索全部知识库；如果不为空，默认只检索这些 ID，除非用户明确要求使用其他知识库。
4. 使用 `scripts/cherry_knowledge.py search "<query>"` 执行检索。
5. 根据返回的片段回答问题。可用时包含来源元数据，尤其是 `knowledge_base_name`、`score`、`metadata.source` 以及文件名、页码等字段。

## 命令

列出已配置的知识库：

```powershell
python "scripts/cherry_knowledge.py" list
```

按 ID 获取单个知识库：

```powershell
python "scripts/cherry_knowledge.py" get "<knowledge-base-id>"
```

检索知识库：

```powershell
python "scripts/cherry_knowledge.py" search "<query>" --count 5
```

单次检索时覆盖配置中的知识库 ID：

```powershell
python "scripts/cherry_knowledge.py" search "<query>" --knowledge-base-id "<id-1>" --knowledge-base-id "<id-2>"
```

## API 端点

脚本封装了以下 Cherry Studio API 端点：

- `GET /v1/knowledge-bases`：列出知识库。查询参数：`limit`、`offset`。
- `GET /v1/knowledge-bases/{id}`：获取单个知识库。
- `POST /v1/knowledge-bases/search`：检索知识库。

检索请求体：

```json
{
  "query": "How do I configure Ollama embedding?",
  "knowledge_base_ids": ["kb-uuid-1", "kb-uuid-2"],
  "document_count": 5
}
```

配置 `auth.api_key` 后，认证使用 `Authorization: Bearer <api_key>`。
