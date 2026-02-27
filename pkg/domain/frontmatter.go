package domain

import (
	"bytes"

	"github.com/m-mizutani/goerr/v2"
	"gopkg.in/yaml.v3"
)

var frontmatterDelimiter = []byte("---")

// ParseFrontmatter extracts YAML frontmatter from markdown content.
// It returns the parsed metadata and the remaining body content.
// If no frontmatter is found, it returns zero-value Metadata and the original content.
func ParseFrontmatter(content []byte) (Metadata, []byte, error) {
	trimmed := bytes.TrimLeft(content, " \t")
	if !bytes.HasPrefix(trimmed, frontmatterDelimiter) {
		return Metadata{}, content, nil
	}

	// Find the end of the opening delimiter line
	afterOpen := trimmed[len(frontmatterDelimiter):]
	nlIdx := bytes.IndexByte(afterOpen, '\n')
	if nlIdx == -1 {
		return Metadata{}, content, nil
	}

	// Check that the rest of the opening line is only whitespace
	openingRest := afterOpen[:nlIdx]
	if len(bytes.TrimRight(openingRest, " \t\r")) > 0 {
		return Metadata{}, content, nil
	}

	yamlStart := afterOpen[nlIdx+1:]

	// Find the closing delimiter
	closeIdx := findClosingDelimiter(yamlStart)
	if closeIdx == -1 {
		return Metadata{}, content, nil
	}

	yamlBlock := yamlStart[:closeIdx]
	remaining := yamlStart[closeIdx+len(frontmatterDelimiter):]

	// Skip the rest of the closing delimiter line
	if nlIdx := bytes.IndexByte(remaining, '\n'); nlIdx != -1 {
		remaining = remaining[nlIdx+1:]
	} else {
		remaining = nil
	}

	var meta Metadata
	if err := yaml.Unmarshal(yamlBlock, &meta); err != nil {
		return Metadata{}, nil, goerr.Wrap(err, "parsing frontmatter YAML")
	}

	return meta, remaining, nil
}

// findClosingDelimiter finds the position of `---` at the start of a line.
func findClosingDelimiter(data []byte) int {
	search := data
	offset := 0
	for len(search) > 0 {
		idx := bytes.Index(search, frontmatterDelimiter)
		if idx == -1 {
			return -1
		}

		// Must be at the start of a line (idx == 0 or preceded by \n)
		if idx == 0 || search[idx-1] == '\n' {
			// Check that the rest of the line is only whitespace
			afterDelim := search[idx+len(frontmatterDelimiter):]
			nlIdx := bytes.IndexByte(afterDelim, '\n')
			var lineRest []byte
			if nlIdx == -1 {
				lineRest = afterDelim
			} else {
				lineRest = afterDelim[:nlIdx]
			}
			if len(bytes.TrimRight(lineRest, " \t\r")) == 0 {
				return offset + idx
			}
		}

		search = search[idx+len(frontmatterDelimiter):]
		offset += idx + len(frontmatterDelimiter)
	}
	return -1
}
