// TitleFlow — local-first overlay pack builder and runtime server.
//
// Three HTTP surfaces:
//   - :8080  Runtime server — serves the composed pack output for OBS browser source
//   - :8081  Editor UI     — code-first pack editor with preview and inspector
//   - :8082  Dock UI       — pack selector and explicit control surface
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"titleflow/internal/appstate"
	"titleflow/internal/dock"
	"titleflow/internal/editor"
	"titleflow/internal/hub"
	"titleflow/internal/runtime"
)

func main() {
	// Determine the default packs directory relative to the executable.
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	packsDir := flag.String("packs", filepath.Join(exeDir, "packs"), "path to the directory containing pack folders")
	runtimePort := flag.String("runtime-port", "8080", "port for the runtime server (OBS browser source)")
	editorPort := flag.String("editor-port", "8081", "port for the editor UI")
	dockPort := flag.String("dock-port", "8082", "port for the dock UI")
	flag.Parse()

	// Ensure the packs directory exists.
	if err := os.MkdirAll(*packsDir, 0755); err != nil {
		log.Fatalf("Cannot create packs directory %q: %v", *packsDir, err)
	}

	log.Printf("TitleFlow starting. Packs directory: %s", *packsDir)

	// Initialise shared application state (scans packs directory).
	state, err := appstate.New(*packsDir)
	if err != nil {
		log.Fatalf("Failed to initialise application state: %v", err)
	}

	packs := state.AllPacks()
	log.Printf("Found %d pack(s).", len(packs))
	for _, p := range packs {
		log.Printf("  pack: %s (%d element(s))", p.Name, len(p.Elements))
	}

	// Verify active pack is set
	if activePack := state.ActivePack(); activePack != nil {
		log.Printf("Active pack set to: %s", activePack.Name)
	} else {
		log.Printf("WARNING: No pack is active!")
	}

	// The SSE hub is shared across the lifetime of the process.
	// The runtime server uses it to push fire/reset events to connected OBS pages.
	sseHub := hub.New()

	// The runtime server is created once. Its handler is cached and reused.
	runtimeSrv := runtime.New(sseHub, state.Controls)
	runtimeHandler := &activePackHandler{
		state:   state,
		srv:     runtimeSrv,
		handler: runtimeSrv.Handler(),
	}

	// Start all three servers.
	errCh := make(chan error, 3)

	go func() {
		addr := ":" + *runtimePort
		log.Printf("Runtime server listening at http://localhost%s (use as OBS browser source)", addr)
		errCh <- http.ListenAndServe(addr, runtimeHandler)
	}()

	go func() {
		addr := ":" + *editorPort
		log.Printf("Editor UI listening at http://localhost%s", addr)
		errCh <- http.ListenAndServe(addr, editor.Handler(state, sseHub))
	}()

	go func() {
		addr := ":" + *dockPort
		log.Printf("Dock UI listening at http://localhost%s", addr)
		errCh <- http.ListenAndServe(addr, dock.Handler(state, sseHub))
	}()

	fmt.Println()
	fmt.Println("TitleFlow is running.")
	fmt.Printf("  Runtime (OBS source) → http://localhost:%s\n", *runtimePort)
	fmt.Printf("  Editor               → http://localhost:%s\n", *editorPort)
	fmt.Printf("  Dock                 → http://localhost:%s\n", *dockPort)
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop.")

	// Block until one of the servers fails.
	if err := <-errCh; err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// activePackHandler keeps the runtime server's active pack in sync with the
// application state on every request, then delegates to the cached handler.
type activePackHandler struct {
	state   *appstate.State
	srv     *runtime.Server
	handler http.Handler
}

func (h *activePackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.srv.SetActivePack(h.state.ActivePack())
	h.handler.ServeHTTP(w, r)
}
