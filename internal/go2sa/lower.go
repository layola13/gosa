// Package go2sa lowering: AST -> SA text (S2t: S2s + string concat).
package go2sa

import (
	"fmt"
	"go/ast"
	"go/token"
	gotypes "go/types"
	"strconv"
	"strings"
)

// saGen holds emission state for one file.
type saGen struct {
	fset     *token.FileSet
	sb       strings.Builder
	imports  []string
	hasImp   map[string]bool
	consts   []string
	strPool  map[string]int
	tmp      int
	lbl      int
	vars     map[string]bool
	varOrder []string
	// structVars maps variable -> struct type name for field access (S2k, per func).
	structVars map[string]string
	// constVals maps const names to inlined values (S2q): int text, bool 1/0, string raw.
	constVals map[string]constVal
	// localConsts marks function-local pinned consts (S2q immutability).
	localConsts map[string]bool
	// topConsts marks top-level const names (S2q shadow check).
	topConsts map[string]bool
	// sliceVars marks slice-header variables (S2l, per func): 16B {ptr,len} head.
	sliceVars map[string]bool
	// mapVars marks btree map-handle variables (S2o, per func): ptr to BTreeMap.
	mapVars map[string]bool
	// strVars marks string-header variables (S2p, per func): 16B {ptr,len} head.
	strVars map[string]bool
	// arrayLens records fixed array lengths (S2s, per func): name -> N.
	arrayLens map[string]int
	// types threads go/types info for P2 type-driven checks (nil = syntax-only fallback).
	types *gotypes.Info
	// structs indexes top-level struct layouts by name (S2k): field order + 8B offsets.
	structs map[string]*structInfo
	// curFunc tracks current function scope (S2g): params/ret + per-func state.
	curFunc *funcScope
	// funcs indexes all top-level funcs for call checking (S2g).
	funcs map[string]*funcScope
	// callTmps tracks call-result temps needing `!` after single use (S2g, per func).
	callTmps map[string]bool
	// term tracks whether current sequential flow already terminated (S2e).
	term bool
	// loopStack tracks enclosing for labels for break/continue (S2e).
	loopStack []loopTarget
	// switchStack tracks enclosing switch end labels for break (S2f).
	switchStack []int
	// nestIsFor orders for(true)/switch(false) nesting for break targeting (S2f).
	nestIsFor []bool
	refusals  []Refusal
}

// structOf reports the struct type of e when e is a composite literal or struct var (S2k).
func (g *saGen) structOf(e ast.Expr) (*structInfo, bool) {
	switch n := e.(type) {
	case *ast.CompositeLit:
		id, ok := n.Type.(*ast.Ident)
		if !ok {
			return nil, false
		}
		si, ok := g.structs[id.Name]
		return si, ok
	case *ast.Ident:
		if g.structVars == nil {
			return nil, false
		}
		name, ok := g.structVars[n.Name]
		if !ok {
			return nil, false
		}
		si, ok := g.structs[name]
		return si, ok
	}
	return nil, false
}

// structVarsSet records/clears struct-ness of a variable after assignment (S2k).
func (g *saGen) structVarsSet(name string, rhs ast.Expr) {
	if g.structVars == nil {
		g.structVars = map[string]string{}
	}
	if si, ok := g.structOf(rhs); ok {
		g.structVars[name] = si.name
		return
	}
	delete(g.structVars, name)
}

// isMakeCall reports whether e is a make(...) call (S2l slice construction).
func (g *saGen) isMakeCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "make"
}

// isAppendCall reports whether e is an append(...) call (S2m: returns new head).
func (g *saGen) isAppendCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Fun.(*ast.Ident)
	return ok && id.Name == "append"
}

// strVarsSet marks/unmarks string-header variables after assignment (S2p).
func (g *saGen) strVarsSet(name string, rhs ast.Expr) {
	if g.strVars == nil {
		g.strVars = map[string]bool{}
	}
	if lit, ok := rhs.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		g.strVars[name] = true
		return
	}
	if id, ok := rhs.(*ast.Ident); ok && g.strVars[id.Name] {
		g.strVars[name] = true
		return
	}
	if g.isStrConcat(rhs) {
		g.strVars[name] = true
		return
	}
	delete(g.strVars, name)
}

// isConst reports whether name is a live const (S2q immutability).
func (g *saGen) isConst(name string) bool {
	if _, ok := g.constVals[name]; !ok {
		return false
	}
	if g.localConsts != nil && g.localConsts[name] {
		return true
	}
	if g.vars[name] {
		return false
	}
	return true
}

// isShadowedVar reports whether a plain local var shadows a const (S2q).
func (g *saGen) isShadowedVar(name string) bool {
	if g.localConsts != nil && g.localConsts[name] {
		return false
	}
	return g.vars[name]
}

// lowerConstStr materializes a string const value as a 16B head temp (S2q).
func (g *saGen) lowerConstStr(s string) string {
	g.addImport("sa_std/io/print.sai")
	name := g.internStr(s)
	h := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
	fmt.Fprintf(&g.sb, "  store %s + 0, &%s as ptr\n", h, name)
	fmt.Fprintf(&g.sb, "  store %s + 8, %d as u64\n", h, len(s))
	return h
}

// isStrVar reports whether e is a tracked string variable (S2p).
func (g *saGen) isStrVar(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && g.strVars != nil && g.strVars[id.Name]
}

// isStrConcat reports whether e is a string `+` expression (S2t chaining).
func (g *saGen) isStrConcat(e ast.Expr) bool {
	be, ok := e.(*ast.BinaryExpr)
	return ok && be.Op == token.ADD && g.isStrAdd(be)
}

// isStrLiteral reports whether e is a string literal (S2p).
func isStrLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING
}

// mapVarsSet marks/unmarks btree map variables after assignment (S2o).
func (g *saGen) mapVarsSet(name string, rhs ast.Expr) {
	if g.mapVars == nil {
		g.mapVars = map[string]bool{}
	}
	if g.isMakeMap(rhs) {
		g.mapVars[name] = true
		return
	}
	if id, ok := rhs.(*ast.Ident); ok && g.mapVars[id.Name] {
		g.mapVars[name] = true
		return
	}
	delete(g.mapVars, name)
}

// isMakeMap reports whether e is make(map[int]int) (S2o).
func (g *saGen) isMakeMap(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok || id.Name != "make" || len(call.Args) == 0 {
		return false
	}
	mt, ok := call.Args[0].(*ast.MapType)
	if !ok {
		return false
	}
	k, ok := mt.Key.(*ast.Ident)
	if !ok || k.Name != "int" {
		return false
	}
	v, ok := mt.Value.(*ast.Ident)
	return ok && v.Name == "int"
}

// sliceVarsSet marks/unmarks slice-header variables after assignment (S2l/S2m).
// S2s: arrays count as slice heads (fixed len recorded separately).
func (g *saGen) sliceVarsSet(name string, rhs ast.Expr) {
	if g.sliceVars == nil {
		g.sliceVars = map[string]bool{}
	}
	if call, ok := rhs.(*ast.CallExpr); ok {
		if id, ok := call.Fun.(*ast.Ident); ok && (id.Name == "make" || id.Name == "append") {
			g.sliceVars[name] = true
			return
		}
	}
	if id, ok := rhs.(*ast.Ident); ok && g.sliceVars[id.Name] {
		g.sliceVars[name] = true
		return
	}
	delete(g.sliceVars, name)
}

// isSliceVar reports whether name is a slice/array head var (S2l/S2s).
func (g *saGen) isSliceVar(name string) bool {
	return g.sliceVars != nil && g.sliceVars[name]
}

// arrayLenOf evaluates `[N]int` length N via go/types const (S2s).
func (g *saGen) arrayLenOf(at *ast.ArrayType) (int, bool) {
	if at.Len == nil {
		return 0, false
	}
	if g.types != nil {
		if tv, ok := g.types.Types[at.Len]; ok && tv.Value != nil {
			var n int
			if _, err := fmt.Sscanf(tv.Value.ExactString(), "%d", &n); err == nil && n > 0 {
				return n, true
			}
		}
	}
	if lit, ok := at.Len.(*ast.BasicLit); ok && lit.Kind == token.INT {
		var n int
		if _, err := fmt.Sscanf(lit.Value, "%d", &n); err == nil && n > 0 {
			return n, true
		}
	}
	return 0, false
}

// constVal is an inlined const value (S2q): kind int|bool|string.
type constVal struct {
	kind string
	ival string
	sval string
}

// structInfo holds an 8-byte-slot struct layout (S2k): fields in source order.
type structInfo struct {
	name   string
	fields []string
	off    map[string]int
	size   int
}

// funcScope holds per-function lowering state (S2g).
type funcScope struct {
	name   string
	params []string
	ret    bool
}

// loopTarget holds top/post/end labels of one enclosing for (S2e).
type loopTarget struct {
	top int
	// post < 0 means no post statement (while-style for).
	post int
	end  int
}

func (g *saGen) refuseAt(pos token.Pos, format string, args ...interface{}) {
	p := g.fset.Position(pos)
	line, col := p.Line, p.Column
	if line <= 0 {
		line = 1
	}
	if col <= 0 {
		col = 1
	}
	g.refusals = append(g.refusals, Refusal{Line: line, Col: col, Msg: fmt.Sprintf(format, args...)})
}

func (g *saGen) addImport(imp string) {
	if g.hasImp[imp] {
		return
	}
	g.hasImp[imp] = true
	g.imports = append(g.imports, imp)
}

func (g *saGen) nextTmp() string {
	g.tmp++
	return fmt.Sprintf("t_%d", g.tmp)
}

