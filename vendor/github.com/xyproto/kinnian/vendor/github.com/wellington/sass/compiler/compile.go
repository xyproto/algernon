package compiler

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/calc"
	"github.com/wellington/sass/parser"
	"github.com/wellington/sass/token"
)

// Context maintains the state of the compiler and handles the output of the
// parser.
type Context struct {
	buf      *bytes.Buffer
	fileName *ast.Ident
	mode     parser.Mode

	err error
	// Records the current level of selectors
	// Each time a selector is encountered, increase
	// by one. Each time a block is exited, remove
	// the last selector
	sels [][]*ast.Ident
	// activeSel maintains the current selector
	// it is never flushed, but will be replaced when the next
	// selstmt is encountered
	activeSel *ast.BasicLit
	// activeMedia maintains the current media query
	// Once flushed, it should never be printed again
	activeMedia *ast.BasicLit
	// indicates that a media closing bracket needs to be
	// flushed
	inMedia     bool
	firstRule   bool // first rules print { otherwise don't
	hiddenBlock bool // @each has hidden blocks, probably other examples of this
	level       int
	printers    map[ast.Node]func(*Context, ast.Node)
	fset        *token.FileSet
	scope       Scope
}

// NewContext returns a new, initialized context
func NewContext() *Context {
	ctx := &Context{}
	ctx.init()
	return ctx
}

// Compile accepts a byte slice and returns a byte slice
func Compile(input []byte) ([]byte, error) {
	ctx := NewContext()
	return ctx.run("", string(input))

}

// Run accepts a path to a Sass file and outputs a string
func Run(path string) (string, error) {
	ctx := NewContext()
	out, err := ctx.run(path, nil)
	if err != nil {
		log.Fatal(err)
	}
	return string(out), err
}

// SetMode modifies the mode that the parser runs in. See parser.Mode for
// available options
func (ctx *Context) SetMode(mode parser.Mode) error {
	ctx.mode = mode
	return nil
}

func (ctx *Context) runString(path string, src interface{}) (string, error) {
	b, err := ctx.run(path, src)
	return string(b), err
}

func (ctx *Context) run(path string, src interface{}) ([]byte, error) {

	ctx.fset = token.NewFileSet()
	// ctx.mode = parser.Trace
	pf, err := parser.ParseFile(ctx.fset, path, src, ctx.mode)
	if err != nil {
		return nil, err
	}

	ast.Walk(ctx, pf)
	lr, _ := utf8.DecodeLastRune(ctx.buf.Bytes())
	_ = lr
	if ctx.buf.Len() > 0 && lr != '\n' {
		ctx.out("\n")
	}
	// ctx.printSels(pf.Decls)
	return ctx.buf.Bytes(), nil
}

// out prints with the appropriate indention, selectors always have indent
// 0
func (ctx *Context) out(v string) {
	fr, _ := utf8.DecodeRuneInString(v)
	if fr == '\n' {
		fmt.Fprintf(ctx.buf, v)
		return
	}
	ws := []byte("                                              ")
	lvl := ctx.level

	format := append(ws[:lvl*2], "%s"...)
	fmt.Fprintf(ctx.buf, string(format), v)
}

// This needs a new name, it prints on every stmt
func (ctx *Context) blockIntro() {

	// this isn't a new block
	if !ctx.firstRule {
		fmt.Fprint(ctx.buf, "\n")
		return
	}

	ctx.firstRule = false

	// Only print newlines if there is text in the buffer
	if ctx.buf.Len() > 0 {
		if ctx.level == 0 {
			fmt.Fprint(ctx.buf, "\n")
		}
	}

	if ctx.activeMedia != nil {
		val := ctx.activeMedia.Value
		ctx.activeMedia = nil
		// media queries have invalid indention, move up one
		ctx.level--
		ctx.out(fmt.Sprintf("%s {\n", val))
		ctx.level++
	}

	sel := "MISSING"
	if ctx.activeSel != nil {
		sel = ctx.activeSel.Value
	}

	ctx.out(fmt.Sprintf("%s {\n", sel))
}

func (ctx *Context) blockOutro() {
	// Remove the innermost selector scope
	// if len(ctx.sels) > 0 {
	// 	ctx.sels = ctx.sels[:len(ctx.sels)-1]
	// }
	// Don't print } if there are no rules at this level
	if ctx.firstRule {
		return
	}

	ctx.firstRule = true
	buf := " }\n"
	if ctx.inMedia {
		ctx.inMedia = false
		buf = " }" + buf
	}
	// if !skipParen {
	fmt.Fprintf(ctx.buf, buf)
	// }
}

