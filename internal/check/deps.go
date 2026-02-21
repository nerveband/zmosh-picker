package check

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DepStatus represents the installation status of a dependency.
type DepStatus struct {
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	Path      string `json:"path,omitempty"`
}

// Result represents the full dependency check result.
type Result struct {
	Zmosh  DepStatus `json:"zmosh"`
	Zoxide DepStatus `json:"zoxide"`
	Fzf    DepStatus `json:"fzf"`
	Shell  string    `json:"shell"`
	OS     string    `json:"os"`
	Arch   string    `json:"arch"`
}

// JSON returns the result as indented JSON.
func (r Result) JSON() (string, error) {
	b, err := json.MarshalIndent(r, "", "  ")
	return string(b), err
}

// Run checks all dependencies and returns the result.
func Run() Result {
	r := Result{
		Shell: detectShell(),
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	}
	r.Zmosh = checkDep("zmosh", "version")
	r.Zoxide = checkDep("zoxide", "--version")
	r.Fzf = checkDep("fzf", "--version")
	return r
}

func checkDep(name, versionFlag string) DepStatus {
	path, err := exec.LookPath(name)
	if err != nil {
		return DepStatus{Installed: false}
	}
	status := DepStatus{Installed: true, Path: path}
	if out, err := exec.Command(name, versionFlag).Output(); err == nil {
		status.Version = strings.TrimSpace(string(out))
	}
	return status
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell)
	}
	return "unknown"
}

// PrintHuman prints the check result in a human-readable format.
func (r Result) PrintHuman() {
	printDep("zmosh", r.Zmosh, true)
	printDep("zoxide", r.Zoxide, false)
	printDep("fzf", r.Fzf, false)
	fmt.Printf("\nPlatform: %s/%s, Shell: %s\n", r.OS, r.Arch, r.Shell)
}

func printDep(name string, d DepStatus, required bool) {
	status := "\u2713"
	if !d.Installed {
		if required {
			status = "\u2717"
		} else {
			status = "\u25CB"
		}
	}
	label := " (optional)"
	if required {
		label = " (required)"
	}
	if d.Installed {
		fmt.Printf("  %s %s%s \u2014 %s\n", status, name, label, d.Version)
	} else {
		fmt.Printf("  %s %s%s \u2014 not found\n", status, name, label)
	}
}
