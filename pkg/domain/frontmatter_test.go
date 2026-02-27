package domain_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/mdex/pkg/domain"
)

func TestParseFrontmatterAllFields(t *testing.T) {
	content := []byte("---\ntitle: My Page\ntags:\n  - go\n  - notion\ncategory: tech\n---\n# Hello\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	gt.S(t, meta.Title).Equal("My Page")
	gt.A(t, meta.Tags).Length(2)
	gt.S(t, meta.Tags[0]).Equal("go")
	gt.S(t, meta.Tags[1]).Equal("notion")
	gt.S(t, meta.Category).Equal("tech")
	gt.S(t, string(body)).Equal("# Hello\n")
}

func TestParseFrontmatterTitleOnly(t *testing.T) {
	content := []byte("---\ntitle: Custom Title\n---\n# Body\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	gt.S(t, meta.Title).Equal("Custom Title")
	gt.A(t, meta.Tags).Length(0)
	gt.S(t, meta.Category).Equal("")
	gt.S(t, string(body)).Equal("# Body\n")
}

func TestParseFrontmatterTagsOnly(t *testing.T) {
	content := []byte("---\ntags:\n  - api\n---\n# Body\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	gt.A(t, meta.Tags).Length(1)
	gt.S(t, meta.Tags[0]).Equal("api")
	gt.S(t, meta.Category).Equal("")
	gt.S(t, string(body)).Equal("# Body\n")
}

func TestParseFrontmatterCategoryOnly(t *testing.T) {
	content := []byte("---\ncategory: blog\n---\n# Body\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	gt.A(t, meta.Tags).Length(0)
	gt.S(t, meta.Category).Equal("blog")
	gt.S(t, string(body)).Equal("# Body\n")
}

func TestParseFrontmatterNoFrontmatter(t *testing.T) {
	content := []byte("# Hello\nNo frontmatter here.\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	gt.A(t, meta.Tags).Length(0)
	gt.S(t, meta.Category).Equal("")
	gt.S(t, string(body)).Equal("# Hello\nNo frontmatter here.\n")
}

func TestParseFrontmatterEmptyContent(t *testing.T) {
	meta, body, err := domain.ParseFrontmatter([]byte{})
	gt.NoError(t, err)
	gt.A(t, meta.Tags).Length(0)
	gt.S(t, meta.Category).Equal("")
	gt.N(t, len(body)).Equal(0)
}

func TestParseFrontmatterMissingClosingDelimiter(t *testing.T) {
	content := []byte("---\ntags:\n  - go\n# No closing delimiter\n")

	meta, body, err := domain.ParseFrontmatter(content)
	gt.NoError(t, err)
	// Treated as no frontmatter
	gt.A(t, meta.Tags).Length(0)
	gt.S(t, string(body)).Equal(string(content))
}

func TestParseFrontmatterInvalidYAML(t *testing.T) {
	content := []byte("---\n: invalid: yaml: [broken\n---\n# Body\n")

	_, _, err := domain.ParseFrontmatter(content)
	gt.Error(t, err)
}
