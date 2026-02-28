package converter_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/mdex/pkg/domain/converter"
)

func helperConvert(t *testing.T, md string) []converter.Block {
	t.Helper()
	blocks, err := converter.Convert([]byte(md), "test.md", "")
	gt.NoError(t, err)
	return blocks
}

func TestConvertHeading(t *testing.T) {
	blocks := helperConvert(t, "# Hello\n## World\n### Sub")
	gt.A(t, blocks).Length(3)

	gt.V(t, blocks[0]["type"]).Equal("heading_1")
	gt.V(t, blocks[1]["type"]).Equal("heading_2")
	gt.V(t, blocks[2]["type"]).Equal("heading_3")
}

func TestConvertParagraph(t *testing.T) {
	blocks := helperConvert(t, "This is a paragraph.")
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("paragraph")
}

func TestConvertBulletedList(t *testing.T) {
	md := "- item 1\n- item 2\n- item 3\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(3)
	for _, b := range blocks {
		gt.V(t, b["type"]).Equal("bulleted_list_item")
	}
}

func TestConvertNumberedList(t *testing.T) {
	md := "1. first\n2. second\n3. third\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(3)
	for _, b := range blocks {
		gt.V(t, b["type"]).Equal("numbered_list_item")
	}
}

func TestConvertFencedCodeBlock(t *testing.T) {
	md := "```go\nfmt.Println(\"hello\")\n```\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("code")

	code := blocks[0]["code"].(map[string]interface{})
	gt.V(t, code["language"]).Equal("go")
}

func TestConvertBlockquote(t *testing.T) {
	md := "> This is a quote\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("quote")
}

func TestConvertHorizontalRule(t *testing.T) {
	md := "---\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("divider")
}

func TestConvertTable(t *testing.T) {
	md := "| A | B |\n|---|---|\n| 1 | 2 |\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("table")
}

func TestConvertTaskList(t *testing.T) {
	md := "- [x] done\n- [ ] todo\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(2)
	gt.V(t, blocks[0]["type"]).Equal("to_do")
	gt.V(t, blocks[1]["type"]).Equal("to_do")

	todo0 := blocks[0]["to_do"].(map[string]interface{})
	gt.V(t, todo0["checked"]).Equal(true)
	todo1 := blocks[1]["to_do"].(map[string]interface{})
	gt.V(t, todo1["checked"]).Equal(false)
}

func TestConvertLocalImageAbsolutePathWithImageBaseDir(t *testing.T) {
	md := "![alt](/images/photo.png)\n"
	blocks, err := converter.Convert([]byte(md), "/workspace/content/docs/page.md", "/workspace/static")
	gt.NoError(t, err)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("image")

	img := blocks[0]["image"].(map[string]interface{})
	gt.V(t, img["type"]).Equal("file_upload")
	gt.V(t, img["local_path"]).Equal("/workspace/static/images/photo.png")
}

func TestConvertLocalImageRelativePathWithImageBaseDir(t *testing.T) {
	md := "![alt](./photo.png)\n"
	blocks, err := converter.Convert([]byte(md), "/workspace/content/docs/page.md", "/workspace/static")
	gt.NoError(t, err)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("image")

	img := blocks[0]["image"].(map[string]interface{})
	gt.V(t, img["type"]).Equal("file_upload")
	// Relative path should still resolve relative to the markdown file's directory
	gt.V(t, img["local_path"]).Equal("/workspace/content/docs/photo.png")
}

func TestConvertLocalImageAbsolutePathWithoutImageBaseDir(t *testing.T) {
	md := "![alt](/photo.png)\n"
	blocks, err := converter.Convert([]byte(md), "/workspace/content/docs/page.md", "")
	gt.NoError(t, err)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("image")

	img := blocks[0]["image"].(map[string]interface{})
	gt.V(t, img["type"]).Equal("file_upload")
	// Without imageBaseDir, absolute path resolves relative to markdown file's directory (backward compat)
	gt.V(t, img["local_path"]).Equal("/workspace/content/docs/photo.png")
}

func TestConvertLocalImagePathTraversalWithImageBaseDir(t *testing.T) {
	md := "![alt](/../../etc/passwd)\n"
	blocks, err := converter.Convert([]byte(md), "/workspace/content/docs/page.md", "/workspace/static")
	gt.NoError(t, err)
	// Path traversal should be blocked, resulting in no image block
	gt.A(t, blocks).Length(0)
}

func TestConvertLocalImagePathTraversalRelative(t *testing.T) {
	md := "![alt](../../../etc/passwd)\n"
	blocks, err := converter.Convert([]byte(md), "/workspace/content/docs/page.md", "")
	gt.NoError(t, err)
	// Path traversal should be blocked
	gt.A(t, blocks).Length(0)
}

func TestConvertExternalImage(t *testing.T) {
	md := "![alt](https://example.com/image.png)\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)
	gt.V(t, blocks[0]["type"]).Equal("image")

	img := blocks[0]["image"].(map[string]interface{})
	gt.V(t, img["type"]).Equal("external")
}

func TestConvertInlineBold(t *testing.T) {
	md := "This is **bold** text.\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)

	para := blocks[0]["paragraph"].(map[string]interface{})
	richTexts := para["rich_text"].([]converter.RichText)

	// Find the bold part
	var foundBold bool
	for _, rt := range richTexts {
		if rt.Text != nil && rt.Text.Content == "bold" {
			gt.V(t, rt.Annotations.Bold).Equal(true)
			foundBold = true
		}
	}
	gt.V(t, foundBold).Equal(true)
}

func TestConvertInlineCode(t *testing.T) {
	md := "Use `fmt.Println` here.\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)

	para := blocks[0]["paragraph"].(map[string]interface{})
	richTexts := para["rich_text"].([]converter.RichText)

	var foundCode bool
	for _, rt := range richTexts {
		if rt.Text != nil && rt.Text.Content == "fmt.Println" {
			gt.V(t, rt.Annotations.Code).Equal(true)
			foundCode = true
		}
	}
	gt.V(t, foundCode).Equal(true)
}

func TestConvertLink(t *testing.T) {
	md := "Click [here](https://example.com) please.\n"
	blocks := helperConvert(t, md)
	gt.A(t, blocks).Length(1)

	para := blocks[0]["paragraph"].(map[string]interface{})
	richTexts := para["rich_text"].([]converter.RichText)

	var foundLink bool
	for _, rt := range richTexts {
		if rt.Text != nil && rt.Text.Content == "here" && rt.Text.Link != nil {
			gt.S(t, rt.Text.Link.URL).Equal("https://example.com")
			foundLink = true
		}
	}
	gt.V(t, foundLink).Equal(true)
}
