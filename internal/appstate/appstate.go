// Package appstate holds the shared application state: the list of known packs
// and which one is currently active. It coordinates between the dock, editor,
// and runtime server without introducing a database.
package appstate

import (
	"fmt"
	"sync"

	"titleflow/internal/models"
	"titleflow/internal/scanner"
	"titleflow/internal/validator"
)

// State is the shared application state. It is safe to use from multiple goroutines.
type State struct {
	mu         sync.RWMutex
	packsDir   string
	packs      []*models.Pack
	activePack *models.Pack
	Controls   *ControlState
}

// New creates a State and performs an initial scan of packsDir.
func New(packsDir string) (*State, error) {
	s := &State{
		packsDir: packsDir,
		Controls: NewControlState(),
	}
	if err := s.ReloadPacks(); err != nil {
		return nil, err
	}
	// Set the first pack as active by default, if any.
	s.mu.Lock()
	if len(s.packs) > 0 {
		s.activePack = s.packs[0]
	}
	s.mu.Unlock()
	return s, nil
}

// PacksDir returns the packs directory path.
func (s *State) PacksDir() string {
	return s.packsDir
}

// ReloadPacks re-scans the packs directory and updates the pack list.
// The previously active pack is re-linked by name if it still exists.
func (s *State) ReloadPacks() error {
	packs, err := scanner.ScanPackDir(s.packsDir)
	if err != nil {
		return fmt.Errorf("appstate: scan packs dir: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	prevActiveName := ""
	if s.activePack != nil {
		prevActiveName = s.activePack.Name
	}

	s.packs = packs
	s.activePack = nil

	// Re-link active pack by name.
	for _, p := range s.packs {
		if p.Name == prevActiveName {
			s.activePack = p
			break
		}
	}

	// Per design docs: live control state is ephemeral and resets on reload.
	s.Controls.Reset()

	return nil
}

// AllPacks returns all known packs. Callers must not mutate the slice.
func (s *State) AllPacks() []*models.Pack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Pack, len(s.packs))
	copy(out, s.packs)
	return out
}

// ActivePack returns the currently active pack, or nil.
func (s *State) ActivePack() *models.Pack {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activePack
}

// SetActivePack sets the active pack by name.
// Returns an error if no pack with that name exists.
func (s *State) SetActivePack(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, p := range s.packs {
		if p.Name == name {
			s.activePack = p
			return nil
		}
	}
	return fmt.Errorf("appstate: no pack named %q", name)
}

// ValidatePack runs validation on the given pack and returns the result.
func (s *State) ValidatePack(pack *models.Pack) *models.ValidationResult {
	return validator.ValidatePack(pack)
}

// ─── ControlState ─────────────────────────────────────────────────────────────

// ControlState stores the current live values for HTML bindings and CSS vars.
// Values are held in memory only — they reset when TitleFlow restarts.
// It is safe to use from multiple goroutines.
type ControlState struct {
	mu      sync.RWMutex
	binds   map[string]map[string]string // "pack--element" → bindKey → value
	cssVars map[string]map[string]string // "pack--element" → varName  → value
}

// NewControlState returns a ready-to-use ControlState.
func NewControlState() *ControlState {
	return &ControlState{
		binds:   make(map[string]map[string]string),
		cssVars: make(map[string]map[string]string),
	}
}

// Reset clears all live values. It is safe to call concurrently.
func (c *ControlState) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.binds = make(map[string]map[string]string)
	c.cssVars = make(map[string]map[string]string)
}

// SetBind stores a live value for an HTML binding.
func (c *ControlState) SetBind(slug, key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.binds[slug] == nil {
		c.binds[slug] = make(map[string]string)
	}
	c.binds[slug][key] = value
}

// SetCSSVar stores a live value for a published CSS custom property.
func (c *ControlState) SetCSSVar(slug, name, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cssVars[slug] == nil {
		c.cssVars[slug] = make(map[string]string)
	}
	c.cssVars[slug][name] = value
}

// ControlSnapshot is a JSON-serialisable deep copy of the current control values.
type ControlSnapshot struct {
	Binds   map[string]map[string]string `json:"binds"`
	CSSVars map[string]map[string]string `json:"css_vars"`
}

// Snapshot returns a deep copy of the current control values.
func (c *ControlState) Snapshot() ControlSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	snap := ControlSnapshot{
		Binds:   make(map[string]map[string]string),
		CSSVars: make(map[string]map[string]string),
	}
	for slug, keys := range c.binds {
		m := make(map[string]string, len(keys))
		for k, v := range keys {
			m[k] = v
		}
		snap.Binds[slug] = m
	}
	for slug, vars := range c.cssVars {
		m := make(map[string]string, len(vars))
		for k, v := range vars {
			m[k] = v
		}
		snap.CSSVars[slug] = m
	}
	return snap
}
