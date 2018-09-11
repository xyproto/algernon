package ast

import (
	"fmt"
	"log"
)

// ToIdent converts expressions to Ident
func ToIdent(expr Expr) *Ident {
	switch v := expr.(type) {
	case *BasicLit:
		return &Ident{
			Name:    v.Value,
			NamePos: v.ValuePos,
		}
	case *Ident:
		return v
	default:
		fmt.Printf("Failed to cast expr to Ident % #v\n", v)
	}
	return nil
}

func ToValue(expr Expr, keys ...string) Expr {
	var key string
	if len(keys) > 0 {
		key = keys[0]
	}
	switch v := expr.(type) {
	case *KeyValueExpr:
		k := v.Key.(*BasicLit)
		val := v.Value.(*Ident)
		if k.Value == key {
			return val
		}
	default:
		log.Printf("failed to cast % #v\n")
	}
	return nil
}
