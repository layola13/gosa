package gosla

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)

// Increment 3: goroutines, channels and select, emitted onto SLA's
// native surface (Chan<T>, thread::spawn, select) per gosa AGENTS.md
// rule 2 — the frontend re-invents no concurrency machinery.
//
// Lifetime discipline (v1): a channel made or a goroutine spawned in a
// block is joined/destroyed at the end of that same block, in spawn
// order (join handles first, then destroy channels). Returning from a
// function with live handles/channels is refused, as is creating them
// inside a loop body (a break could skip the block-end cleanup).

type goScope struct {
	handles []string
	chans   []string
}

func (e *emitter) pushScope() {
	e.scopes = append(e.scopes, goScope{})
}

func (e *emitter) popScopeEmitCleanup() {
	sc := e.scopes[len(e.scopes)-1]
	e.scopes = e.scopes[:len(e.scopes)-1]
	for _, h := range sc.handles {
		e.line("let %s_jr = %s.join();", h, h)
	}
	for _, c := range sc.chans {
		e.line("%s.destroy();", c)
	}
}

func (e *emitter) curScope() *goScope {
	return &e.scopes[len(e.scopes)-1]
}

// liveGo reports whether any scope of the current function still holds
// unjoined handles or undestroyed channels.
func (e *emitter) liveGo() bool {
	for i := e.fnBase; i < len(e.scopes); i++ {
		if len(e.scopes[i].handles) > 0 || len(e.scopes[i].chans) > 0 {
			return true
		}
	}
	return false
}

// makeChan binds `name = Chan::new(cap)` for make(chan T[, cap]) and
// registers the channel for block-end destruction.
func (e *emitter) makeChan(name string, c *ast.CallExpr, define bool) error {
	if !define {
		return e.refuse(c.Pos(), "channel re-make (assignment form) not yet supported")
	}
	if e.loopDepth > 0 {
		return e.refuse(c.Pos(), "make(chan) inside a loop body not yet supported")
	}
	ct := e.info.TypeOf(c)
	u, ok := ct.Underlying().(*types.Chan)
	if !ok {
		return e.refuse(c.Pos(), "make: not a channel type")
	}
	if u.Dir() != types.SendRecv {
		return e.refuse(c.Pos(), "directed channel types (chan<-, <-chan) not yet supported")
	}
	et, ok := e.slaType(u.Elem())
	if !ok {
		return e.refuse(c.Pos(), "unsupported channel element type")
	}
	if len(c.Args) < 1 || len(c.Args) > 2 {
		return e.refuse(c.Pos(), "make(chan) with %d args not yet supported", len(c.Args))
	}
	capStr := "0"
	if len(c.Args) == 2 {
		v, err := e.expr(c.Args[1])
		if err != nil {
			return err
		}
		capStr = v
	}
	e.useChan = true
	e.line("let %s: Chan<%s> = Chan::new(%s);", name, et, capStr)
	e.curScope().chans = append(e.curScope().chans, name)
	return nil
}

// goStmt emits `go f(args)` as a thread::spawn with a generated handle
// registered for block-end join.
func (e *emitter) goStmt(st *ast.GoStmt) error {
	if e.loopDepth > 0 {
		return e.refuse(st.Pos(), "go statement inside a loop body not yet supported")
	}
	c := st.Call
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return e.refuse(st.Pos(), "go of a method value or builtin not yet supported")
	}
	var args []string
	for _, a := range c.Args {
		v, err := e.expr(a)
		if err != nil {
			return err
		}
		args = append(args, v)
	}
	e.useChan = true
	h := e.tmp()
	e.line("let %s = thread::spawn(^|| %s(%s));", h, id.Name, strings.Join(args, ", "))
	e.curScope().handles = append(e.curScope().handles, h)
	return nil
}

// sendStmt emits `ch <- v` as ch.send(v);
func (e *emitter) sendStmt(st *ast.SendStmt) error {
	ch, err := e.expr(st.Chan)
	if err != nil {
		return err
	}
	v, err := e.expr(st.Value)
	if err != nil {
		return err
	}
	e.line("%s.send(%s);", ch, v)
	return nil
}

// recvParts decomposes a receive expression `<-ch`.
func recvChan(x ast.Expr) (ast.Expr, bool) {
	u, ok := x.(*ast.UnaryExpr)
	if !ok || u.Op != token.ARROW {
		return nil, false
	}
	return u.X, true
}

