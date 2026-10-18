// Package go2sa lowering entry: AST -> SA text (S1 placeholder).
package go2sa

import (
	"go/ast"
	"go/token"
)

// saLowerFile lowers one parsed Go file to SA text.
// S1: skeleton only -> loud refusal pointing at package clause (no silent miscompile).
func saLowerFile(fset *token.FileSet, f *ast.File, _ string) *Output {
	p := fset.Position(f.Package)
	line, col := p.Line, p.Column
	if line <= 0 {
		line = 1
	}
	if col <= 0 {
		col = 1
	}
	return &Output{Refusals: []Refusal{{Line: line, Col: col, Msg: "go2sa lowering not implemented yet (S1 skeleton: parse only)"}}}
}
