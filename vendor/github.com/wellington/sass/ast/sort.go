package ast

import (
	"fmt"

	"github.com/wellington/sass/token"
)

var _ token.Pos

type Stmts []Stmt

func (s Stmts) lookup(pos int) int {
	i := 0
	switch s[pos].(type) {
	case *DeclStmt, *IncludeStmt, *EmptyStmt,
		*AssignStmt, *BadStmt, *EachStmt, *IfStmt:
	case *ReturnStmt:
	case *CommStmt:
	case *BlockStmt:
		// This is an error situation, but better errors are
		// reported if it gets sorted
		i = 1000
	case *SelStmt, *MediaStmt:
		// log.Printf("pushing to end % #v\n", v)
		//Print(token.NewFileSet(), v)
		i = 1
	default:
		panic(fmt.Errorf("failed to sort % #v\n", s[pos]))
	}
	return i
}

// Sort statements for most efficient usage of CSS rules
// (rules first, then other tings)
func SortStatements(list Stmts) {
	b := list[:0]
	var notrules []Stmt
	for i := range list {
		if list.lookup(i) == 1 {
			notrules = append(notrules, list[i])
			continue
		}
		b = append(b, list[i])
	}
	b = append(b, notrules...)
}
