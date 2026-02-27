# Using mdex with GitHub Actions

Export your Markdown documentation to Notion automatically on every push using the mdex Docker image.

## Prerequisites

- A Notion integration token (see [README](../README.md#notion-integration-token))
- A Notion database ID (see [README](../README.md#notion-database-id))
- Both values stored as GitHub Actions secrets (`MDEX_NOTION_TOKEN` and `MDEX_NOTION_DATABASE_ID`)

## Basic Example

Export all Markdown files in `docs/` to Notion whenever the `main` branch is updated:

```yaml
name: Export docs to Notion

on:
  push:
    branches: [main]
    paths:
      - "docs/**/*.md"

jobs:
  export:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Export to Notion
        run: |
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:latest \
            export --dir /work/docs
```

## With Domain Scoping

Use `--domain` to share a single Notion database across multiple repositories:

```yaml
      - name: Export to Notion
        run: |
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:latest \
            export --dir /work/docs --domain "${{ github.repository }}"
```

This scopes pages by repository name (e.g., `m-mizutani/mdex`), so multiple repositories can write to the same database without interfering with each other.

## Dry-Run on Pull Requests

Preview what would be exported without making any Notion API calls:

```yaml
name: Preview Notion export

on:
  pull_request:
    paths:
      - "docs/**/*.md"

jobs:
  preview:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Dry-run export
        run: |
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:latest \
            export --dir /work/docs --dry-run
```

## Pinning a Specific Version

It is recommended to pin a specific version tag instead of `latest`:

```yaml
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:0.1.0 \
            export --dir /work/docs
```

## Force Re-export

To re-export all files regardless of whether they have changed:

```yaml
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:latest \
            export --dir /work/docs --force
```

## Complete Workflow Example

A full workflow that exports on push to `main` with domain scoping and version pinning:

```yaml
name: Export docs to Notion

on:
  push:
    branches: [main]
    paths:
      - "docs/**/*.md"

permissions:
  contents: read

jobs:
  export:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Export to Notion
        run: |
          docker run --rm \
            -v "${{ github.workspace }}:/work" \
            -e MDEX_NOTION_TOKEN="${{ secrets.MDEX_NOTION_TOKEN }}" \
            -e MDEX_NOTION_DATABASE_ID="${{ secrets.MDEX_NOTION_DATABASE_ID }}" \
            ghcr.io/m-mizutani/mdex:0.1.0 \
            export --dir /work/docs --domain "${{ github.repository }}"
```
