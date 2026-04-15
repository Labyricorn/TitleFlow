// Package models defines the shared data structures used across TitleFlow.
package models

// Pack represents a discovered and optionally validated overlay pack.
type Pack struct {
	// Name is the directory name of the pack root.
	Name string
	// Root is the absolute path to the pack root directory.
	Root string
	// Elements are the element folders found inside the pack root,
	// sorted alphabetically by slug (the folder name).
	Elements []*Element
}

// Element represents one element folder inside a pack.
type Element struct {
	// Slug is the folder name (lowercase kebab-case).
	Slug string
	// Dir is the absolute path to the element folder.
	Dir string
	// HTMLSource is the raw content of element.html.
	HTMLSource string
	// CSSSource is the raw content of element.css.
	CSSSource string
	// JSSource is the raw content of element.js, or "" if not present.
	JSSource string
	// HasJS is true when element.js exists in the element folder.
	HasJS bool
	// Bindings lists the explicitly declared HTML editable bindings.
	Bindings []*HTMLBinding
	// CSSVars lists the published CSS custom properties (--tf-ui-* prefix).
	CSSVars []*CSSVar
	// FireEvents lists the named fire events declared in element.html.
	FireEvents []*FireEvent
}

// HTMLBinding represents an explicitly declared editable HTML field.
// Both data-tf-bind and data-tf-label must be present.
type HTMLBinding struct {
	// Key is the value of data-tf-bind.
	Key string
	// Label is the value of data-tf-label.
	Label string
	// DefaultValue is the text content of the element as written.
	DefaultValue string
}

// CSSVar represents a published CSS custom property (--tf-ui-* prefix).
type CSSVar struct {
	// Name is the full variable name, e.g. "--tf-ui-accent-color".
	Name string
	// Value is the default value as written in element.css.
	Value string
	// Label is derived from Name by converting the suffix to Title Case words.
	Label string
	// InferredType is a conservative guess: "color", "length", "duration", "text".
	InferredType string
}

// FireEvent represents a named fire event declared in element.html via data-tf-fire.
type FireEvent struct {
	// Name is the value of data-tf-fire (e.g. "show", "hide").
	Name string
	// Label is the value of data-tf-label.
	Label string
}

// ValidationResult holds errors and warnings produced by the validator.
type ValidationResult struct {
	Errors   []string
	Warnings []string
}

// OK returns true when there are no errors.
func (v *ValidationResult) OK() bool {
	return len(v.Errors) == 0
}
