# TitleFlow Pack Specification

Version: v0.4.3b  
Last Updated: 2026-04-15

This document is the authoritative contract for TitleFlow pack authors. It defines how packs are structured, what conventions elements must follow, and how the dock, runtime, and overlay interact. An AI assistant converting a Codepen project to a TitleFlow pack should follow this spec exactly.

---

## Pack Structure

```
packs/
  <pack-name>/
    <element-slug>/
      element.html   required
      element.css    required
      element.js     optional
```

- `<pack-name>` — the folder name shown in the dock's pack switcher
- `<element-slug>` — lowercase kebab-case; used as the DOM element ID (`tf-element-<slug>`); combined with pack name as `<pack-name>--<element-slug>` for API calls and internal routing to prevent namespace collisions

Example:
```
packs/
  news-pack/
    lower-third/
      element.html
      element.css
      element.js
    ticker/
      element.html
      element.css
```

---

## element.html

Contains the markup for one overlay element. TitleFlow wraps this in:

```html
<div id="tf-element-<slug>" class="tf-element" data-tf-pack="<pack>" data-tf-slug="<slug>">
  <!-- your element.html goes here -->
</div>
```

The wrapper's DOM ID uses the slug only (e.g., `tf-element-lower-third`) so CSS selectors work without modification. The pack name is available via `data-tf-pack`. The wrapper starts `display: none`. It becomes `display: block` when a **show** fire event is received (`.tf-fired` class is added). Internally, TitleFlow routes events as `pack--element` (e.g., `my-pack--lower-third`) to prevent collisions when multiple packs share element slugs.

### HTML Bindings

Text fields that should be editable from the dock:

```html
<span data-tf-bind="name" data-tf-label="Name">Default Text</span>
```

- `data-tf-bind` — unique key within the element (used in API calls)
- `data-tf-label` — human-readable label shown in the dock
- Text content of the element is used as the default value

### Fire Events

Named triggers that appear as buttons in the dock control panel:

```html
<button data-tf-fire="show" data-tf-label="Show" style="display:none"></button>
<button data-tf-fire="hide" data-tf-label="Hide" style="display:none"></button>
```

- `data-tf-fire` — event name (string); `show` has built-in runtime behavior
- `data-tf-label` — button label shown in the dock header
- The element itself is hidden (`display:none`) — it exists only as a metadata anchor
- Buttons appear in the dock header in declaration order, always visible regardless of collapsed state

**Built-in event behavior:**

| Event name | Runtime behavior |
|---|---|
| `show` | DOM reinsertion (restarts CSS animations) + adds `.tf-fired` + dispatches `tf:fire` |
| `hide` | Cancels in-flight CSS animations, dispatches `tf:fire`, adds `.tf-hiding`, waits `--tf-ui-hide-duration`, removes `.tf-fired` and `.tf-hiding`. Call `e.preventDefault()` on `tf:fire` in element.js to suppress this default and own the full exit lifecycle. |
| anything else | Dispatches `tf:fire` only — element.js handles all behavior |

**Reset button** — always present in the dock header regardless of declared fire events. Calls `/api/reset` which immediately removes `.tf-fired` (hard-cut, no animation). Use this as a safety fallback if an animation gets stuck.

All events dispatch a `tf:fire` CustomEvent on the element root, regardless of name (see JS API below).

**Recommended `.tf-hiding` pattern for animated exits:**

Define a CSS rule that overrides the element's animations to play the out-keyframes immediately at delay 0ms. **No element.js is required** — the runtime default handles adding `.tf-hiding`, waiting for `--tf-ui-hide-duration`, and removing both classes:

```css
.my-element-root {
  --tf-ui-hide-duration: 600ms;
  /* ... */
}

#tf-element-my-element.tf-hiding .my-bar {
  animation-name: my-out;
  animation-duration: var(--tf-ui-hide-duration, 600ms);
  animation-delay: 0ms;
  animation-fill-mode: forwards;
}
```

**Hide-animation requirements:**

