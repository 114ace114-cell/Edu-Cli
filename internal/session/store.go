package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kocoro/edu-cli/internal/client"
)

// Session is the durable transcript. The full store is internal/session
// (JSON files plus a SQLite FTS index). This store is one JSON file per id.
type Session struct {
	ID        string           `json:"id"`
	RouteKey  string           `json:"route_key"`
	Title     string           `json:"title,omitempty"`
	CWD       string           `json:"cwd,omitempty"`
	Messages  []client.Message `json:"messages"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// SessionSummary is one row for /sessions and the session picker.
type SessionSummary struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MsgCount  int       `json:"msg_count"`
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// ValidID reports whether a client-supplied session id is safe to use as a filename.
func ValidID(id string) bool {
	return id == "" || idPattern.MatchString(id)
}

// Store persists sessions under dir.
type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) LoadOrCreate(id, routeKey string) (*Session, error) {
	if id == "" {
		id = newID()
		return &Session{ID: id, RouteKey: routeKey, CreatedAt: time.Now().UTC()}, nil
	}
	raw, err := os.ReadFile(s.path(id))
	if os.IsNotExist(err) {
		return &Session{ID: id, RouteKey: routeKey, CreatedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, err
	}
	if routeKey != "" {
		sess.RouteKey = routeKey
	}
	return &sess, nil
}

func (s *Store) Save(sess *Session) error {
	sess.UpdatedAt = time.Now().UTC()
	raw, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}
	path := s.path(sess.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ListSummaries returns sessions sorted by UpdatedAt, newest first.
func (s *Store) ListSummaries() ([]SessionSummary, error) {
	ids, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make([]SessionSummary, 0, len(ids))
	for _, id := range ids {
		sess, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, SessionSummary{
			ID:        sess.ID,
			Title:     DisplayTitle(sess),
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
			MsgCount:  len(sess.Messages),
		})
	}
	sortSessionSummaries(out)
	return out, nil
}

func sortSessionSummaries(items []SessionSummary) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && items[j].UpdatedAt.After(items[j-1].UpdatedAt) {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

// Delete removes a session file.
func (s *Store) Delete(id string) error {
	if !ValidID(id) || id == "" {
		return fmt.Errorf("invalid session id")
	}
	return os.Remove(s.path(id))
}

func (s *Store) Load(id string) (*Session, error) {
	raw, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(raw, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

// Summary is one row in the startup "Recent activity" list.
type Summary struct {
	Title     string
	UpdatedAt time.Time
	MsgCount  int
}

// Recent returns the newest sessions, newest first.
func (s *Store) Recent(limit int) ([]Summary, error) {
	if limit <= 0 {
		return nil, nil
	}
	ids, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(ids))
	for _, id := range ids {
		sess, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, Summary{
			Title:     DisplayTitle(sess),
			UpdatedAt: sess.UpdatedAt,
			MsgCount:  len(sess.Messages),
		})
	}
	sortSummaries(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// DisplayTitle returns the stored title or derives one from the first user message.
func DisplayTitle(sess *Session) string {
	if sess == nil {
		return "New session"
	}
	if t := strings.TrimSpace(sess.Title); t != "" && t != "New session" {
		r := []rune(t)
		if len(r) > 48 {
			return string(r[:48]) + "..."
		}
		return t
	}
	return titleFromMessages(sess.Messages, sess.ID)
}

// TitleFromUserLine trims a compose line for use as session title.
func TitleFromUserLine(text string) string {
	text = strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if text == "" {
		return ""
	}
	r := []rune(text)
	if len(r) > 48 {
		return string(r[:48]) + "..."
	}
	return text
}

func titleFromMessages(msgs []client.Message, fallbackID string) string {
	for _, m := range msgs {
		if m.Role != "user" {
			continue
		}
		text := strings.Join(strings.Fields(m.Content), " ")
		if text == "" {
			continue
		}
		r := []rune(text)
		if len(r) > 48 {
			return string(r[:48]) + "..."
		}
		return text
	}
	if fallbackID != "" {
		return fallbackID
	}
	return "New session"
}

func sortSummaries(items []Summary) {
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && items[j].UpdatedAt.After(items[j-1].UpdatedAt) {
			items[j], items[j-1] = items[j-1], items[j]
			j--
		}
	}
}

func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if filepath.Ext(name) == ".json" {
			ids = append(ids, name[:len(name)-len(".json")])
		}
	}
	return ids, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