// saQuoteStr renders a Go string value as SA utf8 literal content.
func saQuoteStr(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '"':
			b.WriteString("\\\"")
		case '\n':
			b.WriteString("\\n")
		case '\t':
			b.WriteString("\\t")
		case '\r':
			b.WriteString("\\r")
		case 0:
			b.WriteString("\\0")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (g *saGen) internStr(s string) string {
	if n, ok := g.strPool[s]; ok {
		return fmt.Sprintf("str_const_%d", n)
	}
	n := len(g.strPool)
	g.strPool[s] = n
	g.consts = append(g.consts, fmt.Sprintf(`@const str_const_%d = utf8:"%s\0"`, n, saQuoteStr(s)))
	return fmt.Sprintf("str_const_%d", n)
}

// saLowerFile lowers one parsed Go file to SA text.
// S2k: S2j lowering + struct type/construct/field (int/bool fields, 8B slots); methods rejected.
func saLowerFile(fset *token.FileSet, f *ast.File, info *gotypes.Info) *Output {
	g := &saGen{fset: fset, hasImp: map[string]bool{}, strPool: map[string]int{}, types: info, structs: map[string]*structInfo{}, constVals: map[string]constVal{}, localConsts: map[string]bool{}, topConsts: map[string]bool{}}
	if f.Name == nil || f.Name.Name != "main" {
		g.refuseAt(f.Package, "go2sa only supports package main (got %q)", f.Name.Name)
		return &Output{Refusals: g.refusals}
	}
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || path != "fmt" {
			g.refuseAt(imp.Pos(), "go2sa S2a only supports import \"fmt\"")
		}
	}
	var order []*ast.FuncDecl
	byName := map[string]*funcScope{}
	var mainFn *ast.FuncDecl
	// Shared pools across functions (aliased, not copied): imports/consts/strPool.
	g.imports = nil
	g.consts = nil
	g.strPool = map[string]int{}
	g.hasImp = map[string]bool{}
	for _, decl := range f.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.IMPORT {
			continue // imports checked above
		}
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			g.collectStructType(gd)
			continue
		}
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.CONST {
			g.collectConst(gd)
			continue
		}
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			g.refuseAt(decl.Pos(), "go2sa S2q only supports const/type/func declarations (no top-level %T)", decl)
			continue
		}
		if fn.Recv != nil {
			g.refuseAt(fn.Pos(), "go2sa S2g methods not supported (got %q)", fn.Name.Name)
			continue
		}
		if _, dup := byName[fn.Name.Name]; dup {
			g.refuseAt(fn.Pos(), "duplicate func %q", fn.Name.Name)
			continue
		}
		params, ret, ok := checkFuncSig(g, fn)
		if !ok {
			continue
		}
		byName[fn.Name.Name] = &funcScope{name: fn.Name.Name, params: params, ret: ret}
		order = append(order, fn)
		if fn.Name.Name == "main" {
			if len(params) != 0 || ret {
				g.refuseAt(fn.Pos(), "go2sa S2g func main must be main() with no result")
				continue
			}
			mainFn = fn
		}
	}
	if len(g.refusals) > 0 {
		return &Output{Refusals: g.refusals}
	}
	if mainFn == nil {
		g.refuseAt(f.Package, "go2sa S2g requires func main")
		return &Output{Refusals: g.refusals}
	}
	g.funcs = byName
	// Emit each function with isolated scope; shared imports/consts/strPool.
	type fnOut struct {
		header string
		body   string
		footer []string
	}
	var outs []fnOut
	for _, fn := range order {
		sc := byName[fn.Name.Name]
		fg := &saGen{
			fset: fset, hasImp: g.hasImp,
			strPool: g.strPool, consts: g.consts, imports: g.imports,
			vars: map[string]bool{}, mapVars: map[string]bool{}, strVars: map[string]bool{},
			constVals: copyConstVals(g.constVals), localConsts: map[string]bool{}, topConsts: g.topConsts,
			arrayLens: copyArrayLens(g.arrayLens),
			funcs:     byName, curFunc: sc, types: g.types, structs: g.structs,
		}
		// Share pools by reference via copying back after each func.
		fg.tmp = g.tmp
		fg.lbl = g.lbl
		var params []string
		for _, p := range sc.params {
			fg.vars[p] = true
			fg.varOrder = append(fg.varOrder, p)
			params = append(params, p+": i32")
		}
		fg.sb.WriteString("")
		fg.lowerBlock(fn.Body)
		if len(fg.refusals) > 0 {
			g.refusals = append(g.refusals, fg.refusals...)
			return &Output{Refusals: g.refusals}
		}
		g.tmp = fg.tmp
		g.lbl = fg.lbl
		g.consts = fg.consts
		g.imports = fg.imports
		var header string
		if fn.Name.Name == "main" {
			header = "@main() -> i32:\n"
		} else if sc.ret {
			header = fmt.Sprintf("@%s(%s) -> i32:\n", fn.Name.Name, strings.Join(params, ", "))
		} else {
			header = fmt.Sprintf("@%s(%s) -> void:\n", fn.Name.Name, strings.Join(params, ", "))
		}
		body := fg.sb.String()
		var footer []string
		if fg.term {
			// Terminated (return/break): no trailing releases (unreachable).
			// Params are owned by caller; locals already released inline.
		} else {
			for _, name := range fg.varOrder {
				footer = append(footer, fmt.Sprintf("  !%s", name))
			}
		}
		if fn.Name.Name == "main" {
			footer = append(footer, "  ret 0")
		} else if !sc.ret && !fg.term {
			footer = append(footer, "  return")
		}
		outs = append(outs, fnOut{header: header, body: body, footer: footer})
		if sc.ret && !fg.term {
			g.refuseAt(fn.Body.Rbrace, "go2sa S2g func %q with i32 result must end with return", fn.Name.Name)
			return &Output{Refusals: g.refusals}
		}
	}
	var out strings.Builder
	out.WriteString("// Generated by go2sa S2t: parse(go/parser) + types(P1 gate + P2 type-driven i32) + bool + struct + slice + append + range + map + string + const + array + concat -> lower -> emit(SA text + sci macros)\n")
	for _, imp := range g.imports {
		fmt.Fprintf(&out, "@import %q\n", imp)
	}
	for _, c := range g.consts {
		out.WriteString(c + "\n")
	}
	for _, o := range outs {
		out.WriteString(o.header)
		out.WriteString(o.body)
		for _, l := range o.footer {
			out.WriteString(l + "\n")
		}
	}
	return &Output{SAI: out.String()}
}

// collectStructType records `type Name struct { f int|bool; ... }` layouts (S2k).
// Non-struct types, embedded fields, tags, generics, methods all refused downstream.
func (g *saGen) collectStructType(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			g.refuseAt(spec.Pos(), "go2sa S2k type block must hold type specs")
			continue
		}
		if ts.TypeParams != nil {
			g.refuseAt(ts.Pos(), "go2sa S2k generic type %q not supported", ts.Name.Name)
			continue
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			g.refuseAt(ts.Pos(), "go2sa S2k only supports struct types (got %q)", ts.Name.Name)
			continue
		}
		if _, dup := g.structs[ts.Name.Name]; dup {
			g.refuseAt(ts.Pos(), "duplicate type %q", ts.Name.Name)
			continue
		}
		si := &structInfo{name: ts.Name.Name, off: map[string]int{}}
		for _, field := range st.Fields.List {
			if len(field.Names) == 0 {
				g.refuseAt(field.Pos(), "go2sa S2k embedded field not supported (type %q)", ts.Name.Name)
				continue
			}
			if field.Tag != nil {
				g.refuseAt(field.Tag.Pos(), "go2sa S2k field tag not supported (type %q)", ts.Name.Name)
				continue
			}
			id, ok := field.Type.(*ast.Ident)
			if !ok || (id.Name != "int" && id.Name != "bool") {
				g.refuseAt(field.Type.Pos(), "go2sa S2k struct field must be int|bool (type %q)", ts.Name.Name)
				continue
			}
			for _, nm := range field.Names {
				if _, dup := si.off[nm.Name]; dup {
					g.refuseAt(nm.Pos(), "go2sa S2k duplicate field %q (type %q)", nm.Name, ts.Name.Name)
					continue
				}
				si.off[nm.Name] = len(si.fields) * 8
				si.fields = append(si.fields, nm.Name)
			}
		}
		si.size = len(si.fields) * 8
		if si.size == 0 {
			g.refuseAt(ts.Pos(), "go2sa S2k struct %q needs at least 1 field", ts.Name.Name)
			continue
		}
		g.structs[ts.Name.Name] = si
	}
}

// copyConstVals clones top-level consts for per-func scopes (S2q).
func copyConstVals(m map[string]constVal) map[string]constVal {
	out := map[string]constVal{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copyArrayLens clones array lengths for per-func scopes (S2s).
func copyArrayLens(m map[string]int) map[string]int {
	out := map[string]int{}
	for k, v := range m {
		out[k] = v
	}
	return out
}

// collectConst records `const Name = lit` evaluated values (S2q P1).
// int/bool via go/types const; string via unquote. iota/expr refused.
func (g *saGen) collectConst(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
			g.refuseAt(spec.Pos(), "go2sa S2q const needs exactly 1 name and 1 value")
			continue
		}
		name := vs.Names[0].Name
		if _, dup := g.constVals[name]; dup {
			g.refuseAt(vs.Pos(), "duplicate const %q", name)
			continue
		}
		val := vs.Values[0]
		if vs.Type != nil {
			if id, ok := vs.Type.(*ast.Ident); !ok || (id.Name != "int" && id.Name != "bool" && id.Name != "string") {
				g.refuseAt(vs.Type.Pos(), "go2sa S2q const %q type must be int|bool|string", name)
				continue
			}
		}
		if lit, ok := val.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				g.refuseAt(val.Pos(), "go2sa bad string literal")
				continue
			}
			g.constVals[name] = constVal{kind: "string", sval: s}
			g.topConsts[name] = true
			continue
		}
		if id, ok := val.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
			if id.Name == "true" {
				g.constVals[name] = constVal{kind: "bool", ival: "1"}
			} else {
				g.constVals[name] = constVal{kind: "bool", ival: "0"}
			}
			g.topConsts[name] = true
			continue
		}
		// int consts (incl. named refs resolved by types): use go/types value.
		if tv, ok := g.evalConst(val); ok {
			g.constVals[name] = tv
			g.topConsts[name] = true
			continue
		}
		g.refuseAt(val.Pos(), "go2sa S2q const %q must be int/bool/string literal", name)
	}
}

// checkFuncSig validates S2g signature via go/types objects (P2 type-driven):

// evalConst evaluates an int/bool const expr via go/types (S2q).
func (g *saGen) evalConst(e ast.Expr) (constVal, bool) {
	if g.types == nil {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.INT {
			return constVal{kind: "int", ival: lit.Value}, true
		}
		return constVal{}, false
	}
	tv, ok := g.types.Types[e]
	if !ok || tv.Type == nil {
		return constVal{}, false
	}
	b, ok := tv.Type.Underlying().(*gotypes.Basic)
	if !ok {
		return constVal{}, false
	}
	switch b.Kind() {
	case gotypes.Int, gotypes.UntypedInt, gotypes.UntypedRune:
		if tv.Value == nil {
			return constVal{}, false
		}
		return constVal{kind: "int", ival: tv.Value.ExactString()}, true
	case gotypes.Bool, gotypes.UntypedBool:
		if tv.Value == nil {
			return constVal{}, false
		}
		if tv.Value.ExactString() == "true" {
			return constVal{kind: "bool", ival: "1"}, true
		}
		return constVal{kind: "bool", ival: "0"}, true
	}
	return constVal{}, false
}

// params all int-typed, at most one int result. Falls back to syntax isIntType.
func checkFuncSig(g *saGen, fn *ast.FuncDecl) ([]string, bool, bool) {
	var params []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if len(field.Names) == 0 {
				g.refuseAt(field.Type.Pos(), "go2sa S2i params need names (func %q)", fn.Name.Name)
				return nil, false, false
			}
			for _, nm := range field.Names {
				if !g.isIntParam(nm) {
					g.refuseAt(nm.Pos(), "go2sa S2i params must be int (func %q, param %q)", fn.Name.Name, nm.Name)
					return nil, false, false
				}
				params = append(params, nm.Name)
			}
		}
	}
	seen := map[string]bool{}
	for _, p := range params {
		if seen[p] {
			g.refuseAt(fn.Pos(), "go2sa S2g duplicate param %q (func %q)", p, fn.Name.Name)
			return nil, false, false
		}
		seen[p] = true
	}
	ret := false
	if fn.Type.Results != nil {
		n := 0
		for _, field := range fn.Type.Results.List {
			k := len(field.Names)
			if k == 0 {
				k = 1
			}
			for i := 0; i < k; i++ {
				n++
				if !g.isIntResult(field, i) {
					g.refuseAt(field.Type.Pos(), "go2sa S2i result must be int (func %q)", fn.Name.Name)
					return nil, false, false
				}
			}
		}
		if n > 1 {
			g.refuseAt(fn.Type.Results.Pos(), "go2sa S2g at most one result (func %q)", fn.Name.Name)
			return nil, false, false
		}
		ret = n == 1
	}
	return params, ret, true
}

// isIntParam reports whether param ident has int/bool type via go/types Defs (P2+S2j).
func (g *saGen) isIntParam(id *ast.Ident) bool {
	if g.types != nil {
		if obj, ok := g.types.Defs[id]; ok && obj != nil {
			if b, ok := obj.Type().Underlying().(*gotypes.Basic); ok {
				return b.Kind() == gotypes.Int || b.Kind() == gotypes.Bool
			}
			return false
		}
	}
	return true // fallback: P1 gate already type-checked; syntax shape secondary
}

// isIntResult reports whether i-th result field has int/bool type (P2+S2j).
func (g *saGen) isIntResult(field *ast.Field, i int) bool {
	if g.types != nil && len(field.Names) > i {
		if obj, ok := g.types.Defs[field.Names[i]]; ok && obj != nil {
			if b, ok := obj.Type().Underlying().(*gotypes.Basic); ok {
				return b.Kind() == gotypes.Int || b.Kind() == gotypes.Bool
			}
			return false
		}
	}
	return isIntType(field.Type) || isBoolType(field.Type)
}

// isIntType reports whether e is plain `int` (P2: type-driven via go/types).
// Falls back to syntax check when types info is absent.
func isIntType(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "int"
}

// isBoolType reports whether e is plain `bool` (S2j syntax fallback).
func isBoolType(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "bool"
}

// isI32Expr reports whether e has i32/bool-lowerable type (P2 type-driven + S2j bool):
// typed int/bool or untyped constant (int literal, true/false, const expr).
func (g *saGen) isI32Expr(e ast.Expr) bool {
	if g.types == nil {
		return true // syntax-only fallback: downstream lowering validates shape
	}
	tv, ok := g.types.Types[e]
	if !ok {
		return false
	}
	t := tv.Type
	if t == nil {
		return false
	}
	if b, ok := t.Underlying().(*gotypes.Basic); ok {
		switch b.Kind() {
		case gotypes.Int, gotypes.UntypedInt, gotypes.UntypedRune,
			gotypes.Bool, gotypes.UntypedBool:
			return true
		}
		// Untyped constants defaultable to int (e.g. `40`, `2+3`) count as i32.
		if tv.IsValue() && b.Info()&gotypes.IsUntyped != 0 {
			return true
		}
	}
	return false
}

