// Package gosla transpiles a single-file Go program (package main) to SLA text.
// Increment 1 scope: int funcs (incl. recursion), var/:=/assign, if/else,
// C-style for, inc/dec, int/string expressions, fmt.Println.
// Anything outside scope is a positioned Refusal — never a silent approximation.
package gosla

import (
	"fmt"
	"go/ast"
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
	fset *token.FileSet
	info *types.Info
	sb   strings.Builder
	ind  int
}

func (e *emitter) refuse(pos token.Pos, format string, args ...any) *Refusal {
	return &Refusal{Pos: e.fset.Position(pos), Msg: fmt.Sprintf(format, args...)}
}

func (e *emitter) line(format string, args ...any) {
	e.sb.WriteString(strings.Repeat("    ", e.ind))
	e.sb.WriteString(fmt.Sprintf(format, args...))
	e.sb.WriteString("\n")
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
	e.sb.WriteString("@import \"sa_std/io/print.sai\"\n@import \"sa_std/fmt.sai\"\n\n")
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if err := e.funcDecl(d); err != nil {
				return "", err
			}
		case *ast.GenDecl:
			// imports only in increment 1; other top-level decls refused.
			if d.Tok != token.IMPORT {
				return "", e.refuse(d.Pos(), "top-level %s declarations not yet supported", d.Tok)
			}
		default:
			return "", e.refuse(decl.Pos(), "unsupported declaration %T", decl)
		}
	}
	return e.sb.String(), nil
}

func (e *emitter) isInt(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.Int
}

func (e *emitter) isString(t types.Type) bool {
	if t == nil {
		return false
	}
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Kind() == types.String
}

func (e *emitter) funcDecl(fn *ast.FuncDecl) error {
	if fn.Recv != nil {
		return e.refuse(fn.Pos(), "methods not yet supported")
	}
	name := fn.Name.Name
	var params []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if !e.isInt(e.info.TypeOf(field.Type)) {
				return e.refuse(field.Pos(), "params must be int (func %q)", name)
			}
			for _, n := range field.Names {
				params = append(params, n.Name+": i64")
			}
		}
	}
	results := 0
	if fn.Type.Results != nil {
		results = fn.Type.Results.NumFields()
		if results > 1 {
			return e.refuse(fn.Type.Results.Pos(), "multiple results not yet supported (func %q)", name)
		}
		if results == 1 && !e.isInt(e.info.TypeOf(fn.Type.Results.List[0].Type)) {
			return e.refuse(fn.Type.Results.Pos(), "result must be int (func %q)", name)
		}
	}
	isMain := name == "main"
	if isMain && (len(params) > 0 || results > 0) {
		return e.refuse(fn.Pos(), "main must have no params or results")
	}
	ret := "i64"
	if results == 0 {
		ret = "void"
	}
	if isMain {
		ret = "i32"
	}
	e.line("fn %s(%s) -> %s {", name, strings.Join(params, ", "), ret)
	e.ind++
	if fn.Body != nil {
		if err := e.block(fn.Body, false); err != nil {
			return err
		}
	}
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
	}
	for _, s := range b.List {
		if err := e.stmt(s); err != nil {
			return err
		}
	}
	if scoped {
		e.ind--
		e.line("}")
	}
	return nil
}

