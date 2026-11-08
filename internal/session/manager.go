package session

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// Manager tracks the in-process current session (Kocoro-style harness).
type Manager struct {
	store   *Store
	current *Session
}

// NewManager wraps an existing store (same dir as daemon.Runner.Sessions).
func NewManager(store *Store) *Manager {
	return &Manager{store: store}
}

// NewSession creates a new empty session and persists it.
func (m *Manager) NewSession() (*Session, error) {
	cwd, _ := os.Getwd()
	m.current = &Session{
		ID:        newSessionID(),
		Title:     "New session",
		CWD:       cwd,
		RouteKey:  "new",
		CreatedAt: time.Now().UTC(),
	}
	if err := m.store.Save(m.current); err != nil {
		return nil, err
	}
	return m.current, nil
}

func (m *Manager) Current() *Session {
	return m.current
}

// Resume loads a session from disk and sets it current.
func (m *Manager) Resume(id string) (*Session, error) {
	sess, err := m.store.Load(id)
	if err != nil {
		return nil, err
	}
	m.current = sess
	return sess, nil
}

// RefreshCurrent reloads the current session from disk after Runner saves transcript.
func (m *Manager) RefreshCurrent() error {
	if m.current == nil || m.current.ID == "" {
		return nil
	}
	savedTitle := strings.TrimSpace(m.current.Title)
	sess, err := m.store.Load(m.current.ID)
	if err != nil {
		return err
	}
	if savedTitle != "" && savedTitle != "New session" {
		sess.Title = savedTitle
	}
	m.current = sess
	return nil
}

// Save writes the current session.
func (m *Manager) Save() error {
	if m.current == nil {
		return nil
	}
	return m.store.Save(m.current)
}

// List returns session summaries for pickers.
func (m *Manager) List() ([]SessionSummary, error) {
	return m.store.ListSummaries()
}

// Delete removes a session file and clears current if it matches.
func (m *Manager) Delete(id string) error {
	if err := m.store.Delete(id); err != nil {
		return err
	}
	if m.current != nil && m.current.ID == id {
		m.current = nil
	}
	return nil
}

// ResumeByListIndex resumes the session at a 1-based index from List().
func (m *Manager) ResumeByListIndex(index int) (*Session, error) {
	summaries, err := m.List()
	if err != nil {
		return nil, err
	}
	if index < 1 || index > len(summaries) {
		return nil, fmt.Errorf("session index out of range")
	}
	return m.Resume(summaries[index-1].ID)
}

func newSessionID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return time.Now().UTC().Format("2006-01-02-150405")
	}
	return fmt.Sprintf("%s-%s", time.Now().UTC().Format("2006-01-02"), hex.EncodeToString(b))
}
