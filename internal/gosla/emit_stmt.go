package gosla

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
)

// isMakeOrAppend reports whether x is a make()/append() call needing
// statement-level emission, and returns the call.
func asCall(x ast.Expr) *ast.CallExpr {
	c, _ := x.(*ast.CallExpr)
	return c
}

func builtinName(c *ast.CallExpr) string {
	if c == nil {
		return ""
	}
	if id, ok := c.Fun.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// bindTarget emits a let/assignment of value to name, handling make() specially.
func (e *emitter) bind(name, value string, define bool) {
	if define {
		e.line("let %s = %s;", name, value)
	} else {
		e.line("%s = %s;", name, value)
	}
}

// emitMakeSlice emits `Vec::new()` + zero fill for make([]T, n).
func (e *emitter) emitMakeSlice(name string, c *ast.CallExpr, define bool) error {
	if len(c.Args) < 1 || len(c.Args) > 2 {
		return e.refuse(c.Pos(), "make with %d args not yet supported", len(c.Args))
	}
	e.useVec = true
	e.bind(name, "Vec::new()", define)
	if len(c.Args) == 2 {
		n, err := e.expr(c.Args[1])
		if err != nil {
			return err
		}
		ctr := e.tmp()
		e.line("let %s = 0;", ctr)
		e.line("while %s < %s {", ctr, n)
		e.ind++
		e.line("%s.push(0);", name)
		e.line("%s = %s + 1;", ctr, ctr)
		e.ind--
		e.line("}")
	}
	return nil
}

func (e *emitter) stmt(s ast.Stmt) error {
	switch st := s.(type) {
	case *ast.DeclStmt:
		gd, ok := st.Decl.(*ast.GenDecl)
		if !ok {
			return e.refuse(s.Pos(), "unsupported declaration statement")
		}
		if gd.Tok == token.CONST {
			return nil // folded at use sites
		}
		if gd.Tok != token.VAR {
			return e.refuse(s.Pos(), "unsupported declaration statement %s", gd.Tok)
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, n := range vs.Names {
				if i < len(vs.Values) {
					if err := e.bindValue(n.Name, vs.Values[i], true); err != nil {
						return err
					}
				} else {
					zt, ok := e.zeroValue(e.info.TypeOf(vs.Type))
					if !ok {
						return e.refuse(s.Pos(), "var without value of this type not yet supported")
					}
					e.bind(n.Name, zt, true)
				}
			}
		}
		return nil
	case *ast.AssignStmt:
		return e.assign(st)
	case *ast.IncDecStmt:
		return e.incDec(st)
	case *ast.GoStmt:
		return e.goStmt(st)
	case *ast.SendStmt:
		return e.sendStmt(st)
	case *ast.SelectStmt:
		return e.selectStmt(st)
	case *ast.ReturnStmt:
		if e.liveGo() {
			return e.refuse(s.Pos(), "return with live goroutines/channels not yet supported")
		}
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
		if c := asCall(st.X); builtinName(c) == "delete" {
			if len(c.Args) != 2 {
				return e.refuse(s.Pos(), "delete needs 2 args")
			}
			mv, err := e.expr(c.Args[0])
			if err != nil {
				return err
			}
			kv, err := e.expr(c.Args[1])
			if err != nil {
				return err
			}
			e.useMap = true
			e.line("%s.remove(%s);", mv, kv)
			return nil
		}
		v, err := e.expr(st.X)
		if err != nil {
			return err
		}
		if v != "" {
			e.line("%s;", v)
		}
		return nil
	case *ast.BranchStmt:
		switch st.Tok {
		case token.BREAK:
			if st.Label != nil {
				return e.refuse(s.Pos(), "labelled break not yet supported")
			}
			e.line("break;")
			return nil
		case token.CONTINUE:
			if st.Label != nil {
				return e.refuse(s.Pos(), "labelled continue not yet supported")
			}
			if len(e.rangeIncr) > 0 {
				e.line("%s", e.rangeIncr[len(e.rangeIncr)-1])
			}
			e.line("continue;")
			return nil
		default:
			return e.refuse(s.Pos(), "unsupported branch statement %s", st.Tok)
		}
	case *ast.IfStmt:
		return e.ifStmt(st)
	case *ast.ForStmt:
		return e.forStmt(st)
	case *ast.RangeStmt:
		return e.rangeStmt(st)
	case *ast.SwitchStmt:
		return e.switchStmt(st)
	case *ast.BlockStmt:
		return e.block(st, true)
	default:
		return e.refuse(s.Pos(), "unsupported statement %T", s)
	}
}

// zeroValue renders the Go zero value for arrays and scalars.
func (e *emitter) zeroValue(t types.Type) (string, bool) {
	if t == nil {
		return "", false
	}
	switch u := t.Underlying().(type) {
	case *types.Basic:
		switch u.Kind() {
		case types.Int:
			return "0", true
		case types.Bool:
			return "false", true
		case types.String:
			return `""`, true
		}
	case *types.Array:
		ez, ok := e.zeroValue(u.Elem())
		if !ok {
			return "", false
		}
		out := "["
		for i := int64(0); i < u.Len(); i++ {
			if i > 0 {
				out += ", "
			}
			out += ez
		}
		return out + "]", true
	}
	return "", false
}

func (e *emitter) bindValue(name string, val ast.Expr, define bool) error {
	if c := asCall(val); c != nil {
		switch builtinName(c) {
		case "make":
			return e.bindMake(name, c, define)
		case "append":
			return e.refuse(val.Pos(), "append outside an assignment not yet supported")
		}
	}
	v, err := e.expr(val)
	if err != nil {
		return err
	}
	e.bind(name, v, define)
	return nil
}

func (e *emitter) bindMake(name string, c *ast.CallExpr, define bool) error {
	ct := e.info.TypeOf(c)
	if ct == nil {
		return e.refuse(c.Pos(), "make: unknown type")
	}
	switch ct.Underlying().(type) {
	case *types.Slice:
		return e.emitMakeSlice(name, c, define)
	case *types.Chan:
		return e.makeChan(name, c, define)
	case *types.Map:
		e.useMap = true
		e.bind(name, "HashMap::new()", define)
		return nil
	default:
		return e.refuse(c.Pos(), "make of %T not yet supported", ct.Underlying())
	}
}

func (e *emitter) assign(st *ast.AssignStmt) error {
	// Receive assignment: v := <-ch / v, ok := <-ch (any target count).
	if len(st.Rhs) == 1 {
		if chX, isRecv := recvChan(st.Rhs[0]); isRecv {
			return e.recvAssign(st.Lhs, chX, st.Tok == token.DEFINE, st.Pos())
		}
	}
	if len(st.Lhs) != 1 || len(st.Rhs) != 1 {
		return e.refuse(st.Pos(), "multiple assignment not yet supported")
	}
	rhs := st.Rhs[0]
	// s = append(s, v) → s.push(v);
	if c := asCall(rhs); builtinName(c) == "append" {
		if len(c.Args) != 2 {
			return e.refuse(st.Pos(), "append with %d args not yet supported", len(c.Args))
		}
		sv, err := e.expr(c.Args[0])
		if err != nil {
			return err
		}
		vv, err := e.expr(c.Args[1])
		if err != nil {
			return err
		}
		e.useVec = true
		e.line("%s.push(%s);", sv, vv)
		return nil
	}
	lhs := st.Lhs[0]
	switch lt := lhs.(type) {
	case *ast.Ident:
		if lt.Name == "_" {
			_, err := e.expr(rhs)
			return err
		}
		if st.Tok == token.DEFINE {
			return e.bindValue(lt.Name, rhs, true)
		}
		if st.Tok == token.ASSIGN {
			return e.bindValue(lt.Name, rhs, false)
		}
		// compound assignment on a plain variable
		v, err := e.expr(rhs)
		if err != nil {
			return err
		}
		op, ok := compoundOp(st.Tok)
		if !ok {
			return e.refuse(st.Pos(), "unsupported assignment operator %s", st.Tok)
		}
		e.line("%s = (%s %s %s);", lt.Name, lt.Name, op, v)
		return nil
	case *ast.IndexExpr:
		v, err := e.expr(rhs)
		if err != nil {
			return err
		}
		if st.Tok != token.ASSIGN {
			return e.refuse(st.Pos(), "compound assignment to an index not yet supported")
		}
		xt := e.info.TypeOf(lt.X)
		if _, isMap := xt.Underlying().(*types.Map); isMap {
			mv, err := e.expr(lt.X)
			if err != nil {
				return err
			}
			kv, err := e.expr(lt.Index)
			if err != nil {
				return err
			}
			e.useMap = true
			e.line("%s.insert(%s, %s);", mv, kv, v)
			return nil
		}
		xv, err := e.expr(lt.X)
		if err != nil {
			return err
		}
		iv, err := e.expr(lt.Index)
		if err != nil {
			return err
		}
		e.line("%s[%s] = %s;", xv, iv, v)
		return nil
	case *ast.SelectorExpr:
		v, err := e.expr(rhs)
		if err != nil {
			return err
		}
		xv, err := e.expr(lt.X)
		if err != nil {
			return err
		}
		if st.Tok != token.ASSIGN {
			// compound on a field: p.x = p.x op v
			op, ok := compoundOp(st.Tok)
			if !ok {
				return e.refuse(st.Pos(), "unsupported assignment operator %s", st.Tok)
			}
			e.line("%s.%s = (%s.%s %s %s);", xv, fieldName(lt.Sel.Name), xv, fieldName(lt.Sel.Name), op, v)
			return nil
		}
		e.line("%s.%s = %s;", xv, fieldName(lt.Sel.Name), v)
		return nil
	default:
		return e.refuse(st.Pos(), "unsupported assignment target %T", lhs)
	}
}

func compoundOp(t token.Token) (string, bool) {
	switch t {
	case token.ADD_ASSIGN:
		return "+", true
	case token.SUB_ASSIGN:
		return "-", true
	case token.MUL_ASSIGN:
		return "*", true
	case token.QUO_ASSIGN:
		return "/", true
	case token.REM_ASSIGN:
		return "%", true
	}
	return "", false
}

func (e *emitter) incDec(st *ast.IncDecStmt) error {
	v, err := e.expr(st.X)
	if err != nil {
		return err
	}
	op := "+"
	if st.Tok == token.DEC {
		op = "-"
	}
	e.line("%s = %s %s 1;", v, v, op)
	return nil
}

func (e *emitter) ifStmt(st *ast.IfStmt) error {
	if st.Init != nil {
		return e.refuse(st.Pos(), "if with init statement not yet supported")
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
		if err := e.ifStmt(els); err != nil {
			return err
		}
	default:
		return e.refuse(st.Pos(), "unsupported else shape")
	}
	return nil
}

func (e *emitter) forStmt(st *ast.ForStmt) error {
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
	e.loopDepth++
	err := e.block(st.Body, false)
	e.loopDepth--
	if err != nil {
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
}

func (e *emitter) rangeStmt(st *ast.RangeStmt) error {
	xv, err := e.expr(st.X)
	if err != nil {
		return err
	}
	ctr := e.tmp()
	incr := fmt.Sprintf("%s = %s + 1;", ctr, ctr)
	e.line("let %s = 0;", ctr)
	e.line("while %s < len(%s) {", ctr, xv)
	e.ind++
	bindOne := func(tok ast.Expr, isValue bool) error {
		id, ok := tok.(*ast.Ident)
		if !ok {
			return e.refuse(st.Pos(), "range variable must be a plain name")
		}
		if id.Name == "_" {
			return nil
		}
		src := ctr
		if isValue {
			src = fmt.Sprintf("%s[%s]", xv, ctr)
		}
		if st.Tok == token.DEFINE {
			e.line("let %s = %s;", id.Name, src)
		} else {
			e.line("%s = %s;", id.Name, src)
		}
		return nil
	}
	if st.Key != nil {
		if err := bindOne(st.Key, false); err != nil {
			return err
		}
	}
	if st.Value != nil {
		if err := bindOne(st.Value, true); err != nil {
			return err
		}
	}
	e.rangeIncr = append(e.rangeIncr, incr)
	e.loopDepth++
	err = e.block(st.Body, false)
	e.loopDepth--
	e.rangeIncr = e.rangeIncr[:len(e.rangeIncr)-1]
	if err != nil {
		return err
	}
	e.line("%s", incr)
	e.ind--
	e.line("}")
	return nil
}

func (e *emitter) switchStmt(st *ast.SwitchStmt) error {
	if st.Init != nil {
		return e.refuse(st.Pos(), "switch with init statement not yet supported")
	}
	if st.Tag == nil {
		return e.refuse(st.Pos(), "tagless switch not yet supported")
	}
	tag, err := e.expr(st.Tag)
	if err != nil {
		return err
	}
	// bind non-trivial tags once
	if _, ok := st.Tag.(*ast.Ident); !ok {
		if _, ok := st.Tag.(*ast.BasicLit); !ok {
			tv := e.tmp()
			e.line("let %s = %s;", tv, tag)
			tag = tv
		}
	}
	e.line("switch %s {", tag)
	e.ind++
	hasDefault := false
	for _, cs := range st.Body.List {
		cc := cs.(*ast.CaseClause)
		if cc.List == nil {
			hasDefault = true
			e.line("default => {")
			e.ind++
			if err := e.caseBody(cc.Body); err != nil {
				return err
			}
			e.ind--
			e.line("},")
			continue
		}
		for _, cv := range cc.List {
			v, err := e.expr(cv)
			if err != nil {
				return err
			}
			e.line("%s => {", v)
			e.ind++
			if err := e.caseBody(cc.Body); err != nil {
				return err
			}
			e.ind--
			e.line("},")
		}
	}
	if !hasDefault {
		e.line("default => {},")
	}
	e.ind--
	e.line("};")
	return nil
}

// caseBody emits a case's statements; a break truncates the rest (Go semantics:
// break exits the switch, and statements after it in the case are unreachable).
func (e *emitter) caseBody(body []ast.Stmt) error {
	for _, s := range body {
		if bs, ok := s.(*ast.BranchStmt); ok && bs.Tok == token.BREAK && bs.Label == nil {
			return nil
		}
		if err := e.stmt(s); err != nil {
			return err
		}
	}
	return nil
}
