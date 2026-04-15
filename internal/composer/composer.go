// Package composer builds the composed runtime HTML output from a pack.
// This is the single source of composition logic used by both the runtime server
// and the editor preview — the two must always use the same rules.
package composer

import (
	"fmt"
	"html/template"
	"strings"

	"titleflow/internal/models"
)

// composedTmpl is the outer document shell that TitleFlow owns.
// Pack authors never write this; it is always generated.
var composedTmpl = template.Must(template.New("composed").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>TitleFlow — {{.PackName}}</title>
<style>
/* TitleFlow base reset — keeps the overlay transparent and full-size */
*, *::before, *::after { box-sizing: border-box; }
html, body {
  margin: 0; padding: 0;
  width: 1920px; height: 1080px;
  background: transparent;
  overflow: hidden;
}

/*
 * :root defaults for all published CSS variables across all elements.
 *
 * These are injected by TitleFlow so that var() references in animation
 * shorthand properties always resolve — even before an element is fired
 * (and therefore still display:none). Without this, some Chromium builds
 * do not compute styles for descendants of a display:none element, causing
 * var() to fall back to the CSS initial value (0s for animation-duration).
 *
 * The element's own CSS declarations have the same specificity as :root
 * but appear later in the stylesheet, so they take precedence for normal
 * property overrides. The dock's live override (ID-level selector) still
 * wins over both.
 */
:root {
{{range .VarDefaults}}  {{.Name}}: {{.Value}}; /* {{.Slug}} */
{{end}}}

/*
 * Elements start hidden. .tf-fired makes them visible and restarts CSS
 * animations via the display:none → display:block cycle.
 */
.tf-element {
  display: none;
  position: absolute;
  top: 0; left: 0;
  width: 1920px; height: 1080px;
}
.tf-element.tf-fired {
  display: block;
}
</style>
{{range .Elements}}
<style data-tf-element="{{.Slug}}">
{{.CSS}}
</style>
{{end}}
</head>
<body>
{{range .Elements}}
<div id="tf-element-{{.Slug}}" class="tf-element" data-tf-pack="{{.Pack}}" data-tf-slug="{{.Slug}}">
{{.HTML}}
</div>
{{end}}
{{range .Elements}}{{if .JS}}
<script data-tf-element="{{.Slug}}">
(function(){
// Element: {{.Slug}}
var __root = document.getElementById('tf-element-{{.Slug}}');
{{.JS}}
})();
</script>
{{end}}{{end}}

<!-- TitleFlow runtime client — SSE events + initial state restore -->
<script>
(function() {

  // applyOverrides injects a dynamic <style> element that overrides CSS custom
  // properties for a given element using ID-level specificity.
  // ID specificity (1,0,0) beats the pack's class-level declarations (0,1,0),
  // so the override wins regardless of where the variable is declared in pack CSS.
  // elementId should be in format "pack--element"; extracts slug for DOM ID lookup.
  function applyOverrides(elementId, vars) {
    var styleId = 'tf-override-' + elementId;
    var styleEl = document.getElementById(styleId);
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = styleId;
      document.head.appendChild(styleEl);
    }
    if (!styleEl._vars) styleEl._vars = {};
    Object.keys(vars).forEach(function(k) { styleEl._vars[k] = vars[k]; });
    var decls = Object.keys(styleEl._vars).map(function(k) {
      return k + ':' + styleEl._vars[k];
    }).join(';');
    // Extract just the element slug (everything after the first "--")
    var slug = elementId.split('--')[1];
    var sel = '#tf-element-' + slug;
    styleEl.textContent = sel + ',' + sel + ' *{' + decls + '}';
  }

  // On page load, fetch the current dock control values from the server and
  // apply them. This restores dock settings after a page reload without
  // requiring the user to re-enter values in the dock.
  fetch('/api/state')
    .then(function(r) { return r.json(); })
    .then(function(state) {
      // Restore HTML binding text values.
      // Keys are in format "pack--element" to prevent namespace collisions.
      var binds = state.binds || {};
      Object.keys(binds).forEach(function(elementId) {
        Object.keys(binds[elementId]).forEach(function(key) {
          // Extract just the element slug (everything after the first "--")
          var slug = elementId.split('--')[1];
          var root = document.getElementById('tf-element-' + slug);
          if (!root) return;
          var node = root.querySelector('[data-tf-bind="' + key + '"]');
          if (node) node.textContent = binds[elementId][key];
        });
      });
      // Restore CSS variable overrides.
      // Keys are in format "pack--element" to prevent namespace collisions.
      var cssVars = state.css_vars || {};
      Object.keys(cssVars).forEach(function(elementId) {
        applyOverrides(elementId, cssVars[elementId]);
      });
    })
    .catch(function() {
      // /api/state is only available from the runtime server (port 8080).
      // Silently ignore when the composed HTML is loaded from another origin
      // (e.g. the editor preview at port 8081).
    });

  // SSE client — receives real-time commands from the dock.
  var es = new EventSource('/events');

  es.addEventListener('fire', function(e) {
    var d = JSON.parse(e.data);
    console.log('[tf] fire SSE received:', JSON.stringify(d));
    // d.element is in format "pack--element" to prevent namespace collisions
    // Extract just the element slug (everything after the first "--")
    var slug = d.element.split('--')[1];
    var el = document.getElementById('tf-element-' + slug);
    if (!el) { console.log('[tf] element not found:', d.element); return; }
    var evt = d.event || 'show';
    console.log('[tf] dispatching tf:fire on', el.id, 'event=', evt);
    if (evt === 'show') {
      // Reliable animation restart: remove the element from the DOM and reinsert
      // the same node. DOM reinsertion always produces a clean animation state.
      // void el.offsetWidth is unreliable here because offsetWidth returns 0 on
      // display:none elements and does not force a real layout pass for the subtree.
      // Reinserting the same node preserves JS references (__root) and any event
      // listeners set up by element.js.
      el.classList.remove('tf-fired');
      var parent = el.parentNode;
      var next = el.nextSibling;
      parent.removeChild(el);
      parent.insertBefore(el, next);
      el.classList.add('tf-fired');
    }
    // Dispatch cancelable CustomEvent — element.js can call preventDefault()
    // to suppress the runtime's default hide behavior when it needs full
    // control (e.g. GSAP staggered animations with custom timing).
    // Extract slug from namespaced element name for backward compatibility with element.js
    var ce = new CustomEvent('tf:fire', {
      detail: { element: slug, event: evt },
      cancelable: true
    });
    document.dispatchEvent(ce);
    // Default hide: add .tf-hiding for the CSS out-animation, then remove
    // .tf-fired after the animation duration to fully hide the element.
    // This mirrors "show" by handling hide at the runtime level so packs
    // do not need element.js boilerplate for basic hide behavior.
    if (evt === 'hide' && !ce.defaultPrevented) {
      // Cancel any in-flight or completed CSS animations on the subtree so the
      // .tf-hiding rule's animation-name list is treated as a brand-new animation
      // set. Without this, Chromium pairs animations by index across the cascade
      // change and the out-animation in .tf-hiding does not reliably restart.
      // Transitions are unnamed and unaffected by this reset.
      el.style.animation = 'none';
      var animNodes = el.querySelectorAll('*');
      for (var i = 0; i < animNodes.length; i++) animNodes[i].style.animation = 'none';
      void el.offsetWidth; // force reflow while animation:none is in effect
      el.style.animation = '';
      for (var j = 0; j < animNodes.length; j++) animNodes[j].style.animation = '';

      el.classList.add('tf-hiding');
      var durStr = getComputedStyle(el).getPropertyValue('--tf-ui-hide-duration').trim();
      var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
      if (!ms || isNaN(ms)) ms = 600;
      setTimeout(function() {
        el.classList.remove('tf-fired', 'tf-hiding');
      }, ms);
    }
  });

  es.addEventListener('reset', function(e) {
    var d = JSON.parse(e.data);
    // d.element is in format "pack--element" to prevent namespace collisions
    // Extract just the element slug (everything after the first "--")
    var slug = d.element.split('--')[1];
    var el = document.getElementById('tf-element-' + slug);
    if (!el) return;
    el.classList.remove('tf-fired');
  });

  es.addEventListener('bind', function(e) {
    var d = JSON.parse(e.data);
    // d.element is in format "pack--element" to prevent namespace collisions
    // Extract just the element slug (everything after the first "--")
    var slug = d.element.split('--')[1];
    var root = document.getElementById('tf-element-' + slug);
    if (!root) return;
    var node = root.querySelector('[data-tf-bind="' + d.key + '"]');
    if (!node) return;
    node.textContent = d.value;
  });

  es.addEventListener('cssvar', function(e) {
    var d = JSON.parse(e.data);
    // d.element is in format "pack--element" to prevent namespace collisions
    var vars = {};
    vars[d.name] = d.value;
    applyOverrides(d.element, vars);
  });

  es.addEventListener('reload-page', function(e) {
    console.log('[tf] reload-page event received');
    location.reload();
  });

})();
</script>
</body>
</html>
`))

// varDefault is one CSS variable default injected at :root level.
type varDefault struct {
	Slug  string
	Name  template.CSS
	Value template.CSS
}

// elementData is the per-element data passed to the template.
type elementData struct {
	Pack string
	Slug string
	HTML template.HTML
	CSS  template.CSS
	JS   template.JS
}

// composedData is the full data passed to composedTmpl.
type composedData struct {
	PackName    string
	VarDefaults []varDefault
	Elements    []elementData
}

// Compose builds the composed runtime HTML for the given pack.
// Elements are already sorted alphabetically by the scanner.
// The returned string is the full HTML document ready to serve.
func Compose(pack *models.Pack) (string, error) {
	data := composedData{
		PackName: pack.Name,
	}

	for _, elem := range pack.Elements {
		for _, v := range elem.CSSVars {
			data.VarDefaults = append(data.VarDefaults, varDefault{
				Slug:  elem.Slug,
				Name:  template.CSS(v.Name),
				Value: template.CSS(v.Value),
			})
		}

		data.Elements = append(data.Elements, elementData{
			Pack: pack.Name,
			Slug: elem.Slug,
			HTML: template.HTML(elem.HTMLSource), // authors own this markup
			CSS:  template.CSS(elem.CSSSource),   // authors own this CSS
			JS:   template.JS(elem.JSSource),
		})
	}

	var buf strings.Builder
	if err := composedTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("composer: template execution failed: %w", err)
	}
	return buf.String(), nil
}
