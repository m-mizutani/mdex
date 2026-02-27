package cli

import (
	"github.com/urfave/cli/v3"
)

// New creates the root CLI command for mdex.
func New() *cli.Command {
	return &cli.Command{
		Name:  "mdex",
		Usage: "Markdown Exporter to Notion",
		Commands: []*cli.Command{
			newExportCommand(),
		},
	}
}
