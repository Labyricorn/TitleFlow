# TitleFlow Architecture

Version: v0.4.4b  
Last Updated: 2026-04-15

This document describes the internal architecture of TitleFlow for developers and contributors. It covers system design, package roles, data flow, and key concepts.

---

## System Overview

TitleFlow is a **single-binary, three-server overlay system** with clean separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                        TitleFlow                            │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │  Editor Server   │  │  Dock Server     │               │
│  │  Port 8081       │  │  Port 8082       │               │
│  │                  │  │                  │               │
│  │  • Code panels   │  │  • Pack switcher │               │
│  │  • Live preview  │  │  • Control UI    │               │
│  │  • Validation    │  │  • Reload trigger│               │
│  │  • Reload trigger│  │                  │               │
│  └─────────┬────────┘  └─────────┬────────┘               │
│            │                     │                         │
│            └─────────┬───────────┘                         │
│                      │                                     │
│            ┌─────────▼──────────┐                         │
│            │  Shared AppState   │                         │
│            │  • Pack list       │                         │
│            │  • Active pack     │                         │
│            │  • Live values     │                         │
│            └─────────┬──────────┘                         │
│                      │                                     │
│            ┌─────────▼──────────┐                         │
│            │ Runtime Server     │                         │
│            │ Port 8080          │                         │
│            │                    │                         │
│            │ • Composer         │                         │
│            │ • SSE hub          │                         │
│            │ • OBS browser src  │                         │
│            └────────────────────┘                         │
│                      ▲                                     │
│                      │ SSE stream                         │
│                      │                                     │
│         ┌────────────┴────────────┐                       │
│         │      OBS Browser        │                       │
│         │      Source             │                       │
│         │  http://localhost:8080  │                       │
│         └─────────────────────────┘                       │
│                                                           │
└─────────────────────────────────────────────────────────────┘
```

**Key insight:** All three servers share a single `appstate.AppState` instance for pack discovery and active pack selection. The runtime server is stateless (composition happens on every request); editors and dock trigger reloads via API calls.

---

## Package Map

### `internal/models` — Data Structures

Defines all shared types used across the system.

**Key structs:**

```go
type Pack struct {
  Name     string      // folder name (e.g., "news-pack")
  Elements []Element   // all elements in pack
}

type Element struct {
  Slug         string        // lowercase kebab-case (e.g., "lower-third")
  HTMLSource   string        // element.html content
  CSSSource    string        // element.css content
  JSSource     string        // element.js content (optional)
  HTMLBindings []HTMLBinding // parsed data-tf-bind controls
  CSSVars      []CSSVar      // parsed --tf-ui-* variables
  FireEvents   []FireEvent   // parsed data-tf-fire buttons
}

type HTMLBinding struct {
  Key   string // data-tf-bind value
  Label string // data-tf-label value
}

type CSSVar struct {
  Name  string // full name with -- prefix (e.g., "--tf-ui-accent-color")
  Value string // default value (e.g., "#58a6ff")
  Type  string // inferred type: "color", "duration", "length", "text"
}

type FireEvent struct {
  Name  string // data-tf-fire value (e.g., "show", "hide", "break")
  Label string // data-tf-label value
}

type ValidationResult struct {
  OK       bool     // true if all errors are empty
  Errors   []string // validation failures (must fix)
  Warnings []string // cautions (for review)
}
```

---

### `internal/scanner` — Pack Discovery & Control Extraction

Discovers packs from the filesystem and parses control declarations from source files.

**Responsibilities:**
- Walk `packs/` directory and validate structure
- Identify valid pack roots (immediate subdirectories)
- Scan each pack for element directories
- Parse HTML bindings from `element.html` (regex + DOM parsing)
- Parse CSS variables from `element.css` (regex)
- Parse fire events from `element.html` (DOM parsing)
- Validate slug format (lowercase kebab-case)

**Key functions:**

```go
func ScanPackDir(packsDir string) ([]*models.Pack, error)
  // Returns all valid packs in the directory

func ScanPack(packRoot string) (*models.Pack, error)
  // Scans a single pack folder and returns Pack with all elements

func parseHTMLBindings(html string) []models.HTMLBinding
  // Uses golang.org/x/net/html to parse and extract data-tf-bind elements

