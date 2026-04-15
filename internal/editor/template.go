package editor

// editorHTML is the full editor page. It is inlined to keep the binary self-contained.
const editorHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>TitleFlow — Pack Editor</title>
<style>
:root {
  --bg: #0d1117;
  --surface: #161b22;
  --surface2: #21262d;
  --border: #30363d;
  --accent: #58a6ff;
  --accent2: #3fb950;
  --warn: #d29922;
  --danger: #f85149;
  --text: #c9d1d9;
  --muted: #8b949e;
  --radius: 6px;
  --font: system-ui, sans-serif;
  --mono: 'Cascadia Code', 'Fira Code', 'Consolas', monospace;

  /* Resizable panel sizes — updated by the JS splitter engine */
  --sidebar-w: 220px;
  --right-w: 320px;
  --code-h: 60%;
}
*, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
html, body { height: 100%; font-family: var(--font); background: var(--bg); color: var(--text); font-size: 13px; overflow: hidden; }

/* ── Top-level layout ── */
body { display: grid; grid-template-rows: 40px 1fr; }
header {
  display: flex; align-items: center; gap: 12px; padding: 0 16px;
  border-bottom: 1px solid var(--border); background: var(--surface);
}
.app-name { font-size: 14px; font-weight: 600; }
.header-sep { flex: 1; }
.header-links a { color: var(--accent); text-decoration: none; margin-left: 12px; font-size: 12px; }
.header-links a:hover { text-decoration: underline; }

/* Workspace: sidebar | splitter | center | splitter | right */
.workspace {
  display: grid;
  grid-template-columns: var(--sidebar-w) 4px 1fr 4px var(--right-w);
  overflow: hidden;
}

/* ── Sidebar ── */
.sidebar {
  display: flex; flex-direction: column; overflow: hidden;
  background: var(--surface); min-width: 0;
}
.sidebar-section { padding: 10px 12px 6px; }
.sidebar-label {
  font-size: 11px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.08em; color: var(--muted); margin-bottom: 8px;
}
.pack-select {
  width: 100%; background: var(--surface2); border: 1px solid var(--border);
  border-radius: var(--radius); color: var(--text); padding: 5px 8px;
  font-size: 12px; cursor: pointer;
}
.pack-select:focus { outline: none; border-color: var(--accent); }
.element-list { flex: 1; overflow-y: auto; padding: 4px 8px; }
.element-item {
  padding: 7px 10px; border-radius: var(--radius); cursor: pointer;
  font-size: 12px; color: var(--muted); border: 1px solid transparent;
  margin-bottom: 2px; transition: background 0.1s; font-family: var(--mono);
}
.element-item:hover { background: var(--surface2); color: var(--text); }
.element-item.active { background: var(--surface2); border-color: var(--accent); color: var(--accent); font-weight: 600; }
.sidebar-footer { padding: 10px 12px; border-top: 1px solid var(--border); }

