package walker

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Entry represents a file discovered by the walker.
type Entry struct {
	Path      string // relative to repo root, forward-slash separated
	AbsPath   string
	Size      int64
	IsDir     bool
	IsBinary  bool
}

// Options configures the walker.
type Options struct {
	// MaxFileSize is the threshold above which files are flagged as skipped.
	// Default: 1MB.
	MaxFileSize int64

	// BinarySampleSize is how many bytes to read to detect binary content.
	// Default: 8192.
	BinarySampleSize int
}

// DefaultOptions returns sensible defaults.
func DefaultOptions() Options {
	return Options{
		MaxFileSize:      1 << 20, // 1 MB
		BinarySampleSize: 8192,
	}
}

// Walk walks the repo at root, respecting ignore rules, and calls fn for each file.
// Directories matching ignore rules are skipped entirely.
func Walk(root string, matcher *IgnoreMatcher, opts Options, fn func(Entry) error) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Skip root itself.
		if relPath == "." {
			return nil
		}

		// Normalize to forward slashes.
		relPath = filepath.ToSlash(relPath)

		// Always skip .git directory.
		if d.IsDir() && d.Name() == ".git" {
			return fs.SkipDir
		}

		// Check ignore rules.
		if matcher.Match(relPath, d.IsDir()) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}

		// We only emit files, not directories.
		// Also resolve symlinks — a symlink to a directory should be skipped.
		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// If this is a symlink, check if it points to a directory.
		if info.Mode()&os.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil {
				return nil // skip broken symlinks
			}
			ri, err := os.Stat(resolved)
			if err != nil {
				return nil
			}
			if ri.IsDir() {
				return nil // skip symlinks to directories
			}
		}

		entry := Entry{
			Path:    relPath,
			AbsPath: path,
			Size:    info.Size(),
			IsDir:   false,
		}

		// Detect binary.
		entry.IsBinary = isBinary(path, opts.BinarySampleSize)

		return fn(entry)
	})
}

// isBinary samples the first n bytes and checks for null bytes
// or a high ratio of non-UTF8 content.
func isBinary(path string, sampleSize int) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, sampleSize)
	n, err := f.Read(buf)
	if n == 0 {
		return false // empty files are not binary
	}
	buf = buf[:n]

	// Check for null bytes — strong signal for binary.
	for _, b := range buf {
		if b == 0 {
			return true
		}
	}

	// Check if the content is valid UTF-8 text.
	// If less than 90% of the content is valid UTF-8, consider it binary.
	if !utf8.Valid(buf) {
		// Count valid runes.
		valid := 0
		for i := 0; i < len(buf); {
			r, size := utf8.DecodeRune(buf[i:])
			if r != utf8.RuneError || size > 1 {
				valid += size
			}
			i += size
		}
		if float64(valid)/float64(len(buf)) < 0.9 {
			return true
		}
	}

	return false
}

// Location returns a short location label for a file path.
// e.g. "root" for top-level files, "src/" for files under src/.
func Location(relPath string) string {
	dir := filepath.Dir(filepath.ToSlash(relPath))
	if dir == "." {
		return "root"
	}
	// Return the top-level directory with a trailing slash.
	parts := strings.SplitN(dir, "/", 2)
	return parts[0] + "/"
}
