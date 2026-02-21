package zmosh

import (
	"strconv"
	"strings"
)

// ParseSessions parses the tab-separated output of `zmosh list`.
// Each line has fields like: session_name=foo\tpid=123\tclients=1\tstarted_in=~/bar
// Lines may have leading whitespace or a → prefix for the current session.
func ParseSessions(output string) []Session {
	var sessions []Session

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		// Strip → prefix (current session indicator)
		line = strings.TrimPrefix(line, "\u2192 ")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var s Session
		s.StartedIn = "~" // default

		for _, field := range strings.Split(line, "\t") {
			field = strings.TrimSpace(field)
			if k, v, ok := strings.Cut(field, "="); ok {
				switch k {
				case "session_name":
					s.Name = v
				case "pid":
					s.PID, _ = strconv.Atoi(v)
				case "clients":
					s.Clients, _ = strconv.Atoi(v)
				case "started_in":
					s.StartedIn = v
				}
			}
		}

		if s.Name == "" {
			continue
		}
		s.Active = s.Clients > 0
		sessions = append(sessions, s)
	}

	return sessions
}
