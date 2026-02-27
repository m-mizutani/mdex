package converter

import (
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

// RichText represents a Notion rich text object.
type RichText struct {
	Type        string              `json:"type"`
	Text        *TextContent        `json:"text,omitempty"`
	Annotations *RichTextAnnotation `json:"annotations,omitempty"`
}

// TextContent is the text content within a rich text object.
type TextContent struct {
	Content string `json:"content"`
	Link    *Link  `json:"link,omitempty"`
}

// Link represents a URL link.
type Link struct {
	URL string `json:"url"`
}

// RichTextAnnotation represents text formatting annotations.
type RichTextAnnotation struct {
	Bold          bool   `json:"bold"`
	Italic        bool   `json:"italic"`
	Strikethrough bool   `json:"strikethrough"`
	Underline     bool   `json:"underline"`
	Code          bool   `json:"code"`
	Color         string `json:"color"`
}

// annotationState tracks the current inline formatting state while walking the AST.
type annotationState struct {
	bold          bool
	italic        bool
	strikethrough bool
	code          bool
	link          string
}

// convertInlineChildren walks the children of an inline node and collects rich text objects.
func convertInlineChildren(n ast.Node, source []byte) []RichText {
	var result []RichText
	state := &annotationState{}
	collectInline(n, source, state, &result)
	return result
}

func collectInline(n ast.Node, source []byte, state *annotationState, result *[]RichText) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch v := child.(type) {
		case *ast.Text:
			text := string(v.Value(source))
			if text != "" {
				*result = append(*result, makeRichText(text, state))
			}
			if v.SoftLineBreak() {
				*result = append(*result, makeRichText("\n", state))
			}
			if v.HardLineBreak() {
				*result = append(*result, makeRichText("\n", state))
			}

		case *ast.CodeSpan:
			// Collect text from code span children
			text := collectTextContent(v, source)
			oldCode := state.code
			state.code = true
			*result = append(*result, makeRichText(text, state))
			state.code = oldCode

		case *ast.Emphasis:
			switch v.Level {
			case 1:
				oldItalic := state.italic
				state.italic = true
				collectInline(v, source, state, result)
				state.italic = oldItalic
			case 2:
				oldBold := state.bold
				state.bold = true
				collectInline(v, source, state, result)
				state.bold = oldBold
			}

		case *east.Strikethrough:
			oldStrike := state.strikethrough
			state.strikethrough = true
			collectInline(v, source, state, result)
			state.strikethrough = oldStrike

		case *ast.Link:
			oldLink := state.link
			state.link = string(v.Destination)
			collectInline(v, source, state, result)
			state.link = oldLink

		case *ast.AutoLink:
			url := string(v.URL(source))
			oldLink := state.link
			state.link = url
			*result = append(*result, makeRichText(url, state))
			state.link = oldLink

		default:
			// For other inline elements, recurse into children
			collectInline(child, source, state, result)
		}
	}
}

func collectTextContent(n ast.Node, source []byte) string {
	var text string
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if t, ok := child.(*ast.Text); ok {
			text += string(t.Value(source))
		}
	}
	return text
}

func makeRichText(text string, state *annotationState) RichText {
	rt := RichText{
		Type: "text",
		Text: &TextContent{
			Content: text,
		},
		Annotations: &RichTextAnnotation{
			Bold:          state.bold,
			Italic:        state.italic,
			Strikethrough: state.strikethrough,
			Code:          state.code,
			Color:         "default",
		},
	}
	if state.link != "" {
		rt.Text.Link = &Link{URL: state.link}
	}
	return rt
}

// splitRichText splits a slice of RichText objects so that no single text content
// exceeds the Notion API limit of 2000 characters.
func splitRichText(rts []RichText) []RichText {
	const maxLen = 2000
	var result []RichText
	for _, rt := range rts {
		if rt.Text == nil || len(rt.Text.Content) <= maxLen {
			result = append(result, rt)
			continue
		}
		content := rt.Text.Content
		for len(content) > 0 {
			end := maxLen
			if end > len(content) {
				end = len(content)
			}
			chunk := RichText{
				Type: rt.Type,
				Text: &TextContent{
					Content: content[:end],
					Link:    rt.Text.Link,
				},
				Annotations: rt.Annotations,
			}
			result = append(result, chunk)
			content = content[end:]
		}
	}
	return result
}
