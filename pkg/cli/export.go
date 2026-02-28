package cli

import (
	"context"
	"log/slog"
	"os"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/infra/fs"
	"github.com/m-mizutani/mdex/pkg/infra/notion"
	"github.com/m-mizutani/mdex/pkg/usecase"
	"github.com/m-mizutani/mdex/pkg/utils/dryrun"
	"github.com/m-mizutani/mdex/pkg/utils/logging"
	"github.com/urfave/cli/v3"
)

func newExportCommand() *cli.Command {
	var (
		notionDatabaseID string
		notionToken      string
		dir              string
		files            []string
		pathProperty     string
		hashProperty     string
		tagsProperty     string
		categoryProperty string
		domainValue      string
		domainProperty   string
		force            bool
		dryRun           bool
		imageBaseDir     string
	)

	return &cli.Command{
		Name:    "export",
		Aliases: []string{"e"},
		Usage:   "Export Markdown files to a Notion database",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "notion-database-id",
				Aliases:     []string{"d"},
				Usage:       "Notion Database ID to export to",
				Sources:     cli.EnvVars("MDEX_NOTION_DATABASE_ID"),
				Destination: &notionDatabaseID,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "notion-token",
				Aliases:     []string{"t"},
				Usage:       "Notion Integration Token",
				Sources:     cli.EnvVars("MDEX_NOTION_TOKEN"),
				Destination: &notionToken,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "dir",
				Usage:       "Directory containing Markdown files",
				Sources:     cli.EnvVars("MDEX_DIR"),
				Destination: &dir,
			},
			&cli.StringSliceFlag{
				Name:        "file",
				Aliases:     []string{"f"},
				Usage:       "Markdown file(s) to export (can be specified multiple times)",
				Sources:     cli.EnvVars("MDEX_FILES"),
				Destination: &files,
			},
			&cli.StringFlag{
				Name:        "path-property",
				Usage:       "Notion property name for file path",
				Sources:     cli.EnvVars("MDEX_PATH_PROPERTY"),
				Value:       "mdex_path",
				Destination: &pathProperty,
			},
			&cli.StringFlag{
				Name:        "hash-property",
				Usage:       "Notion property name for content hash",
				Sources:     cli.EnvVars("MDEX_HASH_PROPERTY"),
				Value:       "mdex_hash",
				Destination: &hashProperty,
			},
			&cli.StringFlag{
				Name:        "tags-property",
				Usage:       "Notion property name for tags (multi_select)",
				Sources:     cli.EnvVars("MDEX_TAGS_PROPERTY"),
				Value:       "Tags",
				Destination: &tagsProperty,
			},
			&cli.StringFlag{
				Name:        "category-property",
				Usage:       "Notion property name for category (select)",
				Sources:     cli.EnvVars("MDEX_CATEGORY_PROPERTY"),
				Value:       "Category",
				Destination: &categoryProperty,
			},
			&cli.StringFlag{
				Name:        "domain",
				Usage:       "Domain value for scoping pages within the database",
				Sources:     cli.EnvVars("MDEX_DOMAIN"),
				Destination: &domainValue,
			},
			&cli.StringFlag{
				Name:        "domain-property",
				Usage:       "Notion property name for domain (select)",
				Sources:     cli.EnvVars("MDEX_DOMAIN_PROPERTY"),
				Value:       "Domain",
				Destination: &domainProperty,
			},
			&cli.BoolFlag{
				Name:        "force",
				Usage:       "Skip hash comparison and re-export all files",
				Sources:     cli.EnvVars("MDEX_FORCE"),
				Destination: &force,
			},
			&cli.BoolFlag{
				Name:        "dry-run",
				Usage:       "Show export plan without making API calls",
				Sources:     cli.EnvVars("MDEX_DRY_RUN"),
				Destination: &dryRun,
			},
			&cli.StringFlag{
				Name:        "image-base-dir",
				Usage:       "Base directory for resolving absolute image paths (e.g., Hugo's static/ directory)",
				Sources:     cli.EnvVars("MDEX_IMAGE_BASE_DIR"),
				Destination: &imageBaseDir,
			},
		},
		Action: func(ctx context.Context, _ *cli.Command) error {
			if dir == "" && len(files) == 0 {
				return goerr.New("either --dir or --file must be specified")
			}
			if dir != "" && len(files) > 0 {
				return goerr.New("--dir and --file cannot be used together")
			}

			// Setup logger
			logger := logging.New(os.Stderr, slog.LevelInfo)
			ctx = logging.With(ctx, logger)

			// Setup dry-run
			if dryRun {
				ctx = dryrun.WithDryRun(ctx)
				logging.From(ctx).Info("dry-run mode enabled")
			}

			config := domain.ExportConfig{
				NotionDatabaseID: notionDatabaseID,
				NotionToken:      notionToken,
				Dir:              dir,
				Files:            files,
				PathProperty:     pathProperty,
				HashProperty:     hashProperty,
				TagsProperty:     tagsProperty,
				CategoryProperty: categoryProperty,
				Domain:           domainValue,
				DomainProperty:   domainProperty,
				Force:            force,
				ImageBaseDir:     imageBaseDir,
			}

			notionClient := notion.New(notionToken)
			fileScanner := fs.New()
			uc := usecase.NewExportUseCase(notionClient, fileScanner)

			return uc.Execute(ctx, config)
		},
	}
}