// requireI32 refuses with position when e is not i32-lowerable (P2).
func (g *saGen) requireI32(e ast.Expr, what string) bool {
	if g.isI32Expr(e) {
		return true
	}
	g.refuseAt(e.Pos(), "go2sa S2i %s needs i32 (int) type", what)
	return false
}

func (g *saGen) lowerBlock(b *ast.BlockStmt) {
	if b == nil {
		return
	}
	for _, s := range b.List {
		if g.term {
			g.refuseAt(s.Pos(), "go2sa S2e unreachable statement after terminator")
			return
		}
		g.lowerStmt(s)
	}
}

func (g *saGen) lowerStmt(s ast.Stmt) {
	switch n := s.(type) {
	case *ast.ExprStmt:
		g.lowerExprStmt(n)
	case *ast.DeclStmt:
		g.lowerDeclStmt(n)
	case *ast.AssignStmt:
		g.lowerAssignStmt(n)
	case *ast.IfStmt:
		g.lowerIfStmt(n)
	case *ast.ForStmt:
		g.lowerForStmt(n)
	case *ast.RangeStmt:
		g.lowerRangeStmt(n)
	case *ast.IncDecStmt:
		g.lowerIncDec(n)
	case *ast.BranchStmt:
		g.lowerBranchStmt(n)
	case *ast.SwitchStmt:
		g.lowerSwitchStmt(n)
	case *ast.ReturnStmt:
		g.lowerReturnStmt(n)
	default:
		g.refuseAt(s.Pos(), "go2sa S2g unsupported statement %T", s)
	}
}

func (g *saGen) declareVar(name string, pos token.Pos) {
	if g.vars == nil {
		g.vars = map[string]bool{}
	}
	if g.vars[name] {
		g.refuseAt(pos, "go2sa S2b duplicate variable %q", name)
		return
	}
	g.vars[name] = true
	g.varOrder = append(g.varOrder, name)
	fmt.Fprintf(&g.sb, "  %s = add 0, 0\n", name)
}

// lowerLocalConst lowers function-body `const x = lit` (S2q): evaluates and pins.
func (g *saGen) lowerLocalConst(gd *ast.GenDecl) {
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || len(vs.Names) != 1 || len(vs.Values) != 1 {
			g.refuseAt(spec.Pos(), "go2sa S2q local const needs exactly 1 name and 1 value")
			continue
		}
		name := vs.Names[0].Name
		if g.vars[name] {
			g.refuseAt(vs.Pos(), "go2sa S2b duplicate variable %q", name)
			continue
		}
		if vs.Type != nil {
			if id, ok := vs.Type.(*ast.Ident); !ok || (id.Name != "int" && id.Name != "bool" && id.Name != "string") {
				g.refuseAt(vs.Type.Pos(), "go2sa S2q const %q type must be int|bool|string", name)
				continue
			}
		}
		val := vs.Values[0]
		if lit, ok := val.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			s, err := strconv.Unquote(lit.Value)
			if err != nil {
				g.refuseAt(val.Pos(), "go2sa bad string literal")
				continue
			}
			g.declareVar(name, vs.Pos())
			h := g.lowerConstStr(s)
			fmt.Fprintf(&g.sb, "  %s = alloc 16\n", name)
			p := g.nextTmp()
			l := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", p, h)
			fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", l, h)
			fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", name, p)
			fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", name, l)
			fmt.Fprintf(&g.sb, "  !%s\n", h)
			g.strVarsSet(name, val)
			g.pinLocalConst(name, constVal{kind: "string", sval: s})
			continue
		}
		if id, ok := val.(*ast.Ident); ok && (id.Name == "true" || id.Name == "false") {
			iv := "0"
			if id.Name == "true" {
				iv = "1"
			}
			g.declareVar(name, vs.Pos())
			fmt.Fprintf(&g.sb, "  !%s\n", name)
			fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, iv)
			g.pinLocalConst(name, constVal{kind: "bool", ival: iv})
			continue
		}
		if cv, ok := g.evalConst(val); ok {
			g.declareVar(name, vs.Pos())
			fmt.Fprintf(&g.sb, "  !%s\n", name)
			fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, cv.ival)
			g.pinLocalConst(name, cv)
			continue
		}
		g.refuseAt(val.Pos(), "go2sa S2q const %q must be int/bool/string literal", name)
	}
}

// pinLocalConst pins a function-local const (S2q immutability + shadowing).
func (g *saGen) pinLocalConst(name string, cv constVal) {
	if g.constVals == nil {
		g.constVals = map[string]constVal{}
	}
	if g.localConsts == nil {
		g.localConsts = map[string]bool{}
	}
	g.constVals[name] = cv
	g.localConsts[name] = true
}

// lowerDeclStmt lowers `var x = <i32>` / `var x int` / `x := <i32>` (single-name, i32 only).
func (g *saGen) lowerDeclStmt(n *ast.DeclStmt) {
	gd, ok := n.Decl.(*ast.GenDecl)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2b only supports var declarations in function body")
		return
	}
	if gd.Tok == token.CONST {
		g.lowerLocalConst(gd)
		return
	}
	if gd.Tok != token.VAR {
		g.refuseAt(n.Pos(), "go2sa S2b only supports var declarations in function body")
		return
	}
	for _, spec := range gd.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			g.refuseAt(spec.Pos(), "go2sa S2b only supports var name = <i32 expr>")
			continue
		}
		if len(vs.Names) != 1 {
			g.refuseAt(spec.Pos(), "go2sa S2b var needs exactly 1 name")
			continue
		}
		name := vs.Names[0].Name
		if len(vs.Values) == 0 {
			// S2s: `var a [N]int` zero array.
			if vs.Type != nil {
				if at, ok := vs.Type.(*ast.ArrayType); ok && at.Len != nil {
					if n, ok := g.lowerArrayDecl(name, vs.Pos(), at, nil); ok {
						_ = n
						continue
					}
					continue
				}
			}
			g.declareVar(name, vs.Pos())
			continue
		}
		if len(vs.Values) != 1 {
			g.refuseAt(spec.Pos(), "go2sa S2b var needs exactly 1 value")
			continue
		}
		// S2s: `var a [N]int = ...` / `a := [N]int{...}`.
		if at, ok := g.arrayTypeOf(vs.Type, vs.Values[0]); ok {
			if _, done := g.lowerArrayDecl(name, vs.Pos(), at, vs.Values[0]); done {
				continue
			}
			continue
		}
		if isStrLiteral(vs.Values[0]) || g.isStrVar(vs.Values[0]) || g.isStrConcat(vs.Values[0]) {
			h := g.lowerStrHead(vs.Values[0])
			if h == "" {
				continue
			}
			g.declareVar(name, vs.Pos())
			if h == name {
				g.strVarsSet(name, vs.Values[0])
				continue
			}
			fmt.Fprintf(&g.sb, "  %s = alloc 16\n", name)
			p := g.nextTmp()
			l := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", p, h)
			fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", l, h)
			fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", name, p)
			fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", name, l)
			fmt.Fprintf(&g.sb, "  !%s\n", h)
			g.strVarsSet(name, vs.Values[0])
			continue
		}
		if _, isStruct := g.structOf(vs.Values[0]); !isStruct && !g.isMakeCall(vs.Values[0]) && !g.isAppendCall(vs.Values[0]) && !g.isMakeMap(vs.Values[0]) {
			if !g.requireI32(vs.Values[0], "var initializer") {
				continue
			}
		}
		v := g.lowerI32(vs.Values[0])
		if v == "" {
			return
		}
		g.declareVar(name, vs.Pos())
		if g.isMakeMap(vs.Values[0]) {
			// Map handle temp copied into var slot: release the temp, mark map.
			fmt.Fprintf(&g.sb, "  !%s\n", name)
			fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, v)
			fmt.Fprintf(&g.sb, "  !%s\n", v)
			g.mapVarsSet(name, vs.Values[0])
			continue
		}
		if _, isStruct := g.structOf(vs.Values[0]); isStruct {
			// Struct alloc temp copied into var slot via add: release the temp.
			fmt.Fprintf(&g.sb, "  !%s\n", name)
			fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, v)
			fmt.Fprintf(&g.sb, "  !%s\n", v)
			g.structVarsSet(name, vs.Values[0])
			g.sliceVarsSet(name, vs.Values[0])
			continue
		}
		if g.isMakeCall(vs.Values[0]) || g.isAppendCall(vs.Values[0]) {
			// Slice head temp copied into var slot: release the temp, mark slice.
			fmt.Fprintf(&g.sb, "  !%s\n", name)
			fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, v)
			fmt.Fprintf(&g.sb, "  !%s\n", v)
			g.sliceVarsSet(name, vs.Values[0])
			continue
		}
		fmt.Fprintf(&g.sb, "  !%s\n", name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, v)
	}
}

// arrayTypeOf resolves `[N]int` from explicit type or composite literal (S2s).
func (g *saGen) arrayTypeOf(typ ast.Expr, val ast.Expr) (*ast.ArrayType, bool) {
	if at, ok := typ.(*ast.ArrayType); ok && at.Len != nil {
		if id, ok := at.Elt.(*ast.Ident); ok && id.Name == "int" {
			return at, true
		}
		if typ != nil {
			g.refuseAt(typ.Pos(), "go2sa S2s array elem must be int")
			return nil, false
		}
	}
	if lit, ok := val.(*ast.CompositeLit); ok {
		if at, ok := lit.Type.(*ast.ArrayType); ok && at.Len != nil {
			if id, ok := at.Elt.(*ast.Ident); !ok || id.Name != "int" {
				g.refuseAt(lit.Type.Pos(), "go2sa S2s array elem must be int")
				return nil, false
			}
			return at, true
		}
	}
	return nil, false
}

// lowerArrayDecl materializes `[N]int` as slice-head + N*8 buffer (S2s).
// init nil = zero; CompositeLit = positional (+ optional keyed) values.
func (g *saGen) lowerArrayDecl(name string, pos token.Pos, at *ast.ArrayType, init ast.Expr) (int, bool) {
	n, ok := g.arrayLenOf(at)
	if !ok {
		g.refuseAt(at.Len.Pos(), "go2sa S2s array length must be a positive const")
		return 0, false
	}
	vals := map[int]ast.Expr{}
	if init != nil {
		lit, ok := init.(*ast.CompositeLit)
		if !ok {
			g.refuseAt(init.Pos(), "go2sa S2s array init must be a composite literal")
			return 0, false
		}
		posIdx := 0
		for _, e := range lit.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				ki, ok := g.constIndex(kv.Key)
				if !ok {
					g.refuseAt(kv.Key.Pos(), "go2sa S2s array key must be a const index")
					return 0, false
				}
				if ki < 0 || ki >= n {
					g.refuseAt(kv.Key.Pos(), "go2sa S2s array index %d out of range", ki)
					return 0, false
				}
				if _, dup := vals[ki]; dup {
					g.refuseAt(kv.Key.Pos(), "go2sa S2s duplicate array index %d", ki)
					return 0, false
				}
				vals[ki] = kv.Value
				continue
			}
			if posIdx >= n {
				g.refuseAt(e.Pos(), "go2sa S2s too many values for array")
				return 0, false
			}
			vals[posIdx] = e
			posIdx++
		}
		for _, e := range vals {
			if !g.requireI32(e, "array element") {
				return 0, false
			}
		}
	}
	head := g.nextTmp()
	buf := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc %d\n", buf, n*8)
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", head)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", head, buf)
	fmt.Fprintf(&g.sb, "  store %s + 8, %d as u64\n", head, n)
	fmt.Fprintf(&g.sb, "  !%s\n", buf)
	g.declareVar(name, pos)
	fmt.Fprintf(&g.sb, "  !%s\n", name)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, head)
	fmt.Fprintf(&g.sb, "  !%s\n", head)
	if g.sliceVars == nil {
		g.sliceVars = map[string]bool{}
	}
	g.sliceVars[name] = true
	if g.arrayLens == nil {
		g.arrayLens = map[string]int{}
	}
	g.arrayLens[name] = n
	for i := 0; i < n; i++ {
		e, ok := vals[i]
		if !ok {
			continue // zero-init
		}
		v := g.lowerI32(e)
		if v == "" {
			return 0, false
		}
		bo := g.nextTmp()
		bp := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", bp, name)
		fmt.Fprintf(&g.sb, "  %s = mul %d, 8\n", bo, i)
		fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", bp, bp, bo)
		fmt.Fprintf(&g.sb, "  store %s + 0, %s as u64\n", bp, v)
		g.releaseIfCallTemp(v)
	}
	return n, true
}

