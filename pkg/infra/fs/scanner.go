package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/mdex/pkg/domain"
)

// Scanner implements domain.FileScanner.
type Scanner struct{}

// New creates a new Scanner.
func New() *Scanner {
	return &Scanner{}
}

// ScanMarkdownFiles recursively scans the given directory and returns all Markdown files.
func (s *Scanner) ScanMarkdownFiles(baseDir string) ([]domain.MarkdownFile, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, goerr.Wrap(err, "resolving absolute path", goerr.V("baseDir", baseDir))
	}

	var files []domain.MarkdownFile

	walkErr := filepath.WalkDir(absBase, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return goerr.Wrap(err, "walking directory", goerr.V("path", path))
		}

		// Skip hidden directories (starting with '.')
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Skip directories and non-markdown files
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Skip symlinks
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}

		content, err := os.ReadFile(path) // #nosec G304 -- path comes from filepath.WalkDir on user-specified directory
		if err != nil {
			return goerr.Wrap(err, "reading file", goerr.V("path", path))
		}

		relPath, err := filepath.Rel(absBase, path)
		if err != nil {
			return goerr.Wrap(err, "computing relative path", goerr.V("path", path))
		}

		// Normalize to forward slashes
		relPath = filepath.ToSlash(filepath.Clean(relPath))

		hash := sha256.Sum256(content)

		files = append(files, domain.MarkdownFile{
			RelPath:  relPath,
			FilePath: path,
			Content:  content,
			Hash:     hex.EncodeToString(hash[:]),
		})

		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	return files, nil
}

// ReadMarkdownFiles reads the specified Markdown files and returns them.
func (s *Scanner) ReadMarkdownFiles(paths []string) ([]domain.MarkdownFile, error) {
	var files []domain.MarkdownFile

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, goerr.Wrap(err, "resolving absolute path", goerr.V("path", p))
		}

		info, err := os.Stat(absPath)
		if err != nil {
			return nil, goerr.Wrap(err, "stat file", goerr.V("path", absPath))
		}

		if info.IsDir() {
			return nil, goerr.New("expected a file but got a directory", goerr.V("path", absPath))
		}

		if !strings.HasSuffix(info.Name(), ".md") {
			return nil, goerr.New("expected a markdown file (.md)", goerr.V("path", absPath))
		}

		content, err := os.ReadFile(absPath) // #nosec G304 -- path comes from user-specified file arguments
		if err != nil {
			return nil, goerr.Wrap(err, "reading file", goerr.V("path", absPath))
		}

		hash := sha256.Sum256(content)

		// Use the user-supplied path (cleaned) as RelPath to avoid collisions
		// when files from different directories share the same basename.
		relPath := filepath.ToSlash(filepath.Clean(p))

		files = append(files, domain.MarkdownFile{
			RelPath:  relPath,
			FilePath: absPath,
			Content:  content,
			Hash:     hex.EncodeToString(hash[:]),
		})
	}

	return files, nil
}
