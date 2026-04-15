# TitleFlow API Reference

Version: v0.4.4b  
Last Updated: 2026-04-15

This document is the complete HTTP API reference for TitleFlow. The system exposes three independent HTTP servers on different ports, each serving a distinct purpose.

---

## Overview

TitleFlow runs three concurrent servers:

| Server | Port | Purpose | Audience |
|--------|------|---------|----------|
| **Runtime** | 8080 | Serves composed overlay to OBS browser source | OBS operators |
| **Editor** | 8081 | Live code editor with preview and validation | Pack authors |
| **Dock** | 8082 | Control surface for active pack | OBS operators |

**CORS:** All `/api/*` endpoints on the Runtime and Editor servers return `Access-Control-Allow-Origin: *` to allow cross-port requests from the dock.

---

## Runtime Server (Port 8080)

The runtime server composes and serves the active pack. It also broadcasts real-time control updates via Server-Sent Events (SSE).

### `GET /`

Serves the composed runtime HTML document or static assets from the active pack.

**Parameters:**
- `path` (optional, query): Relative path to asset within pack root (e.g., `?path=assets/logo.png` or `?path=lower-third/assets/logo.png`). If omitted, returns composed HTML.

**Response:**
- `Content-Type: text/html` — Composed HTML document (if no `path` param)
- `Content-Type: */*` — Static file content (if `path` param provided)
- `Cache-Control: no-store` — Disables caching
- HTTP 503 — If no active pack

**Asset paths** — Assets can be stored at pack level or element level. Paths always resolve from the pack root:
- Pack-level: `packs/<pack>/assets/logo.png` → `GET /?path=assets/logo.png`
- Element-local: `packs/<pack>/<slug>/assets/logo.png` → `GET /?path=<slug>/assets/logo.png`

**Example:**
```bash
# Get composed runtime HTML
curl http://localhost:8080/

# Get asset from pack-level assets folder
curl "http://localhost:8080/?path=assets/logo.png"

# Get asset from element-local assets folder
curl "http://localhost:8080/?path=lower-third/assets/logo.png"

# (Also supported) Path-based asset URL
curl "http://localhost:8080/assets/logo.png"
curl "http://localhost:8080/lower-third/assets/logo.png"
```

**Notes:**
- Composition happens at startup and on every editor save
- All elements are composed in alphabetical slug order
- Each element is wrapped in `<div id="tf-element-<slug>" class="tf-element" data-tf-slug="<slug>">`
- CSS and `--tf-ui-*` defaults are injected at `:root`
- A hidden `<script>` block initializes SSE listeners and fire event handlers

---

### `GET /events`

Opens a Server-Sent Events (SSE) stream for real-time control updates and broadcast events.

**Parameters:** None

**Response:**
- `Content-Type: text/event-stream` — SSE stream
- Events are streamed as they occur (fire, reset, bind, cssvar, reload-page)

**SSE Event Types:**

#### `fire`
A named fire event was triggered from the dock.
```
event: fire
data: {"element":"lower-third","event":"show"}
```

#### `reset`
The reset button was clicked; element should clear `.tf-fired` immediately.
```
event: reset
data: {"element":"lower-third"}
```

#### `bind`
HTML binding text was updated in the dock.
```
event: bind
data: {"element":"lower-third","key":"name","value":"John Doe"}
```

#### `cssvar`
CSS variable was updated in the dock.
```
event: cssvar
data: {"element":"lower-third","name":"--tf-ui-accent-color","value":"#ff0000"}
```

#### `reload-page`
Pack was switched or packs were reloaded; page should reload to get fresh content.
```
event: reload-page
data: {}
```

**Example (JavaScript client):**
```javascript
const es = new EventSource('http://localhost:8080/events');
es.addEventListener('fire', (e) => {
  const { element, event } = JSON.parse(e.data);
  console.log(`Fire: ${element} → ${event}`);
});
es.addEventListener('bind', (e) => {
  const { element, key, value } = JSON.parse(e.data);
  console.log(`Bind: ${element}.${key} = ${value}`);
});
es.addEventListener('reload-page', () => {
  location.reload();
});
```

---

### `GET /api/state`

Returns the current live control values (HTML bindings and CSS variable overrides).

**Parameters:** None

**Response:**
```json
{
  "binds": {
    "lower-third": {
      "name": "John Doe",
      "title": "Breaking News"
    }
  },
  "css_vars": {
    "lower-third": {
      "--tf-ui-accent-color": "#ff6b35",
      "--tf-ui-show-duration": "2800ms"
    }
  }
}
```

