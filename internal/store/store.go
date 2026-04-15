// Package store handles reading and writing pack files on disk.
// The file/folder structure is the source of truth — no hidden database is used.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// slugRe matches valid element slugs (same rule as scanner).
var slugRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// ReadFile returns the contents of a file inside a pack element folder.
// path must be an absolute path.
func ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a file, creating parent directories as needed.
func WriteFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("store: mkdir %q: %w", filepath.Dir(path), err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// CreateElement creates the standard file set for a new element inside packRoot.
// Returns an error if the slug is invalid or the folder already exists.
func CreateElement(packRoot, slug string) error {
	if !slugRe.MatchString(slug) {
		return fmt.Errorf("store: %q is not a valid element slug (must be lowercase kebab-case)", slug)
	}

	elemDir := filepath.Join(packRoot, slug)
	if _, err := os.Stat(elemDir); err == nil {
		return fmt.Errorf("store: element folder %q already exists", slug)
	}

	if err := os.MkdirAll(elemDir, 0755); err != nil {
		return fmt.Errorf("store: create element dir: %w", err)
	}

	files := map[string]string{
		"element.html": defaultHTML(slug),
		"element.css":  defaultCSS(slug),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(elemDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("store: write %s: %w", name, err)
		}
	}
	return nil
}

// CreatePack creates a new empty pack root directory.
func CreatePack(packsDir, packName string) error {
	packRoot := filepath.Join(packsDir, packName)
	if _, err := os.Stat(packRoot); err == nil {
		return fmt.Errorf("store: pack %q already exists", packName)
	}
	return os.MkdirAll(packRoot, 0755)
}

// defaultHTML returns starter HTML for a new element.
func defaultHTML(slug string) string {
	return fmt.Sprintf(`<div class="%s">
  <span class="%s__text"
        data-tf-bind="text"
        data-tf-label="Text">Your text here</span>
</div>
`, slug, slug)
}

// defaultCSS returns starter CSS for a new element.
func defaultCSS(slug string) string {
	return fmt.Sprintf(`.%s {
  position: absolute;
  bottom: 80px;
  left: 60px;

  --tf-ui-text-color: #ffffff;
  --tf-ui-background-color: #000000;
  --tf-ui-font-size: 36px;
}

.%s__text {
  color: var(--tf-ui-text-color);
  background: var(--tf-ui-background-color);
  font-size: var(--tf-ui-font-size);
  padding: 8px 16px;
}
`, slug, slug)
}
