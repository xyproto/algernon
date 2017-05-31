package parser

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/calc"
	"github.com/wellington/sass/scanner"
	"github.com/wellington/sass/strops"
	"github.com/wellington/sass/token"
)

func init() {
	log.SetFlags(log.Llongfile)
}

type stack struct {
	file    *token.File
	scanner scanner.Scanner
	pos     token.Pos
	tok     token.Token
	lit     string
	syncPos token.Pos
	syncCnt int
}

type triplet struct {
	pos token.Pos
	tok token.Token
	lit string
}

type queue struct {
	filename string
	src      interface{}
}

// The parser structure holds the parser's internal state.
type parser struct {
	file    *token.File
	errors  scanner.ErrorList
	scanner scanner.Scanner

	// Parser state is pushed onto importStack while imports
	// are being scanned and parsed.
	imps      []stack
	queue     *queue // queued file for import, starts a new scanner
	lookahead triplet
	inSel     bool // controler selector logic
	prescan   bool // control interpolation joining

	// Tracing/debugging
	mode   Mode // parsing mode
	trace  bool // == (mode & Trace != 0)
	indent int  // indentation used for tracing output

	// Comments
	attachComment []*ast.CommentGroup // Comments to attach to decl/spec
	comments      []*ast.CommentGroup
	leadComment   *ast.CommentGroup // last lead comment
	lineComment   *ast.CommentGroup // last line comment

	// Next token
	pos token.Pos   // token position
	tok token.Token // one token look-ahead
	lit string      // token literal

	// Error recovery
	// (used to limit the number of calls to syncXXX functions
	// w/o making scanning progress - avoids potential endless
	// loops across multiple parser functions during error recovery)
	syncPos token.Pos // last synchronization position
	syncCnt int       // number of calls to syncXXX without progress

	// Non-syntactic parser control
	exprLev int            // < 0: in control clause, >= 0: in expression
	inRhs   bool           // if set, the parser is parsing a rhs expression
	inMixin bool           // special rules for mixins
	sels    []*ast.SelStmt // current list of nested selectors

	// Ordinary identifier scopes
	pkgScope   *ast.Scope        // pkgScope.Outer == nil
	topScope   *ast.Scope        // top-most scope; may be pkgScope
	unresolved []*ast.Ident      // unresolved identifiers
	imports    []*ast.ImportSpec // list of imports

	// Label scopes
	// (maintained by open/close LabelScope)
	labelScope  *ast.Scope     // label scope for current function
	targetStack [][]*ast.Ident // stack of unresolved labels
}

var Globalfset *token.FileSet

func (p *parser) init(fset *token.FileSet, filename string, src []byte, mode Mode) {
	Globalfset = fset
	p.file = fset.AddFile(filename, -1, len(src))
	var m scanner.Mode
	m = scanner.ScanComments
	eh := func(pos token.Position, msg string) { p.errors.Add(pos, msg) }
	p.scanner.Init(p.file, src, eh, m)

	p.mode = mode
	p.trace = mode&Trace != 0 // for convenience (p.trace is used frequently)

	// p.next()
}

