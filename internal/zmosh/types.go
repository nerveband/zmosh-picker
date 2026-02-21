package zmosh

// Session represents a zmosh session from `zmosh list` output.
type Session struct {
	Name      string `json:"name"`
	PID       int    `json:"pid,omitempty"`
	Clients   int    `json:"clients"`
	StartedIn string `json:"started_in"`
	Active    bool   `json:"active"`
}

// ListResult is the JSON output format for `zmosh-picker list --json`.
type ListResult struct {
	Sessions     []Session `json:"sessions"`
	Count        int       `json:"count"`
	ZmoshVersion string    `json:"zmosh_version,omitempty"`
}
