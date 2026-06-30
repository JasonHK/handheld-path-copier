package main

import (
	"github.com/JasonHK/handheld-path-copier/internal/cmd"
)

var version = "dev"

func main() {
	cmd.Execute(version)
}