- `--tf-ui-hide-duration` must be at least as long as the longest hide animation (including any `animation-delay`). The runtime removes `.tf-fired` when the duration expires regardless of whether the animation has finished.
- Always use `animation-fill-mode: forwards` on hide rules so the element holds its final state until the class is removed.
- Do not set an `animation-delay` greater than the duration — the element will be removed before the animation starts.
- The dock **Reset** button removes `.tf-fired` immediately with no animation. Use it to recover if an exit gets stuck.

**Advanced — override with element.js:** Call `e.preventDefault()` on the `tf:fire` event to suppress the runtime default entirely and take manual control (useful for GSAP, staggered sequences, or custom timing):

```js
document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'my-element') return;
  if (e.detail.event !== 'hide') return;
  e.preventDefault(); // suppress runtime default
  var durStr = getComputedStyle(__root).getPropertyValue('--tf-ui-hide-duration').trim();
  var ms = durStr.endsWith('ms') ? parseFloat(durStr) : parseFloat(durStr) * 1000;
  __root.classList.add('tf-hiding');
  setTimeout(function() {
    __root.classList.remove('tf-fired', 'tf-hiding');
  }, ms || 600);
});
```

---

## element.css

Standard CSS. TitleFlow injects it into a `<style data-tf-element="<slug>">` block.

### CSS Variables (Dock Controls)

Any CSS custom property prefixed `--tf-ui-` is automatically surfaced as a dock control:

```css
:root,
.lower-third {
  --tf-ui-accent-color: #58a6ff;
  --tf-ui-show-duration: 2800ms;
  --tf-ui-font-size: 16px;
  --tf-ui-headline: "Breaking News";
}
```

The dock infers the input type from the value:

| Value pattern | Input type |
|---|---|
| Starts with `#`, `rgb`, `hsl` | Color picker |
| Ends with `ms` or `s` | Number (duration) |
| Ends with `px`, `em`, `rem`, `%`, `vw`, `vh`, `pt`, `ch` | Text (length) |
| Anything else | Text |

The label is derived from the variable name: `--tf-ui-accent-color` → **Accent Color**.

TitleFlow injects all `--tf-ui-*` defaults into `:root` in the composed document, ensuring CSS `var()` references always resolve — even before an element is shown.

### Position Variables

Position is exposed using the same `--tf-ui-*` convention with descriptive names that match the element's anchor point. The author decides which CSS property each variable drives; the dock label is derived from the name automatically.

| Anchor | Variables | Applied as |
|---|---|---|
| Bottom-left | `--tf-ui-left`, `--tf-ui-bottom` | `left: var(--tf-ui-left); bottom: var(--tf-ui-bottom);` |
| Centered + nudge | `--tf-ui-x-offset`, `--tf-ui-bottom` | `left: 50%; transform: translateX(calc(-50% + var(--tf-ui-x-offset)));` |
| Bottom-right (bug) | `--tf-ui-right`, `--tf-ui-bottom` | `right: var(--tf-ui-right); bottom: var(--tf-ui-bottom);` |
| Full-width bottom | `--tf-ui-bottom` only | `bottom: var(--tf-ui-bottom); left: 0;` |
| Full-screen | *(none)* | Position is structural — not operator-controlled |

Elements that do not benefit from repositioning simply omit position variables.

### Recommended `--tf-ui-*` naming conventions

These are conventions, not enforcement — the dock surfaces any custom property prefixed `--tf-ui-`. Consistent names make packs easier to operate.

- **Colors** — `--tf-ui-primary-color`, `--tf-ui-accent-color`, `--tf-ui-background-color`, `--tf-ui-stroke-color`. For gradients, use `--tf-ui-gradient-start` / `--tf-ui-gradient-end`.
- **Typography** — `--tf-ui-font-size-primary`, `--tf-ui-font-size-secondary`, `--tf-ui-font-weight-bold`, `--tf-ui-font-weight-regular`.
- **Timing** — `--tf-ui-show-duration`, `--tf-ui-hide-duration`, `--tf-ui-delay-primary`, `--tf-ui-delay-secondary`.
- **Positioning** — `--tf-ui-left`, `--tf-ui-right`, `--tf-ui-top`, `--tf-ui-bottom`, `--tf-ui-x-offset` (see table above).
- **Spacing** — `--tf-ui-padding`, `--tf-ui-margin`, `--tf-ui-gap`.