func parseFireEvents(html string) []models.FireEvent
  // Uses golang.org/x/net/html to extract data-tf-fire elements

func parseCSSVars(css string) []models.CSSVar
  // Uses regex to extract --tf-ui-* and infers type from value

func inferCSSVarType(value string) string
  // Heuristic: checks for color (#/rgb/hsl), duration (ms/s), length (px/em/%), else text
```

**Execution:** Runs once at startup and on every editor save or dock reload.

---

### `internal/appstate` — Shared Application State

Manages pack discovery, active pack selection, and live control values.

**Responsibilities:**
- Hold the authoritative pack list
- Track which pack is currently active
- Store live control values (HTML bindings + CSS var overrides)
- Thread-safe read/write access via `sync.RWMutex`
- Reload packs on demand

**Key structures:**

```go
type AppState struct {
  packs        []*models.Pack        // all discovered packs
  activePack   *models.Pack          // currently active (may be nil)
  controls     *ControlState         // live values
  packsDir     string                // path to packs directory
  mu            sync.RWMutex         // guards access
}

type ControlState struct {
  Bindings map[string]map[string]string    // ["pack--element"][key] = value
  CSSVars  map[string]map[string]string    // ["pack--element"][varname] = value
  mu        sync.RWMutex
}
```

**Key methods:**

```go
func New(packsDir string) (*AppState, error)
  // Initialize: scan packs, set first as active

func (s *AppState) ActivePack() *models.Pack
  // Thread-safe read of active pack

func (s *AppState) SetActivePack(packName string)
  // Thread-safe write; validates pack exists

func (s *AppState) ReloadPacks() error
  // Rescan filesystem; re-link active pack; reset controls

func (s *AppState) SetBinding(element, key, value string)
  // Store HTML binding value

func (s *AppState) SetCSSVar(element, name, value string)
  // Store CSS variable override

func (s *AppState) GetControlState() *ControlState
  // Read live values for /api/state endpoint
```

---

### `internal/composer` — Composition Logic

Builds the final composed HTML document from a pack. This is the **single source of composition truth** — both runtime and editor use it.

**Responsibilities:**
- Take a Pack and produce final HTML
- Wrap each element in `<div id="tf-element-<slug>" class="tf-element" data-tf-pack="<pack>" data-tf-slug="<slug>">`
- Inject CSS variable defaults at `:root`
- Include SSE client and fire event listeners
- Sort elements alphabetically by slug

**Key function:**

```go
func Compose(pack *models.Pack) string
  // Returns HTML document ready to send to browser
```

**Composition algorithm:**

1. Start with HTML boilerplate (`<html><head>...<body>`)
2. Inject `<style>` with `:root { --tf-ui-*: defaults; }`
3. For each element (sorted by slug):
   - Wrap `element.html` in `<div id="tf-element-<slug>" class="tf-element" data-tf-pack="<pack>" data-tf-slug="<slug>">` (DOM IDs are slug-only for CSS compatibility)
   - Inject `<style data-tf-element="<slug>">` with element CSS
4. Inject hidden `<script>` with:
   - SSE listener that opens `/events` and handles fire/bind/cssvar/reset broadcasts
   - `tf:fire` CustomEvent dispatcher (extracts slug from `"pack--element"` SSE event; dispatches with slug only so element.js handlers work without modification)
   - Control state restoration from live values (uses namespaced element keys for lookup, slug for DOM access)
5. Close HTML

**Key insight:** Composition is deterministic and stateless — same Pack + ControlState always produces identical HTML.

---

### `internal/runtime` — OBS Runtime Server

HTTP server on port 8080. Serves composed overlays to OBS and manages SSE broadcasting.

**Responsibilities:**
- HTTP handler for `/`, `/events`, `/api/state`, `/api/fire`, `/api/reset`, `/api/bind`, `/api/cssvar`
- Manage active pack (delegates to appstate)
- Compose and serve HTML (delegates to composer)
- Broadcast real-time control updates via SSE (delegates to hub)
- Serve static assets from active pack

**Key handlers:**

```go
func activePackHandler(w http.ResponseWriter, r *http.Request)
  // GET / - serves composed HTML or assets

func eventsHandler(w http.ResponseWriter, r *http.Request)
  // GET /events - opens SSE connection; subscribes to hub

