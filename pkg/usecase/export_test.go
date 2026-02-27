package usecase_test

import (
	"context"
	"io"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/mdex/pkg/domain"
	"github.com/m-mizutani/mdex/pkg/domain/mock"
	"github.com/m-mizutani/mdex/pkg/usecase"
)

// newMockNotionClient creates a NotionClientMock with sensible defaults.
func newMockNotionClient() *mock.NotionClientMock {
	return &mock.NotionClientMock{
		EnsureDatabasePropertiesFunc: func(_ context.Context, _ string, _ []domain.PropertySpec) error {
			return nil
		},
		QueryDatabaseFunc: func(_ context.Context, _ string, _ string, _ string, _ string, _ string) ([]domain.NotionPage, error) {
			return nil, nil
		},
		GetTitlePropertyFunc: func(_ context.Context, _ string) (string, error) {
			return "Name", nil
		},
		CreatePageFunc: func(_ context.Context, _ string, _ map[string]interface{}, _ []interface{}) (string, error) {
			return "new-page-id", nil
		},
		AppendBlocksFunc: func(_ context.Context, _ string, _ []interface{}) error {
			return nil
		},
		ArchivePageFunc: func(_ context.Context, _ string) error {
			return nil
		},
		UploadFileFunc: func(_ context.Context, _ string, _ string, _ io.Reader) (string, error) {
			return "mock-file-upload-id", nil
		},
	}
}

// newMockFileScanner creates a FileScannerMock that returns the given files.
func newMockFileScanner(files []domain.MarkdownFile) *mock.FileScannerMock {
	return &mock.FileScannerMock{
		ScanMarkdownFilesFunc: func(_ string) ([]domain.MarkdownFile, error) {
			return files, nil
		},
	}
}

func TestComputeExportPlanNewFiles(t *testing.T) {
	files := []domain.MarkdownFile{
		{RelPath: "doc1.md", Hash: "hash1"},
		{RelPath: "doc2.md", Hash: "hash2"},
	}
	var pages []domain.NotionPage

	plan := usecase.ComputeExportPlan(files, pages, false)
	gt.A(t, plan).Length(2)
	gt.V(t, plan[0].Action).Equal(domain.ActionCreate)
	gt.V(t, plan[1].Action).Equal(domain.ActionCreate)
}

func TestComputeExportPlanSkipUnchanged(t *testing.T) {
	files := []domain.MarkdownFile{
		{RelPath: "doc1.md", Hash: "hash1"},
	}
	pages := []domain.NotionPage{
		{ID: "page1", MdexPath: "doc1.md", MdexHash: "hash1"},
	}

	plan := usecase.ComputeExportPlan(files, pages, false)
	gt.A(t, plan).Length(1)
	gt.V(t, plan[0].Action).Equal(domain.ActionSkip)
}

func TestComputeExportPlanUpdateChanged(t *testing.T) {
	files := []domain.MarkdownFile{
		{RelPath: "doc1.md", Hash: "new-hash"},
	}
	pages := []domain.NotionPage{
		{ID: "page1", MdexPath: "doc1.md", MdexHash: "old-hash"},
	}

	plan := usecase.ComputeExportPlan(files, pages, false)
	gt.A(t, plan).Length(1)
	gt.V(t, plan[0].Action).Equal(domain.ActionUpdate)
}

func TestComputeExportPlanDeleteRemoved(t *testing.T) {
	var files []domain.MarkdownFile
	pages := []domain.NotionPage{
		{ID: "page1", MdexPath: "deleted.md", MdexHash: "hash1"},
	}

	plan := usecase.ComputeExportPlan(files, pages, false)
	gt.A(t, plan).Length(1)
	gt.V(t, plan[0].Action).Equal(domain.ActionDelete)
}

func TestComputeExportPlanForceUpdate(t *testing.T) {
	files := []domain.MarkdownFile{
		{RelPath: "doc1.md", Hash: "hash1"},
	}
	pages := []domain.NotionPage{
		{ID: "page1", MdexPath: "doc1.md", MdexHash: "hash1"},
	}

	plan := usecase.ComputeExportPlan(files, pages, true)
	gt.A(t, plan).Length(1)
	gt.V(t, plan[0].Action).Equal(domain.ActionUpdate)
}

func TestExecuteNewFiles(t *testing.T) {
	nc := newMockNotionClient()
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "hello.md", Content: []byte("# Hello"), Hash: "abc123"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.CreatePageCalls()).Length(1)
	gt.S(t, nc.CreatePageCalls()[0].DatabaseID).Equal("db-id")
}

func TestExecuteDeletedFiles(t *testing.T) {
	nc := newMockNotionClient()
	nc.QueryDatabaseFunc = func(_ context.Context, _ string, _ string, _ string, _ string, _ string) ([]domain.NotionPage, error) {
		return []domain.NotionPage{
			{ID: "page-to-delete", MdexPath: "removed.md", MdexHash: "old"},
		}, nil
	}
	fs := newMockFileScanner([]domain.MarkdownFile{})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.ArchivePageCalls()).Length(1)
	gt.S(t, nc.ArchivePageCalls()[0].PageID).Equal("page-to-delete")
}