// constIndex evaluates a const index expr (S2s).
func (g *saGen) constIndex(e ast.Expr) (int, bool) {
	if g.types != nil {
		if tv, ok := g.types.Types[e]; ok && tv.Value != nil {
			var n int
			if _, err := fmt.Sscanf(tv.Value.ExactString(), "%d", &n); err == nil {
				return n, true
			}
		}
	}
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.INT {
		var n int
		if _, err := fmt.Sscanf(lit.Value, "%d", &n); err == nil {
			return n, true
		}
	}
	return 0, false
}

// lowerIfStmt lowers `if cond { } else { }` via EXPAND IF_ELSE (mirrors tsgosa step2).
// S2c scope: no init statement, cond must be i32 comparison or nonzero-i32 value.
func (g *saGen) lowerIfStmt(n *ast.IfStmt) {
	if n.Init != nil {
		g.refuseAt(n.Init.Pos(), "go2sa S2c if init statement not supported")
		return
	}
	cond := g.lowerCond(n.Cond)
	if cond == "" {
		return
	}
	g.addImport("sa_std/control.sal")
	g.lbl++
	thenL := g.lbl
	g.lbl++
	elseL := g.lbl
	g.lbl++
	endL := g.lbl
	fmt.Fprintf(&g.sb, "  EXPAND IF_ELSE %s, L_then_%d, L_else_%d\n", cond, thenL, elseL)
	fmt.Fprintf(&g.sb, "L_then_%d:\n", thenL)
	g.term = false
	g.lowerBlock(n.Body)
	thenTerm := g.term
	if !thenTerm {
		fmt.Fprintf(&g.sb, "  jmp L_end_%d\n", endL)
	}
	fmt.Fprintf(&g.sb, "L_else_%d:\n", elseL)
	g.term = false
	if n.Else != nil {
		switch e := n.Else.(type) {
		case *ast.BlockStmt:
			g.lowerBlock(e)
		case *ast.IfStmt:
			g.lowerIfStmt(e)
		default:
			g.refuseAt(n.Else.Pos(), "go2sa S2c unsupported else %T", n.Else)
			return
		}
	}
	// Join label always emitted (harmless when both arms terminated);
	// flow continues sequentially after the join.
	fmt.Fprintf(&g.sb, "L_end_%d:\n", endL)
	g.term = false
}

// lowerBranchStmt lowers `break` / `continue` (S2e for + S2f switch-break, unlabeled only).
func (g *saGen) lowerBranchStmt(n *ast.BranchStmt) {
	if n.Label != nil {
		g.refuseAt(n.Pos(), "go2sa S2f labeled %s not supported", n.Tok)
		return
	}
	switch n.Tok {
	case token.BREAK:
		if len(g.nestIsFor) == 0 {
			g.refuseAt(n.Pos(), "go2sa S2f break outside for/switch")
			return
		}
		if g.nestIsFor[len(g.nestIsFor)-1] {
			top := g.loopStack[len(g.loopStack)-1]
			fmt.Fprintf(&g.sb, "  jmp L_for_end_%d\n", top.end)
		} else {
			fmt.Fprintf(&g.sb, "  jmp L_switch_end_%d\n", g.switchStack[len(g.switchStack)-1])
		}
		g.term = true
		return
	case token.CONTINUE:
		if len(g.loopStack) == 0 {
			g.refuseAt(n.Pos(), "go2sa S2f continue outside for")
			return
		}
		top := g.loopStack[len(g.loopStack)-1]
		if top.post >= 0 {
			fmt.Fprintf(&g.sb, "  jmp L_for_post_%d\n", top.post)
		} else {
			fmt.Fprintf(&g.sb, "  jmp L_for_top_%d\n", top.top)
		}
		g.term = true
	default:
		g.refuseAt(n.Pos(), "go2sa S2f unsupported branch %s", n.Tok)
	}
}

// lowerReturnStmt lowers `return [expr]` (S2g): main 禁 return；值函数须 1 个 i32，空函数须裸 return。
func (g *saGen) lowerReturnStmt(n *ast.ReturnStmt) {
	if g.curFunc == nil || g.curFunc.name == "main" {
		g.refuseAt(n.Pos(), "go2sa S2g return outside value function")
		return
	}
	if !g.curFunc.ret {
		if len(n.Results) != 0 {
			g.refuseAt(n.Pos(), "go2sa S2g void func %q must use bare return", g.curFunc.name)
			return
		}
		fmt.Fprintf(&g.sb, "  return\n")
		g.term = true
		return
	}
	if len(n.Results) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2g func %q must return exactly 1 value", g.curFunc.name)
		return
	}
	if !g.requireI32(n.Results[0], "return value") {
		return
	}
	v := g.lowerI32(n.Results[0])
	if v == "" {
		return
	}
	// `return v` moves ownership to caller: no trailing `!v`.
	fmt.Fprintf(&g.sb, "  return %s\n", v)
	if g.callTmps != nil {
		delete(g.callTmps, v)
	}
	g.term = true
}

// inForBodyForSwitch is superseded by nestIsFor ordering (kept for reference).
func (g *saGen) inForBodyForSwitch() bool {
	return len(g.loopStack) > 0
}

// lowerSwitchStmt lowers `switch [tag] { case ...: ...; default: ... }` (S2f).
// Tag-less switch (cond chain) stays refused; Init stays refused; fallthrough stays refused.
// Each case list must be i32-comparable to tag; body break jumps to switch end (Go implicit break).
func (g *saGen) lowerSwitchStmt(n *ast.SwitchStmt) {
	if n.Init != nil {
		g.refuseAt(n.Init.Pos(), "go2sa S2f switch init not supported")
		return
	}
	if n.Tag == nil {
		g.refuseAt(n.Pos(), "go2sa S2f tag-less switch not supported")
		return
	}
	if !g.requireI32(n.Tag, "switch tag") {
		return
	}
	tag := g.lowerI32(n.Tag)
	if tag == "" {
		return
	}
	// Call-temp tags are single-use linear: spill to a hidden var for multi-case reuse.
	if g.callTmps != nil && g.callTmps[tag] {
		delete(g.callTmps, tag)
		hidden := fmt.Sprintf("sw_tag_%d", g.lbl+1)
		g.declareVar(hidden, n.Tag.Pos())
		fmt.Fprintf(&g.sb, "  !%s\n", hidden)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", hidden, tag)
		fmt.Fprintf(&g.sb, "  !%s\n", tag)
		tag = hidden
	}
	g.lbl++
	endL := g.lbl
	g.switchStack = append(g.switchStack, endL)
	g.nestIsFor = append(g.nestIsFor, false)
	for _, stmt := range n.Body.List {
		cc, ok := stmt.(*ast.CaseClause)
		if !ok {
			g.refuseAt(stmt.Pos(), "go2sa S2f switch body must be case clauses")
			g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
			g.switchStack = g.switchStack[:len(g.switchStack)-1]
			return
		}
		for _, e := range cc.Body {
			if br, ok := e.(*ast.BranchStmt); ok && br.Tok == token.FALLTHROUGH {
				g.refuseAt(e.Pos(), "go2sa S2f fallthrough not supported")
				g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
				g.switchStack = g.switchStack[:len(g.switchStack)-1]
				return
			}
		}
	}
	for _, stmt := range n.Body.List {
		cc := stmt.(*ast.CaseClause)
		if len(cc.List) == 0 {
			continue // default handled after cases
		}
		var conds []string
		for _, e := range cc.List {
			if !g.requireI32(e, "case value") {
				g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
				g.switchStack = g.switchStack[:len(g.switchStack)-1]
				return
			}
			v := g.lowerI32(e)
			if v == "" {
				g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
				g.switchStack = g.switchStack[:len(g.switchStack)-1]
				return
			}
			t := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = eq %s, %s\n", t, tag, v)
			conds = append(conds, t)
		}
		combined := conds[0]
		for _, c := range conds[1:] {
			// No `or` in witnessed SA shape: normalize via ne-to-i1 then add (any nonzero = match).
			n1 := g.nextTmp()
			n2 := g.nextTmp()
			t := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", n1, combined)
			fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", n2, c)
			fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", t, n1, n2)
			combined = t
		}
		g.addImport("sa_std/control.sal")
		g.lbl++
		bodyL := g.lbl
		g.lbl++
		nextL := g.lbl
		fmt.Fprintf(&g.sb, "  EXPAND IF_ELSE %s, L_switch_case_%d, L_switch_next_%d\n", combined, bodyL, nextL)
		fmt.Fprintf(&g.sb, "L_switch_case_%d:\n", bodyL)
		g.term = false
		for _, e := range cc.Body {
			if g.term {
				break
			}
			g.lowerStmt(e)
		}
		if !g.term {
			fmt.Fprintf(&g.sb, "  jmp L_switch_end_%d\n", endL)
		}
		fmt.Fprintf(&g.sb, "L_switch_next_%d:\n", nextL)
		g.term = false
	}
	for _, stmt := range n.Body.List {
		cc := stmt.(*ast.CaseClause)
		if len(cc.List) != 0 {
			continue
		}
		g.term = false
		for _, e := range cc.Body {
			if g.term {
				break
			}
			g.lowerStmt(e)
		}
		break // single default
	}
	g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
	g.switchStack = g.switchStack[:len(g.switchStack)-1]
	fmt.Fprintf(&g.sb, "L_switch_end_%d:\n", endL)
	g.term = false
}

// lowerRangeStmt lowers `for i[, v] := range s` over slice (S2n P1 desugar):
// i counts 0..len-1, v snapshots s[i] per iteration. Map/string/chan refused.
// Body break/continue reuse for-stack; loop pushes its own for target.
func (g *saGen) lowerRangeStmt(n *ast.RangeStmt) {
	if n.Tok != token.DEFINE {
		g.refuseAt(n.Pos(), "go2sa S2n range must use := (not =)")
		return
	}
	xid, ok := n.X.(*ast.Ident)
	if !ok || !g.isSliceVar(xid.Name) {
		g.refuseAt(n.X.Pos(), "go2sa S2n range needs a slice variable")
		return
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.X.Pos(), "go2sa S2n use of undeclared variable %q", xid.Name)
		return
	}
	var idxName, valName string
	hasVal := false
	switch {
	case n.Key == nil && n.Value == nil:
		g.refuseAt(n.Pos(), "go2sa S2n range needs at least an index variable")
		return
	case n.Key != nil && n.Value == nil:
		kid, ok := n.Key.(*ast.Ident)
		if !ok {
			g.refuseAt(n.Key.Pos(), "go2sa S2n range index must be an identifier")
			return
		}
		idxName = kid.Name
	case n.Key != nil && n.Value != nil:
		kid, ok := n.Key.(*ast.Ident)
		if !ok {
			g.refuseAt(n.Key.Pos(), "go2sa S2n range index must be an identifier")
			return
		}
		vid, ok := n.Value.(*ast.Ident)
		if !ok {
			g.refuseAt(n.Value.Pos(), "go2sa S2n range value must be an identifier")
			return
		}
		if kid.Name == "_" && vid.Name == "_" {
			g.refuseAt(n.Pos(), "go2sa S2n range needs at least an index variable")
			return
		}
		idxName, valName, hasVal = kid.Name, vid.Name, vid.Name != "_"
		if kid.Name == "_" {
			idxName = fmt.Sprintf("rg_ix_%d", g.lbl+1)
		}
	default:
		g.refuseAt(n.Pos(), "go2sa S2n range form not supported")
		return
	}
	if g.vars[idxName] {
		fmt.Fprintf(&g.sb, "  !%s\n", idxName)
		fmt.Fprintf(&g.sb, "  %s = add 0, 0\n", idxName)
	} else {
		g.declareVar(idxName, n.Pos())
		fmt.Fprintf(&g.sb, "  !%s\n", idxName)
		fmt.Fprintf(&g.sb, "  %s = add 0, 0\n", idxName)
	}
	var valDecl string
	if hasVal {
		if g.vars[valName] {
			fmt.Fprintf(&g.sb, "  !%s\n", valName)
			fmt.Fprintf(&g.sb, "  %s = add 0, 0\n", valName)
		} else {
			g.declareVar(valName, n.Pos())
		}
		valDecl = valName
		_ = valDecl
	}
	g.lbl++
	topL := g.lbl
	g.lbl++
	bodyL := g.lbl
	g.lbl++
	endL := g.lbl
	g.lbl++
	postL := g.lbl
	fmt.Fprintf(&g.sb, "L_for_top_%d:\n", topL)
	ln := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", ln, xid.Name)
	c := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = slt %s, %s\n", c, idxName, ln)
	fmt.Fprintf(&g.sb, "  br %s -> L_for_body_%d, L_for_end_%d\n", c, bodyL, endL)
	fmt.Fprintf(&g.sb, "L_for_body_%d:\n", bodyL)
	if hasVal {
		base := g.nextTmp()
		off := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", base, xid.Name)
		fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", off, idxName)
		fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", base, base, off)
		fmt.Fprintf(&g.sb, "  %s = load %s + 0 as u64\n", valName, base)
	} else if idxName != "" && n.Key == nil {
		_ = idxName
	}
	g.loopStack = append(g.loopStack, loopTarget{top: topL, post: postL, end: endL})
	g.nestIsFor = append(g.nestIsFor, true)
	g.lowerBlock(n.Body)
	g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
	g.loopStack = g.loopStack[:len(g.loopStack)-1]
	fmt.Fprintf(&g.sb, "L_for_post_%d:\n", postL)
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = add %s, 1\n", t, idxName)
	fmt.Fprintf(&g.sb, "  !%s\n", idxName)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", idxName, t)
	fmt.Fprintf(&g.sb, "  jmp L_for_top_%d\n", topL)
	fmt.Fprintf(&g.sb, "L_for_end_%d:\n", endL)
}

