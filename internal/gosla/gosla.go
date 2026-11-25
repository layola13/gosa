// Package gosla transpiles a single-file Go program (package main) to SLA text.
// Scope grows by demo-gated increments; anything outside scope is a positioned
// Refusal — never a silent approximation.
package gosla

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

// Refusal is a positioned, loud rejection of an unsupported Go shape.
type Refusal struct {
	Pos token.Position
	Msg string
}

func (r *Refusal) Error() string { return fmt.Sprintf("%s: gosla: %s", r.Pos, r.Msg) }

type emitter struct {
	fset      *token.FileSet
	info      *types.Info
	sb        strings.Builder
	ind       int
	useVec    bool
	useMap    bool
	useChan   bool
	rangeIncr []string // increment lines for enclosing range loops (continue support)
	tmpSeq    int
	scopes    []goScope // goroutine/channel lifetime scopes (emit_chan.go)
	fnBase    int       // scopes index where the current function starts
	loopDepth int
}

func (e *emitter) refuse(pos token.Pos, format string, args ...any) *Refusal {
	return &Refusal{Pos: e.fset.Position(pos), Msg: fmt.Sprintf(format, args...)}
}

func (e *emitter) line(format string, args ...any) {
	e.sb.WriteString(strings.Repeat("    ", e.ind))
	e.sb.WriteString(fmt.Sprintf(format, args...))
	e.sb.WriteString("\n")
}

func (e *emitter) tmp() string {
	e.tmpSeq++
	return fmt.Sprintf("__t%d", e.tmpSeq)
}

// Transpile reads a .go file and returns SLA source text.
func Transpile(path string) (string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return "", err
	}
	if file.Name.Name != "main" {
		return "", &Refusal{Pos: fset.Position(file.Name.Pos()), Msg: "only package main is supported"}
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("main", fset, []*ast.File{file}, info); err != nil {
		return "", &Refusal{Pos: fset.Position(file.Pos()), Msg: "go/types: " + err.Error()}
	}
	e := &emitter{fset: fset, info: info}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if err := e.funcDecl(d); err != nil {
				return "", err
			}
		case *ast.GenDecl:
			if err := e.genDecl(d); err != nil {
				return "", err
			}
		default:
			return "", e.refuse(decl.Pos(), "unsupported declaration %T", decl)
		}
	}
	hdr := "@import \"sa_std/io/print.sai\"\n@import \"sa_std/fmt.sai\"\n"
	if e.useVec {
		hdr += "@import \"sa_std/vec.sa\"\n"
	}
	if e.useMap {
		hdr += "@import \"sa_std/hashmap.sa\"\n@import \"sa_std/core/option.sa\"\n"
	}
	if e.useChan {
		hdr += "@import \"sa_std/task.sai\"\n@import \"sa_std/thread.sa\"\n@import \"sa_std/core/result.sa\"\n"
	}
	return hdr + "\n" + e.sb.String(), nil
}

// slaType maps a Go type to its SLA annotation; ok=false when unmapped.
func (e *emitter) slaType(t types.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Int:
			return "i64", true
		case types.Bool:
			return "bool", true
		case types.String:
			return "Str", true
		}
	case *types.Slice:
		if et, ok := e.slaType(u.Elem()); ok {
			e.useVec = true
			return "Vec<" + et + ">", true
		}
	case *types.Map:
		kt, kok := e.slaType(u.Key())
		vt, vok := e.slaType(u.Elem())
		if kok && vok {
			e.useMap = true
			return "HashMap<" + kt + ", " + vt + ">", true
		}
	case *types.Chan:
		if et, ok := e.slaType(u.Elem()); ok {
			e.useChan = true
			return "Chan<" + et + ">", true
		}
	case *types.Array:
		if et, ok := e.slaType(u.Elem()); ok {
			return fmt.Sprintf("[%s; %d]", et, u.Len()), true
		}
	case *types.Struct:
		if named, ok := t.(*types.Named); ok {
			return named.Obj().Name(), true
		}
	}
	if named, ok := t.(*types.Named); ok {
		if _, isStruct := t.Underlying().(*types.Struct); isStruct {
			return named.Obj().Name(), true
		}
	}
	return "", false
}

