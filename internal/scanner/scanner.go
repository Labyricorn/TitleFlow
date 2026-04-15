// Package scanner walks a pack root directory and identifies element folders.
// It does not perform deep validation — it only finds the structure.
package scanner

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"titleflow/internal/models"
)

// slugRe matches valid element slugs: lowercase, kebab-case, no leading/trailing hyphens.
var slugRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// IsValidSlug returns true if s is a valid element slug.
func IsValidSlug(s string) bool {
	return slugRe.MatchString(s)
}

// ScanPack reads a pack root directory and returns a Pack with its elements.
// Elements with invalid slug names are still included so the validator can report them.
// Files are returned in alphabetical slug order.
func ScanPack(root string) (*models.Pack, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "scan", Path: root, Err: os.ErrInvalid}
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	pack := &models.Pack{
		Name: filepath.Base(root),
		Root: root,
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden directories and the reserved assets folder.
		if strings.HasPrefix(name, ".") {
			continue
		}
		if name == "assets" {
			continue
		}
		elem := scanElement(filepath.Join(root, name), name)
		pack.Elements = append(pack.Elements, elem)
	}

	// Sort alphabetically by slug so load order is deterministic.
	sort.Slice(pack.Elements, func(i, j int) bool {
		return pack.Elements[i].Slug < pack.Elements[j].Slug
	})

	return pack, nil
}

// scanElement reads element files from dir. Missing files result in empty strings.
func scanElement(dir, slug string) *models.Element {
	elem := &models.Element{
		Slug: slug,
		Dir:  dir,
	}

	elem.HTMLSource = readFile(filepath.Join(dir, "element.html"))
	elem.CSSSource = readFile(filepath.Join(dir, "element.css"))

	jsPath := filepath.Join(dir, "element.js")
	if _, err := os.Stat(jsPath); err == nil {
		elem.JSSource = readFile(jsPath)
		elem.HasJS = true
	}

	// Parse controls once at scan-time so models remain read-only at request-time.
	ParseElementControls(elem)

	return elem
}

// readFile returns the file contents as a string, or "" on any error.
func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// ScanPackDir scans a directory for immediate subdirectories that look like pack roots.
// A directory is treated as a pack root if it contains at least one valid element subdirectory
// or simply exists as a directory (we allow empty packs to be reported, validated, and fixed).
func ScanPackDir(dir string) ([]*models.Pack, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var packs []*models.Pack
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		root := filepath.Join(dir, e.Name())
		pack, err := ScanPack(root)
		if err != nil {
			continue
		}
		packs = append(packs, pack)
	}
	return packs, nil
}