// lowerForStmt lowers `for init; cond; post { body }` with i32 init/post and comparison cond.
// S2e adds: while-style `for cond { }` (no init/post) and unlabeled break/continue.
// Cond empty (`for { }`) stays refused (infinite loop).
func (g *saGen) lowerForStmt(n *ast.ForStmt) {
	if n.Init != nil {
		switch in := n.Init.(type) {
		case *ast.AssignStmt:
			g.lowerAssignStmt(in)
		case *ast.DeclStmt:
			g.lowerDeclStmt(in)
		default:
			g.refuseAt(n.Init.Pos(), "go2sa S2e for init must be := or = or var")
			return
		}
	}
	if n.Cond == nil {
		g.refuseAt(n.Pos(), "go2sa S2e for needs a condition")
		return
	}
	g.lbl++
	topL := g.lbl
	g.lbl++
	bodyL := g.lbl
	g.lbl++
	endL := g.lbl
	g.lbl++
	postL := g.lbl
	fmt.Fprintf(&g.sb, "L_for_top_%d:\n", topL)
	cond := g.lowerCond(n.Cond)
	if cond == "" {
		return
	}
	fmt.Fprintf(&g.sb, "  br %s -> L_for_body_%d, L_for_end_%d\n", cond, bodyL, endL)
	fmt.Fprintf(&g.sb, "L_for_body_%d:\n", bodyL)
	g.loopStack = append(g.loopStack, loopTarget{top: topL, post: postL, end: endL})
	g.nestIsFor = append(g.nestIsFor, true)
	g.lowerBlock(n.Body)
	g.nestIsFor = g.nestIsFor[:len(g.nestIsFor)-1]
	g.loopStack = g.loopStack[:len(g.loopStack)-1]
	fmt.Fprintf(&g.sb, "L_for_post_%d:\n", postL)
	if n.Post != nil {
		switch p := n.Post.(type) {
		case *ast.AssignStmt:
			g.lowerAssignStmt(p)
		case *ast.IncDecStmt:
			g.lowerIncDec(p)
		default:
			g.refuseAt(n.Post.Pos(), "go2sa S2e for post must be = or ++/--")
			return
		}
	}
	fmt.Fprintf(&g.sb, "  jmp L_for_top_%d\n", topL)
	fmt.Fprintf(&g.sb, "L_for_end_%d:\n", endL)
}

func (g *saGen) lowerIncDec(n *ast.IncDecStmt) {
	id, ok := n.X.(*ast.Ident)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2e ++/-- target must be an identifier")
		return
	}
	if g.isConst(id.Name) {
		g.refuseAt(n.Pos(), "go2sa S2q cannot assign to const %q", id.Name)
		return
	}
	if !g.vars[id.Name] {
		g.refuseAt(n.Pos(), "go2sa S2e ++/-- of undeclared variable %q", id.Name)
		return
	}
	t := g.nextTmp()
	if n.Tok == token.INC {
		fmt.Fprintf(&g.sb, "  %s = add %s, 1\n", t, id.Name)
	} else if n.Tok == token.DEC {
		fmt.Fprintf(&g.sb, "  %s = sub %s, 1\n", t, id.Name)
	} else {
		g.refuseAt(n.Pos(), "go2sa S2e unsupported incdec %s", n.Tok)
		return
	}
	fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, t)
}

// lowerCond lowers a boolean condition to an i32-valued temp/register for br.
func (g *saGen) lowerCond(e ast.Expr) string {
	if be, ok := e.(*ast.BinaryExpr); ok {
		var op string
		switch be.Op {
		case token.EQL:
			op = "eq"
		case token.NEQ:
			op = "ne"
		case token.LSS:
			op = "slt"
		case token.LEQ:
			op = "sle"
		case token.GTR:
			op = "sgt"
		case token.GEQ:
			op = "sge"
		}
		if op != "" {
			l := g.lowerI32(be.X)
			r := g.lowerI32(be.Y)
			if l == "" || r == "" {
				return ""
			}
			t := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = %s %s, %s\n", t, op, l, r)
			g.releaseIfCallTemp(l)
			g.releaseIfCallTemp(r)
			return t
		}
	}
	v := g.lowerI32(e)
	if v == "" {
		return ""
	}
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", t, v)
	g.releaseIfCallTemp(v)
	return t
}
func (g *saGen) lowerAssignStmt(n *ast.AssignStmt) {
	if n.Tok != token.DEFINE && n.Tok != token.ASSIGN {
		g.refuseAt(n.Pos(), "go2sa S2b only supports := and = assignments")
		return
	}
	if len(n.Lhs) != 1 || len(n.Rhs) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2b assignment needs exactly 1 lhs and 1 rhs")
		return
	}
	id, ok := n.Lhs[0].(*ast.Ident)
	if ok && g.isConst(id.Name) {
		g.refuseAt(n.Pos(), "go2sa S2q cannot assign to const %q", id.Name)
		return
	}
	if !ok {
		// S2k: `p.F = v` field store.
		if sel, ok := n.Lhs[0].(*ast.SelectorExpr); ok {
			g.lowerFieldStore(n, sel)
			return
		}
		// S2o: `m[k] = v` map store (before slice store).
		if ix, ok := n.Lhs[0].(*ast.IndexExpr); ok {
			if xid, ok := ix.X.(*ast.Ident); ok && g.mapVars != nil && g.mapVars[xid.Name] {
				g.lowerMapAssign(n, ix)
				return
			}
		}
		// S2l: `s[i] = v` element store.
		if ix, ok := n.Lhs[0].(*ast.IndexExpr); ok {
			g.lowerIndexStore(n, ix)
			return
		}
		g.refuseAt(n.Lhs[0].Pos(), "go2sa S2b assignment target must be an identifier")
		return
	}
	if _, isStruct := g.structOf(n.Rhs[0]); !isStruct && !g.isMakeCall(n.Rhs[0]) && !g.isAppendCall(n.Rhs[0]) && !g.isMakeMap(n.Rhs[0]) && !isStrLiteral(n.Rhs[0]) && !g.isStrVar(n.Rhs[0]) && !g.isStrConcat(n.Rhs[0]) {
		if _, isArray := g.arrayTypeOf(nil, n.Rhs[0]); !isArray {
			if !g.requireI32(n.Rhs[0], "assignment rhs") {
				return
			}
		}
	}
	if at, ok := g.arrayTypeOf(nil, n.Rhs[0]); ok {
		if n.Tok != token.DEFINE {
			g.refuseAt(n.Pos(), "go2sa S2s array literal must use :=")
			return
		}
		if g.vars[id.Name] {
			g.refuseAt(n.Pos(), "go2sa S2b duplicate variable %q", id.Name)
			return
		}
		if _, done := g.lowerArrayDecl(id.Name, n.Pos(), at, n.Rhs[0]); done {
			return
		}
		return
	}
	if isStrLiteral(n.Rhs[0]) || g.isStrVar(n.Rhs[0]) || g.isStrConcat(n.Rhs[0]) {
		h := g.lowerStrHead(n.Rhs[0])
		if h == "" {
			return
		}
		if n.Tok == token.DEFINE {
			g.declareVar(id.Name, n.Pos())
		} else if !g.vars[id.Name] {
			g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
			return
		}
		if h == id.Name {
			g.strVarsSet(id.Name, n.Rhs[0])
			return
		}
		fmt.Fprintf(&g.sb, "  %s = alloc 16\n", id.Name)
		p := g.nextTmp()
		l := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", p, h)
		fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", l, h)
		fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", id.Name, p)
		fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", id.Name, l)
		if h != id.Name {
			fmt.Fprintf(&g.sb, "  !%s\n", h)
		}
		g.strVarsSet(id.Name, n.Rhs[0])
		return
	}
	v := g.lowerI32(n.Rhs[0])
	if v == "" {
		return
	}
	if _, isStruct := g.structOf(n.Rhs[0]); isStruct {
		// Struct alloc temp copied into var slot: release the temp, record type.
		if n.Tok == token.DEFINE {
			g.declareVar(id.Name, n.Pos())
		} else if !g.vars[id.Name] {
			g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
			return
		}
		fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
		fmt.Fprintf(&g.sb, "  !%s\n", v)
		g.structVarsSet(id.Name, n.Rhs[0])
		g.sliceVarsSet(id.Name, n.Rhs[0])
		return
	}
	if g.isMakeMap(n.Rhs[0]) {
		// Map handle temp copied into var slot: release the temp, mark map.
		if n.Tok == token.DEFINE {
			g.declareVar(id.Name, n.Pos())
		} else if !g.vars[id.Name] {
			g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
			return
		}
		fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
		fmt.Fprintf(&g.sb, "  !%s\n", v)
		g.mapVarsSet(id.Name, n.Rhs[0])
		return
	}
	if g.isMakeCall(n.Rhs[0]) || g.isAppendCall(n.Rhs[0]) {
		// Slice head temp copied into var slot: release the temp, mark slice.
		if n.Tok == token.DEFINE {
			g.declareVar(id.Name, n.Pos())
		} else if !g.vars[id.Name] {
			g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
			return
		}
		fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
		fmt.Fprintf(&g.sb, "  !%s\n", v)
		g.structVarsSet(id.Name, n.Rhs[0])
		g.sliceVarsSet(id.Name, n.Rhs[0])
		return
	}
	if n.Tok == token.DEFINE {
		g.declareVar(id.Name, n.Pos())
		fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
		g.releaseIfCallTemp(v)
		g.structVarsSet(id.Name, n.Rhs[0])
		return
	}
	if !g.vars[id.Name] {
		g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
		return
	}
	fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
	g.releaseIfCallTemp(v)
	g.structVarsSet(id.Name, n.Rhs[0])
}

// lowerFieldStore lowers `p.F = v` (S2k): store v at field offset; no rebind.
func (g *saGen) lowerFieldStore(n *ast.AssignStmt, sel *ast.SelectorExpr) {
	if n.Tok == token.DEFINE {
		g.refuseAt(n.Pos(), "go2sa S2k field store must use = (not :=)")
		return
	}
	if len(n.Rhs) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2k field store needs exactly 1 rhs")
		return
	}
	xid, ok := sel.X.(*ast.Ident)
	if !ok {
		g.refuseAt(sel.Pos(), "go2sa S2k field access needs a variable")
		return
	}
	tname, ok := g.structVars[xid.Name]
	if !ok {
		g.refuseAt(sel.Pos(), "go2sa S2k %q is not a struct variable", xid.Name)
		return
	}
	si, ok := g.structs[tname]
	if !ok {
		g.refuseAt(sel.Pos(), "go2sa S2k unknown struct type %q", tname)
		return
	}
	off, ok := si.off[sel.Sel.Name]
	if !ok {
		g.refuseAt(sel.Pos(), "go2sa S2k unknown field %q (type %q)", sel.Sel.Name, si.name)
		return
	}
	if !g.vars[xid.Name] {
		g.refuseAt(sel.Pos(), "go2sa S2k use of undeclared variable %q", xid.Name)
		return
	}
	if !g.requireI32(n.Rhs[0], "struct field "+sel.Sel.Name) {
		return
	}
	v := g.lowerI32(n.Rhs[0])
	if v == "" {
		return
	}
	fmt.Fprintf(&g.sb, "  store %s + %d, %s as u64\n", xid.Name, off, v)
	g.releaseIfCallTemp(v)
}