func (e *emitter) stmt(s ast.Stmt) error {
	switch st := s.(type) {
	case *ast.DeclStmt:
		gd, ok := st.Decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			return e.refuse(s.Pos(), "unsupported declaration statement")
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			if len(vs.Values) != len(vs.Names) {
				return e.refuse(s.Pos(), "var declaration without explicit value not yet supported")
			}
			for i, n := range vs.Names {
				v, err := e.expr(vs.Values[i])
				if err != nil {
					return err
				}
				e.line("let %s = %s;", n.Name, v)
			}
		}
		return nil
	case *ast.AssignStmt:
		if len(st.Lhs) != 1 || len(st.Rhs) != 1 {
			return e.refuse(s.Pos(), "multiple assignment not yet supported")
		}
		lhs, ok := st.Lhs[0].(*ast.Ident)
		if !ok {
			return e.refuse(s.Pos(), "assignment target must be a plain variable in increment 1")
		}
		rhs, err := e.expr(st.Rhs[0])
		if err != nil {
			return err
		}
		if st.Tok == token.DEFINE {
			e.line("let %s = %s;", lhs.Name, rhs)
		} else if st.Tok == token.ASSIGN {
			e.line("%s = %s;", lhs.Name, rhs)
		} else {
			// compound assignment x op= y  →  x = x op y
			op := map[token.Token]string{
				token.ADD_ASSIGN: "+", token.SUB_ASSIGN: "-", token.MUL_ASSIGN: "*",
				token.QUO_ASSIGN: "/", token.REM_ASSIGN: "%",
			}[st.Tok]
			if op == "" {
				return e.refuse(s.Pos(), "unsupported assignment operator %s", st.Tok)
			}
			e.line("%s = (%s %s %s);", lhs.Name, lhs.Name, op, rhs)
		}
		return nil
	case *ast.IncDecStmt:
		id, ok := st.X.(*ast.Ident)
		if !ok {
			return e.refuse(s.Pos(), "inc/dec target must be a plain variable")
		}
		op := "+"
		if st.Tok == token.DEC {
			op = "-"
		}
		e.line("%s = %s %s 1;", id.Name, id.Name, op)
		return nil
	case *ast.ReturnStmt:
		if len(st.Results) == 0 {
			e.line("return;")
			return nil
		}
		if len(st.Results) > 1 {
			return e.refuse(s.Pos(), "multiple return values not yet supported")
		}
		v, err := e.expr(st.Results[0])
		if err != nil {
			return err
		}
		e.line("return %s;", v)
		return nil
	case *ast.ExprStmt:
		v, err := e.expr(st.X)
		if err != nil {
			return err
		}
		if v != "" {
			e.line("%s;", v)
		}
		return nil
	case *ast.IfStmt:
		if st.Init != nil {
			return e.refuse(s.Pos(), "if with init statement not yet supported")
		}
		cond, err := e.expr(st.Cond)
		if err != nil {
			return err
		}
		e.line("if %s {", cond)
		e.ind++
		if err := e.block(st.Body, false); err != nil {
			return err
		}
		e.ind--
		switch els := st.Else.(type) {
		case nil:
			e.line("};")
		case *ast.BlockStmt:
			e.line("} else {")
			e.ind++
			if err := e.block(els, false); err != nil {
				return err
			}
			e.ind--
			e.line("};")
		case *ast.IfStmt:
			e.line("} else")
			if err := e.stmt(els); err != nil {
				return err
			}
		default:
			return e.refuse(s.Pos(), "unsupported else shape")
		}
		return nil
	case *ast.ForStmt:
		if st.Init != nil {
			if err := e.stmt(st.Init); err != nil {
				return err
			}
		}
		cond := "true"
		if st.Cond != nil {
			c, err := e.expr(st.Cond)
			if err != nil {
				return err
			}
			cond = c
		}
		e.line("while %s {", cond)
		e.ind++
		if err := e.block(st.Body, false); err != nil {
			return err
		}
		if st.Post != nil {
			if err := e.stmt(st.Post); err != nil {
				return err
			}
		}
		e.ind--
		e.line("}")
		return nil
	case *ast.BlockStmt:
		return e.block(st, true)
	default:
		return e.refuse(s.Pos(), "unsupported statement %T", s)
	}
}

func (e *emitter) expr(x ast.Expr) (string, error) {
	switch ex := x.(type) {
	case *ast.BasicLit:
		switch ex.Kind {
		case token.INT, token.STRING:
			return ex.Value, nil
		default:
			return "", e.refuse(x.Pos(), "unsupported literal %s", ex.Kind)
		}
	case *ast.Ident:
		switch ex.Name {
		case "true", "false", "nil":
			return ex.Name, nil
		}
		return ex.Name, nil
	case *ast.ParenExpr:
		return e.expr(ex.X)
	case *ast.UnaryExpr:
		v, err := e.expr(ex.X)
		if err != nil {
			return "", err
		}
		switch ex.Op {
		case token.SUB, token.NOT:
			return fmt.Sprintf("(%s%s)", ex.Op, v), nil
		default:
			return "", e.refuse(x.Pos(), "unsupported unary operator %s", ex.Op)
		}
	case *ast.BinaryExpr:
		if ex.Op == token.ADD && (e.isString(e.info.TypeOf(ex.X)) || e.isString(e.info.TypeOf(ex.Y))) {
			return "", e.refuse(x.Pos(), "string concatenation not yet supported (increment 2: sla string concat)")
		}
		l, err := e.expr(ex.X)
		if err != nil {
			return "", err
		}
		r, err := e.expr(ex.Y)
		if err != nil {
			return "", err
		}
		switch ex.Op {
		case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
			token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ,
			token.LAND, token.LOR:
			return fmt.Sprintf("(%s %s %s)", l, ex.Op, r), nil
		default:
			return "", e.refuse(x.Pos(), "unsupported binary operator %s", ex.Op)
		}
	case *ast.CallExpr:
		return e.call(ex)
	default:
		return "", e.refuse(x.Pos(), "unsupported expression %T", x)
	}
}

func (e *emitter) call(c *ast.CallExpr) (string, error) {
	if sel, ok := c.Fun.(*ast.SelectorExpr); ok {
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "fmt" && sel.Sel.Name == "Println" {
			if len(c.Args) != 1 {
				return "", e.refuse(c.Pos(), "fmt.Println with %d args not yet supported", len(c.Args))
			}
			arg := c.Args[0]
			v, err := e.expr(arg)
			if err != nil {
				return "", err
			}
			if e.isString(e.info.TypeOf(arg)) {
				e.line("println(%s);", v)
			} else if e.isInt(e.info.TypeOf(arg)) {
				e.line("println(\"{}\", %s);", v)
			} else {
				return "", e.refuse(c.Pos(), "fmt.Println arg type not yet supported")
			}
			return "", nil
		}
		return "", e.refuse(c.Pos(), "unsupported selector call")
	}
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return "", e.refuse(c.Pos(), "unsupported call target %T", c.Fun)
	}
	var args []string
	for _, a := range c.Args {
		v, err := e.expr(a)
		if err != nil {
			return "", err
		}
		args = append(args, v)
	}
	callStr := fmt.Sprintf("%s(%s)", id.Name, strings.Join(args, ", "))
	// As an expression statement, emit it as a statement line.
	return callStr, nil
}