**Example:**
```bash
curl http://localhost:8080/api/state
```

**Notes:**
- Values are stored in memory during the session
- Reset to defaults on application restart
- Used by the runtime page on load to restore previous session state

---

### `POST /api/fire`

Triggers a named fire event for an element.

**Parameters (form-encoded):**
- `element` (required): Namespaced element identifier in format `pack--element` (e.g., `my-pack--lower-third`)
- `event` (required): Event name (e.g., `show`, `hide`, or custom name)

**Response:**
- HTTP 204 No Content — Success
- HTTP 400 — Missing parameters
- HTTP 503 — No active pack selected

**Example:**
```bash
curl -X POST http://localhost:8080/api/fire \
  -d "element=my-pack--lower-third&event=show"
```

**Runtime Behavior:**
- Event is broadcast via SSE to all connected clients
- All clients receive `tf:fire` CustomEvent with `{element, event}` detail
- **Built-in behavior for `show`:** Element is removed and reinserted in DOM (restarts animations), then `.tf-fired` class is added
- **Built-in behavior for `hide`:** In-flight CSS animations are cancelled (reflow forced), `tf:fire` CustomEvent is dispatched, `.tf-hiding` is added, and after `--tf-ui-hide-duration` both `.tf-fired` and `.tf-hiding` are removed. Element.js can call `preventDefault()` on the `tf:fire` event to suppress this default and own the full exit lifecycle.
- **Custom events:** Only CustomEvent is dispatched; element.js implements behavior

---

### `POST /api/reset`

Hard-cuts an element by removing the `.tf-fired` class immediately (no animation).

**Parameters (form-encoded):**
- `element` (required): Namespaced element identifier in format `pack--element` (e.g., `my-pack--lower-third`)

**Response:**
- HTTP 204 No Content — Success
- HTTP 400 — Missing parameter
- HTTP 503 — No active pack selected

**Example:**
```bash
curl -X POST http://localhost:8080/api/reset \
  -d "element=my-pack--lower-third"
```

**Notes:**
- No `tf:fire` CustomEvent is dispatched (unlike `/api/fire`)
- Immediately removes `.tf-fired` class (no animation)
- Used as a safety fallback if animation gets stuck
- Always available; always present as a button in dock header

---

### `POST /api/bind`

Updates an HTML binding value and broadcasts it to all clients.

**Parameters (form-encoded):**
- `element` (required): Namespaced element identifier in format `pack--element` (e.g., `my-pack--lower-third`)
- `key` (required): Binding key (e.g., `name`)
- `value` (required): New text value

**Response:**
- HTTP 204 No Content — Success
- HTTP 400 — Missing parameters
- HTTP 503 — No active pack selected

**Example:**
```bash
curl -X POST http://localhost:8080/api/bind \
  -d "element=my-pack--lower-third&key=name&value=Jane%20Smith"
```

**Runtime Behavior:**
- Value is stored in ControlState (in-memory)
- SSE `bind` event is broadcast to all connected clients
- Clients update `[data-tf-bind="key"]` element's textContent
- Value persists for session duration; reset on app restart

---

### `POST /api/cssvar`

Updates a CSS variable override and broadcasts it to all clients.

**Parameters (form-encoded):**
- `element` (required): Namespaced element identifier in format `pack--element` (e.g., `my-pack--lower-third`)
- `name` (required): CSS variable name including `--` prefix (e.g., `--tf-ui-accent-color`)
- `value` (required): New CSS value (e.g., `#ff0000`)

**Response:**
- HTTP 204 No Content — Success
- HTTP 400 — Missing parameters
- HTTP 503 — No active pack selected

**Example:**
```bash
curl -X POST http://localhost:8080/api/cssvar \
  -d "element=my-pack--lower-third&name=--tf-ui-accent-color&value=%23ff0000"
```

**Runtime Behavior:**
- Value is stored in ControlState (in-memory) under namespaced element key
- SSE `cssvar` event is broadcast to all connected clients
- Clients inject a `<style id="tf-override-<pack>--<element>">` block with ID-level specificity (overrides pack-level rules)
- Example injected CSS: `#tf-element-lower-third, #tf-element-lower-third * { --tf-ui-accent-color: #ff0000; }`
- Value persists for session duration; reset on app restart

---

## Editor Server (Port 8081)

