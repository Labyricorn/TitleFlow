# TitleFlow History: v0.1a Design Session

This document preserves the technical decisions and implementation notes from the v0.1a → v0.3.1b development session. It's provided as reference for understanding design rationale.

**Current version:** v0.4.2a  
**Last updated:** 2026-04-14

---

# Session Summary: v0.1a Development

## Completed Work

### 1. Dock Duration Input Controls
- Added `type="number"` inputs with stepper controls (up/down arrows) for duration CSS variables
- Added template helper functions: `parseDurationValue()` (extract numeric part from "2800ms") and `extractDurationUnit()` (extract unit suffix)
- JavaScript helper `tfCSSVarDuration()` reconstructs full value with unit before sending to runtime
- Duration values render as number inputs in dock; others remain as text/color inputs

### 2. Lower-Third Animation Fix
**Problem:** Three-animation chain using `calc(var(...) + var(...))` in `animation-delay` failed in some Chromium versions, causing `lt-bar-out` to start immediately and override the show animation — result: "instantly shows then fades."

**Solution:** Switched to two-animation model:
- `lt-bar-in`: plays for `show-duration`, uses `animation-fill-mode: forwards` to hold end state
- `lt-bar-out`: starts at `onscreen-duration` delay (= show + hold time), plays for `hide-duration`
- Eliminated `calc()` entirely — hold period is implicit gap between animations
- Redefined `--tf-ui-onscreen-duration: 7800ms` to mean "delay from fire to start of hide" (2800ms show + 5000ms hold)

### 3. Ticker Element (news-pack)
Added a full-width scrolling ticker bar at bottom of frame:
- **HTML:** Label binding ("BREAKING") + text binding (scrolling headline)
- **CSS:** Flexbox layout, label with accent color, scrolling text via CSS animation
- **Animation:** Pure CSS, controlled by `animation-play-state: paused/running` — toggled by `.tf-fired` class on parent
- **How it works:** Fire → DOM reinsertion restarts animation → `.tf-fired` added → CSS selector activates animation
- **Dock controls:** Accent Color, Background Color, Text Color, Font Size (length), Scroll Duration (duration)
- No JavaScript needed for scrolling (CSS animation is the driver)

### 4. Named Fire Events + Collapsible Dock

#### Named Fire Events
Fire events are first-class, element-defined controls — declared in `element.html` via `data-tf-fire`/`data-tf-label` attributes, parsed by the scanner, and rendered as buttons in the dock header.

**Declaration (element.html):**
```html
<button data-tf-fire="show" data-tf-label="Show Lower Third" style="display:none"></button>
<button data-tf-fire="hide" data-tf-label="Hide Lower Third" style="display:none"></button>
```

**Runtime behavior:**
- `show` → DOM reinsertion + `.tf-fired` added + `tf:fire` CustomEvent dispatched on `document`
- `hide` → `tf:fire` CustomEvent dispatched only — **element.js is responsible** for animating out and removing `.tf-fired`
- Any other name → `tf:fire` CustomEvent only (element.js handles it)

**element.js listener pattern — listen on `document`, filter by slug:**
```js
document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'my-element') return;
  if (e.detail.event === 'show') { /* ... */ }
  if (e.detail.event === 'hide') { /* ... */ }
});
```

**Why `document` not `__root`:** Dispatching on `el` (from `getElementById`) and listening on `__root` (captured at IIFE time) can reference different JS objects in some browser/CEF contexts after DOM reinsertion. `document` is always the same global reference.

#### Collapsible Dock Sections + Header Layout
- Element header shows: `[slug] [▶ Fire Buttons…] [↺ Reset] [▲]` — always visible
- Clicking the slug area or chevron collapses/expands the control list (bindings + CSS vars)
- All action buttons use `event.stopPropagation()` so they never trigger collapse
- **Reset button** is hardcoded in every header — calls `/api/reset` for an instant hard-cut (no animation); safety fallback if animation gets stuck

