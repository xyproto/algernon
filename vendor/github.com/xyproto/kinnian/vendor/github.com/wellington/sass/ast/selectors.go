package ast

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/wellington/sass/token"
)

var (
	regEql = regexp.MustCompile("\\s*(\\*?=)\\s*").ReplaceAll
	regBkt = regexp.MustCompile("\\s*(\\[)\\s*(\\S+)\\s*(\\])").ReplaceAll
	nilW   = bytes.NewBuffer(nil)
)

// Resolves walks selector operations removing nested Op by prepending X
// on Y.
func (stmt *SelStmt) Resolve(fset *token.FileSet) {
	if stmt.Sel == nil {
		panic(fmt.Errorf("invalid selector: % #v\n", stmt))
	}

	stmt.Resolved = Selector(stmt)
	return
}

func selSplit(s string) []string {
	ss := strings.Split(s, ",")
	for i := range ss {
		ss[i] = strings.TrimSpace(ss[i])
		if !strings.Contains(ss[i], "&") {
			// Add implicit ampersand
			ss[i] = "& " + ss[i]
		}
	}
	return ss
}

func joinParent(delim, parent string, nodes []string) []string {
	rep := "&"
	if len(parent) == 0 {
		rep = "& "
	}
	commadelim := "," + delim
	parts := strings.Split(parent, commadelim)
	var ret []string
	for i := range parts {
		for j := range nodes {
			rep := strings.Replace(nodes[j], rep, parts[i], -1)
			ret = append(ret, rep)
		}
	}
	return ret
}