// add opens a new file and starts scanning it. It preserves the previous
// scanner and position in the importStack stack
func (p *parser) add(filename string, src interface{}) error {
	// imports are relative to the parent
	path := filepath.Join(filepath.Dir(p.file.Name()), filename)
	exts := []string{".scss", ".sass"}
	for i := range exts {
		if !strings.HasSuffix(path, exts[i]) {
			// attempt adding extension
			_, err := os.Stat(path + exts[i])
			if err == nil {
				path += exts[i]
				break
			}

			// Attempt partial
			base := filepath.Base(path)
			p := strings.Replace(path, base, "_"+base+exts[i], 1)
			_, err = os.Stat(p)
			if err == nil {
				path = p
				break
			}
		} else {
			base := filepath.Base(path)
			p := strings.Replace(path, base, "_"+base, 1)
			_, err := os.Stat("_" + p)
			if err == nil {
				path = p
				break
			}
		}

	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	p.queue = &queue{filename: abs, src: src}
	return nil
}

func (p *parser) pop() error {
	if p.queue == nil {
		return fmt.Errorf("pop() called with nil queue")
	}
	stk := stack{
		file:    p.file,
		scanner: p.scanner,
		pos:     p.pos,
		tok:     p.tok,
		lit:     p.lit,
		syncPos: p.syncPos,
		syncCnt: p.syncCnt,
	}
	p.imps = append(p.imps, stk)

	filename, src := p.queue.filename, p.queue.src
	p.queue = nil
	text, err := readSource(filename, src)
	if err != nil {
		abs, ferr := filepath.Abs(filename)
		if ferr != nil {
			log.Println("abs fail", err)
		}
		err = fmt.Errorf("failed to read: %s", err, abs)
		return err
	}
	if p.queue != nil {
		panic("queue hasn't been flushed")
	}
	p.init(Globalfset, filename, text, p.mode)

	return nil
}

// ----------------------------------------------------------------------------
// Scoping support

func (p *parser) openScope() {
	p.topScope = ast.NewScope(p.topScope)
}

func (p *parser) closeScope() {
	if p.topScope == nil {
		panic("scope imbalance")
	}
	p.topScope = p.topScope.Outer
}

func (p *parser) openLabelScope() {
	p.labelScope = ast.NewScope(p.labelScope)
	p.targetStack = append(p.targetStack, nil)
}

func (p *parser) closeLabelScope() {
	// resolve labels
	n := len(p.targetStack) - 1
	scope := p.labelScope
	for _, ident := range p.targetStack[n] {
		ident.Obj = scope.Lookup(ident.Name)
		if ident.Obj == nil && p.mode&DeclarationErrors != 0 {
			p.error(ident.Pos(), fmt.Sprintf("label %s undefined", ident.Name))
		}
	}
	// pop label scope
	p.targetStack = p.targetStack[0:n]
	p.labelScope = p.labelScope.Outer
}

func (p *parser) declare(decl, data interface{}, scope *ast.Scope, kind ast.ObjKind, idents ...*ast.Ident) {
	if p.inMixin {
		log.Fatal("mixin!", idents[0])
		return
	}

	for _, ident := range idents {
		assert(ident != nil, "invalid ident found")
		if ident.Obj != nil {
			fmt.Printf("====== OVERRIDE ==== %s\nold: % #v\nnew: % #v\n",
				ident, ident.Obj.Decl, decl)
		}
		if ident.Obj != nil {
			fmt.Println("not nil!")
			log.Fatal("fail")
		}

		switch d := decl.(type) {
		case *ast.AssignStmt:
		case *ast.FuncDecl:
		default:
			fmt.Printf("Invalid decl % #v\n", decl)
			astPrint(d)
			// FIXME: Why is this being enforced?
			// panic("invalid decl")
		}
		assert(ident.Obj == nil, "identifier already declared or resolved")
		obj := ast.NewObj(kind, ident.Name)
		// remember the corresponding declaration for redeclaration
		// errors and global variable resolution/typechecking phase

		// Catch rules, these should not be declared
		switch decl.(type) {
		case *ast.RuleSpec:
			return
		}
		obj.Decl = decl
		obj.Data = data
		ident.Obj = obj
		if ident.Name == "_" {
			return
		}

		if alt := scope.Insert(obj, ident.Global); alt != nil && p.mode&DeclarationErrors != 0 {

			prevDecl := ""
			if pos := alt.Pos(); pos.IsValid() {
				prevDecl = fmt.Sprintf("\n\tprevious declaration at %s", p.file.Position(pos))
			}
			p.error(ident.Pos(), fmt.Sprintf("%s redeclared in this block%s", ident.Name, prevDecl))
		} else if p.trace {
			fmt.Printf("declared ~> %8s(%p): % #v\n",
				ident, scope, obj.Decl)
		}
	}
}

func (p *parser) shortVarDecl(decl *ast.AssignStmt, list []ast.Expr) {
	// Go spec: A short variable declaration may redeclare variables
	// provided they were originally declared in the same block with
	// the same type, and at least one of the non-blank variables is new.
	n := 0 // number of new variables
	for _, x := range list {
		if ident, isIdent := x.(*ast.Ident); isIdent {
			if ident.Obj != nil {
				fmt.Printf("re-resolved %s\n", ident)
			} else {
				fmt.Printf("new resolve %s\n", ident)
			}
			// assert(ident.Obj == nil, "identifier already declared or resolved")
			obj := ast.NewObj(ast.Var, ident.Name)
			// remember corresponding assignment for other tools
			obj.Decl = decl
			ident.Obj = obj
			if ident.Name != "_" {
				if ident.Global {
					fmt.Println("Storing Global...", obj.Name)
				}
				if alt := p.topScope.Insert(obj, ident.Global); alt != nil {
					if p.trace {
						fmt.Printf("forcefully updated %s (%p): % #v\n", ident,

							ident, decl)
					}
					ident.Obj = alt // redeclaration
				} else {
					n++ // new declaration
				}
			}
		} else {
			p.errorExpected(x.Pos(), "identifier on left side of :=")
		}
	}
	if n == 0 && p.mode&DeclarationErrors != 0 {
		p.error(list[0].Pos(), "no new variables on left side of :=")
	}
}

// The unresolved object is a sentinel to mark identifiers that have been added
// to the list of unresolved identifiers. The sentinel is only used for verifying
// internal consistency.
var unresolved = new(ast.Object)

// If x is an identifier, tryResolve attempts to resolve x by looking up
// the object it denotes. If no object is found and collectUnresolved is
// set, x is marked as unresolved and collected in the list of unresolved
// identifiers.
//
func (p *parser) tryResolve(x ast.Expr, collectUnresolved bool) {
	// nothing to do if x is not an identifier or the blank identifier
	if p.trace {
		fmt.Println("resolve", x)
	}
	ident, _ := x.(*ast.Ident)
	if ident == nil {
		fmt.Printf("cant resolve this: % #v\n", x)
		return
	}

	assert(ident.Obj == nil, "identifier already declared or resolved")
	if ident.Name == "_" {
		return
	}

	// try to resolve the identifier
	for s := p.topScope; s != nil; s = s.Outer {
		if p.trace {
			fmt.Printf("trying %s\n", s)
		}
		if obj := s.Lookup(ident.Name); obj != nil {
			ident.Obj = obj
			return
		}
	}

	// This is a significant failure scenario. However, inside
	// mixins failing to resolve identifiers are perfectly valid.
	// So produce annoying output for somebody to eventually come fix
	// this.

	// all local scopes are known, so any unresolved identifier
	// must be found either in the file scope, package scope
	// (perhaps in another file), or universe scope --- collect
	// them so that they can be resolved later
	if collectUnresolved && !p.inMixin && p.mode&FuncOnly == 0 {
		fmt.Printf("failed to resolve % #v\n", ident)
		// panic("boom")
		ident.Obj = unresolved
		p.unresolved = append(p.unresolved, ident)
	}
}

func (p *parser) resolve(x ast.Expr) {
	p.tryResolve(x, true)
}

// ----------------------------------------------------------------------------
// Parsing support

func (p *parser) printTrace(a ...interface{}) {
	const dots = ". . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . . "
	const n = len(dots)
	pos := p.file.Position(p.pos)
	fmt.Printf("%5d:%3d: ", pos.Line, pos.Column)
	i := 2 * p.indent
	for i > n {
		fmt.Print(dots)
		i -= n
	}
	// i <= n
	fmt.Print(dots[0:i])
	fmt.Println(a...)
}

func trace(p *parser, msg string) *parser {
	p.printTrace(msg, "(")
	p.indent++
	return p
}

// Usage pattern: defer un(trace(p, "..."))
func un(p *parser) {
	p.indent--
	p.printTrace(")")
}

// Advance to the next token.
func (p *parser) next0() {
	// time.Sleep(100 * time.Millisecond)
	// Because of one-token look-ahead, print the previous token
	// when tracing as it provides a more readable output. The
	// very first token (!p.pos.IsValid()) is not initialized
	// (it is token.ILLEGAL), so don't print it .
	if p.trace && p.pos.IsValid() {
		s := p.tok.String()
		switch {
		case p.tok.IsLiteral():
			p.printTrace(s, p.lit)
		case p.tok.IsOperator(), p.tok.IsKeyword():
			p.printTrace("\"" + s + "\"")
		default:
			p.printTrace(s)
		}
	}

	// Sass imports are inline to the parent file. Importing
	// causes the Parser to be reset and can cause invalid positions
	// to be reported. To alleviate this, imports are handled
	// with queueing logic to prevent any un(trace()) calls from
	// from the parent file being executed after the sub-file parser
	// is running.
	if p.queue != nil {
		err := p.pop()
		if err != nil {
			p.error(p.pos, fmt.Sprintf("error reading queue: %s", err))
		}
	}
	p.pos, p.tok, p.lit = p.scanner.Scan()
	// end of declaration, check queue and swap scanner

	// If we have encountered EOF, check the importStack before returning
	// EOF
	if p.tok == token.EOF {
		if len(p.imps) > 0 {
			last := len(p.imps) - 1
			var pop stack
			pop, p.imps, p.imps[last] = p.imps[last], p.imps[:last], stack{}
			p.file = pop.file
			p.scanner = pop.scanner
			p.pos = pop.pos
			p.tok = pop.tok
			p.lit = pop.lit
			p.syncPos = pop.syncPos
			p.syncCnt = pop.syncCnt
			p.next()
		}
	}
}

// Consume a comment and return it and the line on which it ends.
func (p *parser) consumeComment() (comment *ast.Comment, endline int) {
	// /*-style comments may end on a different line than where they start.
	// Scan the comment for '\n' chars and adjust endline accordingly.
	endline = p.file.Line(p.pos)
	tok := token.LINECOMMENT
	if p.lit[1] == '*' {
		// don't use range here - no need to decode Unicode code points
		for i := 0; i < len(p.lit); i++ {
			if p.lit[i] == '\n' {
				endline++
			}
		}
		tok = token.COMMENT
	}

	comment = &ast.Comment{Tok: tok, Slash: p.pos, Text: p.lit}
	p.next0()

	return
}

// Consume a group of adjacent comments, add it to the parser's
// comments list, and return it together with the line at which
// the last comment in the group ends. A non-comment token or n
// empty lines terminate a comment group.
//
func (p *parser) consumeCommentGroup(n int) (comments *ast.CommentGroup, endline int) {
	var list []*ast.Comment
	endline = p.file.Line(p.pos)
	if p.tok == token.COMMENT { //&& p.file.Line(p.pos) <= endline+n {
		var comment *ast.Comment
		comment, endline = p.consumeComment()
		list = append(list, comment)
	}

	// add comment group to the comments list
	comments = &ast.CommentGroup{List: list}
	p.comments = append(p.comments, comments)

	return
}

// Advance to the next non-comment token. In the process, collect
// any comment groups encountered, and remember the last lead and
// and line comments.
//
// A lead comment is a comment group that starts and ends in a
// line without any other tokens and that is followed by a non-comment
// token on the line immediately after the comment group.
//
// A line comment is a comment group that follows a non-comment
// token on the same line, and that has no tokens after it on the line
// where it ends.
//
// Lead and line comments may be considered documentation that is
// stored in the AST.
//
func (p *parser) next() {
	p.leadComment = nil
	p.lineComment = nil
	prev := p.pos
	_ = prev
	p.next0()

	if p.tok == token.COMMENT {
		var comment *ast.CommentGroup
		var endline int

		if p.file.Line(p.pos) == p.file.Line(prev) {
			// The comment is on same line as the previous token; it
			// cannot be a lead comment but may be a line comment.
			comment, endline = p.consumeCommentGroup(0)
			if p.file.Line(p.pos) != endline {
				// The next token is on a different line, thus
				// the last comment group is a line comment.
				p.lineComment = comment
			}
		}

		// consume successor comments, if any
		endline = -1
		if p.tok == token.COMMENT && comment == nil {
			if comment == nil {
				comment = &ast.CommentGroup{}
			}
			var inner *ast.CommentGroup
			inner, endline = p.consumeCommentGroup(1)
			comment.List = append(comment.List, inner.List...)
			// comment, endline = p.consumeCommentGroup(1)
		}

		if endline+1 == p.file.Line(p.pos) {
			// The next token is following on the line immediately after the
			// comment group, thus the last comment group is a lead comment.
			p.leadComment = comment
		}

		// For now, don't report line comments
		for i, cmt := range comment.List {
			if cmt.Tok == token.LINECOMMENT {
				comment.List = append(comment.List[:i], comment.List[i+1:]...)
			}
		}
		p.leadComment = comment
	}
}

// A bailout panic is raised to indicate early termination.
type bailout struct{}

func (p *parser) error(pos token.Pos, msg string) {
	epos := p.file.Position(pos)

	// If AllErrors is not set, discard errors reported on the same line
	// as the last recorded error and stop parsing if there are more than
	// 10 errors.
	if p.mode&AllErrors == 0 {
		n := len(p.errors)
		if n > 0 && p.errors[n-1].Pos.Line == epos.Line {
			return // discard - likely a spurious error
		}
		if n > 10 {
			panic(bailout{})
		}
	}

	p.errors.Add(epos, msg)
}

func (p *parser) errorExpected(pos token.Pos, msg string) {
	msg = "expected " + msg
	if pos == p.pos {
		// the error happened at the current position;
		// make the error message more specific
		if p.tok == token.SEMICOLON && p.lit == "\n" {
			msg += ", found newline"
		} else {
			msg += ", found '" + p.tok.String() + "'"
			if p.tok.IsLiteral() {
				msg += " " + p.lit
			}
		}
	}

	p.error(pos, msg)
}

func (p *parser) expect(tok token.Token) token.Pos {
	pos := p.pos
	if p.tok != tok {
		p.errorExpected(pos, "'"+tok.String()+"'")
	}
	p.next() // make progress
	return pos
}

// expectClosing is like expect but provides a better error message
// for the common case of a missing comma before a newline.
//
func (p *parser) expectClosing(tok token.Token, context string) token.Pos {
	if p.tok != tok && p.tok == token.SEMICOLON && p.lit == "\n" {
		p.error(p.pos, "missing ',' before newline in "+context)
		p.next()
	}
	return p.expect(tok)
}

func (p *parser) expectSemi() {
	// semicolon is optional before a closing ')' or '}'
	if p.tok == token.RPAREN || p.tok == token.RBRACE {
		return
	}
	switch p.tok {
	case token.COMMA:
		// permit a ',' instead of a ';' but complain
		p.errorExpected(p.pos, "';'")
		fallthrough
	case token.SEMICOLON:
		p.next()
	default:
		p.errorExpected(p.pos, "';'")
		syncStmt(p)
	}
}

func (p *parser) atComma(context string, follow token.Token) bool {
	if p.tok == token.COMMA {
		return true
	}
	if p.tok != follow {
		msg := "missing ','"
		if p.tok == token.SEMICOLON && p.lit == "\n" {
			msg += " before newline"
		}
		p.error(p.pos, msg+" in "+context+" expected: "+follow.String()+" got: "+p.tok.String())
		return true // "insert" comma and continue
	}
	return false
}

func assert(cond bool, msg string) {
	if !cond {
		panic("go/parser internal error: " + msg)
	}
}

// syncStmt advances to the next statement.
// Used for synchronization after an error.
//
func syncStmt(p *parser) {
	for {
		switch p.tok {
		case token.FOR, token.IF, token.RETURN:
			// Return only if parser made some progress since last
			// sync or if it has not reached 10 sync calls without
			// progress. Otherwise consume at least one token to
			// avoid an endless parser loop (it is possible that
			// both parseOperand and parseStmt call syncStmt and
			// correctly do not advance, thus the need for the
			// invocation limit p.syncCnt).
			if p.pos == p.syncPos && p.syncCnt < 10 {
				p.syncCnt++
				return
			}
			if p.pos > p.syncPos {
				p.syncPos = p.pos
				p.syncCnt = 0
				return
			}
			// Reaching here indicates a parser bug, likely an
			// incorrect token list in this function, but it only
			// leads to skipping of possibly correct code if a
			// previous error is present, and thus is preferred
			// over a non-terminating parse.
		case token.EOF:
			return
		}
		p.next()
	}
}

// syncDecl advances to the next declaration.
// Used for synchronization after an error.
//
func syncDecl(p *parser) {
	for {
		switch p.tok {
		case token.EOF:
			return
		}
		p.next()
	}
}

// safePos returns a valid file position for a given position: If pos
// is valid to begin with, safePos returns pos. If pos is out-of-range,
// safePos returns the EOF position.
//
// This is hack to work around "artificial" end positions in the AST which
// are computed by adding 1 to (presumably valid) token positions. If the
// token positions are invalid due to parse errors, the resulting end position
// may be past the file's EOF position, which would lead to panics if used
// later on.
//
func (p *parser) safePos(pos token.Pos) (res token.Pos) {
	defer func() {
		if recover() != nil {
			res = token.Pos(p.file.Base() + p.file.Size()) // EOF position
		}
	}()
	_ = p.file.Offset(pos) // trigger a panic if position is out-of-range
	return pos
}

// ----------------------------------------------------------------------------
// Identifiers

func (p *parser) parseInterp() *ast.Interp {
	if p.trace {
		defer un(trace(p, "ParseInterp"))
	}
	pos := p.expect(token.INTERP)
	itp := &ast.Interp{
		Lbrace: pos,
		X:      []ast.Expr{p.inferExprList(false)},
		Rbrace: p.expect(token.RBRACE),
	}
	return itp
}

// Walks through Expressions resolving any CallExpr found
func (p *parser) resolveCall(x ast.Expr) (ast.Expr, error) {
	switch v := x.(type) {
	case *ast.BasicLit:
	case *ast.CallExpr:
		// hold on soldier, first lets resolve all arguments
		for i := range v.Args {
			p.resolveExpr(p.topScope, v.Args[i])
		}
		return evaluateCall(p, p.topScope, v)
	case *ast.BinaryExpr:
		l, err := p.resolveCall(v.X)
		if err != nil {
			return nil, err
		}
		r, err := p.resolveCall(v.Y)
		if err != nil {
			return nil, err
		}
		v.X, v.Y = l, r
		x = v
		// Never called
	// case *ast.UnaryExpr:
	// 	l, _ := p.resolveCall(v.X)
	// 	v.X = l
	// 	x = v
	case *ast.Ident:
		if v.Obj == nil {
			p.resolve(x)
		}
	}
	return x, nil
}

// simplify and resolve expressions inside interpolation.
// strings are always unquoted
func (p *parser) resolveInterp(scope *ast.Scope, itp *ast.Interp) {
	assert(scope == p.topScope, "resolveInterp mismatch scope")
	if len(itp.X) == 0 {
		return
	}
	itp.Obj = ast.NewObj(ast.Var, "")
	ss := make([]string, 0, len(itp.X))
	var merge bool
	for _, x := range itp.X {
		// copied ident won't be resolved, do so
		var err error
		// performing calc
		x, err = p.resolveCall(x)
		if err != nil {
			p.error(x.Pos(), "failed to resolve call: "+err.Error())
		}
		res, err := calc.Resolve(x, true)
		if err != nil {
			p.error(x.Pos(), err.Error())
			continue
		}
		if res.Kind != token.STRING {
			res.Value = strops.Unquote(res.Value)
		}
		// Append value to interp
		if res.Pos() >= itp.End() {
			merge = true
		}
		if !merge {
			ss = append(ss, res.Value)
		} else {
			merge = false
			ss[len(ss)-1] += res.Value
		}
		// Preprend value to interp
		if res.End() <= itp.Pos() {
			merge = true
		}
	}
	// interpolation always outputs a string
	itp.Obj.Decl = &ast.BasicLit{
		Kind:     token.STRING,
		Value:    strings.Join(ss, " "),
		ValuePos: itp.Pos(),
	}
	return
}

func (p *parser) parseDirective() *ast.Ident {
	pos := p.pos
	name := "_"
	switch p.tok {
	case token.MEDIA:
		name = "MEDIA"
	default:
		log.Fatalf("failed to parse directive: %s", p.lit)
	}

	return &ast.Ident{NamePos: pos, Name: name}
}

func (p *parser) parseIdent() *ast.Ident {
	if p.trace {
		defer un(trace(p, "ParseIdent"))
	}
	pos := p.pos
	name := "_"
	// FIXME: parseIdent should not be responding with non-IDENT
	if p.tok == token.IDENT {
		name = p.lit
		p.next()
	} else {
		p.expect(token.IDENT) // use expect() error handling
	}

	return &ast.Ident{
		NamePos: pos,
		Name:    name,
	}
}

func (p *parser) parseIdentList() (list []*ast.Ident) {
	if p.trace {
		defer un(trace(p, "IdentList"))
	}

	list = append(list, p.parseIdent())
	for p.tok == token.COMMA {
		p.next()
		list = append(list, p.parseIdent())
	}

	return
}

// ----------------------------------------------------------------------------
// Common productions

// If lhs is set, result list elements which are identifiers are not resolved.
func (p *parser) parseExprList(lhs bool) (list []ast.Expr) {
	if p.trace {
		defer un(trace(p, "ExpressionList"))
	}

	list = append(list, p.checkExpr(p.parseExpr(lhs)))
	for p.tok == token.COMMA {
		p.next()
		list = append(list, p.checkExpr(p.parseExpr(lhs)))
	}
	return
}

// sass list uses no delimiters, and may optionally be surrounded
// by parens
func (p *parser) parseSassList(lhs, canComma bool) (list []ast.Expr, hasComma, checkParen bool) {
	if p.trace {
		defer un(trace(p, "SassList"))
	}
	if p.tok == token.LPAREN {
		checkParen = true
		p.next()
	}
	if p.tok == token.RULE {
		p.error(p.pos, "sass can not contain a list")
		p.next()
	}
	for p.tok != token.SEMICOLON &&
		// possible closers
		p.tok != token.LBRACE && p.tok != token.RPAREN &&
		p.tok != token.RBRACE &&
		// failure scenario
		p.tok != token.EOF {
		if canComma {
			inner := p.listFromExprs(p.parseSassList(lhs, false))
			list = append(list, inner)
			if p.tok == token.COMMA {
				hasComma = true
				p.next()
			}
		} else if p.tok == token.LPAREN {
			// fuck, new list
			list = append(list, p.listFromExprs(p.parseSassList(lhs, true)))
		} else if p.tok == token.COMMA {
			return
		} else {
			x := p.inferExpr(lhs, checkParen)
			if interp, ok := x.(*ast.Interp); ok {
				p.resolveInterp(p.topScope, interp)
			}
			list = append(list, p.checkExpr(x))
		}
	}

	if p.tok == token.EOF {
		p.error(p.pos, "EOF reached before list end")
	}
	if checkParen {
		p.expect(token.RPAREN)
	}
	return

}

func (p *parser) expandList(in []ast.Expr) []ast.Expr {

	if len(in) != 1 {
		return in
	}

	ident, ok := in[0].(*ast.Ident)
	if !ok {
		return in
	}

	p.tryResolve(ident, false)
	if ident.Obj == nil {
		// uninitialized variable
		return in
	}
	decl := ident.Obj.Decl
	ass, ok := decl.(*ast.AssignStmt)
	if !ok {
		return in
	}

	list, ok := ass.Rhs[0].(*ast.ListLit)
	if !ok {
		return in
	}

	return list.Value
}

func (p *parser) inferLhsList() ast.Expr {
	old := p.inRhs
	p.inRhs = false
	list := p.inferExprList(true)
	switch p.tok {
	case token.DEFINE:
		// lhs of a short variable declaration
		// but doesn't enter scope until later:
		// caller must call p.shortVarDecl(p.makeIdentList(list))
		// at appropriate time.
	case token.COLON:
		// lhs of a label declaration or a communication clause of a select
		// statement (parseLhsList is not called when parsing the case clause
		// of a switch statement):
		// - labels are declared by the caller of parseLhsList
		// - for communication clauses, if there is a stand-alone identifier
		//   followed by a colon, we have a syntax error; there is no need
		//   to resolve the identifier in that case
	default:
		// identifiers must be declared elsewhere
		p.resolveExpr(p.topScope, list)
	}
	p.inRhs = old
	return list
}

func (p *parser) parseRhsList() []ast.Expr {
	old := p.inRhs
	p.inRhs = true
	list := p.parseExprList(false)
	p.inRhs = old
	return list
}

// ----------------------------------------------------------------------------
// Types

func (p *parser) parseType() ast.Expr {
	if p.trace {
		defer un(trace(p, "Type"))
	}

	typ := p.tryType()

	if typ == nil {
		pos := p.pos
		p.errorExpected(pos, "type")
		p.next() // make progress
		return &ast.BadExpr{From: pos, To: p.pos}
	}

	return typ
}

func (p *parser) inferRhsList() ast.Expr {
	old := p.inRhs
	p.inRhs = true
	list := p.inferExprList(false)
	p.inRhs = old

	return list
}

func astPrint(v interface{}) {
	ast.Print(token.NewFileSet(), v)
}

// interpolation can happen inline to a string. In these cases,
// the value should be merged with the previous or subsequent
// value.
func (p *parser) mergeInterps(in []ast.Expr) []ast.Expr {
	if p.trace {
		defer un(trace(p, "MergeInterps"))
	}
	if len(in) < 2 {
		return in
	}
	out := make([]ast.Expr, 0, len(in))
	for i := 0; i < len(in); i++ {
		if in[i].Pos() == 0 {
			log.Fatalf("invalid position %d: % #v\n", i, in[i])
		}
		itp, isInterp := in[i].(*ast.Interp)
		if !isInterp {
			lit, ok := in[i].(*ast.BasicLit)
			// lookbehind if this is a candidate for merge
			if ok && len(out) > 0 {
				l := in[i-1]
				if l.End() == lit.Pos() {
					prev, ok := out[len(out)-1].(*ast.Interp)
					if !ok {
						panic(fmt.Errorf("\nl:% #v\nr:% #v\n",
							l, lit))
					}
					prev.X = append(prev.X, lit)
					// changes to interp require resolution
					p.resolveInterp(p.topScope, prev)
					continue
				}
			}
			out = append(out, in[i])
			continue
		}

		if itp.Obj == nil || itp.Obj.Decl == nil {
			p.error(itp.Pos(), "interpolation is unresolved")
			continue
		}

		lit := itp.Obj.Decl.(*ast.BasicLit)
		if i == 0 {
			if itp.Pos() == 0 {
				log.Fatal("invalid position")
			}
			out = append(out, itp)
			continue
		}

		// Look behind and see if the previous token should
		// be appended to
		if in[i-1].End() == itp.Pos() {
			// Oh shit, it does need to be merged
			// Replace existing preceeding Expr with an interp
			// shouldn't do this, it's going to be a problem
			target, ok := out[len(out)-1].(*ast.BasicLit)
			if ok {
				// merge
				// target.Kind = token.STRING
				itp.X = append([]ast.Expr{target}, itp.X...)
				p.resolveInterp(p.topScope, itp)
				out[len(out)-1] = itp
				continue
			}
			// has to be interp, or we're screwed
			pitp := out[len(out)-1].(*ast.Interp)
			pitp.X = append(pitp.X, itp.X...)
			// changes to interp require resolution
			p.resolveInterp(p.topScope, pitp)
			continue
		}
		itp.Obj.Decl = lit
		out = append(out, itp)
	}
	for _, o := range out {
		if o.Pos() == 0 {
			log.Fatalf("invalid position % #v\n", o)
		}
	}
	return out
}

func (p *parser) inferExprList(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "InferExprList"))
	}
	// lists are weird in Sass
	// list: 1 2 3;
	// list of lists: 1 2, 3

	return p.listFromExprs(p.parseSassList(lhs, true))
}

