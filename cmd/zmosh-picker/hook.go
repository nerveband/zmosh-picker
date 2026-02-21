package main

import (
	"os"

	"github.com/nerveband/zmosh-picker/internal/hook"
)

func runInstallHook() error {
	// Check for --remove flag
	for _, arg := range os.Args[2:] {
		if arg == "--remove" {
			return hook.Remove()
		}
	}
	return hook.Install()
}
