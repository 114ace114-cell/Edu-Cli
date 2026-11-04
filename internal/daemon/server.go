package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Server is the local HTTP API. The full mux is internal/daemon/server.go.
type Server struct {
	runner *Runner
}

func NewServer(runner *Runner) *Server {
	return &Server{runner: runner}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /status", s.handleStatus)
	mux.HandleFunc("GET /sessions", s.handleSessions)
	mux.HandleFunc("GET /sessions/{id}", s.handleSession)
	mux.HandleFunc("POST /message", s.handleMessage)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleStatus(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "ok",
		"workspace":    s.runner.Cfg.Workspace,
		"capabilities": []string{"skeleton_v1"},
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, _ *http.Request) {
	ids, err := s.runner.Sessions.List()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": ids})
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := ValidateSessionID(id); err != nil || id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid session_id"})
		return
	}
	sess, err := s.runner.Sessions.Load(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	var req Request
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad body"})
		return
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "text is required"})
		return
	}
	if err := ValidateSessionID(req.SessionID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if wantsStream(r) {
		s.handleMessageSSE(w, req)
		return
	}
	result, err := s.runner.RunAgent(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error(), "session_id": result.SessionID})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleMessageSSE(w http.ResponseWriter, req Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	result, err := s.runner.RunAgent(req)
	if err != nil {
		writeSSE(w, "error", map[string]string{"type": "error", "error": err.Error()})
		flusher.Flush()
		return
	}
	writeSSE(w, "done", map[string]string{
		"type":       "done",
		"reply":      result.Reply,
		"session_id": result.SessionID,
	})
	flusher.Flush()
}

func wantsStream(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream") || r.URL.Query().Get("stream") == "1"
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	raw, _ := json.Marshal(payload)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, raw)
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