/* ── Buttons ── */
.btn {
  display: inline-flex; align-items: center; gap: 5px; padding: 6px 12px;
  border-radius: var(--radius); border: 1px solid var(--border);
  background: var(--surface2); color: var(--text); cursor: pointer;
  font-size: 12px; transition: border-color 0.15s;
}
.btn:hover { border-color: var(--accent); color: var(--accent); }
.btn-success { background: var(--accent2); border-color: var(--accent2); color: #0d1117; font-weight: 600; }
.btn-success:hover { opacity: 0.85; }
.btn-primary { background: var(--accent); border-color: var(--accent); color: #0d1117; font-weight: 600; }
.btn-primary:hover { background: #79baff; border-color: #79baff; color: #0d1117; }
.btn-sm { padding: 4px 8px; font-size: 11px; }
.btn-full { width: 100%; justify-content: center; margin-bottom: 6px; }

/* ── Center column ── */
/* Rows: code editors | h-splitter | toolbar | bottom area */
.center {
  display: grid;
  grid-template-rows: var(--code-h) 4px auto 1fr;
  overflow: hidden; min-width: 0;
}

/* ── 3-column code area ── */
.code-columns {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  min-height: 0; overflow: hidden;
}
.code-col { display: flex; flex-direction: column; min-width: 0; min-height: 0; overflow: hidden; }
.code-col:not(:last-child) { border-right: 1px solid var(--border); }
.code-col-header {
  display: flex; align-items: center; gap: 8px; padding: 6px 12px;
  border-bottom: 1px solid var(--border); background: var(--surface); flex-shrink: 0;
}
.code-col-label { font-size: 12px; font-weight: 600; color: var(--text); }
.code-col-file { font-size: 11px; color: var(--muted); flex: 1; }
textarea.code {
  flex: 1; resize: none; border: none; outline: none;
  background: var(--bg); color: var(--text); font-family: var(--mono);
  font-size: 13px; line-height: 1.6; padding: 14px 16px; tab-size: 2; min-height: 0;
}
.js-note {
  padding: 4px 12px; font-size: 10px; color: var(--muted);
  background: var(--surface); border-bottom: 1px solid var(--border); flex-shrink: 0;
}
.js-note span { color: var(--warn); }

/* ── Toolbar ── */
.toolbar {
  display: flex; align-items: center; gap: 8px; padding: 8px 12px;
  border-top: 1px solid var(--border); border-bottom: 1px solid var(--border);
  background: var(--surface); flex-shrink: 0;
}
#save-status { font-size: 11px; color: var(--muted); flex: 1; }

/* ── Bottom area: output log + inspector ── */
.bottom-area {
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 0; overflow: hidden;
}
.bottom-left {
  display: flex; flex-direction: column; overflow: hidden;
  border-right: 1px solid var(--border);
}
.bottom-right { display: flex; flex-direction: column; overflow: hidden; }

/* Output log */
.output-log {
  flex: 1; overflow-y: auto; padding: 8px 12px;
  font-family: var(--mono); font-size: 11px; line-height: 1.7;
}
.log-entry { display: flex; gap: 10px; padding: 2px 0; border-bottom: 1px solid rgba(48,54,61,0.4); }
.log-ts { color: var(--muted); flex-shrink: 0; user-select: none; }
.log-ok   { color: var(--accent2); }
.log-err  { color: var(--danger); }
.log-warn { color: var(--warn); }
.log-info { color: var(--text); }
.log-empty { color: var(--muted); font-size: 11px; padding: 8px 0; }

/* Inspector (now inside bottom-right) */
.inspector-body { flex: 1; overflow-y: auto; padding: 8px 12px; }
.inspector-empty { color: var(--muted); font-size: 12px; padding: 8px 0; }
.inspector-group { margin-bottom: 10px; }
.inspector-group-label {
  font-size: 10px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.08em; color: var(--muted); margin-bottom: 6px;
}
.inspector-item { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; flex-wrap: wrap; }
.inspector-item .key { font-family: var(--mono); font-size: 11px; color: var(--accent); }
.inspector-item .lbl { font-size: 12px; color: var(--text); }
.inspector-badge { font-size: 10px; padding: 1px 5px; border-radius: 3px; text-transform: uppercase; letter-spacing: 0.04em; }
.badge-html     { background: #1f3a5f; color: #79c0ff; }
.badge-css      { background: #1f3a2a; color: #56d364; }
.badge-color    { background: #3d2f10; color: #e3b341; }
.badge-length   { background: #2d1f3d; color: #bc8cff; }
.badge-duration { background: #1e3d3d; color: #39d353; }

/* Section header (shared by output, inspector, dock) */
.section-header {
  display: flex; align-items: center; gap: 8px; padding: 7px 12px;
  border-bottom: 1px solid var(--border); background: var(--surface); flex-shrink: 0;
}
.section-title {
  font-size: 11px; font-weight: 600; text-transform: uppercase;
  letter-spacing: 0.06em; color: var(--muted); flex: 1;
}

/* ── Right panel: dock controls only ── */
.right-panel { display: flex; flex-direction: column; overflow: hidden; min-width: 0; }
.dock-section { flex: 1; display: flex; flex-direction: column; overflow: hidden; min-height: 0; }
.dock-body { flex: 1; overflow-y: auto; padding: 8px; }

/* Dock element cards */
.element-section {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); margin-bottom: 8px; overflow: hidden;
}
.element-header {
  padding: 7px 12px; background: var(--surface2); font-size: 12px; font-weight: 600;
  letter-spacing: 0.04em; border-bottom: 1px solid var(--border);
  display: flex; align-items: center; gap: 6px; cursor: pointer; user-select: none;
}
.element-header .slug { flex: 1; font-family: var(--mono); font-size: 11px; color: var(--text); }
.header-controls { display: flex; gap: 4px; }
.header-fire-btn {
  background: var(--accent); border: none; border-radius: 4px;
  color: #0d1117; padding: 3px 8px; font-size: 10px; font-weight: 700;
  cursor: pointer; letter-spacing: 0.03em;
}
.header-fire-btn:hover { opacity: 0.85; }
.header-reset-btn {
  background: none; border: 1px solid var(--border); border-radius: 4px;
  color: var(--muted); padding: 3px 6px; font-size: 10px; cursor: pointer;
}
.header-reset-btn:hover { border-color: var(--text); color: var(--text); }
.collapse-indicator { font-size: 10px; opacity: 0.35; transition: transform 0.15s ease; }
.element-section.collapsed .collapse-indicator { transform: rotate(180deg); }
.element-section.collapsed .control-list { display: none; }
.control-list { padding: 8px 12px; display: flex; flex-direction: column; gap: 8px; }
.control-row { display: flex; flex-direction: column; gap: 3px; }
.control-label { font-size: 11px; color: var(--muted); display: flex; gap: 5px; align-items: center; }
.type-badge {
  background: var(--surface2); border-radius: 3px; padding: 1px 5px;
  font-size: 9px; letter-spacing: 0.04em; text-transform: uppercase; border: 1px solid var(--border);
}
.control-input {
  background: var(--bg); border: 1px solid var(--border); border-radius: 4px;
  color: var(--text); padding: 5px 8px; font-size: 12px; width: 100%;
}
.control-input:focus { outline: none; border-color: var(--accent); }
input[type=color].control-input { height: 32px; padding: 2px 4px; cursor: pointer; }
.dock-hint { color: var(--muted); font-size: 11px; padding: 8px 4px; font-style: italic; }

/* ── Splitters ── */
.splitter { background: var(--border); flex-shrink: 0; z-index: 10; }
.splitter.v { cursor: col-resize; }
.splitter.h { cursor: row-resize; }
.splitter:hover, .splitter.active { background: var(--accent); }

/* ── Modals ── */
.modal-overlay {
  display: none; position: fixed; inset: 0;
  background: rgba(0,0,0,0.65); z-index: 1000;
  align-items: center; justify-content: center;
}
.modal-overlay.open { display: flex; }
.modal {
  background: var(--surface); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 20px 24px; min-width: 320px;
}
.modal h3 { font-size: 14px; margin-bottom: 12px; }
.modal input[type=text] {
  width: 100%; background: var(--bg); border: 1px solid var(--border);
  border-radius: 4px; color: var(--text); padding: 7px 10px;
  font-size: 13px; font-family: var(--mono); margin-bottom: 12px;
}
.modal input:focus { outline: none; border-color: var(--accent); }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; }
.modal-hint { font-size: 11px; color: var(--muted); margin-bottom: 10px; }

/* ── Scrollbars ── */
::-webkit-scrollbar { width: 6px; height: 6px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb { background: var(--border); border-radius: 3px; }
::-webkit-scrollbar-thumb:hover { background: var(--muted); }
</style>
</head>
<body>

<header>
  <span class="app-name">TitleFlow</span>
  <span style="font-size:11px;color:var(--muted)">Pack Editor{{if .CurrentPack}} &mdash; {{.CurrentPack.Name}}{{end}}</span>
  <span class="header-sep"></span>
  <div class="header-links">
    <a href="http://localhost:8080" target="_blank">Overlay ↗</a>
    <a href="http://localhost:8082" target="_blank">Dock ↗</a>
  </div>
</header>

<div class="workspace">

  <!-- ══════════════════════════════ Sidebar ══════════════════════════════ -->
  <aside class="sidebar">
    {{if not .HidePack}}
    <div class="sidebar-section">
      <div class="sidebar-label" style="display:flex;justify-content:space-between;align-items:center;">
        <span>Pack</span>
        <button class="btn btn-sm" onclick="reloadPacks()" title="Reload Packs" style="padding:2px 6px; font-size:10px; border:none; background:transparent;">&#8635; Reload</button>
      </div>
      <select class="pack-select" id="pack-select" onchange="switchPack(this.value)">
        {{range .Packs}}
        <option value="{{.Name}}"{{if $.CurrentPack}}{{if eq .Name $.CurrentPack.Name}} selected{{end}}{{end}}>{{.Name}}</option>
        {{else}}
        <option disabled>No packs found</option>
        {{end}}
      </select>
    </div>
    {{end}}
    <div class="sidebar-section" style="padding-bottom:4px;">
      <div class="sidebar-label">Elements</div>
    </div>
    <div class="element-list" id="element-list">
      {{if .CurrentPack}}
        {{range .CurrentPack.Elements}}
        <div class="element-item{{if $.CurrentElem}}{{if eq .Slug $.CurrentElem.Slug}} active{{end}}{{end}}"
             onclick="selectElement('{{.Slug}}')" data-slug="{{.Slug}}">{{.Slug}}</div>
        {{else}}
        <div style="color:var(--muted);font-size:12px;padding:8px 10px;">No elements yet.</div>
        {{end}}
      {{else}}
      <div style="color:var(--muted);font-size:12px;padding:8px 10px;">Select or create a pack.</div>
      {{end}}
    </div>
    <div class="sidebar-footer">
      <button class="btn btn-full btn-sm" onclick="openNewElementModal()">+ New Element</button>
      {{if not .HidePack}}
      <button class="btn btn-full btn-sm" onclick="openNewPackModal()">+ New Pack</button>
      {{end}}
    </div>
  </aside>

  <!-- Splitter: sidebar | center -->
  <div class="splitter v" data-prop="--sidebar-w"></div>

  <!-- ══════════════════════════ Center column ════════════════════════════ -->
  <div class="center">

    <!-- 3-column code editors -->
    <div class="code-columns">

      <div class="code-col">
        <div class="code-col-header">
          <span class="code-col-label">HTML</span>
          <span class="code-col-file">element.html</span>
          <button class="btn btn-sm" onclick="copyPanel('html')">Copy</button>
        </div>
        <textarea class="code" id="code-html" spellcheck="false" autocomplete="off"
          placeholder="<!-- Write your element markup here.
Use data-tf-bind and data-tf-label to declare editable fields. -->">{{if .CurrentElem}}{{.CurrentElem.HTMLSource}}{{end}}</textarea>
      </div>

      <div class="code-col">
        <div class="code-col-header">
          <span class="code-col-label">CSS</span>
          <span class="code-col-file">element.css</span>
          <button class="btn btn-sm" onclick="copyPanel('css')">Copy</button>
        </div>
        <textarea class="code" id="code-css" spellcheck="false" autocomplete="off"
          placeholder="/* Scope all selectors to this element.
Use --tf-ui-* custom properties to publish editable controls. */">{{if .CurrentElem}}{{.CurrentElem.CSSSource}}{{end}}</textarea>
      </div>

      <div class="code-col">
        <div class="code-col-header">
          <span class="code-col-label">JS</span>
          <span class="code-col-file">element.js</span>
          <button class="btn btn-sm" onclick="copyPanel('js')">Copy</button>
          {{if .CurrentElem}}
          <label style="font-size:11px;color:var(--muted);display:flex;align-items:center;gap:4px;cursor:pointer;white-space:nowrap;">
            <input type="checkbox" id="has-js-check" {{if .CurrentElem.HasJS}}checked{{end}} onchange="toggleJS(this.checked)">
            Include
          </label>
          {{end}}
        </div>
        <div class="js-note"><span>Optional.</span> element.js runs only when included.</div>
        <textarea class="code" id="code-js" spellcheck="false" autocomplete="off"
          placeholder="// Optional JS. __root is the element's wrapper div.
// Listen for CustomEvents dispatched by the runtime client.
">{{if .CurrentElem}}{{.CurrentElem.JSSource}}{{end}}</textarea>
      </div>

    </div>

    <!-- Splitter: code area | bottom area -->
    <div class="splitter h" data-prop="--code-h"></div>

    <!-- Toolbar -->
    <div class="toolbar">
      <button class="btn btn-success" onclick="saveElement()">Save</button>
      <button class="btn" onclick="validatePack()">Validate Pack</button>
      <span id="save-status"></span>
    </div>

    <!-- Bottom area: output log (left) + inspector (right) -->
    <div class="bottom-area">

      <div class="bottom-left">
        <div class="section-header">
          <span class="section-title">Output</span>
          <button class="btn btn-sm" onclick="clearLog()" title="Clear log">&#10005; Clear</button>
        </div>
        <div class="output-log" id="output-log">
          <div class="log-empty">Operations will appear here.</div>
        </div>
      </div>

      <div class="bottom-right">
        <div class="section-header">
          <span class="section-title">Inspector</span>
          <span style="font-size:11px;color:var(--muted);">Selected element controls</span>
        </div>
        <div class="inspector-body" id="inspector-body">
          <div class="inspector-empty">Save the element to see derived controls.</div>
        </div>
      </div>

    </div>

  </div>

  <!-- Splitter: center | right panel -->
  <div class="splitter v" data-prop="--right-w" data-invert="true"></div>

  <!-- ══════════════════════════ Right panel ══════════════════════════════ -->
  <div class="right-panel">
    <div class="dock-section">
      <div class="section-header">
        <span class="section-title">Dock Controls</span>
        {{if .CurrentPack}}<span style="font-size:11px;color:var(--muted);">→ Runtime :8080</span>{{end}}
      </div>
      <div class="dock-body" id="dock-body">
        {{if .CurrentPack}}
          {{range .CurrentPack.Elements}}
          {{$slug := .Slug}}
          <div class="element-section">
            <div class="element-header" onclick="tfToggle(this)">
              <span class="slug">{{$slug}}</span>
              <div class="header-controls">
                {{range .FireEvents}}
                <button class="header-fire-btn"
                        onclick="event.stopPropagation(); tfFire('{{$slug}}', '{{.Name}}')">&#9654; {{.Label}}</button>
                {{end}}
                <button class="header-reset-btn"
                        onclick="event.stopPropagation(); tfReset('{{$slug}}')">&#8635;</button>
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
                <input class="control-input" type="text"
                       value="{{.DefaultValue}}" placeholder="{{.Label}}"
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
                <input class="control-input" type="text"
                       value="{{.Value}}" placeholder="{{.Value}}"
                       oninput="tfCSSVar('{{$slug}}', '{{.Name}}', this.value)">
                {{end}}
              </div>
              {{end}}
              {{if and (not .Bindings) (not .CSSVars)}}
              <p class="dock-hint">No published controls in this element.</p>
              {{end}}
            </div>
          </div>
          {{end}}
        {{else}}
        <p class="dock-hint">Select a pack to see controls.</p>
        {{end}}
      </div>
    </div>
  </div>

</div><!-- /workspace -->

<!-- ══ New Element Modal ══ -->
<div class="modal-overlay" id="modal-new-element">
  <div class="modal">
    <h3>New Element</h3>
    <div class="modal-hint">Slug must be lowercase kebab-case, e.g. <code style="font-family:var(--mono)">lower-third</code></div>
    <input type="text" id="new-element-slug" placeholder="element-slug" oninput="validateSlugInput(this)">
    <div class="modal-actions">
      <button class="btn" onclick="closeModal('modal-new-element')">Cancel</button>
      <button class="btn btn-primary" onclick="createElement()">Create</button>
    </div>
  </div>
</div>

<!-- ══ New Pack Modal ══ -->
<div class="modal-overlay" id="modal-new-pack">
  <div class="modal">
    <h3>New Pack</h3>
    <div class="modal-hint">Pack name becomes the folder name in the packs directory.</div>
    <input type="text" id="new-pack-name" placeholder="my-pack">
    <div class="modal-actions">
      <button class="btn" onclick="closeModal('modal-new-pack')">Cancel</button>
      <button class="btn btn-primary" onclick="createPack()">Create</button>
    </div>
  </div>
</div>

<script>
// ─── State ────────────────────────────────────────────────────────────────────
var currentPack = {{if .CurrentPack}}"{{.CurrentPack.Name}}"{{else}}""{{end}};
var currentElem = {{if .CurrentElem}}"{{.CurrentElem.Slug}}"{{else}}""{{end}};
var hasJS = {{if .CurrentElem}}{{if .CurrentElem.HasJS}}true{{else}}false{{end}}{{else}}false{{end}};

// ─── Output log ───────────────────────────────────────────────────────────────
function appendLog(level, msg) {
  var log = document.getElementById('output-log');
  var empty = log.querySelector('.log-empty');
  if (empty) empty.remove();
  var now = new Date();
  var ts = now.getHours().toString().padStart(2,'0') + ':' +
           now.getMinutes().toString().padStart(2,'0') + ':' +
           now.getSeconds().toString().padStart(2,'0');
  var entry = document.createElement('div');
  entry.className = 'log-entry';
  entry.innerHTML = '<span class="log-ts">' + ts + '</span>' +
                    '<span class="log-' + level + '">' + esc(msg) + '</span>';
  log.appendChild(entry);
  log.scrollTop = log.scrollHeight;
}

function clearLog() {
  var log = document.getElementById('output-log');
  log.innerHTML = '<div class="log-empty">Log cleared.</div>';
}

// ─── Navigation ───────────────────────────────────────────────────────────────
function switchPack(name) {
  var encoded = encodeURIComponent(name);
  fetch('/api/set-active', {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'pack=' + encoded
  }).finally(function() {
    window.location.href = '/?pack=' + encoded;
  });
}

function selectElement(slug) {
  window.location.href = '/?pack=' + encodeURIComponent(currentPack) +
                         '&element=' + encodeURIComponent(slug);
}

function reloadPacks() {
  fetch('/api/reload', { method: 'POST' }).then(function(r) {
    if (r.ok) { window.location.reload(); }
  });
}

// ─── Code panel helpers ───────────────────────────────────────────────────────
function copyPanel(name) {
  var ta = document.getElementById('code-' + name);
  navigator.clipboard.writeText(ta.value).then(function() {
    appendLog('info', 'Copied ' + name.toUpperCase() + ' to clipboard.');
  });
}

function toggleJS(checked) { hasJS = checked; }

// ─── Save ─────────────────────────────────────────────────────────────────────
function saveElement() {
  if (!currentPack || !currentElem) {
    appendLog('err', 'No element selected.');
    return;
  }
  var data = new URLSearchParams();
  data.append('pack',   currentPack);
  data.append('element', currentElem);
  data.append('html',   document.getElementById('code-html').value);
  data.append('css',    document.getElementById('code-css').value);
  data.append('js',     document.getElementById('code-js').value);
  data.append('has_js', hasJS ? 'true' : 'false');

  fetch('/api/save', { method: 'POST', body: data })
    .then(function(r) { return r.json(); })
    .then(function(j) {
      if (j.ok) {
        appendLog('ok', 'Saved \u2014 ' + currentElem);
        // Soft reload to refresh server-rendered dock controls with any new bindings.
        window.location.href = '/?pack=' + encodeURIComponent(currentPack) +
                               '&element=' + encodeURIComponent(currentElem);
      } else {
        appendLog('err', 'Save failed.');
      }
    })
    .catch(function(e) { appendLog('err', 'Save error: ' + e); });
}

// ─── Validate ────────────────────────────────────────────────────────────────
function validatePack() {
  if (!currentPack) { appendLog('err', 'No pack selected.'); return; }
  appendLog('info', 'Validating \u201c' + currentPack + '\u201d\u2026');
  fetch('/api/validate?pack=' + encodeURIComponent(currentPack))
    .then(function(r) { return r.json(); })
    .then(function(j) {
      if (j.ok) {
        appendLog('ok', 'Pack valid \u2014 no errors.');
      }
      j.errors.forEach(function(e)   { appendLog('err',  e); });
      j.warnings.forEach(function(w) { appendLog('warn', w); });
    })
    .catch(function(e) { appendLog('err', 'Validate error: ' + e); });
}

// ─── Dock controls → Runtime :8080 ───────────────────────────────────────────
function tfFire(slug, event) {
  fetch('http://localhost:8080/api/fire?element=' + encodeURIComponent(slug) +
        '&event=' + encodeURIComponent(event), { method: 'POST' })
    .catch(function() { appendLog('err', 'Runtime unreachable. Is TitleFlow running?'); });
}

function tfReset(slug) {
  fetch('http://localhost:8080/api/reset?element=' + encodeURIComponent(slug),
        { method: 'POST' })
    .catch(function() { appendLog('err', 'Runtime unreachable.'); });
}

function tfBind(slug, key, value) {
  fetch('http://localhost:8080/api/bind?element=' + encodeURIComponent(slug) +
        '&key='   + encodeURIComponent(key) +
        '&value=' + encodeURIComponent(value), { method: 'POST' });
}

function tfCSSVar(slug, name, value) {
  fetch('http://localhost:8080/api/cssvar?element=' + encodeURIComponent(slug) +
        '&name='  + encodeURIComponent(name) +
        '&value=' + encodeURIComponent(value), { method: 'POST' });
}

function tfCSSVarDuration(slug, name, value, unit) {
  tfCSSVar(slug, name, value + unit);
}

function tfNumberWheel(e, slug, name, unit) {
  e.preventDefault();
  var input = e.currentTarget;
  var step  = parseFloat(input.step) || 1;
  var delta = e.deltaY < 0 ? step : -step;
  var newVal = (parseFloat(input.value) || 0) + delta;
  input.value = newVal;
  tfCSSVar(slug, name, newVal + unit);
}

function tfToggle(header) {
  header.closest('.element-section').classList.toggle('collapsed');
}

// ─── Inspector ───────────────────────────────────────────────────────────────
function loadInspector() {
  if (!currentPack || !currentElem) return;
  fetch('/api/controls?pack=' + encodeURIComponent(currentPack) +
        '&element=' + encodeURIComponent(currentElem))
    .then(function(r) { return r.json(); })
    .then(function(j) { renderInspector(j); })
    .catch(function() {});
}

function renderInspector(j) {
  var body = document.getElementById('inspector-body');
  if (!j.bindings.length && !j.css_vars.length) {
    body.innerHTML = '<div class="inspector-empty">No explicitly published controls found.' +
      '<br><span style="font-size:11px;">Add data-tf-bind to HTML, or --tf-ui-* to CSS.</span></div>';
    return;
  }
  var html = '';
  if (j.bindings.length) {
    html += '<div class="inspector-group"><div class="inspector-group-label">HTML Bindings</div>';
    j.bindings.forEach(function(b) {
      html += '<div class="inspector-item">';
      html += '<span class="inspector-badge badge-html">HTML</span>';
      html += '<span class="key">' + esc(b.key) + '</span>';
      html += '<span class="lbl">' + esc(b.label) + '</span>';
      if (b.default_value) {
        html += '<span style="color:var(--muted);font-size:11px;">&ldquo;' +
                esc(b.default_value.substring(0,30)) + '&rdquo;</span>';
      }
      html += '</div>';
    });
    html += '</div>';
  }
  if (j.css_vars.length) {
    html += '<div class="inspector-group"><div class="inspector-group-label">CSS Variables</div>';
    j.css_vars.forEach(function(v) {
      var bc = 'badge-css';
      if (v.inferred_type === 'color')    bc = 'badge-color';
      if (v.inferred_type === 'length')   bc = 'badge-length';
      if (v.inferred_type === 'duration') bc = 'badge-duration';
      html += '<div class="inspector-item">';
      html += '<span class="inspector-badge ' + bc + '">' + esc(v.inferred_type) + '</span>';
      html += '<span class="key" style="font-size:10px;">' + esc(v.name) + '</span>';
      html += '<span class="lbl">' + esc(v.label) + '</span>';
      html += '<span style="color:var(--muted);font-size:11px;">' + esc(v.value) + '</span>';
      html += '</div>';
    });
    html += '</div>';
  }
  body.innerHTML = html;
}

// ─── Modals ───────────────────────────────────────────────────────────────────
function openNewElementModal() {
  document.getElementById('new-element-slug').value = '';
  document.getElementById('modal-new-element').classList.add('open');
  setTimeout(function() { document.getElementById('new-element-slug').focus(); }, 50);
}

function openNewPackModal() {
  document.getElementById('new-pack-name').value = '';
  document.getElementById('modal-new-pack').classList.add('open');
  setTimeout(function() { document.getElementById('new-pack-name').focus(); }, 50);
}

function closeModal(id) {
  document.getElementById(id).classList.remove('open');
}

document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') {
    document.querySelectorAll('.modal-overlay.open').forEach(function(m) {
      m.classList.remove('open');
    });
  }
});

function validateSlugInput(input) {
  var ok = /^[a-z][a-z0-9]*(-[a-z0-9]+)*$/.test(input.value);
  input.style.borderColor = input.value === '' ? '' : (ok ? 'var(--accent2)' : 'var(--danger)');
}

function createElement() {
  var slug = document.getElementById('new-element-slug').value.trim();
  if (!slug || !/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/.test(slug)) {
    alert('Invalid slug. Use lowercase kebab-case, e.g. lower-third');
    return;
  }
  var data = new URLSearchParams();
  data.append('pack', currentPack);
  data.append('slug', slug);
  fetch('/api/new-element', { method: 'POST', body: data })
    .then(function(r) {
      if (!r.ok) return r.text().then(function(t) { throw t; });
      return r.json();
    })
    .then(function() {
      closeModal('modal-new-element');
      window.location.href = '/?pack=' + encodeURIComponent(currentPack) +
                             '&element=' + encodeURIComponent(slug);
    })
    .catch(function(e) { alert('Error: ' + e); });
}

function createPack() {
  var name = document.getElementById('new-pack-name').value.trim();
  if (!name) { alert('Pack name is required.'); return; }
  var data = new URLSearchParams();
  data.append('name', name);
  fetch('/api/new-pack', { method: 'POST', body: data })
    .then(function(r) {
      if (!r.ok) return r.text().then(function(t) { throw t; });
      return r.json();
    })
    .then(function() {
      closeModal('modal-new-pack');
      window.location.href = '/?pack=' + encodeURIComponent(name);
    })
    .catch(function(e) { alert('Error: ' + e); });
}

// ─── Helpers ─────────────────────────────────────────────────────────────────
function esc(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;')
    .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

// ─── Splitter engine ─────────────────────────────────────────────────────────
function initSplitters() {
  document.querySelectorAll('.splitter').forEach(function(sp) {
    sp.addEventListener('mousedown', function(e) {
      e.preventDefault();
      var isV    = sp.classList.contains('v');
      var prop   = sp.dataset.prop;
      var invert = sp.dataset.invert === 'true';
      var startPos = isV ? e.clientX : e.clientY;

      // Read the current CSS custom property value.
      var currentVal = parseFloat(
        getComputedStyle(document.documentElement).getPropertyValue(prop)
      );
      if (isNaN(currentVal)) {
        // Fallback: measure the adjacent sibling element.
        var sib = invert ? sp.nextElementSibling : sp.previousElementSibling;
        var rect = sib.getBoundingClientRect();
        currentVal = isV ? rect.width : rect.height;
      }

      sp.classList.add('active');
      document.body.style.cursor    = isV ? 'col-resize' : 'row-resize';
      document.body.style.userSelect = 'none';

      // Transparent overlay prevents any child iframe / element from stealing mouse events.
      var shield = document.createElement('div');
      shield.style.cssText =
        'position:fixed;inset:0;z-index:9999;cursor:' +
        (isV ? 'col-resize' : 'row-resize') + ';';
      document.body.appendChild(shield);

      function onMove(ev) {
        var delta = (isV ? ev.clientX : ev.clientY) - startPos;
        if (invert) delta = -delta;
        var newSize = Math.max(120, currentVal + delta);
        document.documentElement.style.setProperty(prop, newSize + 'px');
      }

      function onUp() {
        sp.classList.remove('active');
        document.body.style.cursor    = '';
        document.body.style.userSelect = '';
        shield.remove();
        document.removeEventListener('mousemove', onMove);
        document.removeEventListener('mouseup',   onUp);
      }

      document.addEventListener('mousemove', onMove);
      document.addEventListener('mouseup',   onUp);
    });
  });
}

// ─── Keyboard shortcuts ───────────────────────────────────────────────────────
document.addEventListener('keydown', function(e) {
  if ((e.ctrlKey || e.metaKey) && e.key === 's') {
    e.preventDefault();
    saveElement();
  }
});

// ─── Init ────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', function() {
  initSplitters();
  if (currentElem) loadInspector();

  // Tab key in textareas inserts 2 spaces instead of leaving the field.
  document.querySelectorAll('textarea.code').forEach(function(ta) {
    ta.addEventListener('keydown', function(e) {
      if (e.key === 'Tab') {
        e.preventDefault();
        var s = ta.selectionStart, en = ta.selectionEnd;
        ta.value = ta.value.substring(0, s) + '  ' + ta.value.substring(en);
        ta.selectionStart = ta.selectionEnd = s + 2;
      }
    });
  });
});
</script>
</body>
</html>
`