// listFromExprs takes a slice of expr to create a ListLit
func (p *parser) listFromExprs(in []ast.Expr, hasComma, inParen bool) ast.Expr {
	if len(in) == 0 {
		return nil
	}
	if len(in) > 1 {

		return &ast.ListLit{
			Paren:    inParen,
			ValuePos: in[0].Pos(),
			EndPos:   in[len(in)-1].End(),
			Value:    p.mergeInterps(in),
			Comma:    hasComma,
		}
	}
	l, ok := in[0].(*ast.ListLit)
	if ok {
		// non-paren list inside paren list
		l.Paren = true
		return l
	}
	if inParen {
		// Doesn't matter always return a list for proper
		// math resolution
		return &ast.ListLit{
			Paren:    inParen,
			ValuePos: in[0].Pos(),
			EndPos:   in[0].End(),
			Value:    in,
			Comma:    hasComma,
		}
	}
	// Unwrap list of 1
	return in[0]
}

func (p *parser) parseString() *ast.StringExpr {
	if p.trace {
		defer un(trace(p, "String"))
	}
	tok := p.tok
	expr := &ast.StringExpr{
		Lquote: p.pos,
		Kind:   tok,
	}
	p.next()
	var list []ast.Expr
	// Only strings and interpolations allowed here
	for p.tok != token.EOF && p.tok != tok {
		x := p.inferExpr(false, false)
		list = append(list, x)
	}
	rquote := p.expectClosing(tok, "string list")
	expr.List = p.mergeInterps(list)
	expr.Rquote = rquote
	return expr
}

// Derive the type from the nature of the value, there is no hint for what
// a value could be. Complete list of types follows
//
// http://sass-lang.com/documentation/file.SASS_REFERENCE.html#data_types
//numbers (e.g. 1.2, 13, 10px)
// strings of text, with and without quotes (e.g. "foo", 'bar', baz)
// colors (e.g. blue, #04a3f9, rgba(255, 0, 0, 0.5))
// booleans (e.g. true, false)
// nulls (e.g. null)
// lists of values, separated by spaces or commas (e.g. 1.5em 1em 0 2em, Helvetica, Arial, sans-serif)
// maps from one value to another (e.g. (key1: value1, key2: value2))
func (p *parser) inferExpr(lhs bool, inParens bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "inferExpr"))
	}
	var expr ast.Expr
	defer func() {
		//fmt.Printf("expr: %d end: %d val: % #v\n",
		//	expr.Pos(), expr.End(), expr)
	}()
	basic := &ast.BasicLit{ValuePos: p.pos, Value: p.lit}
	expr = basic
	switch p.tok {
	case token.RULE:
		basic.Kind = token.RULE
		p.next()
		return expr
	}
	return p.parseBinaryExpr(lhs, inParens, token.LowestPrec+1)
}

func (p *parser) parseSassType() ast.Expr {
	if p.trace {
		defer un(trace(p, "SassType"))
	}
	var expr ast.Expr
	if p.lit[0] == '$' {
		// This is too open, need more checks
		lit := p.lit
		ident := &ast.Ident{
			Name:    lit,
			NamePos: p.pos,
		}
		// variadics need special changes to do resolution
		// temporarily remove ... suffix during resolution
		if strings.HasSuffix(lit, "...") {
			ident.Name = strings.TrimSuffix(p.lit, "...")
			p.resolve(ident)
			ident.Name = p.lit
		} else {
			p.resolve(ident)
		}
		expr = ident
	} else {
		expr = &ast.BasicLit{
			ValuePos: p.pos,
			Value:    p.lit,
			Kind:     token.VAR,
		}
	}
	p.next()
	return expr
}

// If the result is an identifier, it is not resolved.
func (p *parser) parseTypeName() ast.Expr {
	if p.trace {
		defer un(trace(p, "TypeName"))
	}

	ident := p.parseIdent()
	// don't resolve ident yet - it may be a parameter or field name

	if p.tok == token.PERIOD {
		// ident is a package name
		p.next()
		p.resolve(ident)
		sel := p.parseIdent()
		return &ast.SelectorExpr{X: ident, Sel: sel}
	}

	return ident
}

func (p *parser) makeIdentList(list []ast.Expr) []*ast.Ident {
	idents := make([]*ast.Ident, len(list))
	for i, x := range list {
		ident, isIdent := x.(*ast.Ident)
		if !isIdent {
			if _, isBad := x.(*ast.BadExpr); !isBad {
				// only report error if it's a new one
				p.errorExpected(x.Pos(), "identifier")
			}
			ident = &ast.Ident{NamePos: x.Pos(), Name: "_"}
		}
		idents[i] = ident
	}
	return idents
}

func (p *parser) parsePointerType() *ast.StarExpr {
	if p.trace {
		defer un(trace(p, "PointerType"))
	}

	star := p.expect(token.MUL)
	base := p.parseType()

	return &ast.StarExpr{Star: star, X: base}
}

// If the result is an identifier, it is not resolved.
func (p *parser) tryVarType(isParam bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "tryVarType"))
	}
	if isParam {
		typ := p.tryIdentOrType()
		if p.tok == token.COLON {
			// Default arg found!
			pos := p.expect(token.COLON)
			val := p.tryIdentOrType()
			return &ast.KeyValueExpr{
				Key:   typ,
				Colon: pos,
				Value: val,
			}
		}
		return typ
	} else if p.tok == token.INTERP {
		log.Fatalf("interp!", p.lit)
	}

	return p.tryIdentOrType()
}

// If the result is an identifier, it is not resolved.
func (p *parser) parseVarType(isParam bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "ParseVarType"))
	}
	typ := p.tryVarType(isParam)
	if typ == nil {
		pos := p.pos
		p.errorExpected(pos, "type")
		p.next() // make progress
		typ = &ast.BadExpr{From: pos, To: p.pos}
	}
	return typ
}

func (p *parser) parseParameterList(scope *ast.Scope, ellipsisOk bool) (params []*ast.Field) {
	if p.trace {
		defer un(trace(p, "ParameterList"))
	}

	// 1st ParameterDecl
	// A list of identifiers looks like a list of type names.
	var list []ast.Expr
	for {
		list = append(list, p.parseVarType(ellipsisOk))

		if p.tok != token.COMMA {
			break
		}

		p.next()
		if p.tok == token.RPAREN {
			break
		}
	}

	// Type { "," Type } (anonymous parameters)
	params = make([]*ast.Field, len(list))
	for i, typ := range list {
		// ident := ast.ToIdent(typ)
		// if ident != nil && ident.Obj == nil {
		// 	p.resolve(ident)
		// 	typ = ident
		// 	fmt.Printf("wut? % #v\n", typ)
		// }
		params[i] = &ast.Field{Type: typ}
	}

	return
}

