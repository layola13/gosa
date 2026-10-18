// Package go2sa implements the go->sa frontend: parse (go/parser) ->
// analyze (go/types) -> lower (SA text + sci macros).
//
// Pipeline (mirrors tsgosa ts->sa):
//
//	.go --[parser+types: package]--> analyze
//	    --[middle: desugar]--> middle (native Go形)
//	    --[emit: SA text + sci macros]--> .sa + subset-report.txt
//	    --[sci sa build]--> .exe/.wasm
package go2sa

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/importer"
	goparser "go/parser"
	"go/scanner"
	"go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"strings"
)

// Options configures single-file transpilation.
type Options struct {
	// FileName is the name given to the synthesized input file.
	FileName string
}

// Refusal is one loud refusal with source position (mirrors tsgosa SARefusal).
type Refusal struct {
	Line int
	Col  int
	Msg  string
}

// Output contains the emitted SA text and any refusals/warnings.
type Output struct {
	SAI      string
	Refusals []Refusal
	Warnings []Refusal
}

// TranspileSA lowers one Go source file to SA text.
// S2h P1: parse (go/parser, full object resolution) + type gate (go/types)
// before lowering; type errors are loud refusals with positions.
func TranspileSA(_ context.Context, input string, options Options) *Output {
	fname := options.FileName
	if fname == "" {
		fname = "module.go"
	}
	fset := token.NewFileSet()
	f, err := goparser.ParseFile(fset, fname, input, 0)
	if err != nil {
		if el, ok := err.(scanner.ErrorList); ok && len(el) > 0 {
			p := el[0].Pos
			line, col := p.Line, p.Column
			if line <= 0 {
				line = 1
			}
			if col <= 0 {
				col = 1
			}
			return &Output{Refusals: []Refusal{{Line: line, Col: col, Msg: el[0].Msg}}}
		}
		return &Output{Refusals: []Refusal{{Line: 1, Col: 1, Msg: err.Error()}}}
	}
	// P1 type gate: go/types checks undefined vars, arity, type mismatches.
	// importer.Default() reads gc export data (fmt available via toolchain).
	conf := gotypes.Config{Importer: importer.Default()}
	info := &gotypes.Info{
		Defs: make(map[*ast.Ident]gotypes.Object),
		Uses: make(map[*ast.Ident]gotypes.Object),
	}
	if _, err := conf.Check("main", fset, []*ast.File{f}, info); err != nil {
		if el, ok := err.(scanner.ErrorList); ok && len(el) > 0 {
			p := el[0].Pos
			line, col := p.Line, p.Column
			if line <= 0 {
				line = 1
			}
			if col <= 0 {
				col = 1
			}
			return &Output{Refusals: []Refusal{{Line: line, Col: col, Msg: "go/types: " + el[0].Msg}}}
		}
		return &Output{Refusals: []Refusal{{Line: 1, Col: 1, Msg: "go/types: " + err.Error()}}}
	}
	return saLowerFile(fset, f, input)
}

// RunSA implements `go2sa --sa [--out O] <file.go> [...]`.
// Exit codes mirror the gate: 2 usage/IO, 1 refused, 0 lowered.
func RunSA(args []string) int {
	fs := flag.NewFlagSet("sa", flag.ContinueOnError)
	out := fs.String("out", "", "output directory for .sa files (default: <first-input-base>_sa)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	inputs := fs.Args()
	if len(inputs) == 0 {
		fmt.Fprintln(os.Stderr, "usage: go2sa --sa [--out <dir>] <file.go> [...]")
		return 2
	}
	outDir := *out
	if outDir == "" {
		base := strings.TrimSuffix(filepath.Base(inputs[0]), filepath.Ext(inputs[0]))
		outDir = base + "_sa"
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "error: mkdir %s: %v\n", outDir, err)
		return 2
	}
	refused := false
	var report strings.Builder
	for _, f := range inputs {
		text, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %s: %v\n", f, err)
			return 2
		}
		res := TranspileSA(context.Background(), string(text), Options{FileName: f})
		base := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		if err := os.WriteFile(filepath.Join(outDir, base+".sa"), []byte(res.SAI), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "error: write %s: %v\n", f, err)
			return 2
		}
		for _, r := range res.Refusals {
			fmt.Fprintf(&report, "%s:%d:%d: %s\n", f, r.Line, r.Col, r.Msg)
		}
		for _, wr := range res.Warnings {
			fmt.Fprintf(&report, "%s:%d:%d: warning: %s\n", f, wr.Line, wr.Col, wr.Msg)
		}
		if len(res.Refusals) > 0 {
			refused = true
		}
	}
	if err := os.WriteFile(filepath.Join(outDir, "subset-report.txt"), []byte(report.String()), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write report: %v\n", err)
		return 2
	}
	fmt.Printf("wrote SA project to %s\n", outDir)
	if refused {
		return 1
	}
	return 0
}
