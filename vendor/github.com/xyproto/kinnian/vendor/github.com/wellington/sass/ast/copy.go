package ast

import (
	"fmt"
	"log"
)

// StmtCopy performs a deep copy of passed stmt
func StmtCopy(in Stmt) (out Stmt) {
	switch v := in.(type) {
	case *AssignStmt:
		left := ExprsCopy(v.Lhs)
		right := ExprsCopy(v.Rhs)
		stmt := &AssignStmt{
			Lhs: left,
			Rhs: right,
		}
		out = stmt
	case *IfStmt:

		stmt := &IfStmt{}
		stmt.Body = StmtCopy(v.Body).(*BlockStmt)
		stmt.If = v.If
		if v.Else != nil {
			stmt.Else = StmtCopy(v.Else)
		}
		stmt.Cond = ExprCopy(v.Cond)
		out = stmt
	case *ReturnStmt:
		stmt := &ReturnStmt{
			Return:  v.Return,
			Results: ExprsCopy(v.Results),
		}
		out = stmt
	case *DeclStmt:
		stmt := &DeclStmt{}
		if v.Decl != nil {
			stmt.Decl = DeclCopy(v.Decl)
		}
		out = stmt
	case *BlockStmt:
		block := &BlockStmt{}
		list := make([]Stmt, 0, len(v.List))
		for _, stmt := range v.List {
			list = append(list, StmtCopy(stmt))
		}
		block.List = list
		out = block
	case *SelStmt:
		stmt := &SelStmt{
			Name:    IdentCopy(v.Name),
			NamePos: v.NamePos,
			Body:    StmtCopy(v.Body).(*BlockStmt),
			// SelDecl: DeclCopy(v.SelDecl).(*SelDecl),
			Sel: ExprCopy(v.Sel),
		}
		if v.Parent != nil {
			stmt.Parent = &SelStmt{
				Resolved: v.Parent.Resolved,
			}
		}

		names := make([]*Ident, 0, len(v.Names))
		for _, ident := range v.Names {
			names = append(names, IdentCopy(ident))
		}
		stmt.Names = names
		out = stmt
	case *CommStmt:
		out = v
		return
	case *IncludeStmt:
		stmt := &IncludeStmt{
			Spec: SpecCopy(v.Spec).(*IncludeSpec),
		}
		out = stmt
	case *EachStmt:
		stmt := &EachStmt{}
		stmt.X = v.X
		stmt.Body = StmtCopy(v.Body).(*BlockStmt)
		stmt.List = ExprsCopy(v.List)
		stmt.Each = v.Each
		out = stmt
	case *EmptyStmt:
	default:
		log.Fatalf("unsupported stmt copy %T: % #v\n", v, v)
	}
	// fmt.Printf("StmtCopy (%p)% #v\n      ~> (%p)% #v\n", in, in, out, out)
	return
}

func ExprsCopy(in []Expr) []Expr {
	out := make([]Expr, 0, len(in))
	for i := range in {
		if in[i] != nil {
			out = append(out, ExprCopy(in[i]))
		}
	}
	return out
}

func ExprCopy(in Expr) (out Expr) {
	switch expr := in.(type) {
	case *Ident:
		out = IdentCopy(expr)
	case *UnaryExpr:
		out = &UnaryExpr{
			Op:      expr.Op,
			OpPos:   expr.OpPos,
			X:       ExprCopy(expr.X),
			Visited: expr.Visited,
		}
	case *Interp:
		out = &Interp{
			Lbrace: expr.Lbrace,
			Rbrace: expr.Rbrace,
			X:      ExprsCopy(expr.X),
		}
	case *BinaryExpr:
		out = &BinaryExpr{
			X:     ExprCopy(expr.X),
			Op:    expr.Op,
			OpPos: expr.OpPos,
			Y:     ExprCopy(expr.Y),
		}
	case *BasicLit:
		out = &BasicLit{
			Kind:     expr.Kind,
			Value:    expr.Value,
			ValuePos: expr.ValuePos,
		}
	case *CallExpr:
		out = &CallExpr{
			Rparen: expr.Rparen,
			Lparen: expr.Lparen,
			Args:   ExprsCopy(expr.Args),
			Fun:    ExprCopy(expr.Fun),
		}
	case *KeyValueExpr:
		kv := &KeyValueExpr{}
		kv.Colon = expr.Colon
		kv.Key = ExprCopy(expr.Key)
		kv.Value = ExprCopy(expr.Value)
		out = kv
	case *ListLit:
		lit := &ListLit{
			Comma:    expr.Comma,
			ValuePos: expr.Pos(),
			EndPos:   expr.End(),
		}
		lit.Value = ExprsCopy(expr.Value)
		out = lit
	default:
		panic(fmt.Errorf("unsupported expr copy: % #v\n", expr))
	}
	return
}

// IdentCopy does not resolve *Obj, this will need to
// be looked up after the fact
func IdentCopy(in *Ident) (out *Ident) {
	out = NewIdent(in.Name)
	out.NamePos = in.Pos()
	out.Global = in.Global
	return
}

func FieldCopy(in *Field) (out *Field) {
	out = &Field{}
	out.Doc = in.Doc
	out.Names = make([]*Ident, len(in.Names))
	for i := range in.Names {
		out.Names[i] = IdentCopy(in.Names[i])
	}
	out.Type = ExprCopy(in.Type)
	out.Comment = in.Comment
	return
}

func FieldListCopy(in *FieldList) (out *FieldList) {
	out = &FieldList{}
	if in == nil || in.List == nil {
		return
	}
	list := make([]*Field, len(in.List))
	for i := range in.List {
		list[i] = FieldCopy(in.List[i])
	}
	out.List = list
	return
}

func SpecCopy(in Spec) (out Spec) {
	switch v := in.(type) {
	case *RuleSpec:
		spec := &RuleSpec{
			Name: NewIdent(v.Name.Name),
		}
		list := make([]Expr, 0, len(v.Values))
		for i := range v.Values {
			if v.Values[i] != nil {
				list = append(list, ExprCopy(v.Values[i]))
			}
		}
		spec.Values = list
		out = spec
	case *IncludeSpec:
		spec := &IncludeSpec{
			Name:   IdentCopy(v.Name),
			Params: FieldListCopy(v.Params),
		}
		list := make([]Stmt, len(v.List))
		for i := range v.List {
			list[i] = StmtCopy(v.List[i])
		}
		spec.List = list
		out = spec
	default:
		out = v
		log.Fatalf("unsupported spec copy %T: % #v\n", v, v)
		return
	}
	// fmt.Printf("SpecCopy % #v\n      ~> % #v\n", in, out)
	return
}

func DeclCopy(in Decl) (out Decl) {
	switch v := in.(type) {
	case *SelDecl:
		decl := &SelDecl{
			SelStmt: StmtCopy(v.SelStmt).(*SelStmt),
		}
		out = decl
	case *GenDecl:
		decl := *v
		list := make([]Spec, 0, len(decl.Specs))
		for i := range decl.Specs {
			if decl.Specs[i] != nil {
				list = append(list, SpecCopy(decl.Specs[i]))
			} else {
				fmt.Println("nil!")
			}
		}
		decl.Specs = list
		out = &decl
	default:
		log.Fatalf("unsupported decl copy %T: % #v\n", v, v)
	}
	return
}
