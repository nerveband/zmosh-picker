package main

import (
	"github.com/nerveband/zmosh-picker/internal/zmosh"
)

func runKill(name string) error {
	return zmosh.Kill(name)
}