The editor provides a live development environment for creating and editing packs.

### `GET /`

Renders the editor UI.

**Parameters:**
- `pack` (optional, query): Pack name to load (e.g., `?pack=news-pack`)
- `element` (optional, query): Element slug to load (e.g., `?element=lower-third`)

**Response:**
- HTTP 200 — HTML editor page with three-panel UI (HTML, CSS, JS)

**Example:**
```bash
curl http://localhost:8081/?pack=news-pack&element=lower-third
```

---

### `GET /api/preview`

Recomposes and returns the full runtime HTML for the active pack.

**Parameters:**
- `pack` (required, query): Pack name (e.g., `?pack=news-pack`)

**Response:**
- HTTP 200 — Composed HTML (same as runtime server)
- HTTP 400 — Pack not found

**Example:**
```bash
curl "http://localhost:8081/api/preview?pack=news-pack"
```

**Notes:**
- Uses identical composition logic as runtime server
- Embedded in preview iframe within editor UI
- Used for real-time visual feedback as you edit

---

### `POST /api/save`

Saves element source files and triggers pack reload.

**Parameters (form-encoded):**
- `pack` (required): Pack name
- `element` (required): Element slug
- `html` (required): HTML content
- `css` (required): CSS content
- `js` (required): JavaScript content (can be empty string)
- `has_js` (required): `"true"` or `"false"` — whether JS file should be written

**Response:**
- HTTP 200 — `{"ok":true}`
- HTTP 400 — Missing parameters or invalid slug

**Example:**
```bash
curl -X POST http://localhost:8081/api/save \
  -d "pack=news-pack&element=lower-third&html=<h1>Test</h1>&css=h1{color:red;}&js=&has_js=false"
```

**Side Effects:**
- Writes `element.html`, `element.css`, and optionally `element.js`
- Triggers pack reload (`ReloadPacks()`) to update dock and runtime
- Resets ControlState (live values) to defaults

---

### `POST /api/new-element`

Creates a new element folder with default files.

**Parameters (form-encoded):**
- `pack` (required): Pack name (must exist)
- `slug` (required): Element slug (must be valid lowercase kebab-case)

**Response:**
- HTTP 200 — `{"ok":true}`
- HTTP 400 — Invalid parameters or slug

**Example:**
```bash
curl -X POST http://localhost:8081/api/new-element \
  -d "pack=news-pack&slug=my-element"
```

**Side Effects:**
- Creates folder `packs/<pack>/<slug>/`
- Creates default `element.html` with wrapper structure
- Creates default `element.css` with basic styling
- Triggers pack reload

---

### `POST /api/new-pack`

Creates a new pack directory.

**Parameters (form-encoded):**
- `name` (required): Pack name (must be valid folder name)

**Response:**
- HTTP 200 — `{"ok":true}`
- HTTP 400 — Invalid name

**Example:**
```bash
curl -X POST http://localhost:8081/api/new-pack \
  -d "name=my-pack"
```

**Side Effects:**
- Creates folder `packs/<name>/`
- Triggers pack reload to add it to dock switcher

---

### `POST /api/reload`

Rescans the packs directory and reloads state from the editor. Use this to refresh the editor's view of on-disk pack changes.

**Parameters:** None

**Response:**
- HTTP 204 No Content

**Example:**
```bash
curl -X POST http://localhost:8081/api/reload
```

**Side Effects:**
- Rescans `packs/` directory from disk
- Broadcasts `reload-page` SSE event to all runtime clients

---

### `GET /api/validate`

Runs comprehensive validation on a pack.

**Parameters:**
- `pack` (required, query): Pack name (e.g., `?pack=news-pack`)

**Response:**
```json
{
  "ok": false,
  "errors": [
    "pack 'news-pack', element 'lower-third', file 'element.html': editable binding 'name' is missing required data-tf-label."
  ],
  "warnings": [
    "pack 'news-pack', element 'lower-third', file 'element.css': broad selector 'body' may assume page ownership."
  ]
}
```

**Example:**
```bash
curl "http://localhost:8081/api/validate?pack=news-pack"
```

**Validation Rules:**

*Errors (must fix):*
- Pack contains no elements
- Invalid slug format (not lowercase kebab-case)
- Duplicate element slugs within pack
- Missing `element.html` or `element.css`
- HTML bindings missing `data-tf-bind` or `data-tf-label`
- Duplicate binding keys within element
- Invalid `--tf-ui-*` variable name format