#### Hide vs Reset distinction
- **Hide** — author-defined animated exit; element.js adds `.tf-hiding`, waits for animation, then removes `.tf-fired` and `.tf-hiding`
- **Reset** — hardcoded hard-cut; runtime removes `.tf-fired` immediately, no `tf:fire` event dispatched

#### Lower-Third Hide Animation
- `element.css` — added `.tf-hiding` rules that override the timed animation chain, playing out-keyframes immediately at delay 0ms
- `element.js` — reads `--tf-ui-hide-duration` from computed styles, adds `.tf-hiding`, cleans up both classes after timeout

#### Files Modified
- `internal/models/models.go` — added `FireEvent` struct + `FireEvents` field on `Element`
- `internal/scanner/parser.go` — added `parseFireEvents()` DOM walk pass
- `internal/dock/dock.go` — fire buttons + Reset in header, collapse UI, `tfFire(slug, event)` + `tfReset` + `tfToggle` JS, updated CSS
- `internal/runtime/runtime.go` — `/api/fire` accepts and broadcasts `event` param
- `internal/composer/composer.go` — SSE fire handler dispatches `tf:fire` CustomEvent; `hide` no longer auto-removes `.tf-fired`
- `packs/news-pack/lower-third/element.html` — `data-tf-fire` declarations
- `packs/news-pack/lower-third/element.css` — `.tf-hiding` animation override rules
- `packs/news-pack/lower-third/element.js` — hide animation + cleanup
- `packs/news-pack/ticker/element.html` — `data-tf-fire` declarations
- `packs/news-pack/ticker/element.css` — added `--tf-ui-hide-duration`, `ticker-slide-out` keyframe, `.tf-hiding` rules
- `packs/news-pack/ticker/element.js` — hide animation + cleanup (slide down off bottom edge + fade)

### 5. PACK_SPEC.md
New file at repo root — authoritative contract for pack authors and AI converters. Covers:
- Pack folder structure and file requirements
- `data-tf-bind` / `data-tf-label` HTML binding convention
- `data-tf-fire` / `data-tf-label` fire event convention
- `--tf-ui-*` CSS variable convention and type inference rules
- JS `tf:fire` CustomEvent API
- Overlay lifecycle (SSE events, `.tf-fired`, DOM reinsertion)
- Step-by-step Codepen → TitleFlow conversion guide

---

## Key Technical Decisions

1. **Avoided `calc()` in animation-delay** — unreliable in some CEF/Chromium versions; switched to explicit delay on animation-2
2. **CSS `animation-play-state` over JS `transition`** — avoids timing/measurement issues; browser handles animation restart on DOM reinsertion automatically
3. **Fixed scroll duration vs. consistent px/sec speed** — chose fixed duration for simplicity; can measure text width via JS later for pixel-per-second speed if needed
4. **Fire events declared in element.html, not the dock** — elements own their trigger surface; enables multiple named events and makes Codepen conversion mechanical
5. **Server stays a thin relay** — `/api/fire` just broadcasts `{element, event}` via SSE; all animation logic lives in element CSS/JS
6. **`tf:fire` dispatched on `document`, not `el`** — `getElementById` can return different JS wrapper objects across DOM reinsertion cycles in some browser contexts; `document` is always the same global reference and avoids the mismatch entirely; element.js filters by `e.detail.element` slug
7. **Hide is author-owned, Reset is platform-owned** — `hide` dispatches CustomEvent and delegates cleanup to element.js; Reset is a hardcoded hard-cut that bypasses all animation

---

## Build Status
✅ Clean build (`go build -o titleflow.exe .`)

---

## Related Documentation

- **[CHANGELOG.md](CHANGELOG.md)** — Feature evolution from v0.1a to v0.3.1b
- **[PACK_SPEC.md](PACK_SPEC.md)** — Authoritative pack authoring specification
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — System design and package reference
- **[API.md](API.md)** — HTTP endpoint reference
- **[README.md](README.md)** — Getting started guide
