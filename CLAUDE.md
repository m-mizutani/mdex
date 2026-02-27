# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

mdex is a Markdown Exporter to Notion. It converts Markdown files and exports them to Notion pages.

## Common Development Commands

### Building and Testing
- `go vet ./...` - Check for compile errors (do NOT use `go build` for verification)
- `go test ./...` - Run all tests
- `go test ./path/to/package` - Run tests for a specific package
- `go fmt ./...` - Format code
- `golangci-lint run ./...` - Lint check
- `gosec -exclude-generated -quiet ./...` - Security check

### Before Finishing a Task
Always run these checks:
1. `go vet ./...`
2. `go fmt ./...`
3. `golangci-lint run ./...`
4. `gosec -exclude-generated -quiet ./...`
5. `go test ./...`

## Architecture

### Package Structure
- `main.go` - Entry point, calls `pkg/cli`
- `pkg/cli/` - CLI command definitions using [urfave/cli v3](https://github.com/urfave/cli)
- `pkg/domain/` - Domain models and interfaces
- `pkg/usecase/` - Application use cases / business logic
- `pkg/infra/` - Infrastructure adapters (external APIs, storage)
- `pkg/utils/` - Shared utility functions

### CLI Framework
- Uses `github.com/urfave/cli/v3`
- Root command is created in `pkg/cli/cli.go` via `cli.New()`

## Development Guidelines

### Error Handling
- Use `github.com/m-mizutani/goerr/v2` for error handling
- Wrap errors with `goerr.Wrap` to maintain error context
- Add variables with `goerr.V` for debugging
- **NEVER** check error messages with `strings.Contains(err.Error(), ...)`
- **ALWAYS** use `errors.Is` or `errors.As` for error type checking

### Testing
- Use `github.com/m-mizutani/gt` for type-safe testing
- Prefer Helper Driven Testing over Table Driven Tests
- Test package must be `package {name}_test` (external test package)
- Test file naming: `xyz.go` → `xyz_test.go` (no other naming patterns)
- Use `export_test.go` for items that need to be exposed only for testing
- **NEVER** use `t.Skip()` except for missing environment variables
- **NEVER** comment out test assertions

### Code Visibility
- Do not expose unnecessary methods, structs, or variables
- Use `export_test.go` for test-only exports

### Language
- All comments and string literals in source code must be in English
