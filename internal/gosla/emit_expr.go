package gosla

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"
)


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
		if obj := e.info.Uses[ex]; obj != nil {
			if _, isConst := obj.(*types.Const); isConst {
				if lit, ok := constLiteral(obj); ok {
					return lit, nil
				}
				return "", e.refuse(x.Pos(), "unsupported constant %q (only int/bool/string constants fold)", ex.Name)
			}
		}
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
			return "", e.refuse(x.Pos(), "string concatenation not yet supported (needs an SLA-level Str concat surface)")
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
	case *ast.IndexExpr:
		return e.indexExpr(ex)
	case *ast.SelectorExpr:
		xv, err := e.expr(ex.X)
		if err != nil {
			return "", err
		}
		return xv + "." + fieldName(ex.Sel.Name), nil
	case *ast.CompositeLit:
		return e.compositeLit(ex)
	case *ast.CallExpr:
		return e.call(ex)
	default:
		return "", e.refuse(x.Pos(), "unsupported expression %T", x)
	}
}

func (e *emitter) indexExpr(ex *ast.IndexExpr) (string, error) {
	xt := e.info.TypeOf(ex.X)
	if xt != nil && e.isString(xt) {
		return "", e.refuse(ex.Pos(), "string indexing not yet supported (needs an SLA-level Str byte surface)")
	}
	xv, err := e.expr(ex.X)
	if err != nil {
		return "", err
	}
	iv, err := e.expr(ex.Index)
	if err != nil {
		return "", err
	}
	if xt != nil {
		if _, isMap := xt.Underlying().(*types.Map); isMap {
			e.useMap = true
			// Go map read of a missing key yields the zero value.
			return fmt.Sprintf("%s.get(%s).copied().unwrap_or_default()", xv, iv), nil
		}
	}
	return fmt.Sprintf("%s[%s]", xv, iv), nil
}

func (e *emitter) compositeLit(ex *ast.CompositeLit) (string, error) {
	t := e.info.TypeOf(ex)
	if t == nil {
		return "", e.refuse(ex.Pos(), "composite literal of unknown type")
	}
	switch u := t.Underlying().(type) {
	case *types.Struct:
		named, ok := t.(*types.Named)
		if !ok {
			return "", e.refuse(ex.Pos(), "anonymous struct literals not yet supported")
		}
		fields := make([]string, u.NumFields())
		vals := make([]string, u.NumFields())
		for i := 0; i < u.NumFields(); i++ {
			fields[i] = fieldName(u.Field(i).Name())
			zv, _ := e.zeroValue(u.Field(i).Type())
			vals[i] = zv
		}
		for i, elt := range ex.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				key := kv.Key.(*ast.Ident).Name
				for j := 0; j < u.NumFields(); j++ {
					if u.Field(j).Name() == key {
						v, err := e.expr(kv.Value)
						if err != nil {
							return "", err
						}
						vals[j] = v
					}
				}
			} else {
				v, err := e.expr(elt)
				if err != nil {
					return "", err
				}
				vals[i] = v
			}
		}
		var parts []string
		for i := range fields {
			parts = append(parts, fields[i]+": "+vals[i])
		}
		return fmt.Sprintf("%s { %s }", named.Obj().Name(), strings.Join(parts, ", ")), nil
	case *types.Array:
		ez, _ := e.zeroValue(u.Elem())
		vals := make([]string, u.Len())
		for i := range vals {
			vals[i] = ez
		}
		for _, elt := range ex.Elts {
			if kv, ok := elt.(*ast.KeyValueExpr); ok {
				idx, err := e.expr(kv.Key)
				if err != nil {
					return "", err
				}
				var n int
				if _, err := fmt.Sscanf(idx, "%d", &n); err != nil {
					return "", e.refuse(ex.Pos(), "array literal key must be a constant index")
				}
				v, err := e.expr(kv.Value)
				if err != nil {
					return "", err
				}
				vals[n] = v
			}
		}
		pos := 0
		for _, elt := range ex.Elts {
			if _, ok := elt.(*ast.KeyValueExpr); ok {
				continue
			}
			v, err := e.expr(elt)
			if err != nil {
				return "", err
			}
			vals[pos] = v
			pos++
		}
		return "[" + strings.Join(vals, ", ") + "]", nil
	default:
		return "", e.refuse(ex.Pos(), "composite literal of %T not yet supported", t.Underlying())
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
			at := e.info.TypeOf(arg)
			if e.isString(at) {
				e.line("println(%s);", v)
			} else {
				e.line("println(\"{}\", %s);", v)
			}
			return "", nil
		}
		return "", e.refuse(c.Pos(), "unsupported selector call")
	}
	id, ok := c.Fun.(*ast.Ident)
	if !ok {
		return "", e.refuse(c.Pos(), "unsupported call target %T", c.Fun)
	}
	switch id.Name {
	case "len":
		if len(c.Args) != 1 {
			return "", e.refuse(c.Pos(), "len needs 1 arg")
		}
		v, err := e.expr(c.Args[0])
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("len(%s)", v), nil
	case "append", "make", "delete":
		return "", e.refuse(c.Pos(), "%s is only supported in statement position", id.Name)
	}
	var args []string
	for _, a := range c.Args {
		v, err := e.expr(a)
		if err != nil {
			return "", err
		}
		args = append(args, v)
	}
	return fmt.Sprintf("%s(%s)", id.Name, strings.Join(args, ", ")), nil
}
