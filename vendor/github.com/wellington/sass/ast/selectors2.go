package ast

import (
	"strings"

	"github.com/wellington/sass/token"
)

func Selector(stmt *SelStmt) *BasicLit {
	// log.Printf("\n==Selector=====\n")
	// Merge the selector to groups
	delim := " "
	merged := mergeExpr(delim, stmt.Sel, 0)
	var par string
	if stmt.Parent != nil {
		par = stmt.Parent.Resolved.Value
	}
	// log.Printf("Sel                 %q\n", stmt.Name)
	// log.Printf("Merged              %q\n", merged)
	merged = joinParent(delim, par, merged)
	// log.Printf("Adopted             %q\n", merged)
	return &BasicLit{
		Value:    strings.Join(merged, ","+delim),
		ValuePos: stmt.Pos(),
		Kind:     token.STRING,
	}
}

// mergeExpr recursively merges expressions into slice of groups
// a + b, ~ d => ['a + b', '~ d']
func mergeExpr(delim string, expr Expr, round int) []string {
	// fmt.Printf("round %d: %-15T %v\n", round, expr, expr)
	var ret []string
	// defer func() { fmt.Println(round, ret) }()
	switch v := expr.(type) {
	case *BinaryExpr:
		left := mergeExpr(delim, v.X, round+1)
		right := mergeExpr(delim, v.Y, round+1)
		// Based on precedence, non-comma should never contain
		// commas in left or right
		if v.Op != token.COMMA {
			left[0] = left[0] + delim + v.Op.String() + delim + right[0]
			// Implicit NEST
			if round == 0 && !strings.Contains(left[0], "&") {
				left[0] = "& " + left[0]
			}
			ret = append(ret, left...)
		} else {
			var r []string
			r = append(r, left...)
			r = append(r, right...)
			if round == 0 {
				for i := range r {
					if !strings.Contains(r[i], "&") {
						r[i] = "& " + r[i]
					}
				}
			}
			ret = append(ret, r...)
		}
	case *UnaryExpr:
		val := v.X.(*BasicLit).Value
		if v.Op != token.NEST {
			// Implicit backreference ie div { > e {} }
			pieces := []string{"&", v.Op.String(), val}
			ret = append(ret, strings.Join(pieces, delim))
		} else {
			ret = append(ret, val)
		}
		// X is always BasicLit, at some point this will be enforced
	case *BasicLit:
		if round == 0 {
			ret = append(ret, "& "+v.Value)
		} else {
			ret = append(ret, v.Value)
		}

	}
	return ret
}