Prefer descriptive names (`--tf-ui-accent-color`) over numbered slots (`--tf-ui-color-1`), and use `-primary`/`-secondary` for hierarchy rather than `-main`/`-alt`.

### Animating elements in

The wrapper starts `display: none` and becomes `display: block` when `.tf-fired` is added. Child elements inside the wrapper are visible by default — if a child has a delayed or transform-based show animation, set its resting state explicitly or it will flash before the animation runs.

- Declare `opacity: 0` (or a transform initial state like `translateX(-100%)`) on every element that is animated in.
- The selector that sets the resting state and the selector that runs the animation must target the **same DOM node**. A mismatch (e.g. `opacity: 0` on a `<span>`, animation on its parent `<div>`) silently fails to animate the hidden element.

**Simple text fade:**

```css
.my-text span { opacity: 0; }

#tf-element-my-element.tf-fired .my-text span {
  animation: fade-in 0.6s ease-out forwards;
}
@keyframes fade-in { from { opacity: 0 } to { opacity: 1 } }
```

**Container fade (opacity on container, animation on container):**

```css
.my-container { opacity: 0; }

#tf-element-my-element.tf-fired .my-container {
  animation: fade-in 0.6s ease-out forwards;
}
```

**Staggered children:**

```css
.my-list .item { opacity: 0; }

#tf-element-my-element.tf-fired .item:nth-child(1) {
  animation: fade-in 0.6s ease-out 0s forwards;
}
#tf-element-my-element.tf-fired .item:nth-child(2) {
  animation: fade-in 0.6s ease-out 0.2s forwards;
}
```

Elements revealed by a parent clip/mask (e.g. text uncovered by an expanding block) do not need their own `opacity: 0` — the parent handles visibility.

### Overflow and growth animations

`overflow: hidden` on an ancestor will clip any child that scales or grows past the ancestor's box. When animating growth:

- Prefer `transform: scaleX()` (with `transform-origin: left`) over animating `width` from `0` to `100%`. Transforms don't clip, compose cheaply, and survive flex layout.
- If you must animate size, remove `overflow: hidden` from the ancestor, or add padding equal to the growth overshoot so the animated content has room.
- Keep `overflow: hidden` when the effect *is* a clip — wipes, reveals, or text sliding inside a fixed container.

### Flexbox layout notes

Flex defaults override explicit widths/heights on children. When animating inside a flex container:

- Add `flex-shrink: 0` to any flex child whose width animates from `0` — otherwise flex will collapse it.
- Use `min-height` rather than `height: 100%` on flex children; percentage heights are unreliable inside flex.
- Prefer `transform: scaleX()` over width keyframes. Width/height keyframes frequently fail to take effect inside flex containers; transforms always do.

### Renaming an element folder

The folder name **is** the element slug. Renaming `packs/<pack>/old-name/` to `packs/<pack>/new-name/` invalidates:

- Every `#tf-element-old-name` selector in `element.css` (both `.tf-fired` and `.tf-hiding` variants).
- The root class the author chose for the element in `element.html` and `element.css`, if it includes the slug.
- Any `if (e.detail.element !== 'old-name') return;` filter in `element.js`.

Update all three before reloading the dock.

---

## element.js

Optional. Executed once on page load inside a closure with `__root` bound to the element's wrapper div.

```js
// __root is the element's wrapper div: #tf-element-<slug>
// All DOM access should remain within __root.
```

### JS Fire Event API

Listen for named fire events dispatched by the runtime:

