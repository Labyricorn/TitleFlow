// Package validator checks a pack for structural and authoring rule violations.
// All errors and warnings are plain text.
package validator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"titleflow/internal/models"
	"titleflow/internal/scanner"

	"golang.org/x/net/html"
)

// tfUIRe matches valid published CSS variable names (--tf-ui-<one-or-more-segments>).
var tfUIRe = regexp.MustCompile(`^--tf-ui-[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// broadSelectorRe catches obviously forbidden broad selectors in CSS source.
// This is a conservative textual check, not a full CSS parser.
var broadSelectorRe = regexp.MustCompile(`(?m)^\s*(html|body|\*)\s*\{`)

// ValidatePack runs all validation checks on a scanned pack.
// The pack must have been returned by scanner.ScanPack.
func ValidatePack(pack *models.Pack) *models.ValidationResult {
	result := &models.ValidationResult{}

	if len(pack.Elements) == 0 {
		result.Errors = append(result.Errors,
			fmt.Sprintf("ERROR: pack %q: no element folders found in pack root", pack.Name))
	}

	// Track slug uniqueness.
	seen := map[string]bool{}

	for _, elem := range pack.Elements {
		packRef := fmt.Sprintf("pack %q", pack.Name)
		elemRef := fmt.Sprintf("element %q", elem.Slug)

		// Validate slug.
		if !scanner.IsValidSlug(elem.Slug) {
			result.Errors = append(result.Errors,
				fmt.Sprintf("ERROR: %s, %s: slug is not valid lowercase kebab-case", packRef, elemRef))
		}

		// Check slug uniqueness.
		if seen[elem.Slug] {
			result.Errors = append(result.Errors,
				fmt.Sprintf("ERROR: %s: duplicate element slug %q", packRef, elem.Slug))
		}
		seen[elem.Slug] = true

		// Check required files exist.
		htmlPath := filepath.Join(elem.Dir, "element.html")
		cssPath := filepath.Join(elem.Dir, "element.css")

		if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
			result.Errors = append(result.Errors,
				fmt.Sprintf("ERROR: %s, %s, file %q: required file is missing", packRef, elemRef, "element.html"))
		}
		if _, err := os.Stat(cssPath); os.IsNotExist(err) {
			result.Errors = append(result.Errors,
				fmt.Sprintf("ERROR: %s, %s, file %q: required file is missing", packRef, elemRef, "element.css"))
		}

		// Validate HTML source if present.
		if elem.HTMLSource != "" {
			htmlErrors, htmlWarnings := validateHTML(pack.Name, elem)
			result.Errors = append(result.Errors, htmlErrors...)
			result.Warnings = append(result.Warnings, htmlWarnings...)
		}

		// Validate CSS source if present.
		if elem.CSSSource != "" {
			cssErrors, cssWarnings := validateCSS(pack.Name, elem)
			result.Errors = append(result.Errors, cssErrors...)
			result.Warnings = append(result.Warnings, cssWarnings...)
		}
	}

	return result
}

// validateHTML parses element.html and checks editable binding declarations.
func validateHTML(packName string, elem *models.Element) (errors, warnings []string) {
	packRef := fmt.Sprintf("pack %q", packName)
	elemRef := fmt.Sprintf("element %q", elem.Slug)
	fileRef := `file "element.html"`

	doc, err := html.Parse(strings.NewReader(elem.HTMLSource))
	if err != nil {
		errors = append(errors,
			fmt.Sprintf("ERROR: %s, %s, %s: could not parse HTML: %v", packRef, elemRef, fileRef, err))
		return
	}

	// Track binding keys to detect duplicates.
	bindKeys := map[string]bool{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			bind := attrVal(n, "data-tf-bind")
			label := attrVal(n, "data-tf-label")
			fire := attrVal(n, "data-tf-fire")

			hasBind := bind != ""
			hasLabel := label != ""
			hasFire := fire != ""

			if hasBind || hasLabel || hasFire {
				// One or more attributes present — apply full rules.
				if hasBind && !hasLabel {
					errors = append(errors,
						fmt.Sprintf(
							`ERROR: %s, %s, %s: editable binding %q is missing required data-tf-label`,
							packRef, elemRef, fileRef, bind))
				}
				if hasLabel && !hasBind && !hasFire {
					errors = append(errors,
						fmt.Sprintf(
							`ERROR: %s, %s, %s: element with data-tf-label %q is missing required data-tf-bind`,
							packRef, elemRef, fileRef, label))
				}
				if hasBind && hasLabel {
					if bindKeys[bind] {
						errors = append(errors,
							fmt.Sprintf(
								`ERROR: %s, %s, %s: duplicate binding key %q within the same element`,
								packRef, elemRef, fileRef, bind))
					}
					bindKeys[bind] = true
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return
}

// validateCSS does a textual scan of element.css for published variables and forbidden selectors.
func validateCSS(packName string, elem *models.Element) (errors, warnings []string) {
	packRef := fmt.Sprintf("pack %q", packName)
	elemRef := fmt.Sprintf("element %q", elem.Slug)
	fileRef := `file "element.css"`

	// Check for forbidden broad selectors.
	if broadSelectorRe.MatchString(elem.CSSSource) {
		warnings = append(warnings,
			fmt.Sprintf(
				"WARNING: %s, %s, %s: contains a broad selector (html, body, or *) — elements must not assume page ownership",
				packRef, elemRef, fileRef))
	}

	// Extract --tf-ui-* variable declarations and validate their names.
	// Pattern: --tf-ui-<something>: <value>;  (anywhere in the CSS source)
	tfVarRe := regexp.MustCompile(`(--tf-ui-[a-zA-Z0-9_-]+)\s*:`)
	matches := tfVarRe.FindAllStringSubmatch(elem.CSSSource, -1)
	for _, m := range matches {
		varName := m[1]
		if !tfUIRe.MatchString(varName) {
			errors = append(errors,
				fmt.Sprintf(
					"ERROR: %s, %s, %s: published CSS variable %q does not match required --tf-ui-<lowercase-kebab> format",
					packRef, elemRef, fileRef, varName))
		}
	}

	return
}

// attrVal returns the value of an HTML attribute, or "" if not present.
func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}
