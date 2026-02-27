package domain

import (
	"context"
	"io"
)

//go:generate moq -out mock/notion_client.go -pkg mock . NotionClient
//go:generate moq -out mock/file_scanner.go -pkg mock . FileScanner

// NotionClient defines the interface for interacting with the Notion API.
type NotionClient interface {
	// EnsureDatabaseProperties ensures that the specified properties exist in the database schema.
	EnsureDatabaseProperties(ctx context.Context, databaseID string, properties []PropertySpec) error

	// QueryDatabase retrieves all records from the specified Notion database.
	// If domainValue is non-empty, only pages matching the domain filter are returned.
	QueryDatabase(ctx context.Context, databaseID string, pathProperty string, hashProperty string, domainProperty string, domainValue string) ([]NotionPage, error)

	// GetTitleProperty retrieves the name of the title property from the database schema.
	GetTitleProperty(ctx context.Context, databaseID string) (string, error)

	// CreatePage creates a new page in the specified database with the given properties and blocks.
	// If blocks exceed 100, they are split into multiple requests automatically.
	CreatePage(ctx context.Context, databaseID string, properties map[string]interface{}, blocks []interface{}) (string, error)

	// AppendBlocks appends blocks to an existing page (used when blocks exceed 100).
	AppendBlocks(ctx context.Context, pageID string, blocks []interface{}) error

	// ArchivePage archives (soft-deletes) a Notion page.
	ArchivePage(ctx context.Context, pageID string) error

	// UploadFile uploads a file to Notion via the File Upload API.
	// It performs the two-step process: create file upload, then send file content.
	UploadFile(ctx context.Context, filename string, contentType string, body io.Reader) (string, error)
}

// FileScanner defines the interface for scanning Markdown files from a directory.
type FileScanner interface {
	// ScanMarkdownFiles recursively scans the given directory and returns all Markdown files.
	ScanMarkdownFiles(baseDir string) ([]MarkdownFile, error)

	// ReadMarkdownFiles reads the specified Markdown files and returns them.
	ReadMarkdownFiles(paths []string) ([]MarkdownFile, error)
}
