# TitleFlow Troubleshooting Guide

Version: v0.4.4b  
Last Updated: 2026-04-15

This guide covers common issues, debugging techniques, and solutions for TitleFlow operators and pack authors.

---

## Quick Checklist

Before diving into specific issues:

- [ ] TitleFlow is running (`go run .` or `titleflow.exe`)
- [ ] All three URLs are accessible in browser:
  - Runtime: http://localhost:8080
  - Editor: http://localhost:8081
  - Dock: http://localhost:8082
- [ ] OBS has a Browser Source pointing to http://localhost:8080
- [ ] Firewall is not blocking ports 8080, 8081, 8082

---

## OBS Integration Issues

### Browser Source Shows Blank Page / 404 Error

**Symptom:** OBS browser source displays nothing or shows a 404 error.

**Cause:** TitleFlow is not running, wrong port, or firewall is blocking.

**Solution:**

1. Verify TitleFlow is running:
   ```bash
   go run . [flags]
   ```
   
2. Verify the runtime server is listening:
   ```bash
   curl http://localhost:8080
   ```
   You should see HTML content. If it fails, TitleFlow is not running.

3. Check the browser source URL in OBS:
   - Should be: `http://localhost:8080`
   - Not `http://127.0.0.1:8080` (use localhost)
   - Not `http://localhost:8081` (wrong port — that's the editor)

4. If using non-default ports, update flags and OBS URL:
   ```bash
   ./titleflow.exe -runtime-port 9000
   # Then set OBS browser source URL to: http://localhost:9000
   ```

5. Check Windows Firewall:
   - Go to Settings > Privacy & Security > Windows Defender Firewall > Allow an app through firewall
   - Ensure `titleflow.exe` is listed and allowed
   - Or disable firewall temporarily for testing

6. If running on a different machine:
   - Use the machine's IP address instead of `localhost`
   - Example: `http://192.168.1.100:8080`
   - Ensure both machines are on the same network

---

### Controls Don't Appear in OBS / Not Real-Time

**Symptom:** Dock controls exist and update in the dock UI, but OBS browser source doesn't show changes.

**Cause:** OBS browser source is not receiving SSE updates, or page needs manual refresh.

**Solution:**

1. **Refresh the OBS browser source:**
   - Right-click the browser source in OBS
   - Click **Refresh** (or press Ctrl+Shift+R)
   - Wait 2 seconds for SSE connection to establish

2. **Verify SSE connection:**
   - Open browser console (F12) while viewing http://localhost:8080
   - Look for messages like: `SSE connected` or `Listening for events`
   - If you see `SSE failed` or no messages, the connection is broken

3. **Check dock is running:**
   ```bash
   curl http://localhost:8082
   ```
   Should return HTML. If it fails, dock server is not running.

4. **Verify no firewall blocking:**
   - TitleFlow uses SSE (Server-Sent Events), which is HTTP streaming
   - Some corporate firewalls may block streaming; try disabling firewall temporarily

5. **Check browser console errors:**
   - Press F12 in OBS browser source window
   - Look for red error messages
   - Common error: `Failed to fetch /events` — indicates network issue

6. **Note:** Value changes in dock are instant (SSE broadcast), but layout changes (show/hide element) require manual OBS refresh.

---

### Browser Source Content Looks Pixelated / Blurry

**Symptom:** Overlay in OBS is blurry or pixelated.

**Cause:** Browser source resolution mismatch.

**Solution:**

1. In OBS, go to Source Properties for the browser source
2. Set **Width** and **Height** to match your canvas size (typically 1920×1080)
3. Uncheck **Refresh browser when scene becomes active** (causes flicker)
4. Set **Shutdown source when not visible** to On (saves CPU)

---

### Animation Plays Then Immediately Reverses / Looks Wrong

**Symptom:** Show animation plays for a moment, then immediately reverses back.

**Cause:** CSS animation conflict, often due to:
- Broad CSS selector (`body`, `html`, `*`)
- Invalid `calc()` in animation-delay
- Animation duration mismatch

**Solution:**

1. **Validate the pack:**
   ```bash
   curl "http://localhost:8081/api/validate?pack=<pack-name>"
   ```
   Look for warnings about broad selectors. Fix them in `element.css`.

2. **Check CSS for `calc()` in animation timing:**
   - Avoid `animation-delay: calc(var(...) + var(...))`
   - Instead, use explicit delays or stagger animations
   - See [PACK_SPEC.md](PACK_SPEC.md) for recommended animation patterns

3. **Verify animation names match:**
   - Check `element.css` for animation names
   - Check `element.html` for corresponding keyframe names
   - Typos here cause silent failures

4. **Test in browser inspector:**
   - Open http://localhost:8080 in browser (F12)
   - Check Computed tab to see which CSS rules actually apply
   - Look for conflicting rules from broad selectors

---

## Pack Author Issues

### Binding Not Editable in Dock

**Symptom:** A text field should be editable in the dock, but there's no control for it.

**Cause:** Missing or mismatched `data-tf-bind` and `data-tf-label` attributes.

**Solution:**

1. **Check both attributes are present:**
   ```html
   <!-- ✓ Correct -->
   <span data-tf-bind="name" data-tf-label="Name">Default Name</span>
   
   <!-- ✗ Wrong: missing data-tf-label -->
   <span data-tf-bind="name">Default Name</span>
   
   <!-- ✗ Wrong: missing data-tf-bind -->
   <span data-tf-label="Name">Default Name</span>
   ```

2. **Validate the pack:**
   ```bash
   curl "http://localhost:8081/api/validate?pack=<pack-name>"
   ```
   Look for errors mentioning binding names.

3. **Check binding key is unique within element:**
   ```html
   <!-- ✗ Wrong: duplicate 'name' key -->
   <span data-tf-bind="name" data-tf-label="First Name">John</span>
   <span data-tf-bind="name" data-tf-label="Last Name">Doe</span>
   
   <!-- ✓ Correct -->
   <span data-tf-bind="first-name" data-tf-label="First Name">John</span>
   <span data-tf-bind="last-name" data-tf-label="Last Name">Doe</span>
   ```

4. **Verify default text:**
   - The text content of the element is the default value
   - If empty, the control will show an empty field
   - Example: `<span data-tf-bind="name" data-tf-label="Name"></span>` — default is empty string

---

### CSS Variable Not Appearing in Dock Controls

**Symptom:** You added a CSS variable in `element.css`, but it doesn't show as a control in the dock.

**Cause:** Variable name doesn't start with `--tf-ui-` or format is invalid.

**Solution:**

1. **Check the variable name starts with `--tf-ui-`:**
   ```css
   /* ✓ Correct */
   --tf-ui-accent-color: #ff0000;
   --tf-ui-show-duration: 2800ms;
   
   /* ✗ Wrong: missing --tf-ui- prefix */
   --accent-color: #ff0000;
   
   /* ✗ Wrong: uses uppercase */
   --tf-ui-Accent-Color: #ff0000;
   ```

2. **Use lowercase kebab-case:**
   ```css
   /* ✓ Correct */
   --tf-ui-my-custom-value: 10px;
   
   /* ✗ Wrong: camelCase */
   --tf-ui-myCustomValue: 10px;
   
   /* ✗ Wrong: spaces in name */
   --tf-ui-my custom value: 10px;
   ```

3. **Verify variable is applied to an element:**
   ```css
   /* ✓ Correct: variable is on a real element */
   .my-element {
     --tf-ui-accent-color: #ff0000;
     color: var(--tf-ui-accent-color);
   }
   
   /* ⚠ Warning: variable is on :root but not used in element */
   :root {
     --tf-ui-accent-color: #ff0000;
   }
   .my-element {
     color: blue; /* doesn't use the variable */
   }
   ```

4. **Reload the editor:**
   - Click the pack/element selector to refresh
   - Or close/reopen the editor tab
   - Live parse may be stale

5. **Check variable type inference:**
   ```css
   /* Type is inferred from the VALUE */
   --tf-ui-color: #ff0000;           /* → color (starts with #) */
   --tf-ui-duration: 2800ms;         /* → duration (ends with ms) */
   --tf-ui-size: 36px;               /* → length (ends with px) */
   --tf-ui-name: "Hello";            /* → text (anything else) */
   ```

---

### Fire Event Button Not Showing in Dock

**Symptom:** You declared a fire event in HTML, but the button doesn't appear in the dock header.

**Cause:** Missing attributes, validation error, or invalid HTML.

**Solution:**

1. **Check both `data-tf-fire` and `data-tf-label` are present:**
   ```html
   <!-- ✓ Correct -->
   <button data-tf-fire="show" data-tf-label="Show Lower Third" style="display:none"></button>
   
   <!-- ✗ Wrong: missing data-tf-label -->
   <button data-tf-fire="show" style="display:none"></button>
   
   <!-- ✗ Wrong: missing data-tf-fire -->
   <button data-tf-label="Show Lower Third" style="display:none"></button>
   ```

2. **Fire event button must be hidden:**
   ```html
   <!-- ✓ Correct: button is hidden -->
   <button data-tf-fire="show" data-tf-label="Show" style="display:none"></button>
   
   <!-- ✗ Wrong: button is visible -->
   <button data-tf-fire="show" data-tf-label="Show"></button>
   ```

3. **Validate the pack:**
   ```bash
   curl "http://localhost:8081/api/validate?pack=<pack-name>"
   ```
   Look for errors mentioning fire events.

4. **Reload editor after fix:**
   - Click another pack, then click back
   - Or close/reopen the editor tab
   - Parser runs at scan-time

---

### Show Animation Doesn't Play (Element Appears Instantly)

**Symptom:** Fire `show` event, but element appears without animation; or appears briefly then vanishes.

**Cause:** Missing resting state (`opacity: 0`), or animation targets the wrong DOM element.

**Solution:**

1. **Ensure every animated element declares its resting state:**
   ```css
   /* ✓ CORRECT: element has opacity: 0, animation targets same element */
   .my-text span { opacity: 0; }
   #tf-element-my-element.tf-fired .my-text span {
     animation: fade-in 0.6s ease-out forwards;
   }
   
   /* ✗ WRONG: opacity on child, animation on parent — won't animate */
   .my-text span { opacity: 0; }
   #tf-element-my-element.tf-fired .my-text {  /* parent, not child */
     animation: fade-in 0.6s ease-out forwards;
   }
   ```
   
   The selector that sets `opacity: 0` and the selector that runs the animation must target **the same DOM node**.

2. **If using transforms (e.g., slide-in), set initial state on same element:**
   ```css
   .my-block { transform: translateX(-100%); }
   #tf-element-my-element.tf-fired .my-block {
     animation: slide-in 0.6s ease-out forwards;
   }
   ```

3. **For nested elements, check each level:**
   ```css
   /* If you have: <div class="container"><span>Text</span></div> */
   /* And want the span to fade in, animate the span: */
   .container span { opacity: 0; }
   #tf-element-my-element.tf-fired .container span {
     animation: fade-in 0.6s ease-out forwards;
   }
   ```

4. **Check for overflow clipping:**
   - If text scales or grows and appears cut off, remove `overflow: hidden` from ancestors
   - Or use `transform: scaleX()` instead of `width` keyframes
   - See [PACK_SPEC.md § Overflow and growth animations](PACK_SPEC.md#overflow-and-growth-animations)

5. **Inside flex containers, check flex defaults:**
   - Add `flex-shrink: 0` to flex children whose width animates from `0`
   - Use `min-height` instead of `height: 100%` for flex children
   - Prefer `transform: scaleX()` over width keyframes in flex
   - See [PACK_SPEC.md § Flexbox layout notes](PACK_SPEC.md#flexbox-layout-notes)

---

### Hide Animation Doesn't Work

**Symptom:** Fire `hide` event, but element doesn't animate out; just disappears.

**Cause:** Missing `.tf-hiding` CSS rules, or the `.tf-hiding` animation is not restarting cleanly.

**Solution:**

1. **Add `.tf-hiding` CSS override in `element.css`.** This is all that's required — no element.js is needed for standard CSS-based exits. The runtime handles adding `.tf-hiding`, waiting for `--tf-ui-hide-duration`, and cleanup:
   ```css
   .my-element {
     --tf-ui-hide-duration: 600ms;
   }

   #tf-element-my-element.tf-hiding .my-bar {
     animation-name: my-out;
     animation-duration: var(--tf-ui-hide-duration, 600ms);
     animation-delay: 0ms;
     animation-fill-mode: forwards;
   }
   ```

2. **Verify `--tf-ui-hide-duration` is declared** on the element root. The runtime reads this value to know how long to wait before removing `.tf-fired`. Duration must be at least as long as the hide animation (including any `animation-delay`).

3. **Verify animation keyframes exist:**
   ```css
   @keyframes my-out {
     from { opacity: 1; transform: translateY(0); }
     to { opacity: 0; transform: translateY(100px); }
   }
   ```

4. **Check `animation-fill-mode: forwards`** — without it, the element snaps back to its original state when the animation ends.

5. **Do NOT set `animation-delay` longer than the duration** — the element will be removed before the animation starts.

6. **For custom timing or GSAP animations**, add an element.js listener and call `e.preventDefault()` to suppress the runtime default entirely:
   ```js
   document.addEventListener('tf:fire', function(e) {
     if (e.detail.element !== 'my-element') return;
     if (e.detail.event !== 'hide') return;
     e.preventDefault(); // suppress runtime default
     
     var durStr = getComputedStyle(__root)
       .getPropertyValue('--tf-ui-hide-duration').trim();
     var ms = durStr.endsWith('ms') 
       ? parseFloat(durStr) 
       : parseFloat(durStr) * 1000;
     
     __root.classList.add('tf-hiding');
     setTimeout(function() {
       __root.classList.remove('tf-fired', 'tf-hiding');
     }, ms || 600);
   });
   ```

7. **See working examples:**
   - `packs/lower-thirds-pack/lt-slash/element.css` — CSS-only `.tf-hiding` pattern (no element.js)
   - `packs/excuse-me-what/title-intro/element.js` — custom exit with `preventDefault()`

---

### Validation Errors on Save

**Symptom:** Editor shows validation errors when trying to save.

**Cause:** Pack structure or authoring rules violated.

**Solution:**

1. **Check error message carefully:**
   ```
   pack "news-pack", element "lower-third", file "element.html": editable binding "name" is missing required data-tf-label.
   ```

2. **Common validation errors:**

   | Error | Fix |
   |-------|-----|
   | "pack contains no elements" | Create at least one element folder in the pack |
   | "invalid slug format" | Use lowercase kebab-case: `my-element` (no spaces, uppercase, or symbols) |
   | "missing element.html or element.css" | Both files are required for every element |
   | "editable binding missing data-tf-bind or data-tf-label" | Add both attributes to every editable element |
   | "duplicate binding keys" | Each `data-tf-bind` key must be unique within the element |
   | "invalid CSS variable name" | Use `--tf-ui-` prefix with lowercase kebab-case |

3. **Run validation from CLI to see all errors at once:**
   ```bash
   curl "http://localhost:8081/api/validate?pack=<pack-name>"
   ```

4. **Fix errors one by one** and try saving again.

---

### Preview Shows Old Content

**Symptom:** You edit element files, but the editor preview still shows old content.

**Cause:** Browser cache or stale parse.

**Solution:**

1. **Hard-refresh editor preview:**
   - Press Ctrl+Shift+R in the editor preview pane
   - Or use Ctrl+F5 (depends on browser)

2. **Hard-refresh entire editor page:**
   - Go to http://localhost:8081/?pack=<name>&element=<slug>
   - Press Ctrl+Shift+R (force refresh + clear cache)

3. **Close and reopen editor tab:**
   - Close the editor browser tab
   - Open http://localhost:8081 fresh

4. **Clear browser cache:**
   - Press F12 (DevTools)
   - Right-click refresh button, select "Empty cache and hard refresh"

---

## Configuration & Deployment Issues

### Can't Connect to Multiple TitleFlow Instances

**Symptom:** You want to run TitleFlow on different ports but can't access both.

**Cause:** Not specifying custom ports or ports are already in use.

**Solution:**

1. **Specify different ports for each instance:**
   ```bash
   # Instance 1 (default ports)
   ./titleflow.exe -packs "D:\Packs1"
   
   # Instance 2 (different ports, different terminal)
   ./titleflow.exe -packs "D:\Packs2" -runtime-port 9000 -editor-port 9001 -dock-port 9002
   ```

2. **Verify ports are not in use:**
   ```bash
   # Windows: check if port is listening
   netstat -ano | findstr :8080
   ```
   If it shows a process, that port is already in use. Kill the process or use a different port.

3. **Update OBS browser sources:**
   - Instance 1: `http://localhost:8080`
   - Instance 2: `http://localhost:9000`

4. **If running on same machine, use different configs:**
   - Each instance needs its own packs directory
   - Use `-packs` flag to specify separate locations

5. **Remote machines:**
   - If instances are on different machines, use IP addresses
   - Machine A: `http://192.168.1.100:8080`
   - Machine B: `http://192.168.1.101:8080`

---

### Assets Not Loading

**Symptom:** Images, fonts, or other files referenced in elements don't load in OBS.

**Cause:** Asset path is wrong or file is not in the right location.

**Solution:**

1. **Assets must live under the pack root.** Two valid layouts:

   **Pack-level (shared across elements):**
   ```
   packs/
     news-pack/
       assets/
         logo.png
       lower-third/
         element.html
         element.css
   ```

   **Element-local (self-contained, portable):**
   ```
   packs/
     news-pack/
       lower-third/
         assets/
           logo.png
         element.html
         element.css
   ```

2. **Reference via path relative to the pack root** (not the element folder):
   ```html
   <!-- ✓ Pack-level asset -->
   <img src="assets/logo.png" />

   <!-- ✓ Element-local asset — slug prefix is required -->
   <img src="lower-third/assets/logo.png" />

   <!-- ✗ Wrong: absolute path -->
   <img src="/assets/logo.png" />

   <!-- ✗ Wrong: no slug prefix for element-local asset -->
   <img src="assets/logo.png" /> <!-- when file is actually at lower-third/assets/logo.png -->
   ```

3. **Use correct runtime path:**
   ```css
   /* In element.css, same relative path */
   background-image: url(assets/bg.png);
   ```

4. **Avoid reserved folder name:**
   - Do NOT name a folder `assets/` if it should be an element
   - The `assets/` folder is reserved for non-element files
   - Use any other name for element folders (e.g., `my-element`)

5. **Test asset load:**
   ```bash
   # Check if file is served
   curl "http://localhost:8080/?path=assets/logo.png"
   ```
   Should return the file content (binary for images, or error).

6. **Check browser console (F12):**
   - Look for 404 errors on asset URLs
   - Example: `GET http://localhost:8080/assets/logo.png 404`
   - This means the file is not in the pack directory

---

## Performance & Stability

### SSE Connections Keep Dropping

**Symptom:** Controls sometimes stop updating; "SSE failed" appears in console.

**Cause:** Network timeout, slow connection, or system overload.

**Solution:**

1. **Check network connectivity:**
   - Ping the TitleFlow server: `ping localhost`
   - Run a simple test: `curl http://localhost:8080/api/state`

2. **Increase SSE timeout (if possible):**
   - TitleFlow has a 4-message buffer per subscriber
   - If client is very slow, messages may be dropped
   - Ensure OBS is not CPU-throttled

3. **Restart OBS browser source:**
   - Right-click the source in OBS
   - Click **Refresh**
   - Wait 2 seconds for SSE to reconnect

4. **Check system resources:**
   - Open Task Manager (Ctrl+Shift+Esc)
   - Look at CPU and Memory usage
   - If CPU is high, close other applications

5. **Check firewall/antivirus:**
   - Some antivirus software blocks streaming connections
   - Try temporarily disabling to test
   - If that fixes it, add `titleflow.exe` to whitelist

---

### Large Packs Are Slow to Load

**Symptom:** Packs with many elements take a long time to compose or load.

**Cause:** Composition is O(elements); each element requires HTML/CSS parsing and injection.

**Solution:**

1. **Check element count:**
   ```bash
   ls packs\<pack-name>
   ```
   Count the subdirectories (excluding `assets/`).

2. **Optimize CSS size:**
   - Minify CSS in `element.css`
   - Remove unused selectors
   - Avoid repeated values (use CSS variables)

3. **Optimize JS size:**
   - Minify JavaScript in `element.js`
   - Remove debug `console.log()` statements
   - Avoid large library dependencies

4. **Split into multiple packs:**
   - If one pack has 50+ elements, consider splitting
   - Example: `news-pack` and `sports-pack` instead of one mega-pack

5. **Monitor composition time:**
   - Open browser DevTools (F12)
   - Network tab shows response time for `GET /`
   - Typical: < 100ms for small packs, < 500ms for large packs

---

## Debugging Techniques

### View Composed HTML in Browser

```bash
# See exactly what the runtime server is serving
curl http://localhost:8080 > composed.html
# Open in text editor or browser to inspect
```

### Check Live Control State

```bash
# See what controls are currently set
curl http://localhost:8080/api/state
```

**Response shows:**
- All HTML binding values
- All CSS variable overrides
- Persist across page reloads (until app restart)

---

### Validate Pack from CLI

```bash
curl "http://localhost:8081/api/validate?pack=news-pack"
```

**Useful for:**
- Catching all errors at once (not just at save)
- Debugging in scripts or automated builds
- Verifying pack before deployment

---

### Monitor SSE Events

Open http://localhost:8080 in browser and check console:

```javascript
// In browser console (F12)
const es = new EventSource('http://localhost:8080/events');
es.addEventListener('fire', e => console.log('Fire:', e.data));
es.addEventListener('bind', e => console.log('Bind:', e.data));
es.addEventListener('cssvar', e => console.log('CSSVar:', e.data));
es.addEventListener('reset', e => console.log('Reset:', e.data));
es.addEventListener('reload-page', e => console.log('Reload!'));
// Now trigger events in dock, watch console
```

---

### Check Port Availability (Windows)

```bash
# List all listening ports
netstat -ano | findstr LISTENING

# Check specific port (e.g., 8080)
netstat -ano | findstr :8080

# Kill process using a port (find PID above, replace 1234)
taskkill /PID 1234 /F
```

---

### Test API Endpoints with cURL

```bash
# Test runtime server
curl -v http://localhost:8080

# Test dock endpoint
curl -v http://localhost:8082

# Test fire event
curl -v -X POST http://localhost:8080/api/fire -d "element=news-pack--lower-third&event=show"

# Test bind update
curl -v -X POST http://localhost:8080/api/bind -d "element=news-pack--lower-third&key=name&value=TestName"

# Test validation
curl -v "http://localhost:8081/api/validate?pack=news-pack"
```

---

## Getting Help

### Check Logs

TitleFlow logs startup info and errors to stdout. Look for messages like:
```
Starting runtime server on :8080
Starting editor server on :8081
Starting dock server on :8082
Scanned X packs, Y elements
```

### Enable Debug Output (if available)

Currently, TitleFlow does not have a debug flag. To add debugging:
1. Rebuild with debug logging in `main.go`
2. Add `log.Printf()` calls in critical paths
3. Re-run with rebuilt binary

### Report Issues

When reporting a bug, include:
1. TitleFlow version (look in directory path)
2. Operating system (Windows 11, etc.)
3. Port configuration used
4. Steps to reproduce
5. Expected vs. actual behavior
6. Validation output: `curl "http://localhost:8081/api/validate?pack=<name>"`

---

## Related Documentation

- **[API.md](API.md)** — HTTP endpoint reference
- **[ARCHITECTURE.md](ARCHITECTURE.md)** — System design
- **[PACK_SPEC.md](PACK_SPEC.md)** — Pack authoring rules
- **[README.md](README.md)** — Getting started
