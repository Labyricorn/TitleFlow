// Package dock serves the Control Dock UI.
// The dock lists installed packs, lets the user choose the active pack,
// and shows only the explicitly published controls for the active pack.
package dock

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"

	"titleflow/internal/hub"
	"titleflow/internal/models"
)

// StateManager is the minimal interface the dock needs from the application state.
type StateManager interface {
	AllPacks() []*models.Pack
	ActivePack() *models.Pack
	SetActivePack(name string) error
	ReloadPacks() error
	ValidatePack(pack *models.Pack) *models.ValidationResult
}

// Handler returns an http.Handler for the dock UI.
func Handler(state StateManager, h *hub.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveDock(w, r, state)
	})
	mux.HandleFunc("/api/set-active", func(w http.ResponseWriter, r *http.Request) {
		apiSetActive(w, r, state, h)
	})
	mux.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) {
		apiReload(w, r, state, h)
	})
	return mux
}

func serveDock(w http.ResponseWriter, r *http.Request, state StateManager) {
	packs := state.AllPacks()
	active := state.ActivePack()

	var activeParsed *models.Pack
	if active != nil {
		activeParsed = active
	}

	data := struct {
		Packs      []*models.Pack
		Active     *models.Pack
		ActiveName string
	}{
		Packs:  packs,
		Active: activeParsed,
	}
	if active != nil {
		data.ActiveName = active.Name
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := dockTmpl.Execute(w, data); err != nil {
		log.Printf("dock: template error: %v", err)
	}
}

func apiSetActive(w http.ResponseWriter, r *http.Request, state StateManager, h *hub.Hub) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	name := r.FormValue("pack")
	if name == "" {
		http.Error(w, "missing pack name", http.StatusBadRequest)
		return
	}
	if err := state.SetActivePack(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Broadcast page reload event to all connected runtime clients.
	// This forces the OBS browser source to reload and fetch the new pack's HTML.
	h.Publish("reload-page", "{}")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func apiReload(w http.ResponseWriter, r *http.Request, state StateManager, h *hub.Hub) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := state.ReloadPacks(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Broadcast page reload event to all connected runtime clients.
	// This forces the OBS browser source to reload and refresh state.
	h.Publish("reload-page", "{}")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// labelFromType returns a short human-readable type badge.
func labelFromType(t string) string {
	switch t {
	case "color":
		return "Color"
	case "length":
		return "Length"
	case "duration":
		return "Duration"
	default:
		return "Text"
	}
}

// parseLengthValue extracts the numeric part from a CSS length string.
// "100px" → "100", "-50px" → "-50", "1.5em" → "1.5"
func parseLengthValue(val string) string {
	val = strings.TrimSpace(val)
	for _, unit := range []string{"rem", "vw", "vh", "ch", "px", "em", "pt", "%"} {
		if strings.HasSuffix(val, unit) {
			return strings.TrimSuffix(val, unit)
		}
	}
	return val
}

// extractLengthUnit returns the unit suffix from a CSS length string.
// "100px" → "px", "1.5em" → "em", "invalid" → "px" (default)
func extractLengthUnit(val string) string {
	val = strings.TrimSpace(val)
	for _, unit := range []string{"rem", "vw", "vh", "ch", "px", "em", "pt", "%"} {
		if strings.HasSuffix(val, unit) {
			return unit
		}
	}
	return "px"
}

// parseDurationValue extracts the numeric part from a duration string.
// "2800ms" → "2800", "3.5s" → "3.5", "invalid" → "0"
func parseDurationValue(val string) string {
	val = strings.TrimSpace(val)
	val = strings.TrimSuffix(val, "ms")
	val = strings.TrimSuffix(val, "s")
	if val == "" {
		return "0"
	}
	return val
}

// extractDurationUnit returns the unit suffix from a duration string.
// "2800ms" → "ms", "3.5s" → "s", "invalid" → "ms" (default)
func extractDurationUnit(val string) string {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "ms") {
		return "ms"
	}
	if strings.HasSuffix(val, "s") {
		return "s"
	}
	return "ms" // default fallback
}

var dockFuncs = template.FuncMap{
	"typeLabel":           labelFromType,
	"parseDurationValue":  parseDurationValue,
	"extractDurationUnit": extractDurationUnit,
	"parseLengthValue":    parseLengthValue,
	"extractLengthUnit":   extractLengthUnit,
	"join":                strings.Join,
	"sprintf":             fmt.Sprintf,
}

var dockTmpl = template.Must(template.New("dock").Funcs(dockFuncs).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>TitleFlow — Control Dock</title>
<style>
:root {
  --bg: #1a1a1e;
  --surface: #26262b;
  --border: #3a3a42;
  --accent: #58a6ff;
  --text: #e6e6e8;
  --muted: #888;
  --danger: #f85149;
  --success: #3fb950;
  --radius: 6px;
  --font: system-ui, sans-serif;
}
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: var(--font); background: var(--bg); color: var(--text); font-size: 14px; }
header { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid var(--border); }
header h1 { font-size: 16px; font-weight: 600; letter-spacing: 0.02em; }
.links a { color: var(--accent); text-decoration: none; margin-left: 12px; font-size: 13px; }
.links a:hover { text-decoration: underline; }
main { padding: 16px; }
section { margin-bottom: 24px; }
h2 { font-size: 13px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.08em; color: var(--muted); margin-bottom: 10px; }
.pack-list { display: flex; flex-wrap: wrap; gap: 8px; }
.pack-btn {
  background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius);
  color: var(--text); padding: 7px 14px; cursor: pointer; font-size: 13px; transition: border-color 0.15s;
}
.pack-btn:hover { border-color: var(--accent); }
.pack-btn.active { border-color: var(--accent); color: var(--accent); font-weight: 600; }
.reload-btn {
  background: none; border: 1px solid var(--border); border-radius: var(--radius);
  color: var(--muted); padding: 5px 10px; cursor: pointer; font-size: 12px;
}
.reload-btn:hover { border-color: var(--text); color: var(--text); }
.element-section { background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius); margin-bottom: 12px; overflow: hidden; }
.element-header { padding: 8px 14px; background: #2e2e35; font-size: 13px; font-weight: 600; letter-spacing: 0.04em; border-bottom: 1px solid var(--border); display: flex; align-items: center; gap: 8px; }
.element-header .slug { flex: 1; }
.header-controls { display: flex; gap: 6px; margin-left: auto; margin-right: 8px; }
.header-fire-btn { background: var(--accent); border: none; border-radius: 4px; color: #0d1117; padding: 4px 10px; font-size: 11px; font-weight: 700; cursor: pointer; letter-spacing: 0.03em; }
.header-fire-btn:hover { opacity: 0.85; }
.header-reset-btn { background: none; border: 1px solid var(--border); border-radius: 4px; color: var(--muted); padding: 4px 8px; font-size: 11px; cursor: pointer; }
.header-reset-btn:hover { border-color: var(--text); color: var(--text); }
.control-list { padding: 10px 14px; display: flex; flex-direction: column; gap: 10px; }
.control-row { display: flex; flex-direction: column; gap: 4px; }
.control-label { font-size: 12px; color: var(--muted); display: flex; gap: 6px; align-items: center; }
.type-badge { background: #3a3a42; border-radius: 3px; padding: 1px 5px; font-size: 10px; letter-spacing: 0.04em; text-transform: uppercase; }
.control-input { background: #1a1a1e; border: 1px solid var(--border); border-radius: 4px; color: var(--text); padding: 6px 10px; font-size: 13px; width: 100%; }
.control-input:focus { outline: none; border-color: var(--accent); }
input[type=color].control-input { height: 36px; padding: 2px 4px; cursor: pointer; }
.no-active { color: var(--muted); font-size: 13px; padding: 12px 0; }
.hint { color: var(--muted); font-size: 12px; margin-top: 6px; }
.element-header { cursor: pointer; user-select: none; }
.collapse-indicator { font-size: 10px; opacity: 0.4; transition: transform 0.15s ease; margin-left: 4px; }
.element-section.collapsed .collapse-indicator { transform: rotate(180deg); }
.element-section.collapsed .control-list { display: none; }

</style>
</head>
<body>
<header>
  <h1>TitleFlow &mdash; Control Dock</h1>
  <div class="links">
    <a href="http://localhost:8081/?pack={{.ActiveName | urlquery}}&hidepack=true" target="_blank">Editor</a>
    <a href="http://localhost:8080" target="_blank">Overlay</a>
  </div>
</header>
<main>

<section>
  <div style="display:flex;align-items:center;gap:12px;margin-bottom:10px;">
    <h2 style="margin:0">Installed Packs</h2>
    <form method="POST" action="/api/reload">
      <button class="reload-btn" type="submit">&#8635; Reload</button>
    </form>
  </div>
  <div class="pack-list">
    {{range .Packs}}
    <form method="POST" action="/api/set-active">
      <input type="hidden" name="pack" value="{{.Name}}">
      <button class="pack-btn{{if eq .Name $.ActiveName}} active{{end}}" type="submit">{{.Name}}</button>
    </form>
    {{else}}
    <p class="no-active">No packs found. Add a pack folder to the packs directory and reload.</p>
    {{end}}
  </div>
</section>

{{if .Active}}
<section>
  <h2>Controls &mdash; {{.Active.Name}}</h2>
  {{range .Active.Elements}}
  {{$slug := .Slug}}
  <div class="element-section">
    <div class="element-header" onclick="tfToggle(this)">
      <span class="slug">{{$slug}}</span>
      <div class="header-controls">
        {{range .FireEvents}}
        <button class="header-fire-btn" onclick="event.stopPropagation(); tfFire('{{$slug}}', '{{.Name}}')">&#9654; {{.Label}}</button>
        {{end}}
        <button class="header-reset-btn" onclick="event.stopPropagation(); tfReset('{{$slug}}')">&#8635; Reset</button>
      </div>
      <span class="collapse-indicator">&#9650;</span>
    </div>
    <div class="control-list">
      {{range .Bindings}}
      <div class="control-row">
        <div class="control-label">
          <span>{{.Label}}</span>
          <span class="type-badge">HTML</span>
        </div>
        <input class="control-input" type="text" value="{{.DefaultValue}}" placeholder="{{.Label}}"
               oninput="tfBind('{{$slug}}', '{{.Key}}', this.value)">
      </div>
      {{end}}
      {{range .CSSVars}}
      <div class="control-row">
        <div class="control-label">
          <span>{{.Label}}</span>
          <span class="type-badge">{{typeLabel .InferredType}}</span>
        </div>
        {{if eq .InferredType "color"}}
        <input class="control-input" type="color" value="{{.Value}}"
               oninput="tfCSSVar('{{$slug}}', '{{.Name}}', this.value)">
        {{else if eq .InferredType "duration"}}
        <input class="control-input" type="number" step="100"
               value="{{parseDurationValue .Value}}"
               oninput="tfCSSVarDuration('{{$slug}}', '{{.Name}}', this.value, '{{extractDurationUnit .Value}}')"
               onwheel="tfNumberWheel(event, '{{$slug}}', '{{.Name}}', '{{extractDurationUnit .Value}}')">
        {{else if eq .InferredType "length"}}
        <input class="control-input" type="number" step="1"
               value="{{parseLengthValue .Value}}"
               oninput="tfCSSVar('{{$slug}}', '{{.Name}}', this.value + '{{extractLengthUnit .Value}}')"
               onwheel="tfNumberWheel(event, '{{$slug}}', '{{.Name}}', '{{extractLengthUnit .Value}}')">
        {{else}}
        <input class="control-input" type="text" value="{{.Value}}" placeholder="{{.Value}}"
               oninput="tfCSSVar('{{$slug}}', '{{.Name}}', this.value)">
        {{end}}
      </div>
      {{end}}
      {{if and (not .Bindings) (not .CSSVars)}}
      <p class="hint">No explicitly published controls in this element.</p>
      {{end}}
    </div>
  </div>
  {{end}}
</section>
{{else}}
<section>
  <p class="no-active">Select a pack above to see its controls.</p>
</section>
{{end}}

</main>
<script>
function tfFire(slug, event) {
  fetch('http://localhost:8080/api/fire?element=' + encodeURIComponent(slug) +
        '&event=' + encodeURIComponent(event), { method: 'POST' });
}
function tfReset(slug) {
  fetch('http://localhost:8080/api/reset?element=' + encodeURIComponent(slug), { method: 'POST' });
}
function tfToggle(header) {
  header.closest('.element-section').classList.toggle('collapsed');
}
function tfBind(slug, key, value) {
  fetch('http://localhost:8080/api/bind?element=' + encodeURIComponent(slug) +
        '&key=' + encodeURIComponent(key) +
        '&value=' + encodeURIComponent(value), { method: 'POST' });
}
function tfCSSVar(slug, name, value) {
  fetch('http://localhost:8080/api/cssvar?element=' + encodeURIComponent(slug) +
        '&name=' + encodeURIComponent(name) +
        '&value=' + encodeURIComponent(value), { method: 'POST' });
}
function tfCSSVarDuration(slug, name, value, unit) {
  tfCSSVar(slug, name, value + unit);
}
function tfNumberWheel(e, slug, name, unit) {
  e.preventDefault();
  var input = e.currentTarget;
  var step = parseFloat(input.step) || 1;
  var delta = e.deltaY < 0 ? step : -step;
  var newVal = (parseFloat(input.value) || 0) + delta;
  input.value = newVal;
  tfCSSVar(slug, name, newVal + unit);
}
</script>
</body>
</html>
`))

