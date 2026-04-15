# TitleFlow Changelog

All notable changes to TitleFlow are documented here.

---

## v0.4.3b (2026-04-15)

### Features

#### Editor Reload Button
- Added a "↻ Reload" button to the Editor sidebar next to the Pack selection.
- Clicking reload triggers a full rescan of the `packs/` directory and broadcasts a `reload-page` event to all connected runtime clients (OBS sources).
- Added `/api/reload` (POST) endpoint to the Editor server explicitly.

### Bug Fixes

#### Fixed Missing Pack Picker in Editor
- **Problem:** The pack picker and "New Pack" button were missing from the UI when `hidepack=true` was not present in the URL (default state).
- **Root cause:** A custom `not` template function override in `internal/editor/editor.go` and `internal/dock/dock.go` failed to handle boolean values correctly, causing negated checks like `{{if not .HidePack}}` to always evaluate to `false`.
- **Solution:** Removed the redundant custom `not` implementation. Go's native `not` accurately handles both boolean negation and empty slice checks (`len == 0`).
- **Affected files:** `internal/editor/editor.go`, `internal/editor/template.go`, `internal/dock/dock.go`

### Build Status

✅ Clean build: `go build -o titleflow.exe .`

---

## v0.4.2a (2026-04-14)

### Bug Fixes

#### Hide Button Now Plays Exit Animation on CSS-Chain Auto-Exit Elements
- **Problem:** Pressing the dock's Hide button on packs with CSS-chained auto-exit (`animation-name: in-anim, out-anim` with `animation-delay: 0ms, var(--tf-ui-onscreen-duration)`) would cut or fade abruptly instead of playing the out-animation. The same pack animated correctly when the on-screen timer expired naturally.
- **Root cause:** Chromium pairs animations between old/new style by index in the `animation-name` list. When `.tf-hiding` changed the list from `[in, out]` to `[out]`, the browser retained the previously-scheduled delay rather than restarting the out-animation at 0ms. No visible animation played; after `--tf-ui-hide-duration`, `.tf-fired` was removed and the element snapped to `display: none`.
- **Solution:** The runtime default hide path now cancels in-flight CSS animations on the element subtree (`animation: none` + forced reflow) before adding `.tf-hiding`. This ensures the `.tf-hiding` rule's animation-name is treated as a brand-new animation set starting from frame 0. CSS transitions are unaffected by this reset.
- **Affected packs:** Any pack using CSS-chained auto-exit. Confirmed: all `lower-thirds-pack` and `lower-thirds-extras` elements.
- **Not affected:** Packs whose element.js calls `e.preventDefault()` on the `tf:fire hide` event (e.g. `title-intro`) — those own their exit lifecycle entirely and are unchanged.
- **Affected file:** `internal/composer/composer.go`

### Build Status

✅ Clean build: `go build -o titleflow.exe .`

---

## v0.4.0a (2026-04-08)

### Bug Fixes

#### Validator: Fire Event Buttons with Labels No Longer Rejected
- **Problem:** The HTML validator incorrectly required `data-tf-bind` on any element that had `data-tf-label`, causing false errors on fire event buttons that legitimately pair `data-tf-fire` with `data-tf-label` (e.g. `data-tf-fire="show" data-tf-label="Show"`). All five elements in `lower-thirds-pack` and both elements in `news-pack` were affected (14 errors total).
- **Root cause:** Validator checked for `data-tf-label` without distinguishing whether the element was an editable binding or a fire event button. The parser's `parseFireEvents()` already handled the pattern correctly; the validator was out of sync.
- **Solution:** Validator now detects `data-tf-fire` alongside `data-tf-bind`/`data-tf-label`. The "missing `data-tf-bind`" error is only raised when `data-tf-label` is present without either `data-tf-bind` or `data-tf-fire`.
- **Affected file:** `internal/validator/validator.go`

### Build Status

✅ Clean build: `go build -o titleflow.exe .`

---

## v0.4.0b (2026-04-09)

### Fixes

#### Validator: Windows-safe Required File Checks
- **Problem:** Validation built `element.html`/`element.css` paths using hard-coded `/`, which can break required-file checks on Windows.
- **Solution:** Validation now uses `filepath.Join` for required file checks.
- **Affected file:** `internal/validator/validator.go`

#### Runtime API: Method Enforcement + Backward-Compatible Param Parsing
- **Change:** Runtime control endpoints now require `POST` (with `OPTIONS` supported for CORS preflight).
- **Compatibility:** Parameters are parsed via `FormValue`, which accepts both query-string and form-encoded bodies.
- **Affected file:** `internal/runtime/runtime.go`

#### Asset Serving: Support Both `/?path=` and Path-Based URLs
- **Change:** Runtime now serves assets via both `GET /assets/...` and `GET /?path=assets/...` (the documented form). Assets may live at pack level (`assets/`) or element level (`<slug>/assets/`); paths always resolve from pack root.
- **Affected file:** `internal/runtime/runtime.go`

### Stability