// Visit is an internal compiler method. It is exported to allow ast.Walk
// to walk through the parser AST tree.
func (ctx *Context) Visit(node ast.Node) ast.Visitor {
	if ctx.err != nil {
		fmt.Println(ctx.err)
		return nil
	}
	var key ast.Node
	switch v := node.(type) {
	case *ast.BlockStmt:
		if (ctx.scope.RuleLen() > 0 || ctx.activeMedia != nil) &&
			!ctx.hiddenBlock {
			ctx.level = ctx.level + 1
			if !ctx.firstRule {
				fmt.Fprintf(ctx.buf, " }\n")
			}
		}
		ctx.scope = NewScope(ctx.scope)
		if !ctx.hiddenBlock {
			ctx.firstRule = true
		}
		for _, node := range v.List {
			ast.Walk(ctx, node)
		}
		if ctx.level > 0 {
			ctx.level = ctx.level - 1
		}
		ctx.scope = CloseScope(ctx.scope)
		if !ctx.hiddenBlock {
			ctx.blockOutro()
			ctx.firstRule = true
		}
		ctx.hiddenBlock = false
		// ast.Walk(ctx, v.List)
		// fmt.Fprintf(ctx.buf, "}")
		return nil
	case *ast.SelDecl:
	case *ast.File, *ast.GenDecl, *ast.Value:
		// Nothing to print for these
	case *ast.Ident:
		// The first IDENT is always the filename, just preserve
		// it somewhere
		key = ident
	case *ast.PropValueSpec:
		key = propSpec
	case *ast.DeclStmt:
		key = declStmt
	case *ast.IncludeSpec:
		// panic("not supported")
	case *ast.ValueSpec:
		key = valueSpec
	case *ast.RuleSpec:
		key = ruleSpec
	case *ast.SelStmt:
		// We will need to combine parent selectors
		// while printing these
		key = selStmt
		// Nothing to do
	case *ast.CommStmt:
	case *ast.CommentGroup:
	case *ast.Comment:
		key = comment
	case *ast.FuncDecl:
		ctx.printers[funcDecl](ctx, node)
		// Do not traverse mixins in the regular context
		return nil
	case *ast.BasicLit:
		return ctx
	case *ast.CallExpr:
	case nil:
		return ctx
	case *ast.MediaStmt:
		fmt.Println("mediastmt")
		key = mediaStmt
	case *ast.EmptyStmt:
	case *ast.AssignStmt:
		key = assignStmt
	case *ast.EachStmt:
		key = eachStmt
	case *ast.ListLit:
	case *ast.ImportSpec:
	case *ast.IfDecl:
	case *ast.IfStmt:
		key = ifStmt
	default:
		fmt.Printf("add printer for: %T\n", v)
		fmt.Printf("% #v\n", v)
	}
	ctx.printers[key](ctx, node)
	return ctx
}

var (
	ident       *ast.Ident
	expr        ast.Expr
	declStmt    *ast.DeclStmt
	assignStmt  *ast.AssignStmt
	valueSpec   *ast.ValueSpec
	ruleSpec    *ast.RuleSpec
	selDecl     *ast.SelDecl
	selStmt     *ast.SelStmt
	propSpec    *ast.PropValueSpec
	typeSpec    *ast.TypeSpec
	comment     *ast.Comment
	funcDecl    *ast.FuncDecl
	includeSpec *ast.IncludeSpec
	mediaStmt   *ast.MediaStmt
	eachStmt    *ast.EachStmt
	ifStmt      *ast.IfStmt
)

func (ctx *Context) init() {
	ctx.buf = bytes.NewBuffer(nil)
	ctx.printers = make(map[ast.Node]func(*Context, ast.Node))
	ctx.printers[valueSpec] = visitValueSpec
	ctx.printers[funcDecl] = visitFunc
	ctx.printers[assignStmt] = visitAssignStmt
	ctx.printers[ifStmt] = printIfStmt
	ctx.printers[ident] = printIdent
	ctx.printers[includeSpec] = printInclude
	ctx.printers[declStmt] = printDecl
	ctx.printers[ruleSpec] = printRuleSpec
	ctx.printers[selStmt] = printSelStmt
	ctx.printers[propSpec] = printPropValueSpec
	ctx.printers[expr] = printExpr
	ctx.printers[comment] = printComment
	ctx.printers[mediaStmt] = printMedia
	ctx.printers[eachStmt] = printEach
	ctx.scope = NewScope(empty)
	// ctx.printers[typeSpec] = visitTypeSpec
	// assign printers
}

