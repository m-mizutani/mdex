# mdex

Markdown Exporter to Notion. Converts Markdown files into Notion database pages with hash-based change detection, frontmatter support, and local image uploads.

## Features

- **Hash-based diff export** — Only changed files are re-exported (SHA-256 comparison). Use `--force` to re-export all.
- **YAML frontmatter** — Set page title, tags (multi_select), and category (select) from Markdown frontmatter.
- **Local image upload** — PNG, JPEG, GIF, and WebP images are automatically uploaded via the Notion File Upload API. External URLs are referenced directly.
- **Domain scoping** — Share one Notion database across multiple projects using `--domain`.
- **Auto schema setup** — Required database properties are created automatically on first run.
- **Dry-run mode** — Preview the export plan without making any API calls.

## Installation

```bash
go install github.com/m-mizutani/mdex@latest
```

## Setup

### Notion Integration Token

1. Go to [My Integrations](https://www.notion.so/profile/integrations) and create a new integration
2. Copy the **Internal Integration Secret** (starts with `ntn_`)
3. Open your target Notion database, click `...` > **Connections** > add your integration

### Notion Database ID

The database ID is part of the Notion database URL. Open the database as a **full page** in the browser:

```
https://www.notion.so/{workspace}/{database_id}?v={view_id}
                                  ^^^^^^^^^^^^
```

For example, if the URL is `https://www.notion.so/myworkspace/a8b1c2d3e4f5678901234567890abcde?v=...`, the database ID is `a8b1c2d3e4f5678901234567890abcde`.

You can also find it by clicking **Copy link to view** in the database menu and extracting the ID from the URL.

## Quick Start

```bash
export MDEX_NOTION_TOKEN="ntn_..."
export MDEX_NOTION_DATABASE_ID="a8b1c2d3e4f5678901234567890abcde"

mdex export --dir ./docs
```

## Usage

```
mdex export [flags]
```

### Required Flags

| Flag | Alias | Env Var | Description |
|------|-------|---------|-------------|
| `--notion-database-id` | `-d` | `MDEX_NOTION_DATABASE_ID` | Notion Database ID |
| `--notion-token` | `-t` | `MDEX_NOTION_TOKEN` | Notion Integration Token |
| `--dir` | | `MDEX_DIR` | Directory containing Markdown files |

### Optional Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--path-property` | `MDEX_PATH_PROPERTY` | `mdex_path` | Property name for file path (rich_text) |
| `--hash-property` | `MDEX_HASH_PROPERTY` | `mdex_hash` | Property name for content hash (rich_text) |
| `--tags-property` | `MDEX_TAGS_PROPERTY` | `Tags` | Property name for tags (multi_select) |
| `--category-property` | `MDEX_CATEGORY_PROPERTY` | `Category` | Property name for category (select) |
| `--domain` | `MDEX_DOMAIN` | | Domain value for scoping pages |
| `--domain-property` | `MDEX_DOMAIN_PROPERTY` | `Domain` | Property name for domain (select) |
| `--force` | `MDEX_FORCE` | `false` | Re-export all files regardless of hash |
| `--dry-run` | `MDEX_DRY_RUN` | `false` | Preview export plan without API calls |

## Frontmatter

Use YAML frontmatter to set Notion page properties:

```markdown
---
title: Getting Started
tags:
  - guide
  - setup
category: documentation
---

# Your content here
```

- **title** — Page title. Falls back to the filename (without `.md`) if not specified.
- **tags** — Mapped to a multi_select property.
- **category** — Mapped to a select property.

If tags or category are omitted, those properties are not set on the page.

## Domain Scoping

The `--domain` flag allows multiple projects to share a single Notion database without interfering with each other:

```bash
# Project A
mdex export --dir ./project-a/docs --domain project-a

# Project B
mdex export --dir ./project-b/docs --domain project-b
```

Each project only sees and manages its own pages. Delete detection is scoped to the domain — removing a file in project A won't affect project B's pages.

If `--domain` is not specified, no domain property is created and all pages in the database are considered.

## Supported Markdown Elements

### Block-level

- Headings (H1–H3; H4+ mapped to H3)
- Paragraphs
- Fenced code blocks (with language syntax highlighting)
- Blockquotes
- Bulleted lists (with nesting)
- Numbered lists (with nesting)
- Task lists (`- [ ]` / `- [x]`)
- Tables (GFM)
- Horizontal rules
- Images (local and external)

### Inline

- **Bold**, *italic*, ~~strikethrough~~
- `inline code`
- [Links](https://example.com)

## Image Handling

| Type | Behavior |
|------|----------|
| External URL (`https://...`) | Referenced directly as an external image block |
| Local file (PNG, JPEG, GIF, WebP) | Uploaded via Notion File Upload API |
| Unsupported format (e.g., SVG) | Replaced with placeholder text |

Local image paths are resolved relative to the Markdown file's location.

## How It Works

1. **Scan** — Recursively find `.md` files in `--dir`, compute SHA-256 hashes
2. **Schema** — Ensure required properties exist in the Notion database
3. **Query** — Fetch existing pages (filtered by domain if set)
4. **Plan** — Compare local files vs. Notion pages:
   - **Create** — New file, no matching page
   - **Update** — Hash changed, archive old page and create new
   - **Delete** — Page exists but file was removed, archive page
   - **Skip** — Hash matches, no action needed
5. **Execute** — Archive outdated pages, create new pages with blocks and images

## License

See [LICENSE](LICENSE) for details.
