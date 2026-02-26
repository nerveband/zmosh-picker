package main

import (
	"github.com/nerveband/zpick/internal/hook"
	"github.com/nerveband/zpick/internal/update"
)

func runUpgrade() error {
	err := update.Upgrade(version)
	if err == nil {
		hook.CheckSymlink()
	}
	return err
}