func printComment(ctx *Context, n ast.Node) {
	ctx.blockIntro()
	cmt := n.(*ast.Comment)
	// These additional spaces should be handled by out()
	ctx.out("  " + cmt.Text)
}

func printExpr(ctx *Context, n ast.Node) {
	switch v := n.(type) {
	case *ast.File:
	case *ast.BasicLit:
		switch v.Kind {
		case token.STRING:
			fmt.Fprintf(ctx.buf, "%s;", v.Value)
		case token.QSTRING:
			fmt.Fprintf(ctx.buf, `"%s;"`, v.Value)
		default:
			panic("unsupported lit kind")
		}
	case *ast.Value:
	case *ast.GenDecl:
		// Ignoring these for some reason
	default:
		// fmt.Printf("unmatched expr %T: % #v\n", v, v)
	}
}

func printSelStmt(ctx *Context, n ast.Node) {
	stmt := n.(*ast.SelStmt)
	ctx.activeSel = stmt.Resolved
}

func printRuleSpec(ctx *Context, n ast.Node) {
	// Inspect the sel buffer and dump it
	// Also need to track what level was last dumped
	// so selectors don't get printed twice
	ctx.blockIntro()

	spec := n.(*ast.RuleSpec)
	ctx.scope.RuleAdd(spec)
	ctx.out(fmt.Sprintf("  %s: ", spec.Name))
	var s string
	s, ctx.err = simplifyExprs(ctx, spec.Values)
	fmt.Fprintf(ctx.buf, "%s;", s)
}

func printEach(ctx *Context, n ast.Node) {
	// surprise, not media but behavior is same!
	ctx.hiddenBlock = true
	fmt.Println("each...")
	ast.Print(token.NewFileSet(), n)
}

func printMedia(ctx *Context, n ast.Node) {
	stmt := n.(*ast.MediaStmt)
	ctx.activeMedia = stmt.Query
	ctx.inMedia = true
}

func printPropValueSpec(ctx *Context, n ast.Node) {
	spec := n.(*ast.PropValueSpec)
	fmt.Fprintf(ctx.buf, spec.Name.String()+";")
}

func printIfStmt(ctx *Context, n ast.Node) {
	ifStmt := n.(*ast.IfStmt)
	s, err := resolveExpr(ctx, ifStmt.Cond, true)
	if err != nil {
		log.Fatal("failed to resolve @if", err)
	}
	if s == "true" {
		ctx.Visit(ifStmt.Body)
	} else {
		ctx.Visit(ifStmt.Else)
	}
}

// Variable assignments inside blocks ie. mixins
func visitAssignStmt(ctx *Context, n ast.Node) {
	fmt.Println("visit Assign")
	return
	stmt := n.(*ast.AssignStmt)
	var key, val *ast.Ident
	_, _ = key, val
	switch v := stmt.Lhs[0].(type) {
	case *ast.Ident:
		key = v
	default:
		log.Fatalf("unsupported key: % #v", v)
	}

	switch v := stmt.Rhs[0].(type) {
	case *ast.Ident:
		val = v
	default:
		log.Fatalf("unsupported key: % #v", v)
	}

}

// Variable declarations
func visitValueSpec(ctx *Context, n ast.Node) {
	return
}

func calculateExprs(ctx *Context, bin *ast.BinaryExpr, doOp bool) (string, error) {

	lit, err := calc.Resolve(bin, doOp)
	if err != nil {
		return "", err
	}
	return lit.Value, nil
}

func resolveIdent(ctx *Context, ident *ast.Ident) (out string) {
	v := ident
	if ident.Obj == nil {
		out = ident.Name
		return
	}
	switch vv := v.Obj.Decl.(type) {
	case *ast.Ident:
		out = resolveIdent(ctx, vv)
	case *ast.ValueSpec:
		var s []string
		for i := range vv.Values {
			if ident, ok := vv.Values[i].(*ast.Ident); ok {
				// If obj is set, resolve Obj and report
				if ident.Obj != nil {
					spec := ident.Obj.Decl.(*ast.ValueSpec)
					for _, val := range spec.Values {
						s = append(s, fmt.Sprintf("%s", val))
					}
				} else {
					// fmt.Printf("basic ident: % #v\n", ident)
					s = append(s, fmt.Sprintf("%s", ident))
				}
				continue
			}
			lit := vv.Values[i].(*ast.BasicLit)
			if len(lit.Value) > 0 {
				s = append(s, lit.Value)
			}
		}
		out = strings.Join(s, " ")
	case *ast.AssignStmt:
		lits := resolveAssign(ctx, vv)
		out = joinLits(lits, " ")
	case *ast.BasicLit:
		fmt.Printf("assigning %s: % #v\n", ident, vv)
		ident.Obj.Decl = vv
	default:
		fmt.Printf("unsupported VarDecl: % #v\n", vv)
		// Weird stuff here, let's just push the Ident in
		out = v.Name
	}
	return
}

