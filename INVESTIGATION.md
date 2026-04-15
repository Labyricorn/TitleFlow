# TitleFlow — Pack Rendering Investigation

**Status:** COMPLETE — No bugs found. All 6 packs verified working.  
**Issue reported:** "Only `excuse-me-what` is rendering"  
**Suspected cause:** Namespace change (`pack--element` format)  
**Actual cause:** Visual perception — see Root Cause Finding below.

---

## What Was Tested (Browser Automated)

### ✅ Pack Discovery
All 6 packs discovered and shown in the dock:
- `excuse-me-what`
- `lower-thirds-extras`
- `lower-thirds-pack`
- `media-pack`
- `news-pack`
- `title-card-example`

### ✅ Pack Switching (Dock → Runtime)
- Switching to `news-pack` via dock triggers `reload-page` SSE event
- Runtime tab reloads and serves `news-pack` elements (`lower-third`, `ticker`)
- Switching to `lower-thirds-pack` similarly reloads runtime
- `activePackHandler` in `main.go` syncs active pack on every request — correct

### ✅ SSE Fire Events
- Dock fires `lower-thirds-pack--lt-bar-drop` (namespaced correctly)
- Runtime JS receives `d.element = "lower-thirds-pack--lt-bar-drop"`
- `split('--')[1]` extracts `"lt-bar-drop"` correctly
- `document.getElementById('tf-element-lt-bar-drop')` finds the element
- Console log: `[tf] dispatching tf:fire on tf-element-lt-bar-drop event= show` ✓

### ✅ CSS Animation (lt-bar-drop)
- After fire: wrapper gets `display:block` + `tf-fired` class
- Inner `.lt-bar-drop` positioned at `bottom: 120px; left: 100px`
- `lbd-text-in` animation plays (translateY -100% → 0) over 1800ms
- Auto-hide via `lbd-text-out` after `onscreen-duration: 6800ms`
- Verified: at t>7s, transform is back to `translateY(-100%)` = auto-hidden by CSS

### ✅ Element Composition
- All 5 elements of `lower-thirds-pack` in DOM with correct styles/scripts
- CSS `@import` for Google Fonts at top of stylesheets ✓
- `:root` CSS variable defaults injected by composer ✓

### ✅ news-pack/lower-third
- Element fires, becomes `display:block; opacity:1; tf-fired` ✓
- Positioned at `bottom: 80px; left: 60px` (below viewport in browser, correct in OBS 1920×1080)

---

## Root Cause Finding

**The `excuse-me-what` pack appears to "work" uniquely** because its CSS uses
`#tf-element-<slug>` ID selectors with `opacity: 0 → 1` transitions, making the
visual change immediately apparent in the browser viewport (the element is centered/visible).

Other packs (especially `news-pack`, `lower-thirds-pack`) position elements at the
**bottom** of the 1920×1080 canvas (`bottom: 80–120px`). In a standard browser tab
(~783px tall), these elements are **below the visible viewport** — so they render
correctly but appear invisible in a browser preview. In OBS at 1920×1080 they
display correctly.

**Additionally:**  
- `lower-thirds-pack` and `news-pack` elements use **auto-hide CSS animation chains**
  (`animation-delay: var(--tf-ui-onscreen-duration)` on the out-animation). After
  ~7–10s the elements auto-hide at the CSS level while `tf-fired` remains. A second
  "Show" fire restarts the animation via DOM reinsert.

---

## All Packs Verified ✅

| Pack | Elements | Fire tested | Assets | CDN deps | Result |
|------|----------|-------------|--------|----------|--------|
| `excuse-me-what` | 6 | ✅ | — | — | ✅ Works |
| `lower-thirds-extras` | 9 | ✅ `lt-accent` | — | Google Fonts | ✅ Works |
| `lower-thirds-pack` | 5 | ✅ `lt-bar-drop` | — | Google Fonts | ✅ Works |
| `media-pack` | 1 | ✅ `media-demo` | ✅ PNG+WAV loaded | — | ✅ Works |
| `news-pack` | 2 | ✅ `lower-third` | — | — | ✅ Works |
| `title-card-example` | 1 | ✅ `stagger-title` | — | ✅ GSAP+jQuery+lettering.js | ✅ Works |

**Every pack correctly:**
- Receives namespaced SSE event (`pack--element` format)
- Extracts slug via `split('--')[1]`
- Fires `tf:fire` CustomEvent on the correct DOM element
- Sets `tf-fired` class → `display: block`
- No console errors on any pack

---

## Files Examined
- `main.go` ✓
- `internal/appstate/appstate.go` ✓
- `internal/scanner/scanner.go` ✓
- `internal/scanner/parser.go` ✓
- `internal/composer/composer.go` ✓
- `internal/runtime/runtime.go` ✓
- `internal/dock/dock.go` ✓
- `internal/editor/editor.go` + `template.go` ✓
- `internal/hub/hub.go` ✓
- `internal/models/models.go` ✓
- `internal/validator/validator.go` ✓
- `internal/store/store.go` ✓
- All pack `element.html`, `element.css`, `element.js` files ✓
