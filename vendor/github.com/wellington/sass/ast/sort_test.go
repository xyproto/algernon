package ast

import (
	"reflect"
	"testing"
)

func TestSort(t *testing.T) {
	startcmt := &CommStmt{
		Group: &CommentGroup{
			List: []*Comment{
				{Text: "/* start */"},
			},
		},
	}
	endcmt := &CommStmt{
		Group: &CommentGroup{
			List: []*Comment{
				{Text: "/* end */"},
			},
		},
	}

	list := []Stmt{
		&SelStmt{Name: &Ident{Name: "div"}},
		startcmt,
		&DeclStmt{},
		endcmt,
		&SelStmt{Name: &Ident{Name: "p"}},
		&IncludeStmt{},
		&AssignStmt{},
	}

	sorted := []Stmt{
		&CommStmt{},
		&DeclStmt{},
		&CommStmt{},
		&IncludeStmt{},
		&AssignStmt{},
		&SelStmt{},
		&SelStmt{},
	}

	SortStatements(list)

	for i := range list {
		l := reflect.TypeOf(list[i])
		if e := reflect.TypeOf(sorted[i]); e != l {
			t.Errorf("%d got: %v wanted: %v\n", i, l, e)
		}
	}

}
