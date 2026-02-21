package check

import (
	"encoding/json"
	"testing"
)

func TestCheckResultJSON(t *testing.T) {
	result := Result{
		Zmosh:  DepStatus{Installed: true, Version: "0.4.2", Path: "/opt/homebrew/bin/zmosh"},
		Zoxide: DepStatus{Installed: false},
		Fzf:    DepStatus{Installed: false},
		Shell:  "zsh",
		OS:     "darwin",
		Arch:   "arm64",
	}
	j, err := result.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(j) == 0 {
		t.Error("expected non-empty JSON")
	}

	// Verify it's valid JSON that round-trips
	var parsed Result
	if err := json.Unmarshal([]byte(j), &parsed); err != nil {
		t.Fatalf("JSON didn't round-trip: %v", err)
	}
	if !parsed.Zmosh.Installed {
		t.Error("expected zmosh installed after round-trip")
	}
	if parsed.Zmosh.Version != "0.4.2" {
		t.Errorf("expected version 0.4.2, got %s", parsed.Zmosh.Version)
	}
	if parsed.Shell != "zsh" {
		t.Errorf("expected shell zsh, got %s", parsed.Shell)
	}
}

func TestCheckResultJSON_AllInstalled(t *testing.T) {
	result := Result{
		Zmosh:  DepStatus{Installed: true, Version: "0.4.2", Path: "/opt/homebrew/bin/zmosh"},
		Zoxide: DepStatus{Installed: true, Version: "0.9.4", Path: "/opt/homebrew/bin/zoxide"},
		Fzf:    DepStatus{Installed: true, Version: "0.46.0", Path: "/opt/homebrew/bin/fzf"},
		Shell:  "zsh",
		OS:     "darwin",
		Arch:   "arm64",
	}
	j, err := result.JSON()
	if err != nil {
		t.Fatal(err)
	}

	var parsed Result
	if err := json.Unmarshal([]byte(j), &parsed); err != nil {
		t.Fatalf("JSON didn't round-trip: %v", err)
	}
	if !parsed.Zoxide.Installed || !parsed.Fzf.Installed {
		t.Error("expected all deps installed after round-trip")
	}
}

func TestDetectShell(t *testing.T) {
	shell := detectShell()
	if shell == "" {
		t.Error("expected non-empty shell")
	}
}
