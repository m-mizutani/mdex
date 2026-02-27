package fs_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/mdex/pkg/infra/fs"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	dirs := []string{
		"subdir",
		".hidden",
		"subdir/nested",
	}
	for _, d := range dirs {
		gt.NoError(t, os.MkdirAll(filepath.Join(dir, d), 0o755))
	}

	// Create files
	files := map[string]string{
		"readme.md":             "# Hello",
		"subdir/doc.md":         "## Doc",
		"subdir/nested/deep.md": "### Deep",
		".hidden/secret.md":     "secret",
		"not-markdown.txt":      "text file",
	}
	for name, content := range files {
		gt.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}

	return dir
}

func TestScanMarkdownFiles(t *testing.T) {
	dir := setupTestDir(t)
	scanner := fs.New()

	files, err := scanner.ScanMarkdownFiles(dir)
	gt.NoError(t, err)

	// Should find 3 markdown files (excluding hidden dir and non-md file)
	gt.A(t, files).Length(3)

	pathSet := make(map[string]string)
	for _, f := range files {
		pathSet[f.RelPath] = f.Hash
	}

	// Check expected paths exist
	gt.M(t, pathSet).HasKey("readme.md")
	gt.M(t, pathSet).HasKey("subdir/doc.md")
	gt.M(t, pathSet).HasKey("subdir/nested/deep.md")

	// Verify hash
	expectedHash := sha256.Sum256([]byte("# Hello"))
	gt.S(t, pathSet["readme.md"]).Equal(hex.EncodeToString(expectedHash[:]))
}

func TestScanMarkdownFilesSkipsHiddenDirs(t *testing.T) {
	dir := setupTestDir(t)
	scanner := fs.New()

	files, err := scanner.ScanMarkdownFiles(dir)
	gt.NoError(t, err)

	for _, f := range files {
		gt.S(t, f.RelPath).NotContains(".hidden")
	}
}

func TestScanMarkdownFilesEmptyDir(t *testing.T) {
	dir := t.TempDir()
	scanner := fs.New()

	files, err := scanner.ScanMarkdownFiles(dir)
	gt.NoError(t, err)
	gt.A(t, files).Length(0)
}