func TestExecuteUpdatedFiles(t *testing.T) {
	nc := newMockNotionClient()
	nc.QueryDatabaseFunc = func(_ context.Context, _ string, _ string, _ string, _ string, _ string) ([]domain.NotionPage, error) {
		return []domain.NotionPage{
			{ID: "existing-page", MdexPath: "doc.md", MdexHash: "old-hash"},
		}, nil
	}
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "doc.md", Content: []byte("# Updated"), Hash: "new-hash"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	// Should archive old page and create new one
	gt.A(t, nc.ArchivePageCalls()).Length(1)
	gt.A(t, nc.CreatePageCalls()).Length(1)
}

func TestExecuteSkipUnchanged(t *testing.T) {
	nc := newMockNotionClient()
	nc.QueryDatabaseFunc = func(_ context.Context, _ string, _ string, _ string, _ string, _ string) ([]domain.NotionPage, error) {
		return []domain.NotionPage{
			{ID: "existing-page", MdexPath: "doc.md", MdexHash: "same-hash"},
		}, nil
	}
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "doc.md", Content: []byte("# Same"), Hash: "same-hash"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.ArchivePageCalls()).Length(0)
	gt.A(t, nc.CreatePageCalls()).Length(0)
}

func TestExecuteWithFrontmatter(t *testing.T) {
	content := []byte("---\ntitle: Custom Title\ntags:\n  - go\n  - notion\ncategory: tech\n---\n# Hello\n")
	nc := newMockNotionClient()
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "hello.md", Content: content, Hash: "abc123"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.CreatePageCalls()).Length(1)

	props := nc.CreatePageCalls()[0].Properties

	// Verify title uses frontmatter value instead of filename
	titleRaw := props["Name"].(map[string]interface{})
	titleArr := titleRaw["title"].([]map[string]interface{})
	titleText := titleArr[0]["text"].(map[string]interface{})
	gt.S(t, titleText["content"].(string)).Equal("Custom Title")

	// Verify tags multi_select property
	tagsRaw, ok := props["Tags"]
	gt.B(t, ok).True()
	tagsProp := tagsRaw.(map[string]interface{})
	tagsValues := tagsProp["multi_select"].([]map[string]interface{})
	gt.N(t, len(tagsValues)).Equal(2)
	gt.S(t, tagsValues[0]["name"].(string)).Equal("go")
	gt.S(t, tagsValues[1]["name"].(string)).Equal("notion")

	// Verify category select property
	catRaw, ok := props["Category"]
	gt.B(t, ok).True()
	catProp := catRaw.(map[string]interface{})
	catSelect := catProp["select"].(map[string]interface{})
	gt.S(t, catSelect["name"].(string)).Equal("tech")
}

func TestExecuteWithoutFrontmatter(t *testing.T) {
	nc := newMockNotionClient()
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "plain.md", Content: []byte("# No frontmatter"), Hash: "def456"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.CreatePageCalls()).Length(1)

	props := nc.CreatePageCalls()[0].Properties

	// Title should fall back to filename without extension
	titleRaw := props["Name"].(map[string]interface{})
	titleArr := titleRaw["title"].([]map[string]interface{})
	titleText := titleArr[0]["text"].(map[string]interface{})
	gt.S(t, titleText["content"].(string)).Equal("plain")

	// Tags and Category should not be set
	_, hasTags := props["Tags"]
	gt.B(t, hasTags).False()
	_, hasCat := props["Category"]
	gt.B(t, hasCat).False()
}

func TestExecuteWithDomain(t *testing.T) {
	nc := newMockNotionClient()
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "hello.md", Content: []byte("# Hello"), Hash: "abc123"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
		Domain:           "my-project",
		DomainProperty:   "Domain",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.CreatePageCalls()).Length(1)

	props := nc.CreatePageCalls()[0].Properties

	// Domain select property should be set
	domainRaw, ok := props["Domain"]
	gt.B(t, ok).True()
	domainProp := domainRaw.(map[string]interface{})
	domainSelect := domainProp["select"].(map[string]interface{})
	gt.S(t, domainSelect["name"].(string)).Equal("my-project")

	// Verify QueryDatabase was called with domain parameters
	gt.A(t, nc.QueryDatabaseCalls()).Length(1)
	gt.S(t, nc.QueryDatabaseCalls()[0].DomainProperty).Equal("Domain")
	gt.S(t, nc.QueryDatabaseCalls()[0].DomainValue).Equal("my-project")
}

func TestExecuteWithoutDomain(t *testing.T) {
	nc := newMockNotionClient()
	fs := newMockFileScanner([]domain.MarkdownFile{
		{RelPath: "hello.md", Content: []byte("# Hello"), Hash: "abc123"},
	})

	uc := usecase.NewExportUseCase(nc, fs)
	config := domain.ExportConfig{
		NotionDatabaseID: "db-id",
		NotionToken:      "token",
		Dir:              "/tmp/test",
		PathProperty:     "mdex_path",
		HashProperty:     "mdex_hash",
		TagsProperty:     "Tags",
		CategoryProperty: "Category",
	}

	err := uc.Execute(context.Background(), config)
	gt.NoError(t, err)
	gt.A(t, nc.CreatePageCalls()).Length(1)

	props := nc.CreatePageCalls()[0].Properties

	// Domain should not be set when not configured
	_, hasDomain := props["Domain"]
	gt.B(t, hasDomain).False()

	// Verify QueryDatabase was called with empty domain parameters
	gt.A(t, nc.QueryDatabaseCalls()).Length(1)
	gt.S(t, nc.QueryDatabaseCalls()[0].DomainProperty).Equal("")
	gt.S(t, nc.QueryDatabaseCalls()[0].DomainValue).Equal("")
}
