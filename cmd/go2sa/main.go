// Command go2sa is the thin CLI for the go->sa frontend (mirrors tsgosa cmd/tsgo thin dispatch).
package main

import (
	"os"

	"github.com/layola13/gosa/internal/go2sa"
)

func main() {
	os.Exit(runMain())
}

func runMain() int {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "--sa":
			return go2sa.RunSA(args[1:])
		}
	}
	return go2sa.RunSA(args)
}