func (p *parser) parseParameters(scope *ast.Scope, ellipsisOk bool) *ast.FieldList {
	if p.trace {
		defer un(trace(p, "Parameters"))
	}

	var params []*ast.Field
	lparen := p.expect(token.LPAREN)
	if p.tok != token.RPAREN {
		params = p.parseParameterList(scope, ellipsisOk)
	}
	rparen := p.expect(token.RPAREN)

	return &ast.FieldList{Opening: lparen, List: params, Closing: rparen}
}

func (p *parser) parseSignature(scope *ast.Scope) (params, results *ast.FieldList) {
	if p.trace {
		defer un(trace(p, "Signature"))
	}
	// Short circuit if params are not available
	if p.tok != token.LPAREN {
		return
	}
	return p.parseParameters(scope, true), nil
}

// If the result is an identifier, it is not resolved.
func (p *parser) tryIdentOrType() ast.Expr {
	if p.trace {
		defer un(trace(p, "tryIdentOrType"))
	}
	switch p.tok {
	case token.IDENT:
		// return p.parseTypeName()
	case token.MUL:
		return p.parsePointerType()
	case token.VAR:
		return p.parseSassType()
	case token.LPAREN:
		lparen := p.pos
		p.next()
		typ := p.parseType()
		rparen := p.expect(token.RPAREN)
		return &ast.ParenExpr{Lparen: lparen, X: typ, Rparen: rparen}
	case token.RPAREN:
		return nil
	default:
		// FIXME: we should verify types here, ie. UPX, INT, IDENT
		expr := &ast.BasicLit{
			ValuePos: p.pos,
			Kind:     p.tok,
			Value:    p.lit,
		}
		p.next()
		return expr
	}
	// Unreachable
	// no type found
	return nil
}

func (p *parser) tryType() ast.Expr {
	typ := p.tryIdentOrType()
	if typ != nil {
		p.resolve(typ)
	}
	return typ
}

func (p *parser) checkComment() *ast.CommStmt {
	if p.leadComment == nil || len(p.leadComment.List) == 0 {
		return nil
	}
	cmt := &ast.CommStmt{
		Group: p.leadComment,
	}
	p.leadComment = nil
	return cmt
}

func (p *parser) unwrapInclude(in ast.Stmt) []ast.Stmt {
	if inc, ok := in.(*ast.IncludeStmt); ok && !p.inMixin {
		out := make([]ast.Stmt, 0, len(inc.Spec.List)+1)
		for i := range inc.Spec.List {
			out = append(out, p.unwrapInclude(inc.Spec.List[i])...)
		}
		return out
	}
	return []ast.Stmt{in}
}

// ----------------------------------------------------------------------------
// Blocks

func (p *parser) parseStmtList() []ast.Stmt {
	if p.trace {
		defer un(trace(p, "StatementList"))
	}
	var sels []ast.Stmt
	var list []ast.Stmt
	for p.tok != token.RBRACE && p.tok != token.EOF {
		stmt, sel := p.parseStmt()
		if sel {
			sels = append(sels, stmt)
			continue
		}
		// TODO: for some damned reason, semicolons appear here
		// just skip
		if _, ok := stmt.(*ast.EmptyStmt); ok {
			continue
		}
		expand := p.unwrapInclude(stmt)
		list = append(list, expand...)
		if len(expand) > 0 {
			ast.SortStatements(list)
		}
	}
	if cmt := p.checkComment(); cmt != nil {
		list = append(list, cmt)
	}
	list = append(list, sels...)
	// ast.SortStatements(list)
	return list
}

func (p *parser) parseBody(scope *ast.Scope) *ast.BlockStmt {
	if p.trace {
		defer un(trace(p, "Body"))
	}
	lbrace := p.expect(token.LBRACE)
	var list []ast.Stmt
	oldScope := p.topScope
	p.topScope = scope // open function scope
	p.openLabelScope()
	list = append(list, p.parseStmtList()...)
	p.closeLabelScope()
	p.topScope = oldScope
	if cmt := p.checkComment(); cmt != nil {
		list = append(list, cmt)
	}
	rbrace := p.expect(token.RBRACE)

	return &ast.BlockStmt{Lbrace: lbrace, List: list, Rbrace: rbrace}
}

func (p *parser) parseBlockStmt() *ast.BlockStmt {
	if p.trace {
		defer un(trace(p, "BlockStmt"))
	}

	lbrace := p.expect(token.LBRACE)
	var list []ast.Stmt
	p.openScope()
	list = append(list, p.parseStmtList()...)
	p.closeScope()
	rbrace := p.expect(token.RBRACE)

	return &ast.BlockStmt{Lbrace: lbrace, List: list, Rbrace: rbrace}
}

func (p *parser) parseMediaStmt() *ast.MediaStmt {
	if p.trace {
		defer un(trace(p, "MediaStmt"))
	}

	pos := p.expect(token.MEDIA)
	med := &ast.Ident{
		NamePos: pos,
	}
	lit := &ast.BasicLit{
		Kind:     token.STRING,
		Value:    "@media " + p.lit,
		ValuePos: p.pos,
	}
	p.expect(token.STRING)

	return &ast.MediaStmt{
		Name:  med,
		Query: lit,
		Body:  p.parseBody(p.topScope),
	}
}

// parseOperand may return an expression or a raw type (incl. array
// types of the form [...]T. Callers must verify the result.
// If lhs is set and the result is an identifier, it is not resolved.
//
func (p *parser) parseOperand(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "Operand"))
	}

	switch p.tok {
	case token.IDENT:
		x := p.parseIdent()
		// token.LPAREN indicates call expr
		if !lhs && p.tok != token.LPAREN {
			p.resolve(x)
		}
		return x
	case token.INTERP:
		x := p.parseInterp()
		p.resolveInterp(p.topScope, x)
		return x
	case token.QSTRING, token.QSSTRING:
		// TODO: most definitely short sighed
		pos, tok := p.pos, p.tok
		p.next()
		x := &ast.BasicLit{
			Kind:     token.QSTRING,
			Value:    p.lit,
			ValuePos: pos,
		}
		p.next()
		p.expect(tok)
		return x
	case
		token.COLOR,
		token.UEM, token.UPCT, token.UPT, token.UPX, token.UREM,
		token.INT, token.FLOAT, token.STRING:
		x := &ast.BasicLit{ValuePos: p.pos, Kind: p.tok, Value: p.lit}
		p.next()
		return x

	case token.LPAREN:
		ls, _, _ := p.parseSassList(lhs, true)
		if len(ls) > 1 {
			panic("multiple lists found")
		}

		return ls[0]
	case token.VAR:
		// VAR is only hit while parsing function params, so
		// this should only be allowed in that case.
		return p.tryVarType(true) //isParam
	}

	if typ := p.tryIdentOrType(); typ != nil {
		// could be type for composite literal or conversion
		_, isIdent := typ.(*ast.Ident)
		assert(!isIdent, "type cannot be identifier")
		return typ
	}

	// we have an error
	pos := p.pos
	p.errorExpected(pos, "operand")
	syncStmt(p)
	return &ast.BadExpr{From: pos, To: p.pos}
}

func (p *parser) parseIndexOrSlice(x ast.Expr) ast.Expr {
	if p.trace {
		defer un(trace(p, "IndexOrSlice"))
	}

	const N = 3 // change the 3 to 2 to disable 3-index slices
	lbrack := p.expect(token.LBRACK)
	p.exprLev++
	var index [N]ast.Expr
	var colons [N - 1]token.Pos
	if p.tok != token.COLON {
		index[0] = p.parseRhs()
	}
	ncolons := 0
	for p.tok == token.COLON && ncolons < len(colons) {
		colons[ncolons] = p.pos
		ncolons++
		p.next()
		if p.tok != token.COLON && p.tok != token.RBRACK && p.tok != token.EOF {
			index[ncolons] = p.parseRhs()
		}
	}
	p.exprLev--
	rbrack := p.expect(token.RBRACK)

	if ncolons > 0 {
		// slice expression
		slice3 := false
		if ncolons == 2 {
			slice3 = true
			// Check presence of 2nd and 3rd index here rather than during type-checking
			// to prevent erroneous programs from passing through gofmt (was issue 7305).
			if index[1] == nil {
				p.error(colons[0], "2nd index required in 3-index slice")
				index[1] = &ast.BadExpr{From: colons[0] + 1, To: colons[1]}
			}
			if index[2] == nil {
				p.error(colons[1], "3rd index required in 3-index slice")
				index[2] = &ast.BadExpr{From: colons[1] + 1, To: rbrack}
			}
		}
		return &ast.SliceExpr{X: x, Lbrack: lbrack, Low: index[0], High: index[1], Max: index[2], Slice3: slice3, Rbrack: rbrack}
	}

	return &ast.IndexExpr{X: x, Lbrack: lbrack, Index: index[0], Rbrack: rbrack}
}

func (p *parser) parseCallOrConversion(fun ast.Expr) *ast.CallExpr {
	if p.trace {
		defer un(trace(p, "CallOrConversion"))
	}
	// See functions LPAREN have to be next to the previous expression
	// There's other rules too, but no need to apply those just yet
	if fun.End() != p.pos {
		p.error(fun.End(), "Functions are followed immediately by (")
	}
	pos := p.pos
	lparen := p.expect(token.LPAREN)
	p.exprLev++
	var list []ast.Expr
	expr := p.inferExprList(false)
	lit, ok := expr.(*ast.ListLit)
	if ok {
		list = lit.Value
	} else if expr != nil {
		list = []ast.Expr{expr}
	}

	p.exprLev--

	rparen := p.expectClosing(token.RPAREN, "argument list")

	call := &ast.CallExpr{
		Fun:    fun,
		Lparen: lparen,
		Args:   list,
		// Ellipsis: ellipsis,
		Rparen: rparen,
	}
	ident, ok := fun.(*ast.Ident)
	if !ok {
		log.Fatalf("% #v\n", fun)
	}
	if p.mode&FuncOnly == 0 {
		lit, err := evaluateCall(p, p.topScope, call)
		call.Resolved = lit
		// Manually set object, because Ident name isn't unique
		obj := ast.NewObj(ast.Var, ident.Name)
		obj.Decl = lit
		ident.Obj = obj
		if err != nil {
			p.error(pos, err.Error())
		}
	}
	return call
}

func (p *parser) parseValue(keyOk bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "Element"))
	}

	if p.tok == token.LBRACE {
		return p.parseLiteralValue(nil)
	}

	// Because the parser doesn't know the composite literal type, it cannot
	// know if a key that's an identifier is a struct field name or a name
	// denoting a value. The former is not resolved by the parser or the
	// resolver.
	//
	// Instead, _try_ to resolve such a key if possible. If it resolves,
	// it a) has correctly resolved, or b) incorrectly resolved because
	// the key is a struct field with a name matching another identifier.
	// In the former case we are done, and in the latter case we don't
	// care because the type checker will do a separate field lookup.
	//
	// If the key does not resolve, it a) must be defined at the top
	// level in another file of the same package, the universe scope, or be
	// undeclared; or b) it is a struct field. In the former case, the type
	// checker can do a top-level lookup, and in the latter case it will do
	// a separate field lookup.
	x := p.checkExpr(p.parseExpr(keyOk))
	if keyOk {
		if p.tok == token.COLON {
			// Try to resolve the key but don't collect it
			// as unresolved identifier if it fails so that
			// we don't get (possibly false) errors about
			// undeclared names.
			p.tryResolve(x, false)
		} else {
			// not a key
			p.resolve(x)
		}
	}

	return x
}

func (p *parser) parseElement() ast.Expr {
	if p.trace {
		defer un(trace(p, "Element"))
	}

	x := p.parseValue(true)
	if p.tok == token.COLON {
		colon := p.pos
		p.next()
		x = &ast.KeyValueExpr{Key: x, Colon: colon, Value: p.parseValue(false)}
	}

	return x
}

func (p *parser) parseElementList() (list []ast.Expr) {
	if p.trace {
		defer un(trace(p, "ElementList"))
	}

	for p.tok != token.RBRACE && p.tok != token.EOF {
		list = append(list, p.parseElement())
		if !p.atComma("composite literal", token.RBRACE) {
			break
		}
		p.next()
	}

	return
}

func (p *parser) parseLiteralValue(typ ast.Expr) ast.Expr {
	if p.trace {
		defer un(trace(p, "LiteralValue"))
	}

	lbrace := p.expect(token.LBRACE)
	var elts []ast.Expr
	p.exprLev++
	if p.tok != token.RBRACE {
		elts = p.parseElementList()
	}
	p.exprLev--
	rbrace := p.expectClosing(token.RBRACE, "composite literal")
	return &ast.CompositeLit{Type: typ, Lbrace: lbrace, Elts: elts, Rbrace: rbrace}
}

