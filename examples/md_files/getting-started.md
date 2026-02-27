---
title: Getting Started with mdex
tags:
  - guide
  - setup
category: documentation
---

# Getting Started

mdex exports your Markdown files to a Notion database.

## Installation

```bash
go install github.com/m-mizutani/mdex@latest
```

## Quick Start

1. Create a Notion integration and get the token
2. Share a database with the integration
3. Run the export command:

```bash
mdex export \
  --notion-token $NOTION_TOKEN \
  --notion-database-id $DATABASE_ID \
  --dir ./docs
```

## How It Works

mdex scans the specified directory for `.md` files, computes a SHA-256 hash of each file, and compares it with the hash stored in the Notion database. Only changed files are re-exported.

> **Note:** The first run will export all files since no hashes exist in the database yet.
