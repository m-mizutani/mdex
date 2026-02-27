package usecase

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/domain/converter"
	"github.com/m-mizutani/mdex/pkg/utils/logging"
	"github.com/m-mizutani/mdex/pkg/utils/safe"
)

// ExportUseCase orchestrates the Markdown to Notion export process.
type ExportUseCase struct {
	notionClient domain.NotionClient
	fileScanner  domain.FileScanner
}

// NewExportUseCase creates a new ExportUseCase.
func NewExportUseCase(nc domain.NotionClient, fs domain.FileScanner) *ExportUseCase {
	return &ExportUseCase{
		notionClient: nc,
		fileScanner:  fs,
	}
}

// Execute runs the export process.
func (uc *ExportUseCase) Execute(ctx context.Context, config domain.ExportConfig) error {
	logger := logging.From(ctx)

	// 1. Scan local markdown files
	var (
		files []domain.MarkdownFile
		err   error
	)
	if len(config.Files) > 0 {
		logger.Info("reading markdown files", "files", config.Files)
		files, err = uc.fileScanner.ReadMarkdownFiles(config.Files)
		if err != nil {
			return goerr.Wrap(err, "reading markdown files", goerr.V("files", config.Files))
		}
	} else {
		logger.Info("scanning markdown files", "dir", config.Dir)
		files, err = uc.fileScanner.ScanMarkdownFiles(config.Dir)
		if err != nil {
			return goerr.Wrap(err, "scanning markdown files", goerr.V("dir", config.Dir))
		}
	}
	logger.Info("found markdown files", "count", len(files))

	// 2. Ensure required properties exist in the database
	logger.Info("ensuring database properties",
		"path_property", config.PathProperty,
		"hash_property", config.HashProperty,
		"tags_property", config.TagsProperty,
		"category_property", config.CategoryProperty,
		"domain_property", config.DomainProperty,
		"domain", config.Domain,
	)
	propSpecs := []domain.PropertySpec{
		{Name: config.PathProperty, Type: "rich_text"},
		{Name: config.HashProperty, Type: "rich_text"},
		{Name: config.TagsProperty, Type: "multi_select"},
		{Name: config.CategoryProperty, Type: "select"},
	}
	if config.Domain != "" {
		propSpecs = append(propSpecs, domain.PropertySpec{Name: config.DomainProperty, Type: "select"})
	}
	if err := uc.notionClient.EnsureDatabaseProperties(ctx, config.NotionDatabaseID, propSpecs); err != nil {
		return goerr.Wrap(err, "ensuring database properties")
	}

	// 3. Query existing Notion database records
	logger.Info("querying notion database", "database_id", config.NotionDatabaseID)
	pages, err := uc.notionClient.QueryDatabase(ctx, config.NotionDatabaseID, config.PathProperty, config.HashProperty, config.DomainProperty, config.Domain)
	if err != nil {
		return goerr.Wrap(err, "querying notion database")
	}
	logger.Info("found existing pages", "count", len(pages))

	// 4. Get the title property name
	titleProp, err := uc.notionClient.GetTitleProperty(ctx, config.NotionDatabaseID)
	if err != nil {
		return goerr.Wrap(err, "getting title property")
	}

	// 5. Compute export plan
	plan := ComputeExportPlan(files, pages, config.Force)
	logPlan(logger, plan)

	// 6. Execute the plan
	var errs []error

	// Archive pages for updated and deleted files
	for i := range plan {
		entry := &plan[i]
		if (entry.Action == domain.ActionUpdate || entry.Action == domain.ActionDelete) && entry.Page != nil {
			logger.Info("archiving page",
				"page_id", entry.Page.ID,
				"path", entry.Page.MdexPath,
				"action", actionString(entry.Action),
			)
			if err := uc.notionClient.ArchivePage(ctx, entry.Page.ID); err != nil {
				logger.Error("failed to archive page",
					"page_id", entry.Page.ID,
					logging.ErrAttr(err),
				)
				errs = append(errs, err)
			}
		}
	}

	// Create pages for new and updated files
	for i := range plan {
		entry := &plan[i]
		if entry.Action == domain.ActionCreate || entry.Action == domain.ActionUpdate {
			if entry.File == nil {
				continue
			}

			logger.Info("creating page",
				"path", entry.File.RelPath,
				"action", actionString(entry.Action),
			)

			if err := uc.createPage(ctx, config, titleProp, entry.File); err != nil {
				logger.Error("failed to create page",
					"path", entry.File.RelPath,
					logging.ErrAttr(err),
				)
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		messages := make([]string, len(errs))
		for i, e := range errs {
			messages[i] = e.Error()
		}
		return goerr.New("export completed with errors", goerr.V("count", len(errs)), goerr.V("errors", messages))
	}

	logger.Info("export completed successfully")
	return nil
}

func (uc *ExportUseCase) createPage(ctx context.Context, config domain.ExportConfig, titleProp string, file *domain.MarkdownFile) error {
	// Parse frontmatter and extract body
	meta, body, err := domain.ParseFrontmatter(file.Content)
	if err != nil {
		return goerr.Wrap(err, "parsing frontmatter", goerr.V("path", file.RelPath))
	}

	// Convert markdown body (without frontmatter) to Notion blocks
	blocks, err := converter.Convert(body, file.FilePath)
	if err != nil {
		return goerr.Wrap(err, "converting markdown", goerr.V("path", file.RelPath))
	}

	// Process local images: upload them and replace block references
	notionBlocks, err := uc.processBlocks(ctx, blocks)
	if err != nil {
		return goerr.Wrap(err, "processing blocks", goerr.V("path", file.RelPath))
	}

	// Build page properties
	title := meta.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(file.RelPath), ".md")
	}
	properties := map[string]interface{}{
		titleProp: map[string]interface{}{
			"title": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": title,
					},
				},
			},
		},
		config.PathProperty: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": file.RelPath,
					},
				},
			},
		},
		config.HashProperty: map[string]interface{}{
			"rich_text": []map[string]interface{}{
				{
					"type": "text",
					"text": map[string]interface{}{
						"content": file.Hash,
					},
				},
			},
		},
	}

	// Add tags (multi_select) from frontmatter
	if len(meta.Tags) > 0 {
		options := make([]map[string]interface{}, len(meta.Tags))
		for i, tag := range meta.Tags {
			options[i] = map[string]interface{}{"name": tag}
		}
		properties[config.TagsProperty] = map[string]interface{}{
			"multi_select": options,
		}
	}

	// Add category (select) from frontmatter
	if meta.Category != "" {
		properties[config.CategoryProperty] = map[string]interface{}{
			"select": map[string]interface{}{"name": meta.Category},
		}
	}

	// Add domain (select) if configured
	if config.Domain != "" {
		properties[config.DomainProperty] = map[string]interface{}{
			"select": map[string]interface{}{"name": config.Domain},
		}
	}

	_, err = uc.notionClient.CreatePage(ctx, config.NotionDatabaseID, properties, notionBlocks)
	if err != nil {
		return goerr.Wrap(err, "creating notion page", goerr.V("path", file.RelPath))
	}

	return nil
}