func stateHandler(w http.ResponseWriter, r *http.Request)
  // GET /api/state - returns current control values

func fireHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/fire - publishes fire event to hub

func resetHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/reset - publishes reset event to hub

func bindHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/bind - updates binding value, publishes to hub

func cssvarHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/cssvar - updates CSS var, publishes to hub
```

---

### `internal/hub` — SSE Pub/Sub System

Manages Server-Sent Events broadcast to all connected OBS browser sources.

**Responsibilities:**
- Subscribe/unsubscribe SSE clients
- Broadcast events to all subscribers
- Handle slow subscribers (drop silently if buffer full)
- Buffer up to 4 messages per subscriber

**Key structures:**

```go
type Hub struct {
  subscribers map[chan interface{}]bool
  broadcast   chan interface{}
  mu          sync.RWMutex
}
```

**Key methods:**

```go
func (h *Hub) Subscribe() <-chan interface{}
  // Returns a channel for this client; client will receive events

func (h *Hub) Unsubscribe(ch <-chan interface{})
  // Remove subscriber (called on connection close)

func (h *Hub) Publish(event interface{})
  // Broadcast event to all subscribers as JSON
```

**Event flow:**

```
dock API call (e.g., POST /api/fire)
    ↓
runtime handler (retrieves active pack name)
    ↓
constructs namespaced element as "pack--element"
    ↓
hub.Publish({ event: "fire", element: "pack--element", event: "..." })
    ↓
All SSE clients receive event
    ↓
Browser JavaScript extracts slug from "pack--element", looks up #tf-element-<slug>, dispatches tf:fire with slug
```

---

### `internal/editor` — Code Editor Server

HTTP server on port 8081. Provides live development environment.

**Responsibilities:**
- HTTP handlers for `/`, `/api/preview`, `/api/save`, `/api/new-element`, `/api/new-pack`, `/api/validate`, `/api/controls`
- File I/O (delegates to store)
- Composition (delegates to composer)
- Validation (delegates to validator)
- Pack reload trigger (via UI and /api/reload)

**Key handlers:**

```go
func editorHandler(w http.ResponseWriter, r *http.Request)
  // GET / - renders editor UI

func previewHandler(w http.ResponseWriter, r *http.Request)
  // GET /api/preview - returns composed HTML

func saveHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/save - writes files, triggers reload

func newElementHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/new-element - creates element folder

func newPackHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/new-pack - creates pack folder

func validateHandler(w http.ResponseWriter, r *http.Request)
  // GET /api/validate - runs validation

func controlsHandler(w http.ResponseWriter, r *http.Request)
  // GET /api/controls - returns parsed controls for inspector

func reloadHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/reload - rescans packs, broadcasts reload-page
```

---

### `internal/dock` — Control Surface Server

HTTP server on port 8082. Provides minimal pack switcher and control UI.

**Responsibilities:**
- HTTP handlers for `/`, `/api/set-active`, `/api/reload`
- Render dock UI with pack buttons and element controls
- Publish pack switch and reload events to hub

**Key handlers:**

```go
func dockHandler(w http.ResponseWriter, r *http.Request)
  // GET / - renders dock UI

func setActiveHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/set-active - switches active pack, broadcasts reload-page

func reloadHandler(w http.ResponseWriter, r *http.Request)
  // POST /api/reload - rescans packs, broadcasts reload-page
```

---

### `internal/store` — File I/O

Reads and writes pack files to disk.

**Responsibilities:**
- Read `element.html`, `element.css`, `element.js` from disk
- Write files with proper permissions
- Create pack/element directories with defaults
- Validate slugs before write

**Key functions:**

```go
func ReadElement(packRoot, slug string) (*models.Element, error)
  // Read HTML/CSS/JS files for an element

func WriteElement(packRoot, slug string, element *models.Element) error
  // Write element files to disk

func CreateElementDir(packRoot, slug string) error
  // Create element folder with default files

func CreatePackDir(packRoot, packName string) error
  // Create pack folder
