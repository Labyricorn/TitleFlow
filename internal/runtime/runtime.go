// Package runtime serves the composed HTML output for the currently active pack.
// This page is intended to be used as a browser source in OBS or similar software.
package runtime

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"titleflow/internal/appstate"
	"titleflow/internal/composer"
	"titleflow/internal/hub"
	"titleflow/internal/models"
)

// Server holds the state for the runtime HTTP server.
type Server struct {
	mu         sync.RWMutex
	activePack *models.Pack
	hub        *hub.Hub
	controls   *appstate.ControlState
}

// New returns a new runtime Server using the given SSE hub and control store.
func New(h *hub.Hub, c *appstate.ControlState) *Server {
	return &Server{hub: h, controls: c}
}

// SetActivePack updates the active pack. It is safe to call concurrently.
func (s *Server) SetActivePack(pack *models.Pack) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activePack = pack
}

// ActivePack returns the currently active pack, or nil if none is set.
func (s *Server) ActivePack() *models.Pack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activePack
}

// Handler returns an http.Handler for the runtime server.
// Call this once and reuse the result.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveRoot)
	mux.HandleFunc("/events", s.serveEvents)
	mux.HandleFunc("/api/state", s.handleState)
	mux.HandleFunc("/api/fire", s.handleFire)
	mux.HandleFunc("/api/reset", s.handleReset)
	mux.HandleFunc("/api/bind", s.handleBind)
	mux.HandleFunc("/api/cssvar", s.handleCSSVar)
	return mux
}

// cors sets permissive CORS headers so the dock page (different port) can POST here.
func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

// serveRoot serves the composed runtime HTML or pack assets.
func (s *Server) serveRoot(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	pack := s.activePack
	s.mu.RUnlock()

	if pack == nil {
		http.Error(w, "No active pack selected.", http.StatusServiceUnavailable)
		return
	}

	// Asset serving supports both:
	// - `GET /assets/logo.png` (path-based)
	// - `GET /?path=assets/logo.png` (query-based; documented)
	if r.URL.Path != "/" || r.URL.Query().Get("path") != "" {
		rel := r.URL.Path[1:]
		if qp := r.URL.Query().Get("path"); qp != "" {
			rel = qp
		}
		rel = filepath.Clean(rel)
		full := filepath.Join(pack.Root, rel)
		if !isUnder(pack.Root, full) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		http.ServeFile(w, r, full)
		return
	}

	html, err := composer.Compose(pack)
	if err != nil {
		log.Printf("runtime: compose error: %v", err)
		http.Error(w, "Composition error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte(html))
}

// serveEvents streams SSE events to connected runtime pages.
func (s *Server) serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	cors(w)

	ch := s.hub.Subscribe()
	defer s.hub.Unsubscribe(ch)

	w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			w.Write([]byte(msg))
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// handleState returns the current live control values as JSON.
// The runtime page fetches this on load to re-apply dock settings after a page reload.
func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	cors(w)
	snap := s.controls.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snap)
}

// handleFire publishes a fire event for the named element.
func (s *Server) handleFire(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	elem := r.FormValue("element")
	if elem == "" {
		http.Error(w, "missing element parameter", http.StatusBadRequest)
		return
	}
	event := r.FormValue("event")
	if event == "" {
		event = "show"
	}
	s.mu.RLock()
	pack := s.activePack
	s.mu.RUnlock()

	if pack == nil {
		http.Error(w, "No active pack selected", http.StatusServiceUnavailable)
		return
	}

	// Namespace element identifier as "pack--element" to prevent collisions
	namespacedElem := pack.Name + "--" + elem
	data, _ := json.Marshal(map[string]string{"element": namespacedElem, "event": event})
	s.hub.Publish("fire", string(data))
	w.WriteHeader(http.StatusNoContent)
}

// handleReset publishes a reset event for the named element.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	elem := r.FormValue("element")
	if elem == "" {
		http.Error(w, "missing element parameter", http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	pack := s.activePack
	s.mu.RUnlock()

	if pack == nil {
		http.Error(w, "No active pack selected", http.StatusServiceUnavailable)
		return
	}

	// Namespace element identifier as "pack--element" to prevent collisions
	namespacedElem := pack.Name + "--" + elem
	data, _ := json.Marshal(map[string]string{"element": namespacedElem})
	s.hub.Publish("reset", string(data))
	w.WriteHeader(http.StatusNoContent)
}

// handleBind persists and broadcasts a new text value for an HTML binding.
func (s *Server) handleBind(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	elem := r.FormValue("element")
	key := r.FormValue("key")
	value := r.FormValue("value")
	if elem == "" || key == "" {
		http.Error(w, "missing element or key parameter", http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	pack := s.activePack
	s.mu.RUnlock()

	if pack == nil {
		http.Error(w, "No active pack selected", http.StatusServiceUnavailable)
		return
	}

	// Namespace element identifier as "pack--element" to prevent collisions
	namespacedElem := pack.Name + "--" + elem
	s.controls.SetBind(namespacedElem, key, value)
	data, _ := json.Marshal(map[string]string{"element": namespacedElem, "key": key, "value": value})
	s.hub.Publish("bind", string(data))
	w.WriteHeader(http.StatusNoContent)
}

// handleCSSVar persists and broadcasts a new value for a published CSS custom property.
func (s *Server) handleCSSVar(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	_ = r.ParseForm()
	elem := r.FormValue("element")
	name := r.FormValue("name")
	value := r.FormValue("value")
	if elem == "" || name == "" {
		http.Error(w, "missing element or name parameter", http.StatusBadRequest)
		return
	}
	s.mu.RLock()
	pack := s.activePack
	s.mu.RUnlock()

	if pack == nil {
		http.Error(w, "No active pack selected", http.StatusServiceUnavailable)
		return
	}

	// Namespace element identifier as "pack--element" to prevent collisions
	namespacedElem := pack.Name + "--" + elem
	s.controls.SetCSSVar(namespacedElem, name, value)
	data, _ := json.Marshal(map[string]string{"element": namespacedElem, "name": name, "value": value})
	s.hub.Publish("cssvar", string(data))
	w.WriteHeader(http.StatusNoContent)
}

// isUnder returns true if child is inside parent (both should be cleaned paths).
func isUnder(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	if rel == ".." {
		return false
	}
	prefix := ".." + string(os.PathSeparator)
	if strings.HasPrefix(rel, prefix) {
		return false
	}
	return true
}