#### Control Parsing: Parse Once at Scan-Time (Avoid Request-Time Mutation)
- **Problem:** Control parsing mutated shared `*models.Element` structs from multiple request paths, risking data races under concurrent access.
- **Solution:** Elements now parse `Bindings`, `CSSVars`, and `FireEvents` once during scanning; request handlers and the composer no longer mutate elements.
- **Affected files:** `internal/scanner/scanner.go`, `internal/composer/composer.go`, `internal/dock/dock.go`, `internal/editor/editor.go`

#### Control State: Reset on Pack Reload (Matches Docs)
- **Change:** Live control state now resets on `ReloadPacks()`, matching the documented “ephemeral session state” behavior.
- **Affected file:** `internal/appstate/appstate.go`

---

## v0.3.1b (2026-04-07)

### Major Features

#### Named Fire Events
- Fire events are now first-class, element-defined controls
- Declared in `element.html` via `data-tf-fire` and `data-tf-label` attributes
- Render as buttons in dock element headers
- Support custom event names beyond `show`/`hide` (e.g., `break`, `promote`, etc.)
- Each fire event dispatches a `tf:fire` CustomEvent with `{element, event}` detail
- See [PACK_SPEC.md](PACK_SPEC.md) for full specification

#### Collapsible Dock Sections
- Element control panels now collapse/expand
- Click element slug or chevron icon to toggle
- Fire event buttons always visible in header (never collapse)
- Cleaner UI for packs with many elements and controls
- Improved visual hierarchy in dock

#### Hide Animation Pattern
- `hide` fire event now dispatches CustomEvent only (no auto-removal of `.tf-fired`)
- Element.js is responsible for animating out and cleanup
- Introduced `.tf-hiding` CSS class pattern for out-animation overrides
- Recommended pattern: set `.tf-hiding` rules, listen for `hide` event, remove `.tf-fired` after animation
- See [PACK_SPEC.md](PACK_SPEC.md) "Recommended `.tf-hiding` Pattern" section

#### Ticker Element
- New scrolling ticker element in starter pack (`packs/news-pack/ticker/`)
- Full-width scrolling text animation at bottom of frame
- HTML bindings for label and text content
- CSS variables for colors, font size, scroll duration
- Fire events for show/hide with custom animation timing
- Demonstrates pure-CSS animation driven by `.tf-fired` class (no JS needed for scrolling)

### Bug Fixes

#### Lower-Third Animation
- **Problem:** Three-animation chain using `calc(var(...) + var(...))` in `animation-delay` failed in some Chromium versions
- **Solution:** Simplified to two-animation model without `calc()`:
  - `lt-bar-in`: plays for show-duration
  - `lt-bar-out`: starts after show + hold period (no `calc()`, explicit delay)
- Eliminated timing bugs and unreliable behavior
- See `packs/news-pack/lower-third/element.css` for example

### Enhancements

#### Duration Input Controls
- CSS variables ending in `ms` or `s` now render as `<input type="number">` with stepper controls
- Parser detects duration type from value suffix
- Improved UX for adjusting animation timings in dock
- Helper functions: `parseDurationValue()`, `extractDurationUnit()`, `tfCSSVarDuration()`