func (e *emitter) isInt(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Int
}

func (e *emitter) isString(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

// constLiteral folds a *types.Const reference to its SLA literal text.
func constLiteral(obj types.Object) (string, bool) {
	c, ok := obj.(*types.Const)
	if !ok {
		return "", false
	}
	v := c.Val()
	switch v.Kind() {
	case constant.Int:
		return v.ExactString(), true
	case constant.Bool:
		return fmt.Sprintf("%v", constant.BoolVal(v)), true
	case constant.String:
		return fmt.Sprintf("%q", constant.StringVal(v)), true
	}
	return "", false
}

// fieldName maps a Go struct field name to SLA form (first rune lower-cased).
func fieldName(goName string) string {
	if goName == "" {
		return goName
	}
	r := []rune(goName)
	if r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 'a' - 'A'
	}
	return string(r)
}

func (e *emitter) genDecl(d *ast.GenDecl) error {
	switch d.Tok {
	case token.IMPORT:
		return nil
	case token.CONST:
		return nil // constants fold at use sites via go/types
	case token.TYPE:
		for _, spec := range d.Specs {
			ts := spec.(*ast.TypeSpec)
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return e.refuse(ts.Pos(), "only struct type declarations are supported (%q)", ts.Name.Name)
			}
			e.line("struct %s {", ts.Name.Name)
			e.ind++
			for _, f := range st.Fields.List {
				ft, okk := e.slaType(e.info.TypeOf(f.Type))
				if !okk {
					return e.refuse(f.Pos(), "unsupported struct field type (%q)", ts.Name.Name)
				}
				for _, n := range f.Names {
					e.line("%s: %s,", fieldName(n.Name), ft)
				}
			}
			e.ind--
			e.line("}")
			e.sb.WriteString("\n")
		}
		return nil
	default:
		return e.refuse(d.Pos(), "top-level %s declarations not yet supported", d.Tok)
	}
}

func (e *emitter) funcDecl(fn *ast.FuncDecl) error {
	if fn.Recv != nil {
		return e.refuse(fn.Pos(), "methods not yet supported")
	}
	name := fn.Name.Name
	var params []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			pt, ok := e.slaType(e.info.TypeOf(field.Type))
			if !ok {
				return e.refuse(field.Pos(), "unsupported param type (func %q)", name)
			}
			for _, n := range field.Names {
				params = append(params, n.Name+": "+pt)
			}
		}
	}
	results := 0
	ret := "void"
	if fn.Type.Results != nil {
		results = fn.Type.Results.NumFields()
		if results > 1 {
			return e.refuse(fn.Type.Results.Pos(), "multiple results not yet supported (func %q)", name)
		}
		if results == 1 {
			rt, ok := e.slaType(e.info.TypeOf(fn.Type.Results.List[0].Type))
			if !ok {
				return e.refuse(fn.Type.Results.Pos(), "unsupported result type (func %q)", name)
			}
			ret = rt
		}
	}
	isMain := name == "main"
	if isMain {
		if len(params) > 0 || results > 0 {
			return e.refuse(fn.Pos(), "main must have no params or results")
		}
		ret = "i32"
	}
	e.line("fn %s(%s) -> %s {", name, strings.Join(params, ", "), ret)
	e.ind++
	e.fnBase = len(e.scopes)
	e.pushScope()
	if fn.Body != nil {
		if err := e.block(fn.Body, false); err != nil {
			return err
		}
	}
	e.popScopeEmitCleanup()
	if isMain {
		e.line("return 0;")
	}
	e.ind--
	e.line("}")
	e.sb.WriteString("\n")
	return nil
}

func (e *emitter) block(b *ast.BlockStmt, scoped bool) error {
	if scoped {
		e.line("{")
		e.ind++
		e.pushScope()
	}
	for _, s := range b.List {
		if err := e.stmt(s); err != nil {
			return err
		}
	}
	if scoped {
		e.popScopeEmitCleanup()
		e.ind--
		e.line("}")
	}
	return nil
}