```

---

### `internal/validator` — Pack Validation

Validates pack structure and authoring rules. Run by editor and at scan-time.

**Responsibilities:**
- Check pack contains at least one element
- Validate slug format (lowercase kebab-case)
- Check for duplicate slugs
- Verify `element.html` and `element.css` exist
- Validate HTML bindings (both `data-tf-bind` and `data-tf-label` required, no duplicates)
- Validate fire events (both `data-tf-fire` and `data-tf-label` required)
- Validate CSS variables (`--tf-ui-*` format)
- Check for broad selectors (`html`, `body`, `*`) — warnings only

**Key function:**

```go
func Validate(pack *models.Pack) *models.ValidationResult
  // Returns errors and warnings
```

---

## Data Flow

### Pack Discovery (Startup + Reload)

```
appstate.New() or appstate.ReloadPacks()
    ↓
scanner.ScanPackDir("./packs")
    ↓
For each subdirectory:
  scanner.ScanPack()
    ↓
  For each element subdirectory:
    scanner.scanElement()
      ↓
      store.ReadElement()
      scanner.parseHTMLBindings()
      scanner.parseCSSVars()
      scanner.parseFireEvents()
    ↓
  Returns Pack with all elements parsed
    ↓
Set first pack as active (if any exist)
Notify dock/runtime
```

### Composition (Every Request + Save)

```
runtime: GET / or dock: GET /api/preview
    ↓
activePackHandler / previewHandler
    ↓
appstate.ActivePack()
    ↓
composer.Compose(pack, controls)
    ↓
For each element (sorted by slug):
  Wrap HTML + inject CSS + inject JS listeners
    ↓
Inject :root CSS variables (defaults + overrides)
    ↓
Return composed HTML
    ↓
Browser loads HTML
    ↓
SSE connection opens, listens for events
```

### Control Update (Bind/CSSVar)

```
dock UI: user changes color picker
    ↓
POST /api/cssvar {element, name, value}
    ↓
runtime: cssvarHandler()
    ↓
appstate.SetCSSVar()
    ↓
hub.Publish({event: "cssvar", element, name, value})
    ↓
All SSE clients receive event
    ↓
Browser JS injects <style> override
    ↓
Element updates in real-time
```

### Pack Switch

```
dock UI: user clicks pack button
    ↓
POST /api/set-active {pack}
    ↓
dock: setActiveHandler()
    ↓
appstate.SetActivePack()
    ↓
hub.Publish({event: "reload-page"})
    ↓
All SSE clients receive reload-page
    ↓
OBS browser sources reload
    ↓
GET / on runtime server
    ↓
composer.Compose(newActivePack)
    ↓