#### Improved File Structure
- Documentation reorganized for clarity
- New [API.md](API.md) for HTTP endpoint reference
- New [ARCHITECTURE.md](ARCHITECTURE.md) for developer guide
- New [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for operator FAQ
- Renamed session summary to [HISTORY.md](HISTORY.md) for reference

### Technical Changes

#### Code Organization
- `internal/models/models.go` — Added `FireEvent` struct, `FireEvents` field on `Element`
- `internal/scanner/parser.go` — Added `parseFireEvents()` pass for DOM walks
- `internal/dock/dock.go` — Fire event buttons in header, collapse UI, new CSS/JS
- `internal/runtime/runtime.go` — `/api/fire` endpoint accepts `event` parameter
- `internal/composer/composer.go` — CustomEvent dispatch; `hide` no longer auto-removes `.tf-fired`

#### Validation Rules
- New validation for fire event `data-tf-fire` and `data-tf-label` pairing
- Enhanced error messages with file/line context
- Validation now covers: HTML bindings, fire events, CSS variables, slug format, file presence

### Breaking Changes

None. All existing packs continue to work.

- Old `show`/`hide` behavior preserved (backward compatible)
- Existing HTML bindings and CSS variables unchanged
- New features are additive; old code works as-is

### Deprecations

None.

### Dependencies

- Go 1.26.1 (no change from v0.1a)
- `golang.org/x/net v0.52.0` for HTML parsing (no change)

### Build Status

✅ Clean build: `go build -o titleflow.exe .`

---

## v0.1a (Initial Release, 2026-04-06)

### Features

#### Pack Discovery
- File-system based pack structure
- Packs are immediate subdirectories of `packs/` folder
- Elements are subdirectories within each pack
- No configuration file required; folder structure is self-describing

#### Pack Structure
- `packs/<pack-name>/<element-slug>/` directory per element
- `element.html` (required) — element markup
- `element.css` (required) — element styling
- `element.js` (optional) — element behavior

#### HTML Bindings
- Declare editable text fields via `data-tf-bind` and `data-tf-label`
- Text content is default value
- Editable from dock; synchronized via SSE `/api/bind` endpoint
- Validation ensures both attributes are present and keys are unique

#### CSS Variables (Dock Controls)
- Publish `--tf-ui-*` variables to expose controls
- Type inference from value:
  - Color: `#`, `rgb`, `hsl` prefix
  - Duration: `ms`, `s` suffix
  - Length: `px`, `em`, `rem`, `%`, `vw`, `vh`, `pt`, `ch` suffix
  - Text: anything else
- Labels auto-derived from variable name (kebab-case to Title Case)
- Updated via SSE `/api/cssvar` endpoint

#### Fire Events (Show/Hide)
- `show` event: Remove element from DOM, reinsert (restart animations), add `.tf-fired` class
- `hide` event: Element.js must animate out and remove `.tf-fired`
- Built-in reset button: Hard-cut (no animation) fallback
- Dispatches `tf:fire` CustomEvent for JS handling

#### Three HTTP Servers

**Runtime Server (port 8080):**
- Serves composed HTML to OBS browser source
- `/` — composed HTML
- `/events` — SSE stream for real-time updates
- `/api/state` — current control values
- `/api/fire`, `/api/reset`, `/api/bind`, `/api/cssvar` — control triggers

**Editor Server (port 8081):**
- Live code editor with HTML/CSS/JS panels
- Real-time preview with same composition logic as runtime
- `/api/save` — persist changes
- `/api/new-element`, `/api/new-pack` — create new content
- `/api/validate` — check pack for errors
- `/api/controls` — get derived controls for inspector

**Dock Server (port 8082):**
- Control surface for operators
- Pack switcher
- Live control UI (bindings, CSS variables, fire events)
- `/api/set-active` — switch active pack
- `/api/reload` — rescan filesystem

#### Composition
- All elements composed into single HTML document
- Elements sorted alphabetically by slug
- Each element wrapped in `<div id="tf-element-<slug>" class="tf-element">`
- CSS injected per-element with `data-tf-element` attribute
- SSE/JS client embedded for real-time control handling
- Cache-Control: no-store (always fresh)

#### Validation
- Pack contains at least one element
- Slug format: lowercase kebab-case
- No duplicate slugs within pack
- `element.html` and `element.css` required
- HTML bindings must have both `data-tf-bind` and `data-tf-label`
- CSS variables must match `--tf-ui-*` format
- Warnings for broad CSS selectors (`html`, `body`, `*`)

#### Starter Pack
- `packs/news-pack/lower-third/` included as working example
- Two HTML bindings: `name`, `title`
- Five CSS variables: `--tf-ui-accent-color`, `--tf-ui-background-color`, `--tf-ui-name-font-size`, `--tf-ui-title-font-size`, `--tf-ui-show-duration`
- CSS animations driven by `--tf-ui-show-duration`
- Optional `element.js` demonstrating lifecycle events

#### Documentation
- [README.md](README.md) — Getting started, basic reference
- [PACK_SPEC.md](PACK_SPEC.md) — Authoritative spec for pack authors and AI converters
  - Structure, bindings, CSS variables, fire events, lifecycle
  - Step-by-step Codepen conversion guide

### Known Limitations

- **Dock controls are read-only:** Show default values, but edits don't persist to runtime (in scope for later releases)
- **Runtime does not auto-refresh:** Manual OBS refresh required after pack switch
- **CSS validation is textual:** Broad selector check uses regex, may miss some edge cases
- **No live reload:** Saving in editor reloads preview iframe; OBS requires manual refresh
- **No JavaScript live reload:** JS changes require full page reload in OBS

### Build Status

✅ Clean build: `go build -o titleflow.exe .`

---

## Version History Summary

| Version | Date | Focus |
|---------|------|-------|
| v0.1a | 2026-04-06 | Foundation: packs, composition, three servers, basic controls |
| v0.3.1b | 2026-04-07 | Polish: named fire events, hide animations, collapsible dock, ticker element |
| v0.4.0a | 2026-04-08 | Bug fix: validator incorrectly rejected fire event buttons with labels |
| v0.4.0b | 2026-04-09 | Bug fixes: Windows paths, POST enforcement, asset URLs, parse-time control extraction |
| v0.4.2a | 2026-04-14 | Bug fix: hide button animation on CSS-chain auto-exit packs |

---

## Related Documentation

- **[README.md](README.md)** — Quick start and feature overview
- **[API.md](API.md)** — Complete HTTP endpoint reference
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — System design and package guide
- **[PACK_SPEC.md](PACK_SPEC.md)** — Pack authoring rules
- **[TROUBLESHOOTING.md](TROUBLESHOOTING.md)** — FAQ and debugging
- **[HISTORY.md](HISTORY.md)** — Technical decisions and design notes from v0.1a
