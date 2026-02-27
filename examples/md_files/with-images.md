---
title: Image Examples
tags:
  - example
  - images
category: documentation
---

# Image Examples

This page demonstrates how mdex handles images.

## Local Images

Local images are automatically uploaded to Notion via the File Upload API.

### Logo

![mdex logo](images/logo.png)

### Export Flow

The diagram below shows the mdex export pipeline:

![Export flow diagram](images/diagram.png)

## External Images

External URLs are referenced directly without uploading.

![gollem logo](https://raw.githubusercontent.com/m-mizutani/gollem/main/doc/images/logo.png)

## Mixed Content

You can mix images with other Markdown elements:

1. First, scan the files
2. Then compare hashes — here's the flow again:
   ![flow](images/diagram.png)
3. Finally, export to Notion