// checkExpr checks that x is an expression (and not a type).
func (p *parser) checkExpr(x ast.Expr) ast.Expr {
	switch unparen(x).(type) {
	case *ast.BadExpr:
	case *ast.Ident:
	case *ast.BasicLit:
	case *ast.ListLit:
	case *ast.FuncLit:
	case *ast.Interp:
	case *ast.StringExpr:
	case *ast.CompositeLit:
	case *ast.ParenExpr:
		panic("unreachable")
	case *ast.SelectorExpr:
	case *ast.IndexExpr:
	case *ast.SliceExpr:
	case *ast.TypeAssertExpr:
		// If t.Type == nil we have a type assertion of the form
		// y.(type), which is only allowed in type switch expressions.
		// It's hard to exclude those but for the case where we are in
		// a type switch. Instead be lenient and test this in the type
		// checker.
	case *ast.CallExpr:
	case *ast.StarExpr:
	case *ast.UnaryExpr:
	case *ast.BinaryExpr:
	case *ast.KeyValueExpr:
	default:
		panic(fmt.Errorf("fuq % #v\n", x))
		// all other nodes are not proper expressions
		p.errorExpected(x.Pos(), "expression")
		x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
	}
	return x
}

// isTypeName reports whether x is a (qualified) TypeName.
func isTypeName(x ast.Expr) bool {
	switch t := x.(type) {
	case *ast.BadExpr:
	case *ast.Ident:
	case *ast.SelectorExpr:
		_, isIdent := t.X.(*ast.Ident)
		return isIdent
	default:
		return false // all other nodes are not type names
	}
	return true
}

// isLiteralType reports whether x is a legal composite literal type.
func isLiteralType(x ast.Expr) bool {
	switch t := x.(type) {
	case *ast.BadExpr:
	case *ast.Ident:
	case *ast.SelectorExpr:
		_, isIdent := t.X.(*ast.Ident)
		return isIdent
	case *ast.ArrayType:
	case *ast.StructType:
	case *ast.MapType:
	default:
		return false // all other nodes are not legal composite literal types
	}
	return true
}

// If x is of the form (T), unparen returns unparen(T), otherwise it returns x.
func unparen(x ast.Expr) ast.Expr {
	if p, isParen := x.(*ast.ParenExpr); isParen {
		x = unparen(p.X)
	}
	return x
}

// checkExprOrType checks that x is an expression or a type
// (and not a raw type such as [...]T).
//
func (p *parser) checkExprOrType(x ast.Expr) ast.Expr {
	switch t := unparen(x).(type) {
	case *ast.ParenExpr:
		panic("unreachable")
	case *ast.UnaryExpr:
	case *ast.ArrayType:
		if len, isEllipsis := t.Len.(*ast.Ellipsis); isEllipsis {
			p.error(len.Pos(), "expected array length, found '...'")
			x = &ast.BadExpr{From: x.Pos(), To: p.safePos(x.End())}
		}
	}

	// all other nodes are expressions or types
	return x
}

func (p *parser) printf(format string, v ...interface{}) {
	if p.mode&FuncOnly != 0 {
		return
	}
	fmt.Printf(format, v...)
}

// If lhs is set and the result is an identifier, it is not resolved.
func (p *parser) parsePrimaryExpr(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "PrimaryExpr"))
	}

	x := p.parseOperand(lhs)

L:
	for {
		switch p.tok {
		case token.LPAREN:
			if lhs {
				p.resolve(x)
			}
			if x.End() != p.pos {
				return x
			}
			x = p.parseCallOrConversion(p.checkExprOrType(x))
		default:
			break L
		}
		lhs = false // no need to try to resolve again
	}

	return x
}

// If lhs is set and the result is an identifier, it is not resolved.
func (p *parser) parseUnaryExpr(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "UnaryExpr"))
	}

	switch p.tok {
	case token.ADD, token.SUB, token.NOT, token.XOR, token.AND,
		token.MUL, token.QUO:
		pos, op := p.pos, p.tok
		p.next()
		x := p.parseUnaryExpr(false)
		un := &ast.UnaryExpr{OpPos: pos, Op: op, X: p.checkExpr(x)}
		return un
	case token.QSTRING, token.QSSTRING:
		return p.parseString()
	}

	return p.parsePrimaryExpr(lhs)
}

func (p *parser) tokPrec() (token.Token, int) {
	tok := p.tok
	if p.inRhs && tok == token.ASSIGN {
		tok = token.EQL
	}
	return tok, tok.Precedence()
}

// If lhs is set and the result is an identifier, it is not resolved.
func (p *parser) parseBinaryExpr(lhs bool, inParens bool, prec1 int) ast.Expr {
	if p.trace {
		defer un(trace(p, "BinaryExpr"))
	}

	x := p.parseUnaryExpr(lhs)
	for _, prec := p.tokPrec(); prec >= prec1; prec-- {
		for {
			op, oprec := p.tokPrec()
			if oprec != prec {
				break
			}
			pos := p.expect(op)
			if lhs {
				p.resolve(x)
				lhs = false
			}
			y := p.parseBinaryExpr(false, inParens, prec+1)
			x = &ast.BinaryExpr{
				X:     p.checkExpr(x),
				OpPos: pos,
				Op:    op,
				Y:     p.checkExpr(y),
			}
		}
	}
	return x
}

// If lhs is set and the result is an identifier, it is not resolved.
// The result may be a type or even a raw type ([...]int). Callers must
// check the result (using checkExpr or checkExprOrType), depending on
// context.
func (p *parser) parseExpr(lhs bool) ast.Expr {
	if p.trace {
		defer un(trace(p, "Expression"))
	}

	return p.parseBinaryExpr(lhs, false, token.LowestPrec+1)
}

func (p *parser) parseRhs() ast.Expr {
	old := p.inRhs
	p.inRhs = true
	x := p.checkExpr(p.parseExpr(false))
	p.inRhs = old
	return x
}

func (p *parser) parseRhsOrKV() ast.Expr {
	old := p.inRhs
	p.inRhs = true
	x := p.parseExpr(false)
	p.inRhs = old
	return x
}

func (p *parser) parseRhsOrType() ast.Expr {
	old := p.inRhs
	p.inRhs = true
	x := p.checkExprOrType(p.parseExpr(false))
	p.inRhs = old
	return x
}

// ----------------------------------------------------------------------------
// Statements

// Parsing modes for parseSimpleStmt.
const (
	basic = iota
	labelOk
	rangeOk
)

// parseSimpleStmt returns true as 2nd result if it parsed the assignment
// of a range clause (with mode == rangeOk). The returned statement is an
// assignment with a right-hand side that is a single unary expression of
// the form "range x". No guarantees are given for the left-hand side.
func (p *parser) parseSimpleStmt(mode int) (ast.Stmt, bool) {
	if p.trace {
		defer un(trace(p, "SimpleStmt"))
	}
	tok, pos := p.tok, p.pos
	x := []ast.Expr{p.inferLhsList()}
	var stmt ast.Stmt
	isRange := false
	switch tok {
	case token.VAR:
		p.expect(token.COLON)
		y := []ast.Expr{p.inferRhsList()}
		stmt = &ast.AssignStmt{Lhs: x, TokPos: pos, Rhs: y}
	default:
		fmt.Println("wut", p.tok, p.lit)
		panic("?")
		p.error(p.pos, "SimpleStmt failed")
		stmt = &ast.BadStmt{From: x[0].Pos(), To: p.pos + 1}
	}

	if len(x) > 1 {
		for _, expr := range x {
			fmt.Printf("% #v\n", expr)
		}
		panic("boom town")
		p.errorExpected(x[0].Pos(), "1 expression")
		// continue with first expression
	}

	return stmt, isRange
}

func (p *parser) parseCallExpr(callType string) *ast.CallExpr {
	x := p.parseRhsOrType() // could be a conversion: (some type)(x)
	if call, isCall := x.(*ast.CallExpr); isCall {
		return call
	}
	if _, isBad := x.(*ast.BadExpr); !isBad {
		// only report error if it's a new one
		p.error(p.safePos(x.End()), fmt.Sprintf("function must be invoked in %s statement", callType))
	}
	return nil
}

func (p *parser) parseReturnStmt() *ast.ReturnStmt {
	if p.trace {
		defer un(trace(p, "ReturnStmt"))
	}

	pos := p.pos
	p.expect(token.RETURN)
	var x []ast.Expr
	if p.tok != token.SEMICOLON && p.tok != token.RBRACE {
		x = p.parseRhsList()
	}
	p.expectSemi()

	return &ast.ReturnStmt{Return: pos, Results: x}
}

func (p *parser) makeExpr(s ast.Stmt, kind string) ast.Expr {
	if s == nil {
		return nil
	}
	if es, isExpr := s.(*ast.ExprStmt); isExpr {
		return p.checkExpr(es.X)
	}
	p.error(s.Pos(), fmt.Sprintf("expected %s, found simple statement (missing parentheses around composite literal?)", kind))
	return &ast.BadExpr{From: s.Pos(), To: p.safePos(s.End())}
}

func (p *parser) parseIfStmt() *ast.IfStmt {
	if p.trace {
		defer un(trace(p, "IfStmt"))
	}

	// Look for @if and @else if
	var pos token.Pos
	switch p.tok {
	case token.IF:
		fallthrough
	case token.ELSEIF:
		pos = p.pos
	default:
		p.errorExpected(p.pos, "expected @if or @else if")
	}

	p.openScope()
	defer p.closeScope()

	var s ast.Stmt
	var x ast.Expr
	{
		prevLev := p.exprLev
		p.exprLev = -1
		p.next()
		x = p.parseRhs()
		p.exprLev = prevLev
	}
	body := p.parseBlockStmt()
	var else_ ast.Stmt
	if p.tok == token.ELSE {
		p.next()
		else_ = p.parseBlockStmt()
	} else if p.tok == token.ELSEIF {
		else_ = p.parseIfStmt()
	}
	return &ast.IfStmt{If: pos, Init: s, Cond: x, Body: body, Else: else_}
}

func (p *parser) resolveIfStmt(scope *ast.Scope, in *ast.IfStmt) []ast.Stmt {
	fmt.Println("resolve ifstmt!")
	var ret []ast.Stmt
	decl := in
	cond := in.Cond
	// TODO: This is weird, but it resolves idents
	p.resolveExpr(scope, cond)

	// Check if this points to a binary expr, otherwise
	// compare with false (and maybe nil?)
	var compFalse bool
	if ident, ok := cond.(*ast.Ident); ok {
		if _, ok := ident.Obj.Decl.(*ast.BinaryExpr); !ok {
			compFalse = true
		}
	}

	fmt.Printf("cond % #v\n", decl.Cond)
	lit, err := calc.Resolve(decl.Cond, true)
	if err != nil {
		panic(fmt.Sprint("failed to understand condition: ", err))
	}
	switch {
	// This checks just for presence of a variable
	case compFalse && lit.Value != "false":
		fmt.Println("compfalse", lit.Value)
		fallthrough
	case lit.Value == "true":
		fmt.Println("true...")
		resList := p.resolveStmts(scope, decl.Body.List)
		ret = append(ret, resList...)
	default:
		el := decl.Else
		if blk, ok := el.(*ast.BlockStmt); ok {
			res := p.resolveStmts(scope, blk.List)
			ret = append(ret, res...)
		} else {
			ret = append(ret, p.resolveStmts(scope,
				[]ast.Stmt{el})...)
		}
	}

	return ret
}

func (p *parser) parseTypeList() (list []ast.Expr) {
	if p.trace {
		defer un(trace(p, "TypeList"))
	}

	list = append(list, p.parseType())
	for p.tok == token.COMMA {
		p.next()
		list = append(list, p.parseType())
	}

	return
}

// @each $i in (1 2 3)
// @each $i in a b c
func (p *parser) parseEachStmt() *ast.EachStmt {
	if p.trace {
		defer un(trace(p, "EachStmt"))
	}

	pos := p.expect(token.EACH)
	// each variable iterator
	itr := p.parseVarType(true).(*ast.Ident)

	// in
	if p.lit != "in" {
		p.errorExpected(p.pos, "in after iterator ie @each $n in")
	} else {
		p.next()
	}

	list, _, _ := p.parseSassList(true, false)

	body := p.parseBody(p.topScope)
	each := &ast.EachStmt{
		Each: pos,
		X:    itr,
		List: list,
		Body: body,
	}

	// FIXME: decide when to resolve the each stmt
	if !p.inMixin {
		p.resolveEachStmt(p.topScope, each)
	}
	return each
}

