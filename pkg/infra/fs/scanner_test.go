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

func TestReadMarkdownFiles(t *testing.T) {
	dir := setupTestDir(t)
	scanner := fs.New()

	paths := []string{
		filepath.Join(dir, "readme.md"),
		filepath.Join(dir, "subdir", "doc.md"),
	}

	files, err := scanner.ReadMarkdownFiles(paths)
	gt.NoError(t, err)
	gt.A(t, files).Length(2)

	// Verify hash
	expectedHash := sha256.Sum256([]byte("# Hello"))
	gt.V(t, files[0].Hash).Equal(hex.EncodeToString(expectedHash[:]))

	// Verify FilePath is set to absolute path
	absPath, _ := filepath.Abs(paths[0])
	gt.V(t, files[0].FilePath).Equal(absPath)
}

func TestReadMarkdownFilesRelPath(t *testing.T) {
	dir := t.TempDir()
	gt.NoError(t, os.MkdirAll(filepath.Join(dir, "a"), 0o755))
	gt.NoError(t, os.MkdirAll(filepath.Join(dir, "b"), 0o755))
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "a", "doc.md"), []byte("# A"), 0o644))
	gt.NoError(t, os.WriteFile(filepath.Join(dir, "b", "doc.md"), []byte("# B"), 0o644))

	scanner := fs.New()
	files, err := scanner.ReadMarkdownFiles([]string{
		filepath.Join(dir, "a", "doc.md"),
		filepath.Join(dir, "b", "doc.md"),
	})
	gt.NoError(t, err)
	gt.A(t, files).Length(2)

	// RelPaths should be distinct even though basenames are the same
	gt.V(t, files[0].RelPath).NotEqual(files[1].RelPath)
}

func TestReadMarkdownFilesRejectsDirectory(t *testing.T) {
	dir := t.TempDir()
	scanner := fs.New()

	_, err := scanner.ReadMarkdownFiles([]string{dir})
	gt.Error(t, err)
}

func TestReadMarkdownFilesRejectsNonMarkdown(t *testing.T) {
	dir := t.TempDir()
	txtFile := filepath.Join(dir, "note.txt")
	gt.NoError(t, os.WriteFile(txtFile, []byte("hello"), 0o644))

	scanner := fs.New()
	_, err := scanner.ReadMarkdownFiles([]string{txtFile})
	gt.Error(t, err)
}

func TestReadMarkdownFilesFollowsSymlink(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(dir, "real.md")
	gt.NoError(t, os.WriteFile(realFile, []byte("# Real"), 0o644))

	linkFile := filepath.Join(dir, "link.md")
	gt.NoError(t, os.Symlink(realFile, linkFile))

	scanner := fs.New()
	files, err := scanner.ReadMarkdownFiles([]string{linkFile})
	gt.NoError(t, err)
	gt.A(t, files).Length(1)

	expectedHash := sha256.Sum256([]byte("# Real"))
	gt.V(t, files[0].Hash).Equal(hex.EncodeToString(expectedHash[:]))
}

func TestReadMarkdownFilesRejectsSymlinkToDirectory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "sub")
	gt.NoError(t, os.MkdirAll(subDir, 0o755))

	linkDir := filepath.Join(dir, "link-dir")
	gt.NoError(t, os.Symlink(subDir, linkDir))

	scanner := fs.New()
	_, err := scanner.ReadMarkdownFiles([]string{linkDir})
	gt.Error(t, err)
}
