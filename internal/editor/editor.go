// Package editor serves the Pack Editor UI.
// The editor presents HTML, CSS, and optional JS panels alongside an output log,
// an inspector, and inline dock controls that target the runtime server directly.
package editor

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"titleflow/internal/composer"
	"titleflow/internal/hub"
	"titleflow/internal/models"
	"titleflow/internal/scanner"
	"titleflow/internal/store"
	"titleflow/internal/validator"
)

// StateManager is the interface the editor needs from the application.
type StateManager interface {
	AllPacks() []*models.Pack
	ActivePack() *models.Pack
	SetActivePack(name string) error
	ReloadPacks() error
	PacksDir() string
}

// Handler returns an http.Handler for the editor UI.
func Handler(state StateManager, h *hub.Hub) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveEditor(w, r, state)
	})
	mux.HandleFunc("/api/preview", func(w http.ResponseWriter, r *http.Request) {
		apiPreview(w, r, state)
	})
	mux.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		apiSave(w, r, state)
	})
	mux.HandleFunc("/api/new-element", func(w http.ResponseWriter, r *http.Request) {
		apiNewElement(w, r, state)
	})
	mux.HandleFunc("/api/new-pack", func(w http.ResponseWriter, r *http.Request) {
		apiNewPack(w, r, state)
	})
	mux.HandleFunc("/api/validate", func(w http.ResponseWriter, r *http.Request) {
		apiValidate(w, r, state)
	})
	mux.HandleFunc("/api/controls", func(w http.ResponseWriter, r *http.Request) {
		apiControls(w, r, state)
	})
	mux.HandleFunc("/api/set-active", func(w http.ResponseWriter, r *http.Request) {
		apiSetActive(w, r, state, h)
	})
	mux.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) {
		apiReload(w, r, state, h)
	})
	return mux
}

// apiSetActive updates the shared active pack and broadcasts a reload-page
// event to connected overlays, mirroring the dock's /api/set-active behavior.
// Responds with 204 so fetch() callers don't follow a redirect.
func apiSetActive(w http.ResponseWriter, r *http.Request, state StateManager, h *hub.Hub) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	name := r.FormValue("pack")
	if name == "" {
		http.Error(w, "missing pack name", http.StatusBadRequest)
		return
	}
	if err := state.SetActivePack(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.Publish("reload-page", "{}")
	w.WriteHeader(http.StatusNoContent)
}

// apiReload triggers a full disk reload of packs and broadcasts to the runtime.
func apiReload(w http.ResponseWriter, r *http.Request, state StateManager, h *hub.Hub) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := state.ReloadPacks(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.Publish("reload-page", "{}")
	w.WriteHeader(http.StatusNoContent)
}

// serveEditor renders the main editor page.
func serveEditor(w http.ResponseWriter, r *http.Request, state StateManager) {
	packs := state.AllPacks()
	active := state.ActivePack()

	selectedPack := r.URL.Query().Get("pack")
	selectedElem := r.URL.Query().Get("element")
	hidePack := r.URL.Query().Get("hidepack") == "true"

	// Find the requested pack or fall back to the active pack.
	var currentPack *models.Pack
	for _, p := range packs {
		if p.Name == selectedPack {
			currentPack = p
			break
		}
	}
	if currentPack == nil {
		currentPack = active
	}

	// Find the selected element.
	var currentElem *models.Element
	if currentPack != nil {
		for _, e := range currentPack.Elements {
			if e.Slug == selectedElem {
				currentElem = e
				break
			}
		}
		if currentElem == nil && len(currentPack.Elements) > 0 {
			currentElem = currentPack.Elements[0]
		}
	}

	data := struct {
		Packs       []*models.Pack
		ActiveName  string
		CurrentPack *models.Pack
		CurrentElem *models.Element
		HidePack    bool
	}{
		Packs: packs,
	}
	if active != nil {
		data.ActiveName = active.Name
	}
	data.CurrentPack = currentPack
	data.CurrentElem = currentElem
	data.HidePack = hidePack

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := editorTmpl.Execute(w, data); err != nil {
		log.Printf("editor: template error: %v", err)
	}
}