func (p *parser) resolveEachStmt(outscope *ast.Scope, each *ast.EachStmt) {
	litsToExprs := func(in []*ast.BasicLit) []ast.Expr {
		out := make([]ast.Expr, len(in))
		for i := range in {
			out[i] = in[i]
		}
		return out
	}

	// attempt expansion of $var in $vars
	list := p.expandList(each.List)
	itrName := each.X.Name
	r := ast.NewIdent(itrName)

	// at some point, all decl are enforced as AssignStmt
	ass := &ast.AssignStmt{
		Lhs:    []ast.Expr{r},
		TokPos: list[0].Pos(),
		Rhs:    litsToExprs(p.resolveExpr(outscope, list[0])),
	}

	var stmts []ast.Stmt
	// walk through first iterator
	scope := ast.NewScope(outscope)
	p.declare(ass, nil, scope, ast.Var, r)
	copy := make([]ast.Stmt, len(each.Body.List))
	for i := range each.Body.List {
		copy[i] = ast.StmtCopy(each.Body.List[i])
	}
	stmts = append(stmts, p.resolveStmts(scope, copy)...)

	for _, l := range list[1:] {
		// Copy the body from list 1
		copy := make([]ast.Stmt, len(each.Body.List))
		for i := range each.Body.List {
			copy[i] = ast.StmtCopy(each.Body.List[i])
		}

		r := ast.NewIdent(itrName)
		// at some point, all decl are enforced as AssignStmt
		ass := &ast.AssignStmt{
			Lhs:    []ast.Expr{r},
			TokPos: list[0].Pos(),
			Rhs:    litsToExprs(p.resolveExpr(scope, l)),
		}
		scope := ast.NewScope(outscope)
		p.declare(ass, nil, scope, ast.Var, r)
		stmts = append(stmts, p.resolveStmts(scope, copy)...)
	}
	// Modify body with new stmts
	each.Body.List = stmts
}

func (p *parser) parseForStmt() ast.Stmt {
	if p.trace {
		defer un(trace(p, "ForStmt"))
	}

	pos := p.expect(token.FOR)
	p.openScope()
	defer p.closeScope()

	var s1, s2, s3 ast.Stmt
	var isRange bool
	if p.tok != token.LBRACE {
		prevLev := p.exprLev
		p.exprLev = -1
		if p.tok != token.SEMICOLON {
			s2, isRange = p.parseSimpleStmt(rangeOk)

		}
		if !isRange && p.tok == token.SEMICOLON {
			p.next()
			s1 = s2
			s2 = nil
			if p.tok != token.SEMICOLON {
				s2, _ = p.parseSimpleStmt(basic)
			}
			p.expectSemi()
			if p.tok != token.LBRACE {
				s3, _ = p.parseSimpleStmt(basic)
			}
		}
		p.exprLev = prevLev
	}

	body := p.parseBlockStmt()
	p.expectSemi()

	if isRange {
		as := s2.(*ast.AssignStmt)
		// check lhs
		var key, value ast.Expr
		switch len(as.Lhs) {
		case 0:
			// nothing to do
		case 1:
			key = as.Lhs[0]
		case 2:
			key, value = as.Lhs[0], as.Lhs[1]
		default:
			p.errorExpected(as.Lhs[len(as.Lhs)-1].Pos(), "at most 2 expressions")
			return &ast.BadStmt{From: pos, To: p.safePos(body.End())}
		}
		// parseSimpleStmt returned a right-hand side that
		// is a single unary expression of the form "range x"
		x := as.Rhs[0].(*ast.UnaryExpr).X
		return &ast.RangeStmt{
			For:    pos,
			Key:    key,
			Value:  value,
			TokPos: as.TokPos,
			Tok:    as.Tok,
			X:      x,
			Body:   body,
		}
	}

	// regular for statement
	return &ast.ForStmt{
		For:  pos,
		Init: s1,
		Cond: p.makeExpr(s2, "boolean or range expression"),
		Post: s3,
		Body: body,
	}
}

func (p *parser) parseStmt() (s ast.Stmt, isSelector bool) {
	if p.trace {
		defer un(trace(p, "Statement"))
	}
	if cmt := p.checkComment(); cmt != nil {
		fmt.Printf("checkComment % #v\n", cmt.Group.List[0])
		s = cmt
		return
	}

	switch p.tok {
	case token.IDENT, token.RULE:
		s = &ast.DeclStmt{Decl: p.parseDecl(syncStmt)}
		// p.expectSemi()
	case token.COMMENT:
		group, _ := p.consumeCommentGroup(0)
		s = &ast.CommStmt{Group: group}
	case
		token.VAR:
		spec := p.inferValueSpec(p.leadComment, p.tok, 0).(*ast.ValueSpec)
		stmt := &ast.AssignStmt{}
		for _, name := range spec.Names {
			stmt.Lhs = append(stmt.Lhs, name)
		}
		for _, val := range spec.Values {
			stmt.Rhs = append(stmt.Rhs, val)
		}
		// FIXME: lost the Token and Position in this transformation
		s = stmt
	case
		// TODO: Not sure any of these cases ever exist in Sass
		// tokens that may start an expression
		token.INT, token.FLOAT, token.STRING, token.FUNC, token.LPAREN, // operands
		token.LBRACK,
		// composite types
		token.GTR, token.TIL,
		token.ADD, token.SUB, token.MUL, token.AND, token.XOR, token.NOT: // unary operators
		s, _ = p.parseSimpleStmt(labelOk)
		// because of the required look-ahead, labeled statements are
		// parsed by parseSimpleStmt - don't expect a semicolon after
		// them
		if _, isLabeledStmt := s.(*ast.LabeledStmt); !isLabeledStmt {
			p.expectSemi()
		}
	case token.EACH:
		s = p.parseEachStmt()
	case token.RETURN:
		s = p.parseReturnStmt()
	case token.MEDIA:
		s = p.parseMediaStmt()
	case token.LBRACE:
		s = p.parseBlockStmt()
		p.expectSemi()
	case token.IF:
		s = p.parseIfStmt()
	case token.FOR:
		s = p.parseForStmt()
	case token.IMPORT:
		s = &ast.DeclStmt{Decl: p.parseGenDecl("", token.IMPORT, p.parseImportSpec)}
	case token.INCLUDE:
		s = &ast.IncludeStmt{Spec: p.parseIncludeSpec(!p.inMixin)}
	case token.SELECTOR:
		s = p.parseRuleSelStmt()
		isSelector = true
	case token.SEMICOLON:
		// Is it ever possible to have an implicit semicolon
		// producing an empty statement in a valid program?
		// (handle correctly anyway)
		s = &ast.EmptyStmt{Semicolon: p.pos, Implicit: p.lit == "\n"}
		p.next()
	case token.RBRACE:
		// a semicolon may be omitted before a closing "}"
		s = &ast.EmptyStmt{Semicolon: p.pos, Implicit: true}
	default:
		// no statement found
		pos := p.pos
		p.errorExpected(pos, "statement")
		syncStmt(p)
		s = &ast.BadStmt{From: pos, To: p.pos}
	}

	return
}

// ----------------------------------------------------------------------------
// Declarations

type parseSpecFunction func(doc *ast.CommentGroup, keyword token.Token, iota int) ast.Spec

func isValidImport(lit string) bool {
	const illegalChars = `!"#$%&'()*,:;<=>?[\]^{|}` + "`\uFFFD"
	s, _ := strconv.Unquote(lit) // go/scanner returns a legal string literal
	for _, r := range s {
		if !unicode.IsGraphic(r) || unicode.IsSpace(r) || strings.ContainsRune(illegalChars, r) {
			return false
		}
	}
	return s != ""
}

func (p *parser) parseImportSpec(doc *ast.CommentGroup, _ token.Token, _ int) ast.Spec {
	if p.trace {
		// defer un(trace(p, "ImportSpec"))
	}

	var ident *ast.Ident
	switch p.tok {
	case token.PERIOD:
		ident = &ast.Ident{NamePos: p.pos, Name: "."}
		p.next()
	case token.IDENT:
		ident = p.parseIdent()
	}

	p.expect(token.IMPORT)
	x := p.parseOperand(false)
	pathlit, ok := x.(*ast.BasicLit)
	if !ok {
		p.errorExpected(x.Pos(), "expected import to be string or quoted string")
	}

	// collect imports
	spec := &ast.ImportSpec{
		// Doc:     doc,
		Name:    ident,
		Path:    pathlit,
		Comment: p.lineComment,
	}
	// Parse and insert the results into the current parser
	p.imports = append(p.imports, spec)
	err := p.processImport(spec.Path.Value)
	if err != nil {
		log.Fatalf("failed to import: %s", spec.Name)
	}
	return spec
}

func (p *parser) processImport(path string) error {
	return p.add(path, nil)
}

func (p *parser) inferSelSpec(doc *ast.CommentGroup, keyword token.Token, iota int) ast.Spec {
	if p.trace {
		defer un(trace(p, keyword.String()+"InferSelSpec"))
	}
	decl := p.parseRuleSelDecl()

	return &ast.SelSpec{
		Decl: decl,
	}
}

// checkForGlobal inspects variable declarations looking for the
// !global keyword
func checkForGlobal(vals []ast.Expr) bool {
	if len(vals) == 0 {
		return false
	}

	if len(vals) < 2 {
		if list, ok := vals[0].(*ast.ListLit); ok {
			vals = list.Value
		} else {
			return false
		}
	}
	lit, ok := vals[len(vals)-1].(*ast.BasicLit)
	if !ok {
		return false
	}
	if lit.Kind == token.STRING && lit.Value == "!global" {
		return true
	}

	return false
}

func (p *parser) inferValueSpec(doc *ast.CommentGroup, keyword token.Token, iota int) ast.Spec {
	if p.trace {
		defer un(trace(p, "inferValue"+keyword.String()+"Spec"))
	}

	lit := p.lit

	// Move this out of inferValueSpec
	switch p.tok {
	case token.INCLUDE:
		return p.parseIncludeSpec(!p.inMixin)
	}

	name := &ast.Ident{
		Name:    lit,
		NamePos: p.pos,
	}

	// Type has to be derived from the values being set
	// typ := p.tryType()
	// var typ ast.Expr
	var values []ast.Expr
	lhs := true
	p.next()
	pos, tok := p.pos, p.tok
	switch p.tok {
	case token.LPAREN:
		ret := p.parseCallOrConversion(p.checkExprOrType(name))
		values = append(values, ret)
	case token.COLON:
		lhs = false
		p.next()
		fallthrough
	default:
		x := p.inferExprList(lhs)
		if p.tok == token.SEMICOLON {
			values = append(values, x)
			break
		}
		// check for string math against a list...
		y := p.parseUnaryExpr(false)
		if un, ok := y.(*ast.UnaryExpr); ok {
			bin := &ast.BinaryExpr{
				X:     x,
				Y:     un.X,
				Op:    un.Op,
				OpPos: un.OpPos,
			}
			values = append(values, bin)
		} else {
			l, _, _ := p.parseSassList(lhs, true)
			if l == nil {
				panic("dont know now")
			}
			panic(fmt.Errorf("non-unary discovered: % #v", y))
		}
	}

	// p.expectSemi()
	// Go spec: The scope of a constant or variable identifier declared inside
	// a function begins at the end of the ConstSpec or VarSpec and ends at
	// the end of the innermost containing block.
	// (Global identifiers are resolved in a separate phase after parsing.)
	var spec ast.Spec

	switch keyword {
	case token.VAR:
		name.Global = checkForGlobal(values)
		// Assignment happening
		spec = &ast.ValueSpec{
			// Doc:   doc,
			Names:   []*ast.Ident{name},
			Comment: p.lineComment,
			Values:  values,
		}
		decl := &ast.AssignStmt{}
		decl.Lhs = []ast.Expr{name}
		decl.Rhs = values
		decl.Tok = tok
		decl.TokPos = pos

		p.shortVarDecl(decl, decl.Lhs)
	default:
		spec = &ast.RuleSpec{
			Name:    name,
			Comment: p.lineComment,
			Values:  values,
		}
	}
	return spec

}

func (p *parser) parseGenDecl(lit string, keyword token.Token, f parseSpecFunction) *ast.GenDecl {
	if p.trace {
		defer un(trace(p, "GenDecl("+keyword.String()+")"))
	}
	pos := p.pos
	assert(pos != 0, "0 position found")
	// doc := p.leadComment

	var lparen, rparen token.Pos
	var list []ast.Spec
	// TODO: can probably remove this
	if p.tok == token.LPAREN {
		lparen = p.pos
		p.next()
		for iota := 0; p.tok != token.RPAREN && p.tok != token.EOF; iota++ {
			list = append(list, f(p.leadComment, keyword, iota))
		}
		rparen = p.expect(token.RPAREN)
	} else {
		list = append(list, f(nil, keyword, 0))
	}
	p.expectSemi()

	return &ast.GenDecl{
		// Doc:    doc,
		TokPos: pos,
		Tok:    keyword,
		Lparen: lparen,
		Specs:  list,
		Rparen: rparen,
	}
}

