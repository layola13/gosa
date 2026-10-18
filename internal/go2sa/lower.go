// Package go2sa lowering: AST -> SA text (S2g: S2f + multi-func i32 params/return).
package go2sa

import (
	"fmt"
	"go/ast"
	"go/token"
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
// S2g: S2f + top-level `func name(p,...i32) [i32]` + calls + return; all else loud refusal.
func saLowerFile(fset *token.FileSet, f *ast.File, _ string) *Output {
	g := &saGen{fset: fset, hasImp: map[string]bool{}, strPool: map[string]int{}}
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
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			g.refuseAt(decl.Pos(), "go2sa S2g only supports func declarations (no top-level %T)", decl)
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
			vars:  map[string]bool{},
			funcs: byName, curFunc: sc,
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
	out.WriteString("// Generated by go2sa S2g: parse(go/parser) -> lower -> emit(SA text + sci macros)\n")
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

// checkFuncSig validates S2g signature: params all `name int`, at most one `int` result.
func checkFuncSig(g *saGen, fn *ast.FuncDecl) ([]string, bool, bool) {
	var params []string
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			if !isIntType(field.Type) {
				g.refuseAt(field.Type.Pos(), "go2sa S2g params must be int (func %q)", fn.Name.Name)
				return nil, false, false
			}
			if len(field.Names) == 0 {
				g.refuseAt(field.Type.Pos(), "go2sa S2g params need names (func %q)", fn.Name.Name)
				return nil, false, false
			}
			for _, nm := range field.Names {
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
				typ := field.Type
				if !isIntType(typ) {
					g.refuseAt(field.Type.Pos(), "go2sa S2g result must be int (func %q)", fn.Name.Name)
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

// isIntType reports whether e is plain `int`.
func isIntType(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "int"
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

// lowerDeclStmt lowers `var x = <i32>` / `var x int` / `x := <i32>` (single-name, i32 only).
func (g *saGen) lowerDeclStmt(n *ast.DeclStmt) {
	gd, ok := n.Decl.(*ast.GenDecl)
	if !ok || gd.Tok != token.VAR {
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
			g.declareVar(name, vs.Pos())
			continue
		}
		if len(vs.Values) != 1 {
			g.refuseAt(spec.Pos(), "go2sa S2b var needs exactly 1 value")
			continue
		}
		v := g.lowerI32(vs.Values[0])
		if v == "" {
			return
		}
		g.declareVar(name, vs.Pos())
		fmt.Fprintf(&g.sb, "  !%s\n", name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", name, v)
	}
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
	if !ok {
		g.refuseAt(n.Lhs[0].Pos(), "go2sa S2b assignment target must be an identifier")
		return
	}
	v := g.lowerI32(n.Rhs[0])
	if v == "" {
		return
	}
	if n.Tok == token.DEFINE {
		g.declareVar(id.Name, n.Pos())
		fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
		fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
		g.releaseIfCallTemp(v)
		return
	}
	if !g.vars[id.Name] {
		g.refuseAt(n.Pos(), "go2sa S2b assignment to undeclared variable %q", id.Name)
		return
	}
	fmt.Fprintf(&g.sb, "  !%s\n", id.Name)
	fmt.Fprintf(&g.sb, "  %s = add %s, 0\n", id.Name, v)
	g.releaseIfCallTemp(v)
}

func (g *saGen) lowerExprStmt(n *ast.ExprStmt) {
	call, ok := n.X.(*ast.CallExpr)
	if !ok {
		g.refuseAt(n.Pos(), "go2sa S2g only supports calls as statements")
		return
	}
	if id, ok := call.Fun.(*ast.Ident); ok {
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
	g.emitPrintI32(v)
	g.releaseIfCallTemp(v)
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

// lowerI32 lowers an integer expression, returning the value temp/literal.
func (g *saGen) lowerI32(e ast.Expr) string {
	switch n := e.(type) {
	case *ast.BasicLit:
		if n.Kind != token.INT {
			g.refuseAt(e.Pos(), "go2sa S2a fmt.Println needs string or integer expression")
			return ""
		}
		return n.Value
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
		g.refuseAt(e.Pos(), "go2sa S2a unsupported unary %s", n.Op)
		return ""
	case *ast.BinaryExpr:
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
		default:
			g.refuseAt(e.Pos(), "go2sa S2b unsupported binary %s", n.Op)
			return ""
		}
		t := g.nextTmp()
		fmt.Fprintf(&g.sb, "  %s = %s %s, %s\n", t, op, l, r)
		return t
	case *ast.Ident:
		if !g.vars[n.Name] {
			g.refuseAt(e.Pos(), "go2sa S2b use of undeclared variable %q", n.Name)
			return ""
		}
		return n.Name
	case *ast.CallExpr:
		return g.lowerCallValue(n)
	default:
		g.refuseAt(e.Pos(), "go2sa S2b Println needs string or integer expression")
		return ""
	}
}

// lowerCallValue lowers a call to a user value-func returning i32 (S2g).
func (g *saGen) lowerCallValue(n *ast.CallExpr) string {
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
