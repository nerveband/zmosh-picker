package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		// Default: interactive picker
		if err := runPicker(); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "list":
		if err := runList(); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
	case "check":
		if err := runCheck(); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
	case "attach":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: zmosh-picker attach <name> [--dir <path>]")
			os.Exit(1)
		}
		if err := runAttach(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
	case "kill":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: zmosh-picker kill <name>")
			os.Exit(1)
		}
		if err := runKill(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
	case "install-hook":
		if err := runInstallHook(); err != nil {
			fmt.Fprintf(os.Stderr, "zmosh-picker: %v\n", err)
			os.Exit(1)
		}
	case "version":
		fmt.Printf("zmosh-picker %s\n", version)
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "zmosh-picker: unknown command %q\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`zmosh-picker — session launcher for zmosh

Usage:
  zmosh-picker              Interactive TUI picker (default)
  zmosh-picker list         List sessions (--json for machine-readable)
  zmosh-picker check        Check dependencies (--json for machine-readable)
  zmosh-picker attach <n>   Attach or create session
  zmosh-picker kill <name>  Kill a session
  zmosh-picker install-hook Add shell hook to .zshrc/.bashrc
  zmosh-picker version      Print version`)
}