```js
document.addEventListener('tf:fire', function(e) {
  if (e.detail.element !== 'my-element') return; // filter to this element
  var event = e.detail.event;   // e.g. "show", "hide", "break"

  if (event === 'show') { /* optional: runs after .tf-fired is added */ }
  if (event === 'hide') { /* animate out, then remove .tf-fired */ }
});
```

The `tf:fire` event is dispatched on `document` for every named fire event, after the runtime has applied its built-in class changes. Always filter by `e.detail.element` to avoid handling another element's events.

---

## Overlay Lifecycle

1. Pack is loaded → composer builds full HTML document
2. All elements start hidden (`display: none`)
3. On page load, `/api/state` is fetched and dock control values are restored
4. SSE stream at `/events` is opened — dock commands arrive in real time
5. **show** event → element removed + reinserted in DOM (restarts CSS animations) → `.tf-fired` added → `tf:fire` dispatched
6. **hide** event → in-flight CSS animations cancelled → `tf:fire` dispatched → `.tf-hiding` added → after `--tf-ui-hide-duration`, `.tf-fired` and `.tf-hiding` removed (element.js can call `preventDefault()` to suppress this and own the full lifecycle)
7. **reset** (dock Reset button) → `.tf-fired` removed immediately, no animation
8. **bind** event → `[data-tf-bind="key"]` textContent updated live
9. **cssvar** event → ID-level `<style>` override injected (beats pack class-level specificity)

---

## Converting a Codepen to a TitleFlow Element

1. **Copy HTML** into `element.html`. Remove `<html>/<body>/<head>` wrappers.

2. **Copy CSS** into `element.css`. Replace hardcoded tuneable values with `--tf-ui-*` variables:
   ```css
   /* before */
   color: #ff6b35;
   /* after */
   --tf-ui-accent-color: #ff6b35;
   color: var(--tf-ui-accent-color);
   ```

3. **Declare fire events** in `element.html`. Find the codepen's trigger points (usually button `onclick` or a class toggle) and map them:
   ```html
   <button data-tf-fire="show" data-tf-label="Play" style="display:none"></button>
   <button data-tf-fire="hide" data-tf-label="Reset" style="display:none"></button>
   ```

4. **Wire element.js**. Replace the codepen's button handlers with `tf:fire` listeners:
   ```js
   // codepen had: btn.onclick = () => el.classList.add('active')
   __root.addEventListener('tf:fire', function(e) {
     if (e.detail.event === 'show') {
       __root.querySelector('.my-element').classList.add('active');
     }
     if (e.detail.event === 'hide') {
       __root.querySelector('.my-element').classList.remove('active');
     }
   });
   ```
   If the animation is pure CSS (triggered by `.tf-fired`), element.js may not be needed at all.

5. **Mark editable text** with `data-tf-bind` + `data-tf-label` on any text node the operator should control.

6. **Build and reload** the dock to confirm controls appear.

---

## Troubleshooting

**Element doesn't display when show fires**
- Confirm the CSS ID selector matches the folder name: `#tf-element-<folder-name>`.
- If an animated child has `opacity: 0` in its resting state, make sure the `.tf-fired` rule targets that same child, not its parent.
- Check that no stray `display: none` is set on a child inside the wrapper — only the wrapper itself should start hidden.

**Text gets clipped during the show animation**
- Remove `overflow: hidden` from the animated element's ancestors, or switch the animation from `width` keyframes to `transform: scaleX()`.
- Inside flex containers, add `flex-shrink: 0` to the animated child and use `min-height` rather than `height: 100%`.
- If the element scales up past `1.0`, add padding equal to the overshoot.

**Hide animation doesn't play or cuts off early**
- Verify `--tf-ui-hide-duration` is set on the element root and is at least as long as the hide animation (plus any `animation-delay`).
- Confirm the hide selector uses `.tf-hiding`, not `.tf-fired`, and includes `animation-fill-mode: forwards`.
- Ensure no `animation-delay` exceeds the duration — the runtime removes the classes on schedule regardless.