// apiPreview composes and returns the full runtime HTML for the current pack.
// This uses the exact same composition logic as the runtime server.
func apiPreview(w http.ResponseWriter, r *http.Request, state StateManager) {
	packName := r.URL.Query().Get("pack")

	var pack *models.Pack
	for _, p := range state.AllPacks() {
		if p.Name == packName {
			pack = p
			break
		}
	}
	if pack == nil {
		pack = state.ActivePack()
	}
	if pack == nil {
		http.Error(w, "No pack selected", http.StatusBadRequest)
		return
	}

	// Re-scan so the preview reflects the latest saved files.
	freshPack, err := scanner.ScanPack(pack.Root)
	if err != nil {
		http.Error(w, "Scan error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	html, err := composer.Compose(freshPack)
	if err != nil {
		http.Error(w, "Compose error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(html))
}

// apiSave writes updated element source files to disk.
func apiSave(w http.ResponseWriter, r *http.Request, state StateManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form data", http.StatusBadRequest)
		return
	}

	packName := r.FormValue("pack")
	elemSlug := r.FormValue("element")
	htmlSrc := r.FormValue("html")
	cssSrc := r.FormValue("css")
	jsSrc := r.FormValue("js")
	hasJS := r.FormValue("has_js") == "true"

	var pack *models.Pack
	for _, p := range state.AllPacks() {
		if p.Name == packName {
			pack = p
			break
		}
	}
	if pack == nil {
		http.Error(w, "pack not found", http.StatusBadRequest)
		return
	}

	elemDir := filepath.Join(pack.Root, elemSlug)

	if err := store.WriteFile(filepath.Join(elemDir, "element.html"), htmlSrc); err != nil {
		http.Error(w, "write html: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := store.WriteFile(filepath.Join(elemDir, "element.css"), cssSrc); err != nil {
		http.Error(w, "write css: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if hasJS {
		if err := store.WriteFile(filepath.Join(elemDir, "element.js"), jsSrc); err != nil {
			http.Error(w, "write js: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	state.ReloadPacks()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// apiNewElement creates a new element folder with the standard file set.
func apiNewElement(w http.ResponseWriter, r *http.Request, state StateManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	packName := r.FormValue("pack")
	slug := strings.TrimSpace(r.FormValue("slug"))

	var pack *models.Pack
	for _, p := range state.AllPacks() {
		if p.Name == packName {
			pack = p
			break
		}
	}
	if pack == nil {
		http.Error(w, "pack not found", http.StatusBadRequest)
		return
	}

	if err := store.CreateElement(pack.Root, slug); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state.ReloadPacks()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// apiNewPack creates a new pack root directory.
func apiNewPack(w http.ResponseWriter, r *http.Request, state StateManager) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	packName := strings.TrimSpace(r.FormValue("name"))
	if packName == "" {
		http.Error(w, "missing pack name", http.StatusBadRequest)
		return
	}

	if err := store.CreatePack(state.PacksDir(), packName); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	state.ReloadPacks()
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// apiValidate runs the validator on the named pack and returns plain-text results.
func apiValidate(w http.ResponseWriter, r *http.Request, state StateManager) {
	packName := r.URL.Query().Get("pack")

	var pack *models.Pack
	for _, p := range state.AllPacks() {
		if p.Name == packName {
			pack = p
			break
		}
	}
	if pack == nil {
		pack = state.ActivePack()
	}
	if pack == nil {
		http.Error(w, "No pack selected", http.StatusBadRequest)
		return
	}

	// Fresh scan before validating.
	freshPack, err := scanner.ScanPack(pack.Root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	result := validator.ValidatePack(freshPack)

	type response struct {
		OK       bool     `json:"ok"`
		Errors   []string `json:"errors"`
		Warnings []string `json:"warnings"`
	}
	resp := response{
		OK:       result.OK(),
		Errors:   result.Errors,
		Warnings: result.Warnings,
	}
	if resp.Errors == nil {
		resp.Errors = []string{}
	}
	if resp.Warnings == nil {
		resp.Warnings = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// apiControls parses and returns the derived controls for the selected element.
func apiControls(w http.ResponseWriter, r *http.Request, state StateManager) {
	packName := r.URL.Query().Get("pack")
	elemSlug := r.URL.Query().Get("element")

	var pack *models.Pack
	for _, p := range state.AllPacks() {
		if p.Name == packName {
			pack = p
			break
		}
	}
	if pack == nil {
		pack = state.ActivePack()
	}

	var elem *models.Element
	if pack != nil {
		for _, e := range pack.Elements {
			if e.Slug == elemSlug {
				elem = e
				break
			}
		}
	}
	if elem == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"bindings":[],"css_vars":[]}`))
		return
	}

	type bindingOut struct {
		Key          string `json:"key"`
		Label        string `json:"label"`
		DefaultValue string `json:"default_value"`
	}
	type cssVarOut struct {
		Name         string `json:"name"`
		Value        string `json:"value"`
		Label        string `json:"label"`
		InferredType string `json:"inferred_type"`
	}
	type response struct {
		Bindings []bindingOut `json:"bindings"`
		CSSVars  []cssVarOut  `json:"css_vars"`
	}

	resp := response{}
	for _, b := range elem.Bindings {
		resp.Bindings = append(resp.Bindings, bindingOut{b.Key, b.Label, b.DefaultValue})
	}
	for _, c := range elem.CSSVars {
		resp.CSSVars = append(resp.CSSVars, cssVarOut{c.Name, c.Value, c.Label, c.InferredType})
	}
	if resp.Bindings == nil {
		resp.Bindings = []bindingOut{}
	}
	if resp.CSSVars == nil {
		resp.CSSVars = []cssVarOut{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// ─── Template helper functions ────────────────────────────────────────────────
// These are used by the editor template to render inline dock controls.
// Ported from the dock package so the editor package stays independent.

func labelFromType(t string) string {
	switch t {
	case "color":
		return "Color"
	case "length":
		return "Length"
	case "duration":
		return "Duration"
	default:
		return "Text"
	}
}

func parseLengthValue(val string) string {
	val = strings.TrimSpace(val)
	for _, unit := range []string{"rem", "vw", "vh", "ch", "px", "em", "pt", "%"} {
		if strings.HasSuffix(val, unit) {
			return strings.TrimSuffix(val, unit)
		}
	}
	return val
}

func extractLengthUnit(val string) string {
	val = strings.TrimSpace(val)
	for _, unit := range []string{"rem", "vw", "vh", "ch", "px", "em", "pt", "%"} {
		if strings.HasSuffix(val, unit) {
			return unit
		}
	}
	return "px"
}

func parseDurationValue(val string) string {
	val = strings.TrimSpace(val)
	val = strings.TrimSuffix(val, "ms")
	val = strings.TrimSuffix(val, "s")
	if val == "" {
		return "0"
	}
	return val
}

func extractDurationUnit(val string) string {
	val = strings.TrimSpace(val)
	if strings.HasSuffix(val, "ms") {
		return "ms"
	}
	if strings.HasSuffix(val, "s") {
		return "s"
	}
	return "ms"
}



var editorFuncs = template.FuncMap{
	"typeLabel":           labelFromType,
	"parseDurationValue":  parseDurationValue,
	"extractDurationUnit": extractDurationUnit,
	"parseLengthValue":    parseLengthValue,
	"extractLengthUnit":   extractLengthUnit,
	"join":                strings.Join,
	"sprintf":             fmt.Sprintf,
}

var editorTmpl = template.Must(template.New("editor").Funcs(editorFuncs).Parse(editorHTML))