// resolve the selector against its parent and add to the selector stack
func (p *parser) openSelector(sel *ast.SelStmt) {
	// sel.Collapse(p.sels, len(p.sels) != 0, p.error)
	p.sels = append(p.sels, sel)
}

// remove the selector from the nested stack
func (p *parser) closeSelector() {
	p.sels = p.sels[:len(p.sels)-1]
}

func (p *parser) parseSelStmt(backrefOk bool) *ast.SelStmt {
	if p.trace {
		defer un(trace(p, "SelStmt"))
	}
	lit := p.lit
	pos := p.expect(token.SELECTOR)
	assert(pos != 0, "invalid selector position")
	scope := ast.NewScope(p.topScope)
	// idents := p.processSelectors(scope, lit, pos, backrefOk)
	sel := &ast.SelStmt{
		Name: &ast.Ident{
			NamePos: pos,
			Name:    lit,
		},
	}

	if len(p.sels) > 0 {
		sel.Parent = p.sels[len(p.sels)-1]
	}

	var xs []ast.Expr
	for p.tok != token.LBRACE {
		x := p.parseCombSel(token.LowestPrec + 1)
		// if xx, ok := x.(*ast.Interp); ok {
		// lit := xx.Obj.Decl.(*ast.BasicLit)
		// fmt.Printf("%s:%s\n", lit.Kind, lit.Value)
		// }
		xs = append(xs, x)
	}

	if len(xs) == 0 {
		p.error(p.pos, "no selector found...")
		return sel
	}
	sel.Sel = xs[0]
	s, ok := itpMerge(xs)
	if ok {
		fmt.Println("itpMerge", s)
		stmt, err := reparseSelector(s)
		if err != nil {
			p.error(pos, err.Error())
		}
		sel.Sel = stmt.Sel
		sel.Resolved = stmt.Resolved
	}
	sel.Resolve(Globalfset)
	p.openSelector(sel)
	sel.Body = p.parseBody(scope)
	p.closeSelector()

	return sel
}

// reparseSelector starts an entirely new scanner/parser to generate an ast for
// This is entirely overkill and stupid, but interpolation support
// is not at a place where selectors can support them without a
// reparse after interpolation merging.
func reparseSelector(orig string) (*ast.SelStmt, error) {
	orig += "{}" // ensures scanner processes this as a selector
	pf, err := ParseFile(token.NewFileSet(), "nope", orig, 0)
	if err != nil {
		log.Fatal("reparse fail", err)
	}

	if len(pf.Decls) == 0 {
		return nil, errors.New("No declarations found")
	}
	decl := pf.Decls[0].(*ast.SelDecl)

	return decl.SelStmt, nil
}

// similar to inferExpr, but for selectors
func (p *parser) parseSel() ast.Expr {
	if p.trace {
		defer un(trace(p, "Sel"))
	}

	switch p.tok {
	case token.AND:
		// Backreference create a nested Op
		// with the parent as X and child as Y
		pos, lit := p.pos, p.lit
		p.next()
		return &ast.UnaryExpr{
			OpPos: pos,
			Op:    token.NEST,
			X: &ast.BasicLit{
				Kind:     token.STRING,
				ValuePos: pos,
				Value:    lit,
			},
		}
	case token.ADD, token.GTR, token.TIL:
		pos, op := p.pos, p.tok
		p.next()
		x := p.parseSel()
		return &ast.UnaryExpr{OpPos: pos, Op: op, X: p.checkExpr(x)}
	case token.STRING, token.ATTRIBUTE:
		pos := p.pos
		var lits []string
		// eat all the strings
		for p.tok == token.STRING || p.tok == token.ATTRIBUTE {
			lits = append(lits, p.lit)
			p.next()
		}
		s := strings.Join(lits, " ")

		// TODO: inferExpr should be creating this or the scanner
		// should combine adjacent strings
		return &ast.BasicLit{
			Kind:     token.STRING,
			Value:    s,
			ValuePos: pos,
		}
	case token.INTERP:
		x := p.parseInterp()
		p.resolveInterp(p.topScope, x)
		return x
	default:
		log.Fatalf("unsupported sel type %s:%q\n", p.tok, p.lit)
	}
	return &ast.BasicLit{}
}

// Selectors fall in two buckets Combinators and Groups
// Combinators: + > ~
// Groups: ,
// https://www.w3.org/TR/selectors/#selectors
func (p *parser) parseCombSel(prec1 int) ast.Expr {
	if p.trace {
		defer un(trace(p, "CombSel"))
	}

	x := p.parseSel()
	for prec := p.tok.SelPrecedence(); prec >= prec1; prec-- {
		for {
			tok := p.tok
			oprec := tok.SelPrecedence()
			if oprec != prec {
				break
			}
			pos := p.expect(tok)
			y := p.parseCombSel(prec + 1)
			x = &ast.BinaryExpr{
				X:     x,
				OpPos: pos,
				Op:    tok,
				Y:     y,
			}
		}
	}
	return x
}

func (p *parser) parseRuleSelStmt() ast.Stmt {
	if p.trace {
		defer un(trace(p, "RuleSelStmt"))
	}
	return p.parseSelStmt(true)
}

func (p *parser) parseRuleSelDecl() *ast.SelDecl {
	if p.trace {
		defer un(trace(p, "RuleSelDecl"))
	}

	stmt := p.parseSelStmt(false)

	return &ast.SelDecl{
		SelStmt: stmt,
	}

}

func (p *parser) parseRuleDecl() *ast.GenDecl {
	if p.trace {
		defer un(trace(p, "RuleDecl"))
	}
	var list []ast.Spec
	// list = append(list, &ast.RuleSpec{
	// 	Comment: p.lineComment,
	// 	Name: &ast.Ident{
	// 		NamePos: p.pos,
	// 		Name:    p.lit,
	// 		// Comment: p.comments,
	// 	},
	// })
	// pos := p.expect(token.RULE)
	// p.expect(token.COLON)
	pos := p.pos

	f := p.inferValueSpec
	for iota := 0; p.tok != token.SEMICOLON &&
		p.tok != token.RBRACE &&
		// p.tok != token.RULE &&
		p.tok != token.EOF; iota++ {

		var spec ast.Spec

		switch p.tok {
		case token.SELECTOR:
			f = p.inferSelSpec
			spec = f(p.leadComment, p.tok, iota)
		case token.LPAREN:
			spec = f(p.leadComment, p.tok, iota)
			p.expect(token.RPAREN)
		default:
			spec = f(p.leadComment, p.tok, iota)
		}
		if spec != nil {
			list = append(list, spec)
		}
	}
	p.expectSemi()

	return &ast.GenDecl{
		TokPos: pos,
		Tok:    p.tok,
		Specs:  list,
	}

}

func (p *parser) parseIncludeSpecFn(doc *ast.CommentGroup, keyword token.Token, iota int) ast.Spec {
	// The top level must be rules, or this is a failure
	return p.parseIncludeSpec(!p.inMixin)
}

func sigPosition(pos int, list []*ast.Field, isVdc bool) (*ast.Field, error) {
	l := len(list)
	switch {
	case pos < l-1:
		fallthrough
	case pos == l-1 && !isVdc:
		return list[pos], nil
	}

	if !isVdc {
		return nil, errors.New("variable outside bounds of field signature")
	}

	// nil indicates multiple Fields apply
	return nil, nil
}

// processFuncArgs walks through the arguments declaring each signature
// in the provided scope
func (p *parser) processFuncArgs(scope *ast.Scope, signature *ast.FieldList, arguments *ast.FieldList) {
	// assert(p.topScope == scope, "Invalid function argument scope")
	var sigs []*ast.Ident

	toDeclare := make(map[*ast.Ident]interface{})

	var isVariadic bool
	// Process the signature and defaults, toDeclaring the defaults
	for i, sig := range signature.List {
		var key *ast.Ident
		fmt.Printf("typ... %T % #v %t\n", sig.Type, sig.Type, isVariadic)

		// Convert ident or basiclit to ident
		switch v := sig.Type.(type) {
		default:
			log.Fatalf("unsupported sig type % #v\n", v)
		case *ast.Ident:
			if isVariadic {
				log.Fatal("only last argument can be variadic")
			}
			if strings.HasSuffix(v.Name, "...") {
				v.Name = strings.TrimSuffix(v.Name, "...")
				isVariadic = true
			}
			field, err := sigPosition(i, signature.List, isVariadic)
			if err != nil {
				log.Fatalf("failed to process arguments: %s", err)
			}

			if field == nil {
				continue
			}
			key = v
		case *ast.KeyValueExpr:
			var val interface{}
			// Default arg!
			key = v.Key.(*ast.Ident)
			// Set default value, if found
			switch vv := v.Value.(type) {
			case nil:
			case *ast.BasicLit:
				val = vv
				// p.declare(val, nil, scope, ast.Var, ident)
			case *ast.Ident:
				p.resolve(vv)
				// TODO: this may need to recursively search for BasicLit
				val = vv.Obj.Decl
			default:
				log.Fatalf("unsupported default value % #v\n", vv)
			}
			toDeclare[key] = val
		}
		// Preserve sig
		sigs = append(sigs, key)
	}

	// Hold variadic arguments, saving to a list
	var lastArg []ast.Expr

	// Now walk through passed arguments and toDeclare finding the
	// appropriate matching arg
	if arguments != nil {
		for i, arg := range arguments.List {
			var ident *ast.Ident
			if i < len(sigs) {
				ident = sigs[i]
			}

			var val interface{}
			switch v := arg.Type.(type) {
			case *ast.BasicLit:
				val = &ast.AssignStmt{
					Lhs:    []ast.Expr{ident},
					TokPos: arg.Pos(),
					Rhs:    []ast.Expr{v},
				}
			case *ast.Ident:
				p.resolve(v)
				val = v.Obj.Decl
			case *ast.KeyValueExpr:
				ident = v.Key.(*ast.Ident)
				val = v.Value
				if valdent, ok := val.(*ast.Ident); ok {
					p.resolve(valdent)
					val = valdent.Obj.Decl
				}
			}
			if val == nil {
				fmt.Printf("skipped argument %s\n", sigs[i])
				continue
			}

			if isVariadic && i >= len(sigs)-1 {
				// these fucking assignstmt need to go away
				v := val
				if ass, ok := v.(*ast.AssignStmt); ok {
					v = ass.Rhs[0]
				}
				lastArg = append(lastArg, v.(ast.Expr))
				continue
			}
			toDeclare[ident] = val
		}
	}

	if len(lastArg) > 0 {
		ident := signature.List[len(signature.List)-1].Type.(*ast.Ident)
		ident.Name = strings.TrimSuffix(ident.Name, "...")

		list := p.listFromExprs(lastArg, true, true)
		ass := &ast.AssignStmt{
			Lhs:    []ast.Expr{ident},
			TokPos: ident.Pos(),
			Rhs:    []ast.Expr{list},
		}
		fmt.Println("declaring...", ident, ass)
		p.declare(ass, nil, scope, ast.Var, ident)
		fmt.Println("done decl")
	}

	for k, v := range toDeclare {
		if k == nil {
			fmt.Printf("FIXME: nil declaration...", k)
			astPrint(v)
			continue
		}
		p.declare(v, nil, scope, ast.Var, k)
	}
}

// walks through statements resolving them with the provided
// scope
func (p *parser) resolveStmts(scope *ast.Scope, stmts []ast.Stmt) []ast.Stmt {

	oldScope := p.topScope
	p.topScope = scope
	defer func() { p.topScope = oldScope }()

	ret := make([]ast.Stmt, 0, len(stmts))

	for i := range stmts {
		switch decl := stmts[i].(type) {
		case *ast.DeclStmt:
			p.resolveDecl(scope, decl)
			stmts[i] = decl
		case *ast.AssignStmt:
			p.shortVarDecl(decl, decl.Lhs)
		case *ast.CommStmt:
		case *ast.EachStmt:
			p.resolveEachStmt(scope, decl)
		case *ast.IncludeStmt:
			p.resolveIncludeSpec(decl.Spec)
		case *ast.SelStmt:
			if len(p.sels) > 0 {
				decl.Parent = p.sels[len(p.sels)-1]
			}
			decl.Resolve(Globalfset)
			p.openSelector(decl)
			decl.Body.List = p.resolveStmts(scope, decl.Body.List)
			p.closeSelector()
		case *ast.EmptyStmt, nil:
			fmt.Println("empty shit!", stmts[i])
			// Trim from result
			continue
		case *ast.IfStmt:
			ret = append(ret, p.resolveIfStmt(scope, decl)...)
			continue
		case *ast.ReturnStmt:
			// TODO: something to do here?
		case *ast.BlockStmt:
			list := p.resolveStmts(scope, decl.List)
			ret = append(ret, list...)
		default:
			log.Fatalf("unsupported stmt: % #v\n", stmts[i])
		}
		ret = append(ret, stmts[i])
	}

	return ret
}