// processBlocks converts converter.Block to []interface{} and handles local image uploads.
func (uc *ExportUseCase) processBlocks(ctx context.Context, blocks []converter.Block) ([]interface{}, error) {
	result := make([]interface{}, 0, len(blocks))
	for _, block := range blocks {
		processed, err := uc.processBlock(ctx, block)
		if err != nil {
			return nil, err
		}
		result = append(result, processed)
	}
	return result, nil
}

func (uc *ExportUseCase) processBlock(ctx context.Context, block converter.Block) (interface{}, error) {
	// Check if this is a local image that needs uploading
	if blockType, ok := block["type"].(string); ok && blockType == "image" {
		if imgData, ok := block["image"].(map[string]interface{}); ok {
			if imgType, ok := imgData["type"].(string); ok && imgType == "file_upload" {
				if localPath, ok := imgData["local_path"].(string); ok {
					return uc.uploadAndReplaceImage(ctx, localPath)
				}
			}
		}
	}
	return block, nil
}

func (uc *ExportUseCase) uploadAndReplaceImage(ctx context.Context, localPath string) (interface{}, error) {
	filename := filepath.Base(localPath)
	contentType := detectContentType(filename)

	// Skip unsupported image formats — Notion File Upload API only supports png, jpeg, gif, webp
	if !isSupportedImageType(contentType) {
		logging.From(ctx).Warn("skipping unsupported image format",
			"path", localPath,
			"content_type", contentType,
		)
		return map[string]interface{}{
			"type": "paragraph",
			"paragraph": map[string]interface{}{
				"rich_text": []map[string]interface{}{
					{
						"type": "text",
						"text": map[string]interface{}{
							"content": fmt.Sprintf("[unsupported image: %s]", filename),
						},
					},
				},
			},
		}, nil
	}

	cleanPath := filepath.Clean(localPath)
	file, err := os.Open(cleanPath) // #nosec G304 -- path is derived from user-specified directory and markdown content
	if err != nil {
		return nil, goerr.Wrap(err, "opening local image", goerr.V("path", localPath))
	}
	defer safe.Close(ctx, file)

	fileUploadID, err := uc.notionClient.UploadFile(ctx, filename, contentType, file)
	if err != nil {
		return nil, goerr.Wrap(err, "uploading image", goerr.V("path", localPath))
	}

	return map[string]interface{}{
		"type": "image",
		"image": map[string]interface{}{
			"type": "file_upload",
			"file_upload": map[string]interface{}{
				"id": fileUploadID,
			},
		},
	}, nil
}

