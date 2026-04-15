---
name: titleflow-pack-creator
description: >
  Converts an HTML/CSS/JS project (e.g. a CodePen animation) into a valid TitleFlow v0.4.3b pack,
  writing element.html, element.css, and (if needed) element.js directly to disk.
  Use this skill whenever the user pastes or references HTML/CSS/JS and wants to use it
  as a TitleFlow overlay, lower-third, ticker, bug, or any other broadcast graphic element.
  Also trigger when the user says "make this a TitleFlow pack", "convert this to TitleFlow",
  "add this to my packs folder", or describes wanting to use a web animation in OBS via TitleFlow.
---

# TitleFlow Pack Creator v0.4.3b

You are converting an HTML/CSS/JS project into a TitleFlow overlay pack. TitleFlow is a
local-first overlay system for OBS — packs are collections of animated elements controlled
from a dock UI in real time during a live broadcast.

Read `references/pack_spec.md` for the full spec before proceeding. The summary below is a
quick orientation; the spec is authoritative.

**TitleFlow v0.4.3b includes:**
- Named fire events (custom animation triggers beyond show/hide)
- Collapsible dock sections
- Hide animation pattern (`.tf-hiding` CSS with optional element.js cleanup)
- Duration controls (number inputs for ms/s values)
- CSS-variable naming conventions and animation best practices

---

## Quick orientation

**Pack folder structure:**
```
<packs-folder>/
  <pack-name>/
    <element-slug>/
      element.html   ← required
      element.css    ← required
      element.js     ← optional (only if JS is needed)
```

- `<pack-name>` — shown in the dock's pack switcher
- `<element-slug>` — lowercase kebab-case; becomes the element's DOM ID and is used in all API calls

**The runtime wraps your element.html in:**
```html
<div id="tf-element-<slug>" class="tf-element" data-tf-slug="<slug>">
  <!-- element.html goes here -->
</div>
```
The wrapper starts `display: none` and gets `display: block` + class `tf-fired` when the Show event fires.

---

## Step 1 — Gather inputs

You need four things. Ask for anything missing:

1. **HTML** — the markup to convert
2. **CSS** — the styles
3. **JS** — optional; source interactivity handlers if the animation has them
4. **Pack name** — e.g. `sports-pack` (shown in dock switcher)
5. **Element slug** — e.g. `score-bug` (lowercase kebab-case; the element's identity)
6. **Packs folder path** — where to write the output, e.g. `D:\.obs\TitleFlow\v0.3.1b\packs`

Validate the slug: must be lowercase, hyphens only (no spaces, underscores, or uppercase). If the
user provides something invalid, suggest the corrected form and confirm before proceeding.

---

## Step 2 — Plan the conversion (think before writing)

Before generating any code, read the source and make a plan:

- Which text nodes should be operator-editable? (names, titles, headlines — anything a broadcast
  operator would change between takes)
- Which hardcoded values are tuneable? (colors, font sizes, animation durations, positions —
  anything a designer or operator might want to adjust without touching code)
- Does the animation need JS, or is it pure CSS? If it's pure CSS with a class toggle driven by
  `.tf-fired`, element.js may be unnecessary.
- If there's a hide animation: does it need JS to remove `.tf-fired` after the out-animation
  completes? (Almost always yes — implement the `.tf-hiding` pattern.)
- Are there custom fire events beyond show/hide (e.g. "break", "highlight", "flash")?

---

## Step 3 — Build element.html

**Clean the HTML:**
- Remove `<html>`, `<head>`, `<body>`, `<style>`, and `<script>` wrappers — element.html is
  inner markup only.
- Keep the structural content: divs, spans, SVGs, etc.

**Add fire event buttons** at the top of element.html (before the markup):
```html
<button data-tf-fire="show" data-tf-label="Show" style="display:none"></button>
<button data-tf-fire="hide" data-tf-label="Hide" style="display:none"></button>
```
These are invisible metadata anchors — they appear as buttons in the dock header. The labels
should describe the action clearly for a broadcast operator (e.g. "Show Lower Third", not just "Show").

**Custom fire events** (v0.3.1b feature):
You can define any event name beyond `show`/`hide`. Examples:
```html
<button data-tf-fire="break" data-tf-label="Break News" style="display:none"></button>
<button data-tf-fire="promote" data-tf-label="Promote" style="display:none"></button>
<button data-tf-fire="flash" data-tf-label="Flash Alert" style="display:none"></button>
```
Custom events dispatch a `tf:fire` CustomEvent that element.js handles. See Step 5 for listener pattern.

**Mark editable text** with `data-tf-bind` + `data-tf-label`:
```html
<span data-tf-bind="name" data-tf-label="Name">Guest Name</span>
```
- `data-tf-bind` — unique key within this element (used in API calls)
- `data-tf-label` — human label shown in the dock
- Keep the default text as a fallback

---

## Step 4 — Build element.css

**Extract tuneable values into `--tf-ui-*` variables:**

The dock auto-surfaces any CSS custom property starting with `--tf-ui-` as a control. The type
is inferred from the value:
- `#...` / `rgb(...)` / `hsl(...)` → color picker
- `...ms` / `...s` → duration stepper
- `...px` / `...em` / `...rem` / `%` / etc. → text (length)
- anything else → text input

Label is auto-derived: `--tf-ui-accent-color` → **Accent Color**.

Declare all variables at the selector that contains them (typically the root element class or `:root`):
```css
.my-element {
  --tf-ui-accent-color: #58a6ff;
  --tf-ui-font-size: 36px;
  --tf-ui-show-duration: 800ms;
  --tf-ui-hide-duration: 600ms;
}
```

Then reference them throughout the file:
```css
color: var(--tf-ui-accent-color);
animation-duration: var(--tf-ui-show-duration, 800ms);
```

Always include a fallback in `var()` calls matching the declared default — this guards against
timing edge cases before the dock has applied its overrides.

**Recommended `--tf-ui-*` naming conventions** (not enforced, but makes packs more consistent):
- **Colors:** `--tf-ui-primary-color`, `--tf-ui-accent-color`, `--tf-ui-background-color`, `--tf-ui-stroke-color`; for gradients use `--tf-ui-gradient-start` / `-end`
- **Typography:** `--tf-ui-font-size-primary`, `--tf-ui-font-size-secondary`, `--tf-ui-font-weight-bold`, `--tf-ui-font-weight-regular`
- **Timing:** `--tf-ui-show-duration`, `--tf-ui-hide-duration`, `--tf-ui-delay-primary`, `--tf-ui-delay-secondary`
- **Positioning:** `--tf-ui-left`, `--tf-ui-right`, `--tf-ui-bottom`, `--tf-ui-top`, `--tf-ui-x-offset` (depending on anchor point)
- **Spacing:** `--tf-ui-padding`, `--tf-ui-margin`, `--tf-ui-gap`

**Animating elements in:**

The wrapper starts `display: none` and becomes `display: block` when `.tf-fired` is added. Child elements are visible by default — if an element is animated in, declare its resting state explicitly:
- Set `opacity: 0` on elements that fade in, or a transform initial state (e.g. `translateX(-100%)`)
- **Critical:** The selector that sets `opacity: 0` and the selector that runs the animation must target the same DOM node. Parent/child mismatch silently fails.

```css
/* ✓ CORRECT — opacity and animation on same element */
.my-text span { opacity: 0; }
#tf-element-my-element.tf-fired .my-text span { animation: fade-in 0.6s ease-out forwards; }

/* ✗ WRONG — opacity on span, animation on parent div */
.my-text span { opacity: 0; }
#tf-element-my-element.tf-fired .my-text { animation: fade-in 0.6s ease-out forwards; }
```

**Overflow and growth animations:**

`overflow: hidden` on an ancestor will clip any child that scales up or grows past the ancestor's box. When animating growth:
- Prefer `transform: scaleX()` (with `transform-origin: left`) over animating `width` from `0` to `100%`. Transforms don't clip and survive flex layout.
- If you must animate size, remove `overflow: hidden` from the ancestor, or add padding equal to the growth overshoot.
- Keep `overflow: hidden` for effects where it is intentional — wipes, reveals, or text sliding inside a fixed container.

**Flexbox layout notes:**

Flex defaults override explicit widths/heights. When animating inside a flex container:
- Add `flex-shrink: 0` to any flex child whose width animates from `0`.
- Use `min-height` rather than `height: 100%` on flex children; percentage heights are unreliable in flex.
- Prefer `transform: scaleX()` over width keyframes. Width/height keyframes frequently fail in flex containers; transforms always work.

**Animation structure for show/hide:**

The recommended pattern uses a two-animation chain per element — one in, one out — with the out
animation delayed by `onscreen-duration` (which equals `show-duration + desired hold time`):
```css
animation-name: my-in, my-out;
animation-duration: var(--tf-ui-show-duration, 800ms), var(--tf-ui-hide-duration, 600ms);
animation-delay: 0ms, var(--tf-ui-onscreen-duration, 5000ms);
animation-fill-mode: forwards;
```
Avoid `calc()` in `animation-delay` — it's unreliable in some Chromium Embedded Framework versions.

**Add `.tf-hiding` overrides** for on-demand hide (when the hide button is pressed before the
timed out-animation would fire):
```css
#tf-element-<slug>.tf-hiding .my-element {
  animation-name: my-out;
  animation-duration: var(--tf-ui-hide-duration, 600ms);
  animation-delay: 0ms;
  animation-fill-mode: forwards;
}
```
Use the full `#tf-element-<slug>` ID in these selectors — this specificity ensures they beat
pack-level class selectors.

**Hide-animation requirements:**
- `--tf-ui-hide-duration` must be at least as long as the longest hide animation (including any `animation-delay`). The runtime removes `.tf-fired` when the duration expires regardless of animation completion.
- Always use `animation-fill-mode: forwards` on hide rules so the element holds its final state.
- Do not set an `animation-delay` greater than the duration — the element will be removed before animation starts.
- The dock Reset button removes `.tf-fired` immediately with no animation — a safety fallback if an exit gets stuck.

**Animations triggered by `.tf-fired`:**
If the source animation is triggered by a class toggle (common in CodePens), wire it to `.tf-fired`:
```css
.tf-element .my-element { opacity: 0; }
.tf-element.tf-fired .my-element { opacity: 1; transition: opacity 0.4s ease; }
```

**Position variables:**
Expose position as `--tf-ui-*` if repositioning is useful for the operator:
```css
.my-element {
  --tf-ui-left: 60px;
  --tf-ui-bottom: 80px;
  position: absolute;
  left: var(--tf-ui-left);
  bottom: var(--tf-ui-bottom);
}
```
Omit position variables for full-screen or structurally-positioned elements.

---

## Step 5 — Build element.js (only if needed)

**Skip element.js entirely** if the animation is pure CSS driven by `.tf-fired`. The runtime
handles adding/removing that class and manages the `.tf-hiding` lifecycle — no JS is required.

**Write element.js when:**
- The source has JS-driven animation (timelines, counters, canvas, etc.)
- You need to add custom behavior to standard hide events (beyond the CSS `.tf-hiding` pattern)
- There are custom fire events with JS behavior

**Standard hide handler (implement if CSS `.tf-hiding` pattern isn't sufficient):**
```js
// __root is the element's wrapper div: #tf-element-<slug>
// All DOM access should remain within __root.

document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== '<slug>') return;
  if (e.detail.event !== 'hide') return;
  e.preventDefault(); // Suppress runtime default; take manual control

  var durStr = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration').trim();
  var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
  if (!ms || isNaN(ms)) ms = 600;

  __root.classList.add('tf-hiding');
  setTimeout(function() {
    __root.classList.remove('tf-fired', 'tf-hiding');
  }, ms);
});
```

In most cases, **don't call `preventDefault()`** — the runtime default handles `.tf-hiding` and class removal automatically. Only call it if you need custom exit behavior (e.g. GSAP, staggered sequences, or timing that differs from the CSS animation).

Key rules for element.js:
- Always filter `e.detail.element !== '<slug>'` first — `tf:fire` is broadcast to all listeners
- Listen on `document`, not `__root` — `__root` is reinserted into the DOM on each show, which
  would detach event listeners added to it
- Scope all DOM queries to `__root` (use `__root.querySelector(...)`)
- Read `--tf-ui-hide-duration` from `getComputedStyle(__root)` so it stays in sync with dock overrides
- Don't wrap your code in an IIFE — the runtime already does this and provides `__root`

---

## Step 6 — Write the files

Create the directory and write the three files:

```
<packs-folder>/<pack-name>/<element-slug>/element.html
<packs-folder>/<pack-name>/<element-slug>/element.css
<packs-folder>/<pack-name>/<element-slug>/element.js   (omit if not needed)
```

After writing, confirm to the user:
- What was written and where
- Which text bindings are now editable in the dock (list the `data-tf-bind` keys and labels)
- Which CSS variables are now surfaced as dock controls (list them with their inferred type)
- What fire event buttons will appear in the dock header
- Whether element.js was included and why (or why it was omitted)
- Remind them to reload the TitleFlow dock to pick up the new pack

---

## Reference example — news-pack/lower-third

This is the gold standard pack included with TitleFlow. Reference its patterns when making decisions:

**element.html pattern:**
```html
<button data-tf-fire="show" data-tf-label="Show Lower Third" style="display:none"></button>
<button data-tf-fire="hide" data-tf-label="Hide Lower Third" style="display:none"></button>
<div class="lower-third">
  <div class="lower-third__bar"></div>
  <div class="lower-third__content">
    <span class="lower-third__name"
          data-tf-bind="name"
          data-tf-label="Name">Christopher G. Chambers</span>
    <span class="lower-third__title"
          data-tf-bind="title"
          data-tf-label="Title">Senior Correspondent</span>
  </div>
</div>
```

**CSS variable block pattern:**
```css
.lower-third {
  --tf-ui-left:             60px;
  --tf-ui-bottom:           80px;
  --tf-ui-accent-color:     #58a6ff;
  --tf-ui-background-color: #0d1117;
  --tf-ui-name-font-size:   36px;
  --tf-ui-title-font-size:  22px;
  --tf-ui-show-duration:    2800ms;
  --tf-ui-onscreen-duration: 7800ms;
  --tf-ui-hide-duration:    1500ms;
}
```

**element.js hide handler pattern:**
```js
document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'lower-third') return;
  if (e.detail.event !== 'hide') return;
  var durStr = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration').trim();
  var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
  if (!ms || isNaN(ms)) ms = 1500;
  __root.classList.add('tf-hiding');
  setTimeout(function() {
    __root.classList.remove('tf-fired', 'tf-hiding');
  }, ms);
});
```

---

## Common conversion decisions

| Source pattern | TitleFlow approach |
|---|---|
| `btn.onclick = () => el.classList.add('active')` | Replace with `tf:fire` listener for `show`; add `active` inside |
| `el.classList.toggle('visible')` | Map to show/hide fire events |
| Hardcoded `color: #ff6b35` | Extract to `--tf-ui-accent-color: #ff6b35` |
| Hardcoded `animation-duration: 0.8s` | Extract to `--tf-ui-show-duration: 800ms` |
| `setTimeout(() => el.classList.remove('active'), 600)` | Use CSS `.tf-hiding` pattern; omit element.js if pure CSS |
| Static text ("Breaking News") | Add `data-tf-bind="headline" data-tf-label="Headline"` |
| JS-driven scroll/ticker | Keep JS; just replace trigger with `tf:fire` listener |
| CSS `@keyframes` animation on class toggle | Wire to `.tf-fired` and `.tf-hiding`; omit element.js if no other JS |
| Text fades in but appears briefly first | Set `opacity: 0` on the element, animation on same element (not parent) |
| Text clips during scale animation | Remove `overflow: hidden` from ancestor, or use `transform: scaleX()` |
| Animation inside flex container doesn't work | Add `flex-shrink: 0` to animated child; prefer transforms over width keyframes |

## Troubleshooting while converting

**Element doesn't display when show fires**
- Confirm the CSS ID selector matches the folder name: `#tf-element-<folder-name>`.
- If an animated child has `opacity: 0` in its resting state, make sure the `.tf-fired` rule targets that same child, not its parent.

**Text gets clipped during the show animation**
- Remove `overflow: hidden` from the animated element's ancestors, or switch from `width` keyframes to `transform: scaleX()`.
- Inside flex containers, add `flex-shrink: 0` to the animated child and use `min-height` rather than `height: 100%`.

**Hide animation doesn't play or cuts off early**
- Verify `--tf-ui-hide-duration` is set and is at least as long as the hide animation (plus any `animation-delay`).
- Confirm the hide selector uses `.tf-hiding`, not `.tf-fired`, and includes `animation-fill-mode: forwards`.
- Ensure no `animation-delay` exceeds the duration — the runtime removes classes on schedule regardless.