// lowerIndexStore lowers `s[i] = v` (S2l): store v at base+i*8.
func (g *saGen) lowerIndexStore(n *ast.AssignStmt, ix *ast.IndexExpr) {
	if n.Tok == token.DEFINE {
		g.refuseAt(n.Pos(), "go2sa S2l element store must use = (not :=)")
		return
	}
	if len(n.Rhs) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2l element store needs exactly 1 rhs")
		return
	}
	xid, ok := ix.X.(*ast.Ident)
	if !ok || !g.isSliceVar(xid.Name) {
		g.refuseAt(ix.Pos(), "go2sa S2l index needs a slice variable")
		return
	}
	if !g.vars[xid.Name] {
		g.refuseAt(ix.Pos(), "go2sa S2l use of undeclared variable %q", xid.Name)
		return
	}
	if !g.requireI32(ix.Index, "slice index") {
		return
	}
	if !g.requireI32(n.Rhs[0], "slice element") {
		return
	}
	i := g.lowerI32(ix.Index)
	if i == "" {
		return
	}
	v := g.lowerI32(n.Rhs[0])
	if v == "" {
		return
	}
	base := g.nextTmp()
	off := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", base, xid.Name)
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", off, i)
	fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", base, base, off)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as u64\n", base, v)
	g.releaseIfCallTemp(i)
	g.releaseIfCallTemp(v)
}

func (g *saGen) lowerExprStmt(n *ast.ExprStmt) {
	call, ok := n.X.(*ast.CallExpr)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2g only supports calls as statements")
		return
	}
	if id, ok := call.Fun.(*ast.Ident); ok {
		if id.Name == "delete" {
			g.lowerDelete(call)
			return
		}
		sc, found := g.funcs[id.Name]
		if !found {
			g.refuseAt(call.Pos(), "go2sa S2g call to undefined func %q", id.Name)
			return
		}
		if sc.ret {
			g.refuseAt(call.Pos(), "go2sa S2g value func %q result must be used", id.Name)
			return
		}
		args := g.lowerCallArgs(call, sc)
		if args == nil {
			return
		}
		fmt.Fprintf(&g.sb, "  call @%s(%s)\n", id.Name, strings.Join(args, ", "))
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		g.refuseAt(call.Pos(), "go2sa S2a only supports fmt.Println calls")
		return
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "fmt" || sel.Sel.Name != "Println" {
		g.refuseAt(call.Pos(), "go2sa S2a only supports fmt.Println calls")
		return
	}
	if len(call.Args) != 1 {
		g.refuseAt(call.Pos(), "go2sa S2a fmt.Println takes exactly 1 argument")
		return
	}
	g.lowerPrintlnArg(call.Args[0])
}

// lowerPrintlnArg prints one value + trailing newline (mirrors tsgosa console.log shape).
// S2j: bool prints Go words true/false (types-driven); int keeps numeric shape.
func (g *saGen) lowerPrintlnArg(arg ast.Expr) {
	if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			g.refuseAt(arg.Pos(), "go2sa bad string literal")
			return
		}
		g.addImport("sa_std/io/print.sai")
		name := g.internStr(s)
		h := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
		fmt.Fprintf(&g.sb, "  store %s + 0, &%s as ptr\n", h, name)
		fmt.Fprintf(&g.sb, "  store %s + 8, %d as u64\n", h, len(s))
		vp := g.nextTmp()
		vl := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", vp, h)
		fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", vl, h)
		fmt.Fprintf(&g.sb, "  call @sa_print_bytes(&%s, %s)\n", vp, vl)
		fmt.Fprintf(&g.sb, "  !%s\n", h)
		g.printNewline()
		return
	}
	v := g.lowerI32(arg)
	if v == "" {
		return
	}
	if g.isBoolTypeOf(arg) {
		g.emitPrintBool(v)
		g.releaseIfCallTemp(v)
		return
	}
	if g.isStrTypeOf(arg) {
		g.emitPrintStrHead(v)
		// Temp heads (literal/const materialization start with t_); var heads are registers.
		if len(v) > 2 && v[0] == 't' && v[1] == '_' {
			fmt.Fprintf(&g.sb, "  !%s\n", v)
		}
		g.releaseIfCallTemp(v)
		return
	}
	g.emitPrintI32(v)
	g.releaseIfCallTemp(v)
}

// isStrTypeOf reports whether arg has string type via go/types (S2p dispatch).
// S2q: string consts count as string.
func (g *saGen) isStrTypeOf(e ast.Expr) bool {
	if isStrLiteral(e) {
		return true
	}
	if id, ok := e.(*ast.Ident); ok {
		if g.strVars != nil && g.strVars[id.Name] {
			return true
		}
		if cv, ok := g.constVals[id.Name]; ok && !g.isShadowedVar(id.Name) && cv.kind == "string" {
			return true
		}
	}
	if g.types == nil {
		return false
	}
	tv, ok := g.types.Types[e]
	if !ok || tv.Type == nil {
		return false
	}
	if b, ok := tv.Type.Underlying().(*gotypes.Basic); ok {
		return b.Kind() == gotypes.String || b.Kind() == gotypes.UntypedString
	}
	return false
}

// emitPrintStrHead prints a 16B {ptr,len} string head + newline (S2p).
// Does NOT release the head; caller releases temp heads only.
func (g *saGen) emitPrintStrHead(h string) {
	g.addImport("sa_std/io/print.sai")
	vp := g.nextTmp()
	vl := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", vp, h)
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", vl, h)
	fmt.Fprintf(&g.sb, "  call @sa_print_bytes(&%s, %s)\n", vp, vl)
	g.printNewline()
}

// lowerStrHead materializes a string literal/var as a 16B head temp (S2p).
func (g *saGen) lowerStrHead(e ast.Expr) string {
	if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		s, err := strconv.Unquote(lit.Value)
		if err != nil {
			g.refuseAt(e.Pos(), "go2sa bad string literal")
			return ""
		}
		g.addImport("sa_std/io/print.sai")
		name := g.internStr(s)
		h := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
		fmt.Fprintf(&g.sb, "  store %s + 0, &%s as ptr\n", h, name)
		fmt.Fprintf(&g.sb, "  store %s + 8, %d as u64\n", h, len(s))
		return h
	}
	if id, ok := e.(*ast.Ident); ok && g.strVars != nil && g.strVars[id.Name] {
		if !g.vars[id.Name] {
			g.refuseAt(e.Pos(), "go2sa S2p use of undeclared variable %q", id.Name)
			return ""
		}
		return id.Name
	}
	if id, ok := e.(*ast.Ident); ok {
		if cv, ok := g.constVals[id.Name]; ok && !g.isShadowedVar(id.Name) && cv.kind == "string" {
			return g.lowerConstStr(cv.sval)
		}
	}
	if be, ok := e.(*ast.BinaryExpr); ok && be.Op == token.ADD && g.isStrAdd(be) {
		return g.lowerStrConcat(be)
	}
	g.refuseAt(e.Pos(), "go2sa S2p string value must be a literal or string variable")
	return ""
}

// isBoolTypeOf reports whether arg has bool type via go/types (S2j Println dispatch).
func (g *saGen) isBoolTypeOf(e ast.Expr) bool {
	if id, ok := e.(*ast.Ident); ok {
		if cv, ok := g.constVals[id.Name]; ok && !g.isShadowedVar(id.Name) && cv.kind == "bool" {
			return true
		}
		if id.Name == "true" || id.Name == "false" {
			return true
		}
	}
	if g.types == nil {
		return false
	}
	tv, ok := g.types.Types[e]
	if !ok || tv.Type == nil {
		return false
	}
	if b, ok := tv.Type.Underlying().(*gotypes.Basic); ok {
		return b.Kind() == gotypes.Bool || b.Kind() == gotypes.UntypedBool
	}
	return false
}

// emitPrintBool prints i32-as-bool as Go words true/false + newline (S2j).
func (g *saGen) emitPrintBool(v string) {
	g.addImport("sa_std/io/print.sai")
	g.addImport("sa_std/control.sal")
	tWord := g.internStr("true")
	fWord := g.internStr("false")
	g.lbl++
	thenL := g.lbl
	g.lbl++
	elseL := g.lbl
	g.lbl++
	endL := g.lbl
	nz := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", nz, v)
	fmt.Fprintf(&g.sb, "  EXPAND IF_ELSE %s, L_then_%d, L_else_%d\n", nz, thenL, elseL)
	fmt.Fprintf(&g.sb, "L_then_%d:\n", thenL)
	g.emitPrintStrConst(tWord, 4)
	fmt.Fprintf(&g.sb, "  jmp L_end_%d\n", endL)
	fmt.Fprintf(&g.sb, "L_else_%d:\n", elseL)
	g.emitPrintStrConst(fWord, 5)
	fmt.Fprintf(&g.sb, "L_end_%d:\n", endL)
	g.printNewline()
}

// emitPrintStrConst prints an interned utf8 const with byte length (S2j helper).
func (g *saGen) emitPrintStrConst(name string, ln int) {
	h := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
	fmt.Fprintf(&g.sb, "  store %s + 0, &%s as ptr\n", h, name)
	fmt.Fprintf(&g.sb, "  store %s + 8, %d as u64\n", h, ln)
	vp := g.nextTmp()
	vl := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", vp, h)
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", vl, h)
	fmt.Fprintf(&g.sb, "  call @sa_print_bytes(&%s, %s)\n", vp, vl)
	fmt.Fprintf(&g.sb, "  !%s\n", h)
}

func (g *saGen) emitPrintI32(v string) {
	g.addImport("sa_std/io/print.sai")
	g.addImport("sa_std/fmt.sai")
	g.addImport("sa_std/string.sai")
	i64 := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = sext %s as i64\n", i64, v)
	buf := g.nextTmp()
	slot := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 64\n", buf)
	fmt.Fprintf(&g.sb, "  %s = alloc 8\n", slot)
	n := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = call @sa_fmt_i64_into(%s, 10, %s, 64, &%s)\n", n, i64, buf, slot)
	ln := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as u64\n", ln, slot)
	h := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", h, buf)
	fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", h, ln)
	fmt.Fprintf(&g.sb, "  !%s\n", n)
	vp := g.nextTmp()
	vl := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", vp, h)
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", vl, h)
	fmt.Fprintf(&g.sb, "  call @sa_print_bytes(&%s, %s)\n", vp, vl)
	fmt.Fprintf(&g.sb, "  !%s\n", h)
	g.printNewline()
	fmt.Fprintf(&g.sb, "  !%s\n", slot)
	fmt.Fprintf(&g.sb, "  !%s\n", buf)
}

func (g *saGen) printNewline() {
	g.addImport("sa_std/io/print.sai")
	nl := g.internStr("\n")
	h := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", h)
	fmt.Fprintf(&g.sb, "  store %s + 0, &%s as ptr\n", h, nl)
	fmt.Fprintf(&g.sb, "  store %s + 8, 1 as u64\n", h)
	vp := g.nextTmp()
	vl := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", vp, h)
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", vl, h)
	fmt.Fprintf(&g.sb, "  call @sa_print_bytes(&%s, %s)\n", vp, vl)
	fmt.Fprintf(&g.sb, "  !%s\n", h)
}

