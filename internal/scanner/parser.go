// parser.go extracts the explicitly declared editable controls from element source files.
// This is separate from validation — it populates the Bindings and CSSVars fields
// so that the editor, dock, and inspector can render controls without re-parsing every time.
package scanner

import (
	"regexp"
	"strings"

	"titleflow/internal/models"

	"golang.org/x/net/html"
)

// ParseElementControls fills in elem.Bindings, elem.CSSVars, and elem.FireEvents from the element source.
// It does not validate — it only extracts what is present and explicitly declared.
func ParseElementControls(elem *models.Element) {
	elem.Bindings = parseHTMLBindings(elem.HTMLSource)
	elem.CSSVars = parseCSSVars(elem.CSSSource)
	elem.FireEvents = parseFireEvents(elem.HTMLSource)
}

// parseHTMLBindings walks the HTML and collects nodes that have both
// data-tf-bind and data-tf-label attributes.
func parseHTMLBindings(src string) []*models.HTMLBinding {
	if src == "" {
		return nil
	}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil
	}

	var bindings []*models.HTMLBinding
	seen := map[string]bool{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			bind := attrValP(n, "data-tf-bind")
			label := attrValP(n, "data-tf-label")
			if bind != "" && label != "" && !seen[bind] {
				seen[bind] = true
				bindings = append(bindings, &models.HTMLBinding{
					Key:          bind,
					Label:        label,
					DefaultValue: textContent(n),
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return bindings
}

// parseFireEvents walks the HTML and collects nodes that have data-tf-fire attributes.
func parseFireEvents(src string) []*models.FireEvent {
	if src == "" {
		return nil
	}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil
	}

	var events []*models.FireEvent
	seen := map[string]bool{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			name := attrValP(n, "data-tf-fire")
			label := attrValP(n, "data-tf-label")
			if name != "" && !seen[name] {
				seen[name] = true
				if label == "" {
					label = name
				}
				events = append(events, &models.FireEvent{
					Name:  name,
					Label: label,
				})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return events
}

// tfVarDeclRe matches a --tf-ui-* property declaration in CSS source.
// Captures: (varName, value).
var tfVarDeclRe = regexp.MustCompile(`(--tf-ui-[a-zA-Z0-9_-]+)\s*:\s*([^;}\n]+)`)

// parseCSSVars scans the CSS source for --tf-ui-* declarations.
func parseCSSVars(src string) []*models.CSSVar {
	if src == "" {
		return nil
	}
	matches := tfVarDeclRe.FindAllStringSubmatch(src, -1)
	seen := map[string]bool{}
	var vars []*models.CSSVar
	for _, m := range matches {
		name := m[1]
		value := strings.TrimSpace(m[2])
		if seen[name] {
			continue
		}
		seen[name] = true
		vars = append(vars, &models.CSSVar{
			Name:         name,
			Value:        value,
			Label:        deriveCSSLabel(name),
			InferredType: inferCSSType(name, value),
		})
	}
	return vars
}

// deriveCSSLabel converts --tf-ui-accent-color → "Accent Color".
func deriveCSSLabel(name string) string {
	// Strip the required prefix.
	suffix := strings.TrimPrefix(name, "--tf-ui-")
	parts := strings.Split(suffix, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

// inferCSSType makes a conservative guess at the control type from the value and name.
func inferCSSType(name, value string) string {
	v := strings.TrimSpace(value)
	// Color: #rrggbb, #rgb, rgb(...), hsl(...), named colors.
	if strings.HasPrefix(v, "#") || strings.HasPrefix(v, "rgb") || strings.HasPrefix(v, "hsl") {
		return "color"
	}
	// Duration: ends with ms or s.
	if strings.HasSuffix(v, "ms") || (strings.HasSuffix(v, "s") && !strings.HasSuffix(v, "px")) {
		return "duration"
	}
	// Length: ends with px, em, rem, %, vw, vh, etc.
	lengthSuffixes := []string{"px", "em", "rem", "%", "vw", "vh", "pt", "ch"}
	for _, suf := range lengthSuffixes {
		if strings.HasSuffix(v, suf) {
			return "length"
		}
	}
	// Font family hints in the name.
	if strings.Contains(name, "font-family") || strings.Contains(name, "font-face") {
		return "text"
	}
	return "text"
}

// attrValP returns an attribute value from an html.Node.
func attrValP(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textContent returns the concatenated text content of a node and its descendants.
func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(b.String())
}