*Warnings (for review):*
- Broad selectors (`html`, `body`, `*`) in CSS

---

### `GET /api/controls`

Returns derived dock controls for an element (HTML bindings + CSS variables).

**Parameters:**
- `pack` (required, query): Pack name (e.g., `?pack=news-pack`)
- `element` (required, query): Element slug (e.g., `?element=lower-third`)

**Response:**
```json
{
  "bindings": [
    {
      "key": "name",
      "label": "Name",
      "defaultValue": "Default Name"
    }
  ],
  "css_vars": [
    {
      "name": "--tf-ui-accent-color",
      "label": "Accent Color",
      "defaultValue": "#58a6ff",
      "type": "color"
    },
    {
      "name": "--tf-ui-show-duration",
      "label": "Show Duration",
      "defaultValue": "2800ms",
      "type": "duration"
    }
  ]
}
```

**Example:**
```bash
curl "http://localhost:8081/api/controls?pack=news-pack&element=lower-third"
```

**Type Inference:**
- `color` — Values starting with `#`, `rgb`, `hsl`
- `duration` — Values ending with `ms` or `s`
- `length` — Values ending with `px`, `em`, `rem`, `%`, `vw`, `vh`, `pt`, `ch`
- `text` — All other values

---

## Dock Server (Port 8082)

The dock is a minimal control surface for switching packs and triggering published controls.

### `GET /`

Renders the dock UI.

**Parameters:** None

**Response:**
- HTTP 200 — HTML dock page with pack switcher and element controls

**Example:**
```bash
curl http://localhost:8082/
```

---

### `POST /api/set-active`

Sets the active pack and reloads runtime pages.

**Parameters (form-encoded):**
- `pack` (required): Pack name (e.g., `news-pack`)

**Response:**
- HTTP 303 Redirect to `/`

**Example:**
```bash
curl -X POST http://localhost:8082/api/set-active \
  -d "pack=news-pack"
```

**Side Effects:**
- Sets active pack in appstate
- Broadcasts `reload-page` SSE event to all runtime clients
- All OBS browser sources will reload to show new pack

---

### `POST /api/reload`

Rescans the packs directory and reloads state.

**Parameters:** None

**Response:**
- HTTP 303 Redirect to `/`

**Example:**
```bash
curl -X POST http://localhost:8082/api/reload
```

**Side Effects:**
- Rescans `packs/` directory from disk
- Updates pack list and elements
- Broadcasts `reload-page` SSE event to all runtime clients
- Resets ControlState (live values) to defaults

---

## Error Responses

All endpoints may return these HTTP status codes:

| Code | Meaning | Example |
|------|---------|---------|
| 200 | OK | Successful request |
| 204 | No Content | Fire/reset/bind/cssvar success (no body) |
| 303 | Redirect | Pack switch or reload (redirect to `/`) |
| 400 | Bad Request | Missing or invalid parameters |
| 404 | Not Found | Pack/element not found |
| 503 | Service Unavailable | No active pack on runtime |

Error responses typically include a plain-text message body explaining the issue.

---

## Security & Isolation

- **CORS:** All `/api/*` endpoints return permissive CORS headers for cross-port access
- **Port Isolation:** Three separate servers; firewall can restrict individual ports
- **Event Filtering:** element.js **must filter by `e.detail.element`** to avoid cross-element interference
- **No Authentication:** TitleFlow is designed for local networks; use firewall rules to restrict access

---

## Common Workflows

### Adding Text to Lower-Third
```bash
curl -X POST http://localhost:8080/api/bind \
  -d "element=my-pack--lower-third&key=name&value=Breaking%20News"
```

### Changing Accent Color
```bash
curl -X POST http://localhost:8080/api/cssvar \
  -d "element=my-pack--lower-third&name=--tf-ui-accent-color&value=%23ff0000"
```

### Triggering Show Animation
```bash
curl -X POST http://localhost:8080/api/fire \
  -d "element=my-pack--lower-third&event=show"
```

### Switching to Different Pack
```bash
curl -X POST http://localhost:8082/api/set-active \
  -d "pack=sports-pack"
```

### Validating a Pack Before Deployment
```bash
curl "http://localhost:8081/api/validate?pack=news-pack"
```

---

## Related Documentation

- **[PACK_SPEC.md](PACK_SPEC.md)** — Authoring guide for pack creators
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — System design and package reference
- **[README.md](README.md)** — Getting started guide