func (p *parser) resolveExpr(scope *ast.Scope, expr ast.Expr) (out []*ast.BasicLit) {
	oldScope := p.topScope
	p.topScope = scope
	defer func() { p.topScope = oldScope }()

	assert(p.topScope == scope, "resolveExpr scope mismatch")
	switch v := expr.(type) {
	case *ast.BasicLit:
		out = append(out, v)
	case *ast.CallExpr:
		x, _ := p.resolveCall(v)
		out = append(out, x.(*ast.BasicLit))
	case *ast.Interp:
		p.resolveInterp(scope, v)
		fmt.Println("resolved...", v.Obj.Decl.(*ast.BasicLit))
		out = append(out, v.Obj.Decl.(*ast.BasicLit))
	case *ast.Ident:
		if v.Obj != nil {
			log.Println("ident had previous value, this is an error")
			return basicLitFromIdent(v)
		}
		assert(v.Obj == nil, "statement had previous value, was it copied correctly?")
		p.resolve(v)
		out = basicLitFromIdent(v)
	case *ast.ListLit:
		for _, x := range v.Value {
			out = append(out, p.resolveExpr(scope, x)...)
		}
	default:
		panic(fmt.Errorf("unsupported expr % #v", v))
	}
	return
}

// resolveDecl reevalutes all found IDENTs with new scope provided by
// arg list.
func (p *parser) resolveDecl(scope *ast.Scope, decl *ast.DeclStmt) {
	if p.trace {
		defer un(trace(p, "ResolveDecl"))
	}

	assert(p.topScope == scope, "resolveDecl scope mismatch")
	switch v := decl.Decl.(type) {
	case *ast.GenDecl:
		for _, spec := range v.Specs {
			switch sv := spec.(type) {
			case *ast.RuleSpec:
				var lits []*ast.BasicLit
				for i := range sv.Values {
					val := sv.Values[i]
					fmt.Printf("% #v\n", val)
					lits = append(lits, p.resolveExpr(scope, val)...)
				}
				sv.Values = make([]ast.Expr, len(lits))
				for i := range lits {
					sv.Values[i] = lits[i]
				}
			default:
				log.Fatalf("spec not supported % #v\n", v)
			}
		}
	default:
		log.Fatalf("decl not supported %T: % #v\n", v, v)
	}
}

// TODO: delete this, calc.Resolve can do it
// basicLitFromIdent recursively resolves an Ident until a
// basic lit is uncovered.
func basicLitFromIdent(ident *ast.Ident) (lit []*ast.BasicLit) {
	assert(ident.Obj != nil, "ident has not been resolved")
	decl := ident.Obj.Decl
	switch typ := decl.(type) {
	case *ast.Ident:
		return basicLitFromIdent(typ)
	case *ast.AssignStmt:
		var lits []*ast.BasicLit
		//lits := make([]*ast.BasicLit, 0, len(typ.Rhs))
		for i := range typ.Rhs {
			rhs := typ.Rhs[i]
			var lit *ast.BasicLit
			switch rtyp := rhs.(type) {
			case *ast.Ident:
				lits = append(lits, basicLitFromIdent(rtyp)...)
				continue
			case *ast.BasicLit:
				lit = rtyp
			case *ast.ListLit:
				var err error
				lit, err = calc.Resolve(rtyp, rtyp.Paren)
				assert(err == nil, "calc resolve failed")
			default:
				log.Fatalf("illegal Rhs expr % #v\n", rtyp)
			}
			lits = append(lits, lit)
		}
		if len(lits) > 1 {
			log.Println("good thing this looked at multiple rhs values",
				lits)
		}

		return lits
	case *ast.BasicLit:
		return []*ast.BasicLit{typ}
	case nil:
		fmt.Printf("% #v\n", ident)
		panic("ident is nil")
	default:

		panic(fmt.Sprintf("invalid ident: % #v Obj.Decl % #v", ident, typ))
	}
}

// joinLits acts like strings.Join
func joinLits(a []*ast.BasicLit, sep string) string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = a[i].Value
	}
	return strings.Join(s, sep)
}

func (p *parser) resolveFuncDecl(scope *ast.Scope, call *ast.CallExpr) (ast.Expr, error) {
	ident := call.Fun.(*ast.Ident)

	p.tryResolve(ident, false)
	assert(ident.Obj != nil, "failed to locate function: "+ident.Name)
	args := call.Args
	fnDecl := ident.Obj.Decl.(*ast.FuncDecl)

	// setup parameters within a new scope
	// Walk through all statements performing a copy of each
	list := fnDecl.Body.List
	stmts := make([]ast.Stmt, 0, len(list))

	for i := range list {
		if list[i] != nil {
			stmt := ast.StmtCopy(list[i])
			stmts = append(stmts, stmt)
		} else {
			fmt.Println("it is nil", i)
		}
	}
	copyparams := ast.FieldListCopy(fnDecl.Type.Params)
	fields := make([]*ast.Field, 0, len(args))
	for _, arg := range args {
		fields = append(fields, &ast.Field{
			Type: arg,
		})
	}
	copyargs := ast.FieldListCopy(&ast.FieldList{List: fields})
	// All the identifiers within this list need to be re-resolved
	// with the args passed in the include
	p.openScope()
	p.processFuncArgs(scope, copyparams, copyargs)
	stmts = p.resolveStmts(scope, stmts)
	p.closeScope()
	// The last statement should be @return
	ret, ok := stmts[len(stmts)-1].(*ast.ReturnStmt)
	if !ok {
		return nil, errors.New("failed to locate return statement")
	}

	return p.listFromExprs(ret.Results, false, false), nil
}

func (p *parser) resolveIncludeSpec(spec *ast.IncludeSpec) {
	if p.trace {
		defer un(trace(p, "ResolveIncludeSpec"))
	}
	ident := spec.Name
	p.resolve(ident)
	assert(ident.Obj != nil,
		fmt.Sprintf(
			"failed to retrieve mixin: %s(%p)",
			ident.Name,
			p.topScope,
		))
	args := spec.Params
	fnDecl := ident.Obj.Decl.(*ast.FuncDecl)

	// Walk through all statements performing a copy of each
	list := fnDecl.Body.List

	for i := range list {
		stmt := ast.StmtCopy(list[i])
		spec.List = append(spec.List, stmt)
	}

	copyparams := ast.FieldListCopy(fnDecl.Type.Params)
	copyargs := ast.FieldListCopy(args)

	// All the identifiers within this list need to be re-resolved
	// with the args passed in the include
	p.openScope()
	p.processFuncArgs(p.topScope, copyparams, copyargs)
	spec.List = p.resolveStmts(p.topScope, spec.List)
	p.closeScope()
}

// @include foo(second, third);
// @include foo($x: second, $y: third);
func (p *parser) parseIncludeSpec(doResolve bool) *ast.IncludeSpec {
	if p.trace {
		defer un(trace(p, "ParseIncludeSpec"))
	}
	p.expect(token.INCLUDE)
	expr := p.parseOperand(true)
	// Receives ident or basiclit here
	// @include foo(); // ident
	// @include hux;   // basiclit
	ident := ast.ToIdent(expr)
	assert(ident.Name != "_", "invalid include identifier")
	args, _ := p.parseSignature(p.topScope)
	spec := &ast.IncludeSpec{
		Name:   ident,
		Params: args,
	}

	if doResolve {
		p.resolveIncludeSpec(spec)
	} else {
		// Inside mixin, just bail we will come back here later
		fmt.Println("bailed on", ident)
	}

	return spec
}

// @mixin foo($x, $y) {
//   hugabug: $y $x;
// }
func (p *parser) parseMixinDecl() *ast.FuncDecl {
	if p.trace {
		defer un(trace(p, "MixinDecl"))
	}

	// doc := p.leadComment
	pos := p.expect(token.MIXIN)
	// Mixins do not resolve or define variables in any scope
	var scope *ast.Scope //ast.NewScope(nil) // function scope
	ident := p.parseIdent()

	params, _ := p.parseSignature(scope)

	var body *ast.BlockStmt
	if p.tok == token.LBRACE {
		p.inMixin = true
		body = p.parseBody(scope)
		p.inMixin = false
	}

	decl := &ast.FuncDecl{
		// Doc: doc,
		// Recv: recv,
		Tok:  token.MIXIN,
		Name: ident,
		Type: &ast.FuncType{
			Func:   pos,
			Params: params,
		},
		Body: body,
	}

	// Mixins are available to everything parsed from here on out
	p.declare(decl, nil, p.pkgScope, ast.Fun, ident)

	return decl
}

var sentinel bool

func (p *parser) parseFuncDecl() *ast.FuncDecl {
	sentinel = true
	defer func() { sentinel = false }()
	if p.trace {
		defer un(trace(p, "FunctionDecl"))
	}

	// doc := p.leadComment
	pos := p.expect(token.FUNC)
	// function scope is not shared until execution
	var scope *ast.Scope
	// scope := ast.NewScope(p.topScope) // function scope

	var recv *ast.FieldList
	if p.tok == token.LPAREN {
		recv = p.parseParameters(scope, false)
	}

	ident := p.parseIdent()

	params, results := p.parseSignature(scope)

	var body *ast.BlockStmt
	if p.tok == token.LBRACE {
		// prevent resolution of variables in this scope
		p.inMixin = true
		body = p.parseBody(scope)
		p.inMixin = false
	}
	decl := &ast.FuncDecl{
		Recv: recv,
		Tok:  token.FUNC,
		Name: ident,
		Type: &ast.FuncType{
			Func:    pos,
			Params:  params,
			Results: results,
		},
		Body: body,
	}
	p.declare(decl, nil, p.topScope, ast.Var, ident)
	return decl
}

func (p *parser) parseDecl(sync func(*parser)) ast.Decl {
	if p.trace {
		defer un(trace(p, "Declaration"))
	}

	var f parseSpecFunction
	switch p.tok {
	case token.SEMICOLON:
		p.next()
		return nil
	case token.VAR:
		f = p.inferValueSpec
	case token.FUNC:
		return p.parseFuncDecl()
	case token.SELECTOR:
		// Regular CSS
		return p.parseRuleSelDecl()
	case token.INCLUDE:
		return p.parseGenDecl("", token.INCLUDE, p.parseIncludeSpecFn)
	case token.RULE, token.IDENT:
		return p.parseRuleDecl()
	case token.IMPORT:
		// s := &ast.DeclStmt{Decl: p.parse}
		return p.parseGenDecl("", token.IMPORT, p.parseImportSpec)
	case token.MIXIN:
		return p.parseMixinDecl()
	case token.IF:
		stmt := p.parseIfStmt()
		return &ast.IfDecl{IfStmt: stmt}
	default:
		pos := p.pos
		p.errorExpected(pos, "declaration")
		sync(p)
		return &ast.BadDecl{From: pos, To: p.pos}
	}
	return p.parseGenDecl(p.lit, p.tok, f)
}

// ----------------------------------------------------------------------------
// Source files

func (p *parser) parseFile() *ast.File {
	if p.trace {
		defer un(trace(p, "File"))
	}

	// Don't bother parsing the rest if we had errors scanning the first token.
	// Likely not a Go source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

	// Don't bother parsing the rest if we had errors parsing the package clause.
	// Likely not a Go source file at all.
	if p.errors.Len() != 0 {
		return nil
	}

	p.openScope()
	p.pkgScope = p.topScope
	var decls []ast.Decl
	// Bypass importing for now
	// if p.mode&PackageClauseOnly == 0 {
	// 	// import decls
	// 	for p.tok == token.IMPORT {
	// 		decls = append(decls, p.parseGenDecl(token.IMPORT, p.parseImportSpec))
	// 	}

	if p.mode&ImportsOnly == 0 {
		// rest of package body
		for p.tok != token.EOF {
			decls = append(decls, p.parseDecl(syncDecl))
		}
	}

	// }
	p.closeScope()
	assert(p.topScope == nil, "unbalanced scopes")
	assert(p.labelScope == nil, "unbalanced label scopes")

	// resolve global identifiers within the same file
	i := 0
	for _, ident := range p.unresolved {
		// i <= index for current ident
		// assert(ident.Obj == unresolved, "object already resolved")
		// ident.Obj = p.pkgScope.Lookup(ident.Name) // also removes unresolved sentinel
		if ident.Obj == nil {
			// Don't add anything as unresolved yet
			// p.unresolved[i] = ident
			i++
		}
	}

	return &ast.File{
		// Doc: doc,
		// Package:    pos,
		Name:       &ast.Ident{Name: p.file.Name()},
		Decls:      decls,
		Scope:      p.pkgScope,
		Imports:    p.imports,
		Unresolved: p.unresolved[0:i],
		Comments:   p.comments,
	}
}
