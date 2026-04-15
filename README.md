# TitleFlow v0.4.3b

Local-first overlay pack builder and runtime server for OBS and similar software.

**Last Updated:** 2026-04-15

---

## Quick Links

- **New to TitleFlow?** Start with [How to run](#how-to-run) and [How to create a pack](#how-to-create-a-pack)
- **Building packs?** Read [PACK_SPEC.md](PACK_SPEC.md) — the authoritative reference
- **Stuck?** Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues
- **Need API docs?** See [API.md](API.md) for all HTTP endpoints
- **Understanding the code?** Read [ARCHITECTURE.md](ARCHITECTURE.md) for system design
- **History & decisions?** See [HISTORY.md](HISTORY.md) for v0.1a→v0.3.1b evolution

---

## What's New in v0.4.3b

- **Editor Reload Button** — Manual scan trigger in the editor UI to pick up external pack changes.
- **Pack Picker Visibility Fix** — Resolved a template logic bug that caused the pack selection dropdown to disappear during normal use.
- **Improved Boolean Template Logic** — Unified `not` template function behavior across all servers.

For full history, see [CHANGELOG.md](CHANGELOG.md).

---

## How to run

```
go run . [flags]
```

Or build first:

```
go build -o titleflow.exe .
./titleflow.exe [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-packs` | `./packs` | Path to the directory containing pack folders |
| `-runtime-port` | `8080` | Port for the runtime server (use as OBS browser source) |
| `-editor-port` | `8081` | Port for the editor UI |
| `-dock-port` | `8082` | Port for the dock UI |

### Default URLs

| Surface | URL | Purpose | Audience |
|---------|-----|---------|----------|
| **Runtime** | http://localhost:8080 | OBS browser source overlay | OBS operators |
| **Editor** | http://localhost:8081 | Live code editor with preview | Pack authors |
| **Dock** | http://localhost:8082 | Pack switcher & control surface | OBS operators |

**In OBS:** Add a Browser Source pointing to `http://localhost:8080`. Right-click → Refresh to sync after pack changes.

---

### Custom Ports

To run multiple instances or use non-default ports:

```bash
./titleflow.exe -runtime-port 9000 -editor-port 9001 -dock-port 9002 -packs "D:\MyPacks"
```

Then update OBS browser source URL to match the runtime port (e.g., `http://localhost:9000`).

---

## How packs are discovered

TitleFlow scans the packs directory on startup and whenever you click **Reload** in the dock or save in the editor.

Each immediate subdirectory of the packs directory is treated as a pack root.

```
packs/
  news-pack/          ← pack root (name = "news-pack")
    lower-third/      ← element folder (slug = "lower-third")
      element.html
      element.css
      element.js      ← optional
    ticker/
      element.html
      element.css
  sports-pack/
    ...
```

No configuration file is required. TitleFlow reads the folder structure directly.

---

## How to create a pack

### Option A — Use the editor

1. Open http://localhost:8081
2. Click **+ New Pack**, enter a name
3. Click **+ New Element**, enter a slug (lowercase kebab-case)
4. Edit HTML, CSS, and optional JS in the panels
5. Click **Save**

### Option B — Create folders manually

```
mkdir packs/my-pack/my-element
echo "" > packs/my-pack/my-element/element.html
echo "" > packs/my-pack/my-element/element.css
```

Then click **Reload** in the dock or restart TitleFlow.

---

## Pack structure rules

- Each element lives in its own slug-named subfolder.
- Slug must be **lowercase kebab-case**: `lower-third`, `ticker`, `alert`.
- Each element folder **must** contain `element.html` and `element.css`.
- `element.js` is **optional**.
- Assets must live under the pack root and are referenced by paths relative to the pack root.
- Two valid layouts:
  - **Pack-level** — `packs/<pack>/assets/foo.png`, referenced as `assets/foo.png`. The `assets/` folder name is reserved at the pack level and not treated as an element.
  - **Element-local** — `packs/<pack>/<slug>/assets/foo.png`, referenced as `<slug>/assets/foo.png`. Keeps the element self-contained and portable.

---

## Declaring editable controls

### HTML bindings

Add both `data-tf-bind` and `data-tf-label` to any element you want to be editable.

```html
<span class="lower-third__name"
      data-tf-bind="name"
      data-tf-label="Name">Default Name</span>
```

- `data-tf-bind` is the binding key (must be unique within the element).
- `data-tf-label` is the label shown in the dock and inspector.
- The text content is the default value.

Both attributes are **required**. One without the other is a validation error.

### Published CSS variables

Use the `--tf-ui-` prefix to expose CSS custom properties as controls.

```css
.my-element {
  --tf-ui-accent-color: #58a6ff;
  --tf-ui-font-size: 36px;
  --tf-ui-show-duration: 2800ms;
}
```

Labels are derived from the variable name:
- `--tf-ui-accent-color` → **Accent Color**
- `--tf-ui-font-size` → **Font Size**

Only `--tf-ui-*` variables are exposed. All other CSS remains internal.

---

## How validation works

Run validation from the editor (click **Validate Pack**) or via the API:

```
GET http://localhost:8081/api/validate?pack=news-pack
```

Returns JSON with `ok`, `errors`, and `warnings` arrays.

Validation checks:

- Invalid pack root (no element folders)
- Invalid slug names (not lowercase kebab-case)
- Duplicate element slugs
- Missing `element.html` or `element.css`
- HTML editable bindings missing `data-tf-bind` or `data-tf-label`
- Duplicate binding keys within an element
- Invalid `--tf-ui-*` variable name format
- Broad CSS selectors (`*`, `body`, `html`) — reported as warnings
- Unreadable files

Each error identifies the pack, element slug, file, and a plain explanation.

Example output:
```
ERROR: pack "news-pack", element "lower-third", file "element.html": editable binding "name" is missing required data-tf-label.
```

---

## Composition rules

All elements in a pack are composed into one HTML document in **alphabetical slug order**.

TitleFlow creates an outer wrapper `<div>` for each element:

```html
<div id="tf-element-lower-third" class="tf-element" data-tf-slug="lower-third">
  <!-- your element.html content goes here -->
</div>
```

In `element.js`, the variable `__root` refers to this wrapper div.

CSS is loaded per-element with a `data-tf-element` attribute.

The preview in the editor uses **exactly the same composition logic** as the runtime server.

---

## Starter pack

`packs/news-pack/lower-third/` is included as a working example.

It demonstrates:
- Two HTML bindings (`name`, `title`) with explicit labels
- Five published CSS variables (`--tf-ui-accent-color`, `--tf-ui-background-color`, `--tf-ui-name-font-size`, `--tf-ui-title-font-size`, `--tf-ui-show-duration`)
- CSS animations driven by `--tf-ui-show-duration`
- An optional `element.js` that logs when the element mounts

---

## Implementation notes and design decisions

- **Pack discovery is filesystem-based.** Scans folder structure; no config files required. Supports any folder layout inside a pack root.
- **Composition is deterministic.** Same pack + live values always produce identical HTML. Composition happens per-request (no caching of composed output).
- **Control state is ephemeral.** Live values (bindings, CSS variables) stored in memory only. Reset on app restart. Use the dock to set values; they don't persist between sessions.
- **Hide animations have a runtime default.** The runtime cancels in-flight CSS animations, adds `.tf-hiding`, waits `--tf-ui-hide-duration`, then removes `.tf-fired` and `.tf-hiding`. Packs only need to define `.tf-hiding` CSS rules — no element.js required for standard exits. Element.js can call `e.preventDefault()` on `tf:fire` to suppress the default and own the full exit lifecycle (e.g., for GSAP or custom sequencing).
- **Runtime does not auto-refresh.** When you switch packs in the dock, OBS browser sources require a manual refresh (right-click → Refresh) to pick up the change. This avoids unwanted flicker during pack configuration.
- **CSS validation is textual.** The broad-selector check uses regex. It catches common mistakes (`body {`, `html {`, `* {`) but relies on pack authors to avoid overly-broad selectors.
- **JS is wrapped in an IIFE.** Each element's JS runs as `(function(){ var __root = ...; YOUR CODE })();` for element isolation. Pack authors don't need to add the wrapper.
- **SSE is the real-time channel.** All control updates broadcast via Server-Sent Events. Dock and runtime connect on startup; OBS browser sources listen for updates (bindings, CSS vars, fire/reset events, page reloads).
- **No authentication or encryption.** TitleFlow is designed for trusted local networks. Use firewall rules to restrict access if deploying to shared infrastructure.

---

## Next Steps

**Just getting started?**
1. Read [How to create a pack](#how-to-create-a-pack)
2. Edit the starter pack in `packs/news-pack/lower-third/`
3. Open http://localhost:8081 to preview changes

**Building custom overlays?**
1. Read [PACK_SPEC.md](PACK_SPEC.md) for the complete authoring guide
2. Check out `packs/news-pack/ticker/` for a more complex example
3. Use [API.md](API.md) to integrate with external scripts

**Stuck or learning the code?**
1. See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for common issues
2. Read [ARCHITECTURE.md](ARCHITECTURE.md) to understand the system design
3. Check [HISTORY.md](HISTORY.md) for design decisions and rationale