OBS displays new pack
```

---

## Key Concepts

### Control Lifecycle

**Declared controls** (static, parsed at scan-time):
- HTML bindings via `data-tf-bind` + `data-tf-label`
- CSS variables via `--tf-ui-*` naming convention
- Fire events via `data-tf-fire` + `data-tf-label`

**Live values** (ephemeral, in-memory only):
- Stored in `ControlState` (synchronized via SSE)
- Reset to defaults on app restart
- Updated via `/api/bind` and `/api/cssvar`

**Rendering logic:**
- Default values injected at composition-time (`:root` CSS, HTML text content)
- Live overrides broadcast via SSE and applied by client (ID-level CSS, text replacement)
- Browser is the source of truth for current visual state

### Slug Semantics

A **slug** is a unique lowercase kebab-case identifier for an element within a pack.

- Format: `[a-z][a-z0-9]*(-[a-z0-9]+)*` (lowercase, hyphens only)
- Used in folder names: `packs/<pack>/<slug>/`
- Used in DOM IDs: `id="tf-element-<slug>"`
- Used in API calls: `/api/fire?element=<slug>`
- Used in fire event listeners: `if (e.detail.element !== '<slug>') return;`

### Asset Handling

- Assets live under the pack root. Two conventional layouts:
  - **Pack-level** — `packs/<pack>/assets/logo.png`, referenced as `url(assets/logo.png)`. The folder name `assets/` is reserved at the pack level and not treated as an element.
  - **Element-local** — `packs/<pack>/<slug>/assets/logo.png`, referenced as `url(<slug>/assets/logo.png)`. Paths always resolve from the pack root, so the slug prefix is required. Preferred for self-contained, portable elements.
- Served by runtime via `GET /<relative-path>` or `GET /?path=<relative-path>`
- The scanner only reserves `assets/` at the pack root; nested `assets/` under an element folder is ignored by scanning and served normally.

### Animation Restart Behavior

**Show:** When a `show` fire event is triggered:
1. Runtime removes the element from DOM
2. Reinserts it immediately (same node, fresh layout context)
3. Adds `.tf-fired` class
4. All CSS animations restart from 0%

DOM reinsertion is used because `void el.offsetWidth` is unreliable on `display: none` elements and does not force a real layout pass for the subtree.

**Hide (runtime default):** When a `hide` fire event is triggered and element.js does not call `preventDefault()`:
1. Runtime sets `animation: none` inline on the element and all descendants
2. Forces reflow (`void el.offsetWidth`)
3. Clears the inline override, restoring stylesheet control
4. Adds `.tf-hiding` — the `.tf-hiding` CSS rule's animation-name now registers as a brand-new animation set, starting at delay 0ms
5. After `--tf-ui-hide-duration`, removes `.tf-fired` and `.tf-hiding`

The animation reset step is needed because Chromium pairs animations between old/new `animation-name` lists by index. Without it, changing the list from `[in, out]` to `[out]` causes the browser to retain the original scheduled delay rather than restarting the out-animation immediately.

### Hide vs. Reset

- **Hide:** Cancels in-flight CSS animations on the element subtree, dispatches `tf:fire` CustomEvent, adds `.tf-hiding`, waits `--tf-ui-hide-duration`, removes `.tf-fired` and `.tf-hiding`. Element.js can call `e.preventDefault()` on the `tf:fire` event to suppress the default and own the full lifecycle (e.g. for GSAP or custom sequencing).
- **Reset:** Immediately removes `.tf-fired` (no event, no animation, hard-cut safety fallback)

---

## Development Workflow: Adding a New Endpoint

To add a new `/api/custom` endpoint to the runtime server:

1. **Define in `runtime.go`:**
   ```go
   func customHandler(w http.ResponseWriter, r *http.Request) {
     // Get parameters from query or form
     element := r.FormValue("element")
     // Do something with appstate, hub, etc.
     hub.Publish(CustomEvent{...})
     w.WriteHeader(http.StatusNoContent)
   }
   ```

2. **Register in router:**
   ```go
   http.HandleFunc("/api/custom", customHandler)
   ```

3. **Document in API.md** with parameters, response, and example cURL

4. **Broadcast via SSE in hub** (if runtime state changes)

5. **Handle on client side** in browser JavaScript (listen to EventSource)

---

## Development Workflow: Adding a New Validation Rule

To add a new validation check:

1. **Add check to `validator.go`:**
   ```go
   func Validate(pack *models.Pack) *models.ValidationResult {
     // ... existing checks ...
     
     // New check
     for _, elem := range pack.Elements {
       if !isValidCustomRule(elem) {
         result.Errors = append(result.Errors, fmt.Sprintf("..."))
       }
     }
   }
   ```

2. **Run validation from editor:**
   ```bash
   curl "http://localhost:8081/api/validate?pack=<name>"
   ```

3. **Document in [PACK_SPEC.md](PACK_SPEC.md)** if it's a pack authoring rule

---

## Thread Safety

- **AppState:** Guarded by `sync.RWMutex`; safe for concurrent reads, exclusive writes
- **ControlState:** Guarded by `sync.RWMutex`; thread-safe read/write
- **Hub:** Synchronous publish (no buffering at hub level); 4-message buffer per subscriber
- **Scanner:** Runs sequentially on demand (not concurrent)

---

## Build & Run

**Build:**
```bash
go build -o titleflow.exe .
```

**Run with custom paths:**
```bash
./titleflow.exe -packs D:\MyPacks -runtime-port 9000 -editor-port 9001 -dock-port 9002
```

**Startup sequence:**
1. Parse flags
2. Create AppState (scans packs directory)
3. Create Hub (SSE broker)
4. Start runtime server (port 8080)
5. Start editor server (port 8081)
6. Start dock server (port 8082)
7. Block until any server exits

---

## Related Documentation

- **[API.md](API.md)** — HTTP API reference
- **[PACK_SPEC.md](PACK_SPEC.md)** — Pack authoring guide
- **[README.md](README.md)** — Getting started
- **[HISTORY.md](HISTORY.md)** — Design decisions and evolution