// lowerI32 lowers an integer/bool expression, returning the value temp/literal.
// S2j: bool rides as i32 0/1 (`true/false`, `!`, `&&`, `||` short-circuit-free expanded).
func (g *saGen) lowerI32(e ast.Expr) string {
	switch n := e.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT {
			g.refuseAt(e.Pos(), "go2sa S2a fmt.Println needs string or integer expression")
			return ""
		}
		return n.Value
	case *ast.Ident:
		if n.Name == "true" {
			return "1"
		}
		if n.Name == "false" {
			return "0"
		}
		if cv, ok := g.constVals[n.Name]; ok && !g.isShadowedVar(n.Name) {
			switch cv.kind {
			case "int", "bool":
				return cv.ival
			case "string":
				// String consts lower via head path; signal via special marker.
				return g.lowerConstStr(cv.sval)
			}
		}
		if !g.vars[n.Name] {
			g.refuseAt(e.Pos(), "go2sa S2b use of undeclared variable %q", n.Name)
			return ""
		}
		return n.Name
	case *ast.ParenExpr:
		return g.lowerI32(n.X)
	case *ast.UnaryExpr:
		if n.Op == token.SUB {
			v := g.lowerI32(n.X)
			if v == "" {
				return ""
			}
			t := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = sub 0, %s\n", t, v)
			return t
		}
		if n.Op == token.NOT {
			v := g.lowerI32(n.X)
			if v == "" {
				return ""
			}
			t := g.nextTmp()
			fmt.Fprintf(&g.sb, "  %s = eq %s, 0\n", t, v)
			g.releaseIfCallTemp(v)
			return t
		}
		g.refuseAt(e.Pos(), "go2sa S2a unsupported unary %s", n.Op)
		return ""
	case *ast.BinaryExpr:
		if n.Op == token.LAND || n.Op == token.LOR {
			return g.lowerBoolOp(n)
		}
		if n.Op == token.ADD && g.isStrAdd(n) {
			return g.lowerStrConcat(n)
		}
		l := g.lowerI32(n.X)
		r := g.lowerI32(n.Y)
		if l == "" || r == "" {
			return ""
		}
		var op string
		switch n.Op {
		case token.ADD:
			op = "add"
		case token.SUB:
			op = "sub"
		case token.MUL:
			op = "mul"
		case token.QUO:
			op = "div"
		case token.REM:
			op = "srem"
		case token.EQL:
			op = "eq"
		case token.NEQ:
			op = "ne"
		case token.LSS:
			op = "slt"
		case token.LEQ:
			op = "sle"
		case token.GTR:
			op = "sgt"
		case token.GEQ:
			op = "sge"
		default:
			g.refuseAt(e.Pos(), "go2sa S2b unsupported binary %s", n.Op)
			return ""
		}
		t := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = %s %s, %s\n", t, op, l, r)
		g.releaseIfCallTemp(l)
		g.releaseIfCallTemp(r)
		return t
	case *ast.CallExpr:
		return g.lowerCallValue(n)
	case *ast.CompositeLit:
		return g.lowerCompositeLit(n)
	case *ast.SelectorExpr:
		return g.lowerFieldLoad(n)
	case *ast.IndexExpr:
		return g.lowerIndexLoad(n)
	default:
		g.refuseAt(e.Pos(), "go2sa S2b Println needs string or integer expression")
		return ""
	}
}

// isSliceType reports whether e is []int via go/types (S2l).
func (g *saGen) isSliceType(e ast.Expr) bool {
	if g.types == nil {
		if at, ok := e.(*ast.ArrayType); ok && at.Len == nil {
			return true
		}
		return false
	}
	tv, ok := g.types.Types[e]
	if !ok || tv.Type == nil {
		return false
	}
	if s, ok := tv.Type.Underlying().(*gotypes.Slice); ok {
		if b, ok := s.Elem().Underlying().(*gotypes.Basic); ok {
			return b.Kind() == gotypes.Int
		}
	}
	return false
}

// lowerMakeSlice lowers `make([]int, n[, cap])` to 16B {ptr,len} head + n*8 buffer (S2l).
func (g *saGen) lowerMakeSlice(n *ast.CallExpr) string {
	if len(n.Args) < 2 || len(n.Args) > 3 {
		g.refuseAt(n.Pos(), "go2sa S2l make([]int, n[, cap]) needs 2-3 args")
		return ""
	}
	t0, ok := n.Args[0].(*ast.ArrayType)
	if !ok || t0.Len != nil {
		g.refuseAt(n.Args[0].Pos(), "go2sa S2l make needs []int type")
		return ""
	}
	if id, ok := t0.Elt.(*ast.Ident); !ok || id.Name != "int" {
		g.refuseAt(t0.Elt.Pos(), "go2sa S2l slice elem must be int")
		return ""
	}
	if !g.requireI32(n.Args[1], "make length") {
		return ""
	}
	ln := g.lowerI32(n.Args[1])
	if ln == "" {
		return ""
	}
	if len(n.Args) == 3 {
		if !g.requireI32(n.Args[2], "make cap") {
			return ""
		}
		cp := g.lowerI32(n.Args[2])
		if cp == "" {
			return ""
		}
		g.releaseIfCallTemp(cp)
		_ = cp // cap accepted (=len used); over-cap stays refused in P2
	}
	head := g.nextTmp()
	buf := g.nextTmp()
	nb := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", nb, ln)
	fmt.Fprintf(&g.sb, "  %s = alloc %s\n", buf, nb)
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", head)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", head, buf)
	fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", head, ln)
	fmt.Fprintf(&g.sb, "  !%s\n", buf)
	fmt.Fprintf(&g.sb, "  !%s\n", nb)
	g.releaseIfCallTemp(ln)
	return head
}

// lowerMakeMap lowers `make(map[int]int)` to a btree handle (S2o).
func (g *saGen) lowerMakeMap(n *ast.CallExpr) string {
	if len(n.Args) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2o make(map[int]int) needs exactly 1 arg")
		return ""
	}
	if _, ok := n.Args[0].(*ast.MapType); !ok {
		return ""
	}
	g.addImport("sa_std/collections/btree_map.sa")
	g.addImport("sa_std/core/slice.sa")
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  EXPAND BTREE_MAP_NEW %s\n", t)
	return t
}

// lowerMapKey encodes an int key as an 8-byte Slice head for btree (S2o).
func (g *saGen) lowerMapKey(k string) (string, string) {
	kb := g.nextTmp()
	kh := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 8\n", kb)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as u64\n", kb, k)
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", kh)
	fmt.Fprintf(&g.sb, "  EXPAND SLICE_NEW %s, %s, 8\n", kh, kb)
	return kh, kb
}

// lowerMapAssign lowers `m[k] = v` via BTREE_MAP_INSERT (S2o).
func (g *saGen) lowerMapAssign(n *ast.AssignStmt, ix *ast.IndexExpr) {
	if n.Tok == token.DEFINE {
		g.refuseAt(n.Pos(), "go2sa S2o map element store must use = (not :=)")
		return
	}
	if len(n.Rhs) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2o map element store needs exactly 1 rhs")
		return
	}
	xid, ok := ix.X.(*ast.Ident)
	if !ok || g.mapVars == nil || !g.mapVars[xid.Name] {
		g.refuseAt(ix.Pos(), "go2sa S2o index needs a map variable")
		return
	}
	if !g.vars[xid.Name] {
		g.refuseAt(ix.Pos(), "go2sa S2o use of undeclared variable %q", xid.Name)
		return
	}
	if !g.requireI32(ix.Index, "map key") || !g.requireI32(n.Rhs[0], "map value") {
		return
	}
	k := g.lowerI32(ix.Index)
	if k == "" {
		return
	}
	v := g.lowerI32(n.Rhs[0])
	if v == "" {
		return
	}
	g.addImport("sa_std/collections/btree_map.sa")
	g.addImport("sa_std/core/slice.sa")
	kh, kb := g.lowerMapKey(k)
	fmt.Fprintf(&g.sb, "  EXPAND BTREE_MAP_INSERT %s, %s, %s\n", xid.Name, kh, v)
	fmt.Fprintf(&g.sb, "  !%s\n", kh)
	fmt.Fprintf(&g.sb, "  !%s\n", kb)
	g.releaseIfCallTemp(k)
	g.releaseIfCallTemp(v)
}

// lowerMapLoad lowers `m[k]` reads via try_get slot (S2o, missing key = 0).
func (g *saGen) lowerMapLoad(n *ast.IndexExpr) string {
	xid, ok := n.X.(*ast.Ident)
	if !ok || g.mapVars == nil || !g.mapVars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2o index needs a map variable")
		return ""
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2o use of undeclared variable %q", xid.Name)
		return ""
	}
	if !g.requireI32(n.Index, "map key") {
		return ""
	}
	k := g.lowerI32(n.Index)
	if k == "" {
		return ""
	}
	g.addImport("sa_std/collections/btree_map.sa")
	g.addImport("sa_std/core/slice.sa")
	kh, kb := g.lowerMapKey(k)
	slot := g.nextTmp()
	found := g.nextTmp()
	out := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc 8\n", slot)
	fmt.Fprintf(&g.sb, "  %s = call @sa_btree_map_try_get(&%s, &%s, &%s)\n", found, xid.Name, kh, slot)
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as u64\n", out, slot)
	fmt.Fprintf(&g.sb, "  !%s\n", found)
	fmt.Fprintf(&g.sb, "  !%s\n", slot)
	fmt.Fprintf(&g.sb, "  !%s\n", kh)
	fmt.Fprintf(&g.sb, "  !%s\n", kb)
	g.releaseIfCallTemp(k)
	if g.callTmps == nil {
		g.callTmps = map[string]bool{}
	}
	g.callTmps[out] = true
	return out
}

// lowerMapLen lowers `len(m)` for map vars via btree len (S2o).
func (g *saGen) lowerMapLen(n *ast.CallExpr) string {
	xid := n.Args[0].(*ast.Ident)
	if !g.vars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2o use of undeclared variable %q", xid.Name)
		return ""
	}
	g.addImport("sa_std/collections/btree_map.sa")
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = call @sa_btree_map_len(&%s)\n", t, xid.Name)
	if g.callTmps == nil {
		g.callTmps = map[string]bool{}
	}
	g.callTmps[t] = true
	return t
}

// lowerDelete lowers `delete(m, k)` (S2o): remove returns old value (dropped).
func (g *saGen) lowerDelete(n *ast.CallExpr) {
	if len(n.Args) != 2 {
		g.refuseAt(n.Pos(), "go2sa S2o delete(m, k) needs exactly 2 args")
		return
	}
	xid, ok := n.Args[0].(*ast.Ident)
	if !ok || g.mapVars == nil || !g.mapVars[xid.Name] {
		g.refuseAt(n.Args[0].Pos(), "go2sa S2o delete needs a map variable")
		return
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Args[0].Pos(), "go2sa S2o use of undeclared variable %q", xid.Name)
		return
	}
	if !g.requireI32(n.Args[1], "map key") {
		return
	}
	k := g.lowerI32(n.Args[1])
	if k == "" {
		return
	}
	g.addImport("sa_std/collections/btree_map.sa")
	g.addImport("sa_std/core/slice.sa")
	kh, kb := g.lowerMapKey(k)
	old := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = call @sa_btree_map_remove(&%s, &%s)\n", old, xid.Name, kh)
	fmt.Fprintf(&g.sb, "  !%s\n", old)
	fmt.Fprintf(&g.sb, "  !%s\n", kh)
	fmt.Fprintf(&g.sb, "  !%s\n", kb)
	g.releaseIfCallTemp(k)
}

// lowerLen lowers `len(s)` for slice vars (S2l) and strings (S2p head len).
func (g *saGen) lowerLen(n *ast.CallExpr) string {
	if len(n.Args) != 1 {
		g.refuseAt(n.Pos(), "go2sa S2l len needs exactly 1 arg")
		return ""
	}
	xid, ok := n.Args[0].(*ast.Ident)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2l len needs a slice variable")
		return ""
	}
	if g.strVars != nil && g.strVars[xid.Name] {
		if !g.vars[xid.Name] {
			g.refuseAt(n.Pos(), "go2sa S2p use of undeclared variable %q", xid.Name)
			return ""
		}
		t := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", t, xid.Name)
		return t
	}
	if !g.isSliceVar(xid.Name) {
		g.refuseAt(n.Pos(), "go2sa S2l len needs a slice variable")
		return ""
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2l use of undeclared variable %q", xid.Name)
		return ""
	}
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", t, xid.Name)
	return t
}

