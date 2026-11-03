package daemon

import "fmt"

// RouteKey is the sticky lane for a request.
// The full function is daemon.ComputeRouteKey: pinned schedule, session id,
// thread, sender, agent, then channel. This skeleton keeps the session pin,
// which is the rule one-shot and HTTP resume share.
func RouteKey(sessionID string) string {
	if sessionID == "" {
		return "new"
	}
	return "session:" + sessionID
}

// ValidateSessionID rejects ids that would escape the session directory.
func ValidateSessionID(id string) error {
	if id == "" {
		return nil
	}
	for _, r := range id {
		ok := r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !ok {
			return fmt.Errorf("invalid session_id")
		}
	}
	if len(id) > 64 {
		return fmt.Errorf("invalid session_id")
	}
	return nil
}