// recvAssign emits a receive assignment `v := <-ch` / `v, ok := <-ch`.
// SLA recv yields (value, closed); Go's ok is the inverse of closed,
// so ok is bound inside from a fresh closed temporary.
func (e *emitter) recvAssign(lhs []ast.Expr, chExpr ast.Expr, define bool, pos token.Pos) error {
	if !define {
		return e.refuse(pos, "receive assignment to existing variables not yet supported")
	}
	if len(lhs) < 1 || len(lhs) > 2 {
		return e.refuse(pos, "receive with %d targets not yet supported", len(lhs))
	}
	name := func(x ast.Expr) (string, error) {
		id, ok := x.(*ast.Ident)
		if !ok {
			return "", e.refuse(pos, "receive target must be a plain name")
		}
		if id.Name == "_" {
			return e.tmp(), nil
		}
		return id.Name, nil
	}
	ch, err := e.expr(chExpr)
	if err != nil {
		return err
	}
	vn, err := name(lhs[0])
	if err != nil {
		return err
	}
	cl := e.tmp()
	e.line("let (%s, %s) = %s.recv();", vn, cl, ch)
	if len(lhs) == 2 {
		okn, err := name(lhs[1])
		if err != nil {
			return err
		}
		e.line("let %s = %s == false;", okn, cl)
	}
	return nil
}

// selectStmt emits a Go select onto an SLA select. Case forms:
//   case v := <-ch:    -> case let (v, __cl) = ch.recv() {
//   case v, ok := <-ch:-> same, plus `let ok = __cl == false;` inside
//   case <-ch:         -> case let (__t, __cl) = ch.recv() {
//   case ch <- v:      -> case ch.send(v) {
// A default clause is refused (its non-blocking semantics have no
// verified SLA select form yet).
func (e *emitter) selectStmt(st *ast.SelectStmt) error {
	e.useChan = true
	e.line("select {")
	e.ind++
	for _, s := range st.Body.List {
		cc, ok := s.(*ast.CommClause)
		if !ok {
			return e.refuse(s.Pos(), "unsupported select clause %T", s)
		}
		if cc.Comm == nil {
			return e.refuse(cc.Pos(), "select default clause not yet supported")
		}
		var preBody []string
		switch comm := cc.Comm.(type) {
		case *ast.AssignStmt:
			chX, isRecv := recvChan(comm.Rhs[0])
			if len(comm.Rhs) != 1 || !isRecv {
				return e.refuse(comm.Pos(), "unsupported select comm %T", comm)
			}
			if comm.Tok != token.DEFINE {
				return e.refuse(comm.Pos(), "select receive assignment (not :=) not yet supported")
			}
			if len(comm.Lhs) < 1 || len(comm.Lhs) > 2 {
				return e.refuse(comm.Pos(), "select receive with %d targets not yet supported", len(comm.Lhs))
			}
			idOf := func(x ast.Expr) (string, error) {
				id, ok := x.(*ast.Ident)
				if !ok {
					return "", e.refuse(comm.Pos(), "select receive target must be a plain name")
				}
				if id.Name == "_" {
					return e.tmp(), nil
				}
				return id.Name, nil
			}
			ch, err := e.expr(chX)
			if err != nil {
				return err
			}
			vn, err := idOf(comm.Lhs[0])
			if err != nil {
				return err
			}
			cl := e.tmp()
			e.line("case let (%s, %s) = %s.recv() {", vn, cl, ch)
			if len(comm.Lhs) == 2 {
				okn, err := idOf(comm.Lhs[1])
				if err != nil {
					return err
				}
				preBody = append(preBody, fmt.Sprintf("let %s = %s == false;", okn, cl))
			}
		case *ast.ExprStmt:
			chX, isRecv := recvChan(comm.X)
			if !isRecv {
				return e.refuse(comm.Pos(), "unsupported select comm expression")
			}
			ch, err := e.expr(chX)
			if err != nil {
				return err
			}
			e.line("case let (%s, %s) = %s.recv() {", e.tmp(), e.tmp(), ch)
		case *ast.SendStmt:
			ch, err := e.expr(comm.Chan)
			if err != nil {
				return err
			}
			v, err := e.expr(comm.Value)
			if err != nil {
				return err
			}
			e.line("case %s.send(%s) {", ch, v)
		default:
			return e.refuse(cc.Comm.Pos(), "unsupported select comm %T", cc.Comm)
		}
		e.ind++
		for _, l := range preBody {
			e.line("%s", l)
		}
		for _, bs := range cc.Body {
			if err := e.stmt(bs); err != nil {
				return err
			}
		}
		e.ind--
		e.line("}")
	}
	e.ind--
	e.line("}")
	return nil
}