// joinLits acts like strings.Join
func joinLits(a []*ast.BasicLit, sep string) string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = a[i].Value
	}
	return strings.Join(s, sep)
}

func resolveAssign(ctx *Context, astmt *ast.AssignStmt) (lits []*ast.BasicLit) {

	for _, rhs := range astmt.Rhs {
		switch v := rhs.(type) {
		case *ast.Ident:
			assign := v.Obj.Decl.(*ast.AssignStmt)
			// Replace Ident with underlying BasicLit
			lits = append(lits, resolveAssign(ctx, assign)...)
		case *ast.CallExpr:
			lits = append(lits, v.Fun.(*ast.Ident).Obj.Decl.(*ast.BasicLit))
		case *ast.BasicLit:
			lits = append(lits, v)
		case *ast.StringExpr:
			list := make([]*ast.BasicLit, len(v.List))
			for i := range v.List {
				list[i] = v.List[i].(*ast.BasicLit)
			}
			list[0].Value = `"` + list[0].Value
			last := len(list) - 1
			list[last].Value = list[last].Value + `"`
			lits = append(lits, list...)
		case *ast.ListLit:
			out, err := simplifyExprs(ctx, v.Value)
			if err != nil {
				log.Fatal(err)
			}
			lits = append(lits, &ast.BasicLit{
				Value: out,
			})
		default:
			log.Fatalf("default rhs %s % #v\n", rhs, rhs)
		}
	}
	return
}

func resolveExpr(ctx *Context, expr ast.Expr, doOp bool) (out string, err error) {
	switch v := expr.(type) {
	case *ast.Interp:
		return resolveExpr(ctx, v.Obj.Decl.(ast.Expr), doOp)
	case *ast.Value:
		panic("ast.Value")
	case *ast.BinaryExpr:
		out, err = calculateExprs(ctx, v, doOp)
	case *ast.CallExpr:
		fn, ok := v.Fun.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("unable to read func: % #v", v.Fun)
		}
		return resolveExpr(ctx, fn.Obj.Decl.(ast.Expr), doOp)
	case *ast.StringExpr:
		out, err = simplifyExprs(ctx, v.List)
		return `"` + out + `"`, nil
	case *ast.ParenExpr:
		out, ctx.err = simplifyExprs(ctx, []ast.Expr{v.X})
	case *ast.Ident:
		out = resolveIdent(ctx, v)
	case *ast.BasicLit:
		switch v.Kind {
		case token.VAR:
			// s, ok := ctx.scope.Lookup(v.Value).(string)
			// if ok {
			// 	sums = append(sums, s)
			// }
		case token.QSTRING:
			out = `"` + v.Value + `"`
		default:
			out = v.Value
		}
	case *ast.ListLit:
		vals := make([]string, len(v.Value))
		delim := " "
		if v.Comma {
			delim = ", "
		}
		for i, x := range v.Value {
			o, err := resolveExpr(ctx, x, v.Paren)
			_ = err // fuq this error
			vals[i] = o
		}
		return strings.Join(vals, delim), nil
	default:
		panic(fmt.Sprintf("unhandled expr: % #v\n", v))
	}
	return
}

func simplifyExprs(ctx *Context, exprs []ast.Expr) (string, error) {
	sums := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		s, err := resolveExpr(ctx, expr, false)
		if err != nil {
			return "", err
		}
		sums = append(sums, s)
	}

	return strings.Join(sums, " "), nil
}

func printDecl(ctx *Context, node ast.Node) {
	// I think... nothing to print we'll see
}

func printIdent(ctx *Context, node ast.Node) {
	// ident := node.(*ast.Ident)
	// don't print these
	// fmt.Printf("ignoring % #v\n", ident)
}

func (c *Context) makeStrings(exprs []ast.Expr) (list []string) {
	list = make([]string, 0, len(exprs))
	for _, expr := range exprs {
		switch e := expr.(type) {
		case *ast.Ident:
			list = append(list, e.Name)
		}
	}
	return
}
