package domain

// MarkdownFile represents a Markdown file collected from the local directory.
type MarkdownFile struct {
	// RelPath is the normalized relative path (using '/' separator) from the base directory.
	RelPath string
	// Content is the raw content of the Markdown file.
	Content []byte
	// Hash is the SHA-256 hex digest of the file content.
	Hash string
}

// NotionPage represents a page record in the Notion database.
type NotionPage struct {
	// ID is the Notion page ID.
	ID string
	// MdexPath is the value of the mdex_path property.
	MdexPath string
	// MdexHash is the value of the mdex_hash property.
	MdexHash string
}

// ExportAction represents the action to take for a file.
type ExportAction int

const (
	ActionCreate ExportAction = iota // New file, create page
	ActionUpdate                     // Existing file changed, archive + recreate
	ActionDelete                     // File removed locally, archive page
	ActionSkip                       // Hash matches, no change needed
)

// ExportPlanEntry represents a single planned action.
type ExportPlanEntry struct {
	Action ExportAction
	File   *MarkdownFile // nil for delete actions
	Page   *NotionPage   // nil for create actions
}

// Metadata holds frontmatter values extracted from a markdown file.
type Metadata struct {
	Title    string   `yaml:"title"`
	Tags     []string `yaml:"tags"`
	Category string   `yaml:"category"`
}

// PropertySpec describes a database property to ensure exists.
type PropertySpec struct {
	Name string
	Type string // "rich_text", "multi_select", "select"
}

// ExportConfig holds the configuration for an export operation.
type ExportConfig struct {
	NotionDatabaseID string
	NotionToken      string
	Dir              string
	PathProperty     string
	HashProperty     string
	TagsProperty     string
	CategoryProperty string
	Domain           string
	DomainProperty   string
	Force            bool
}