func isSupportedImageType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

func detectContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}

// ComputeExportPlan determines what actions to take for each file/page.
func ComputeExportPlan(files []domain.MarkdownFile, pages []domain.NotionPage, force bool) []domain.ExportPlanEntry {
	// Build a map of existing pages by path
	pageMap := make(map[string]*domain.NotionPage, len(pages))
	for i := range pages {
		pageMap[pages[i].MdexPath] = &pages[i]
	}

	var plan []domain.ExportPlanEntry

	// Check each local file
	processedPaths := make(map[string]bool)
	for i := range files {
		file := &files[i]
		processedPaths[file.RelPath] = true

		page, exists := pageMap[file.RelPath]
		if !exists {
			// New file
			plan = append(plan, domain.ExportPlanEntry{
				Action: domain.ActionCreate,
				File:   file,
			})
		} else if force || page.MdexHash != file.Hash {
			// Existing file, hash changed (or force mode)
			plan = append(plan, domain.ExportPlanEntry{
				Action: domain.ActionUpdate,
				File:   file,
				Page:   page,
			})
		} else {
			// Hash matches, skip
			plan = append(plan, domain.ExportPlanEntry{
				Action: domain.ActionSkip,
				File:   file,
				Page:   page,
			})
		}
	}

	// Check for deleted files (in DB but not locally)
	for i := range pages {
		if !processedPaths[pages[i].MdexPath] {
			plan = append(plan, domain.ExportPlanEntry{
				Action: domain.ActionDelete,
				Page:   &pages[i],
			})
		}
	}

	return plan
}

func actionString(a domain.ExportAction) string {
	switch a {
	case domain.ActionCreate:
		return "create"
	case domain.ActionUpdate:
		return "update"
	case domain.ActionDelete:
		return "delete"
	case domain.ActionSkip:
		return "skip"
	default:
		return "unknown"
	}
}

func logPlan(logger interface{ Info(string, ...any) }, plan []domain.ExportPlanEntry) {
	var creates, updates, deletes, skips int
	for _, entry := range plan {
		switch entry.Action {
		case domain.ActionCreate:
			creates++
		case domain.ActionUpdate:
			updates++
		case domain.ActionDelete:
			deletes++
		case domain.ActionSkip:
			skips++
		}
	}
	logger.Info("export plan computed",
		"create", creates,
		"update", updates,
		"delete", deletes,
		"skip", skips,
	)
}