// lowerAppend lowers `append(s, v)` (S2m P1 single value): growth-copy to
// new (len+1) buffer, stores v at tail, returns new 16B head. `s...` spread refused.
func (g *saGen) lowerAppend(n *ast.CallExpr) string {
	if n.Ellipsis.IsValid() {
		g.refuseAt(n.Pos(), "go2sa S2m append spread not supported")
		return ""
	}
	if len(n.Args) != 2 {
		g.refuseAt(n.Pos(), "go2sa S2m append(s, v) needs exactly 2 args")
		return ""
	}
	xid, ok := n.Args[0].(*ast.Ident)
	if !ok || !g.isSliceVar(xid.Name) {
		g.refuseAt(n.Args[0].Pos(), "go2sa S2m append needs a slice variable")
		return ""
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Args[0].Pos(), "go2sa S2m use of undeclared variable %q", xid.Name)
		return ""
	}
	if !g.requireI32(n.Args[1], "append value") {
		return ""
	}
	v := g.lowerI32(n.Args[1])
	if v == "" {
		return ""
	}
	oldLen := g.nextTmp()
	oldBuf := g.nextTmp()
	newLen := g.nextTmp()
	newNb := g.nextTmp()
	newBuf := g.nextTmp()
	newHead := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 8 as u64\n", oldLen, xid.Name)
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", oldBuf, xid.Name)
	fmt.Fprintf(&g.sb, "  %s = add %s, 1\n", newLen, oldLen)
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", newNb, newLen)
	fmt.Fprintf(&g.sb, "  %s = alloc %s\n", newBuf, newNb)
	fmt.Fprintf(&g.sb, "  %s = alloc 16\n", newHead)
	// Copy loop over old elems (raw SA labels; loop-local temps).
	g.lbl++
	topL := g.lbl
	g.lbl++
	bodyL := g.lbl
	g.lbl++
	endL := g.lbl
	ix := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = add 0, 0\n", ix)
	fmt.Fprintf(&g.sb, "L_ap_top_%d:\n", topL)
	c := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = slt %s, %s\n", c, ix, oldLen)
	fmt.Fprintf(&g.sb, "  br %s -> L_ap_body_%d, L_ap_end_%d\n", c, bodyL, endL)
	fmt.Fprintf(&g.sb, "L_ap_body_%d:\n", bodyL)
	so := g.nextTmp()
	do := g.nextTmp()
	sp := g.nextTmp()
	dp := g.nextTmp()
	ev := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", so, ix)
	fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", sp, oldBuf, so)
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as u64\n", ev, sp)
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", do, ix)
	fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", dp, newBuf, do)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as u64\n", dp, ev)
	ni := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = add %s, 1\n", ni, ix)
	fmt.Fprintf(&g.sb, "  !%s\n", ix)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", ix, ni)
	fmt.Fprintf(&g.sb, "  jmp L_ap_top_%d\n", topL)
	fmt.Fprintf(&g.sb, "L_ap_end_%d:\n", endL)
	fmt.Fprintf(&g.sb, "  !%s\n", ix)
	to := g.nextTmp()
	tp := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", to, oldLen)
	fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", tp, newBuf, to)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as u64\n", tp, v)
	fmt.Fprintf(&g.sb, "  store %s + 0, %s as ptr\n", newHead, newBuf)
	fmt.Fprintf(&g.sb, "  store %s + 8, %s as u64\n", newHead, newLen)
	fmt.Fprintf(&g.sb, "  !%s\n", newBuf)
	fmt.Fprintf(&g.sb, "  !%s\n", newNb)
	g.releaseIfCallTemp(v)
	return newHead
}

// lowerIndexLoad lowers `s[i]` to element temp (S2l, 8B int slots).
// S2o: map vars dispatch to btree try_get.
func (g *saGen) lowerIndexLoad(n *ast.IndexExpr) string {
	if xid, ok := n.X.(*ast.Ident); ok && g.mapVars != nil && g.mapVars[xid.Name] {
		return g.lowerMapLoad(n)
	}
	xid, ok := n.X.(*ast.Ident)
	if !ok || !g.isSliceVar(xid.Name) {
		g.refuseAt(n.Pos(), "go2sa S2l index needs a slice variable")
		return ""
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2l use of undeclared variable %q", xid.Name)
		return ""
	}
	if !g.requireI32(n.Index, "slice index") {
		return ""
	}
	ix := g.lowerI32(n.Index)
	if ix == "" {
		return ""
	}
	base := g.nextTmp()
	off := g.nextTmp()
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as ptr\n", base, xid.Name)
	fmt.Fprintf(&g.sb, "  %s = mul %s, 8\n", off, ix)
	fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", base, base, off)
	fmt.Fprintf(&g.sb, "  %s = load %s + 0 as u64\n", t, base)
	g.releaseIfCallTemp(ix)
	return t
}

// lowerCompositeLit lowers `T{f: v, ...}` / `T{v, ...}` (S2k, int/bool fields).
// Allocates size bytes, stores each field at its 8B offset. Missing fields zero-init.
func (g *saGen) lowerCompositeLit(n *ast.CompositeLit) string {
	id, ok := n.Type.(*ast.Ident)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k composite literal needs a named struct type")
		return ""
	}
	si, ok := g.structs[id.Name]
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k unknown struct type %q", id.Name)
		return ""
	}
	vals := map[string]ast.Expr{}
	posVals := []ast.Expr{}
	for _, e := range n.Elts {
		if kv, ok := e.(*ast.KeyValueExpr); ok {
			kid, ok := kv.Key.(*ast.Ident)
			if !ok {
				g.refuseAt(kv.Key.Pos(), "go2sa S2k struct key must be a field name")
				return ""
			}
			if _, ok := si.off[kid.Name]; !ok {
				g.refuseAt(kv.Key.Pos(), "go2sa S2k unknown field %q (type %q)", kid.Name, si.name)
				return ""
			}
			if _, dup := vals[kid.Name]; dup {
				g.refuseAt(kv.Key.Pos(), "go2sa S2k duplicate field %q", kid.Name)
				return ""
			}
			vals[kid.Name] = kv.Value
			continue
		}
		posVals = append(posVals, e)
	}
	if len(posVals) > 0 && len(vals) > 0 {
		g.refuseAt(n.Pos(), "go2sa S2k mixed positional/keyed literal not supported")
		return ""
	}
	if len(posVals) > len(si.fields) {
		g.refuseAt(n.Pos(), "go2sa S2k too many values for struct %q", si.name)
		return ""
	}
	for i, e := range posVals {
		vals[si.fields[i]] = e
	}
	base := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = alloc %d\n", base, si.size)
	for _, f := range si.fields {
		e, ok := vals[f]
		if !ok {
			continue // zero-init: alloc is zeroed
		}
		if !g.requireI32(e, "struct field "+f) {
			return ""
		}
		v := g.lowerI32(e)
		if v == "" {
			return ""
		}
		fmt.Fprintf(&g.sb, "  store %s + %d, %s as u64\n", base, si.off[f], v)
		g.releaseIfCallTemp(v)
	}
	return base
}

// lowerFieldLoad lowers `p.F` to a loaded temp (S2k).
func (g *saGen) lowerFieldLoad(n *ast.SelectorExpr) string {
	xid, ok := n.X.(*ast.Ident)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k field access needs a variable")
		return ""
	}
	tname, ok := g.structVars[xid.Name]
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k %q is not a struct variable", xid.Name)
		return ""
	}
	si, ok := g.structs[tname]
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k unknown struct type %q", tname)
		return ""
	}
	off, ok := si.off[n.Sel.Name]
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2k unknown field %q (type %q)", n.Sel.Name, si.name)
		return ""
	}
	if !g.vars[xid.Name] {
		g.refuseAt(n.Pos(), "go2sa S2k use of undeclared variable %q", xid.Name)
		return ""
	}
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = load %s + %d as u64\n", t, xid.Name, off)
	return t
}

// isStrAdd reports whether `+` is string concat (S2t): either side string-typed.
func (g *saGen) isStrAdd(n *ast.BinaryExpr) bool {
	return g.isStrOperand(n.X) || g.isStrOperand(n.Y)
}

// isStrOperand reports whether e is a string literal/var/const/concat (S2t, no lowering).
func (g *saGen) isStrOperand(e ast.Expr) bool {
	if isStrLiteral(e) {
		return true
	}
	if be, ok := e.(*ast.BinaryExpr); ok && be.Op == token.ADD {
		return g.isStrOperand(be.X) && g.isStrOperand(be.Y)
	}
	if id, ok := e.(*ast.Ident); ok {
		if g.strVars != nil && g.strVars[id.Name] {
			return true
		}
		if cv, ok := g.constVals[id.Name]; ok && !g.isShadowedVar(id.Name) && cv.kind == "string" {
			return true
		}
	}
	if g.types != nil {
		if tv, ok := g.types.Types[e]; ok && tv.Type != nil {
			if b, ok := tv.Type.Underlying().(*gotypes.Basic); ok {
				return b.Kind() == gotypes.String || b.Kind() == gotypes.UntypedString
			}
		}
	}
	return false
}

// lowerStrConcat lowers string `a + b` via STR_CONCAT (S2t, returns 16B head).
// Non-string side refused (no implicit int→string).
func (g *saGen) lowerStrConcat(n *ast.BinaryExpr) string {
	if !g.isStrOperand(n.X) || !g.isStrOperand(n.Y) {
		g.refuseAt(n.Pos(), "go2sa S2t string concat needs two strings")
		return ""
	}
	lh := g.lowerStrHead(n.X)
	if lh == "" {
		return ""
	}
	rh := g.lowerStrHead(n.Y)
	if rh == "" {
		return ""
	}
	g.addImport("sa_std/string.sa")
	g.addImport("sa_std/core/slice.sa")
	out := g.nextTmp()
	fmt.Fprintf(&g.sb, "  EXPAND STR_CONCAT %s, %s, %s\n", out, lh, rh)
	if len(lh) > 2 && lh[0] == 't' && lh[1] == '_' {
		fmt.Fprintf(&g.sb, "  !%s\n", lh)
	}
	if len(rh) > 2 && rh[0] == 't' && rh[1] == '_' {
		fmt.Fprintf(&g.sb, "  !%s\n", rh)
	}
	return out
}

// lowerBoolOp lowers `a && b` / `a || b` over i32-as-bool (S2j, no short-circuit).
// `&&` = normalized mul; `||` = normalized add (any nonzero = true, matches S2f case-merge).
func (g *saGen) lowerBoolOp(n *ast.BinaryExpr) string {
	l := g.lowerI32(n.X)
	r := g.lowerI32(n.Y)
	if l == "" || r == "" {
		return ""
	}
	ln := g.nextTmp()
	rn := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", ln, l)
	fmt.Fprintf(&g.sb, "  %s = ne %s, 0\n", rn, r)
	g.releaseIfCallTemp(l)
	g.releaseIfCallTemp(r)
	t := g.nextTmp()
	if n.Op == token.LAND {
		fmt.Fprintf(&g.sb, "  %s = mul %s, %s\n", t, ln, rn)
	} else {
		fmt.Fprintf(&g.sb, "  %s = add %s, %s\n", t, ln, rn)
	}
	return t
}

// lowerCallValue lowers a call to a user value-func returning i32 (S2g).
// S2l: `make([]int,n)` and `len(s)` dispatched before user funcs.
// S2m: `append(s, v...)` dispatched here (growth-copy, returns new head).
func (g *saGen) lowerCallValue(n *ast.CallExpr) string {
	if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "make" {
		if g.isMakeMap(n) {
			return g.lowerMakeMap(n)
		}
		return g.lowerMakeSlice(n)
	}
	if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "len" {
		if len(n.Args) == 1 {
			if xid, ok := n.Args[0].(*ast.Ident); ok && g.mapVars != nil && g.mapVars[xid.Name] {
				return g.lowerMapLen(n)
			}
		}
		return g.lowerLen(n)
	}
	if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "delete" {
		g.refuseAt(n.Pos(), "go2sa S2o delete must be a statement")
		return ""
	}
	if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "append" {
		return g.lowerAppend(n)
	}
	id, ok := n.Fun.(*ast.Ident)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2g only supports direct calls to local funcs")
		return ""
	}
	sc, ok := g.funcs[id.Name]
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2g call to undefined func %q", id.Name)
		return ""
	}
	if !sc.ret {
		g.refuseAt(n.Pos(), "go2sa S2g void func %q cannot be used as a value", id.Name)
		return ""
	}
	args := g.lowerCallArgs(n, sc)
	if args == nil {
		return ""
	}
	t := g.nextTmp()
	fmt.Fprintf(&g.sb, "  %s = call @%s(%s)\n", t, id.Name, strings.Join(args, ", "))
	for _, a := range args {
		g.releaseIfCallTemp(a)
	}
	if g.callTmps == nil {
		g.callTmps = map[string]bool{}
	}
	g.callTmps[t] = true
	return t
}

// releaseIfCallTemp emits `!v` if v is a pending call-result temp (S2g single-use chain).
func (g *saGen) releaseIfCallTemp(v string) {
	if g.callTmps != nil && g.callTmps[v] {
		delete(g.callTmps, v)
		fmt.Fprintf(&g.sb, "  !%s\n", v)
	}
}

// lowerCallArgs checks arity and lowers i32 args (S2g).
func (g *saGen) lowerCallArgs(n *ast.CallExpr, sc *funcScope) []string {
	if len(n.Args) != len(sc.params) {
		g.refuseAt(n.Pos(), "go2sa S2g func %q wants %d args (got %d)", sc.name, len(sc.params), len(n.Args))
		return nil
	}
	if n.Ellipsis.IsValid() {
		g.refuseAt(n.Pos(), "go2sa S2g variadic call not supported (func %q)", sc.name)
		return nil
	}
	var args []string
	for _, a := range n.Args {
		v := g.lowerI32(a)
		if v == "" {
			return nil
		}
		args = append(args, v)
	}
	return args
}
