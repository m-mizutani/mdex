---
title: Architecture Overview
tags:
  - architecture
  - design
category: technical
---

# Architecture

mdex follows a clean architecture pattern with four layers.

## Package Structure

| Package | Responsibility |
|---------|---------------|
| `pkg/cli` | CLI command definitions |
| `pkg/domain` | Models, interfaces, frontmatter parser |
| `pkg/usecase` | Export orchestration logic |
| `pkg/infra` | Notion API client, file scanner |

## Export Flow

1. **Scan** local Markdown files and compute hashes
2. **Ensure** required database properties exist
3. **Query** existing pages from Notion
4. **Compare** hashes to determine create/update/delete/skip
5. **Execute** the plan (archive old pages, create new ones)

## Frontmatter Support

Each Markdown file can include YAML frontmatter:

```yaml
---
title: Custom Page Title
tags:
  - api
  - golang
category: backend
---
```

- `title` overrides the filename-based title
- `tags` maps to a Notion **multi_select** property
- `category` maps to a Notion **select** property
