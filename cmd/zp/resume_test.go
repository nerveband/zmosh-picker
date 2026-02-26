package main

import (
	"os/exec"
	"testing"
)

// TestResumeBuildAndSubcommand verifies that "zp resume" is a recognized
// subcommand and exits 0 when no switch-target file exists.
func TestResumeBuildAndSubcommand(t *testing.T) {
	// Build the binary.
	buildCmd := exec.Command("go", "build", "-o", t.TempDir()+"/zp", "./")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	// Run "zp resume" — should exit 0 with no output (no switch-target file).
	bin := t.TempDir() + "/zp"
	// Need to rebuild to the new tempdir since first one is used
	buildCmd2 := exec.Command("go", "build", "-o", bin, "./")
	buildCmd2.Dir = "."
	if out, err := buildCmd2.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	resumeCmd := exec.Command(bin, "resume")
	out, err := resumeCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("'zp resume' should exit 0 when no switch-target exists, got error: %v\n%s", err, out)
	}
	if len(out) > 0 {
		t.Errorf("'zp resume' should produce no output when no switch-target exists, got: %q", out)
	}
}

// TestResumeNotUnknownCommand verifies "resume" isn't treated as an unknown command.
func TestResumeNotUnknownCommand(t *testing.T) {
	bin := t.TempDir() + "/zp"
	buildCmd := exec.Command("go", "build", "-o", bin, "./")
	buildCmd.Dir = "."
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}

	resumeCmd := exec.Command(bin, "resume")
	stderr, err := resumeCmd.CombinedOutput()
	// Should NOT contain "unknown command"
	if err != nil {
		output := string(stderr)
		if contains(output, "unknown command") {
			t.Errorf("'zp resume' was treated as unknown command: %s", output)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
