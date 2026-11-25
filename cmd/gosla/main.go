// gosla — go → sla frontend CLI (thin dispatch).
// Usage: gosla emit <file.go> [-o <out.sla>]
// Exit: 0 ok, 1 refused (positioned diagnostic on stderr), 2 usage/IO error.
package main

import (
	"fmt"
	"os"

	"github.com/layola13/gosa/internal/gosla"
)

func main() {
	os.Exit(runMain(os.Args[1:]))
}

func runMain(args []string) int {
	if len(args) < 2 || args[0] != "emit" {
		fmt.Fprintln(os.Stderr, "usage: gosla emit <file.go> [-o <out.sla>]")
		return 2
	}
	src := args[1]
	out := ""
	for i := 2; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			out = args[i+1]
			i++
		} else {
			fmt.Fprintf(os.Stderr, "gosla: unknown flag %q\n", args[i])
			return 2
		}
	}
	if out == "" {
		out = src[:len(src)-3] + ".sla"
	}
	sla, err := gosla.Transpile(src)
	if err != nil {
		if re, ok := err.(*gosla.Refusal); ok {
			fmt.Fprintln(os.Stderr, re.Error())
			return 1
		}
		fmt.Fprintln(os.Stderr, "gosla:", err)
		return 2
	}
	if err := os.WriteFile(out, []byte(sla), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "gosla:", err)
		return 2
	}
	return 0
}
