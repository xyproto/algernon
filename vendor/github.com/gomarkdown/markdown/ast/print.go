package ast

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// Print is for debugging. It prints a string representation of parsed
// markdown doc (result of parser.Parse()) to dst.
//
// To make output readable, it shortens text output.
func Print(dst io.Writer, doc Node) {
	PrintWithPrefix(dst, doc, "  ")
}

// PrintWithPrefix is like Print but allows customizing prefix used for
// indentation. By default it's 2 spaces. You can change it to e.g. tab
// by passing "\t"
func PrintWithPrefix(w io.Writer, doc Node, prefix string) {
	// for more compact output, don't print outer Document
	if _, ok := doc.(*Document); ok {
		for _, c := range doc.GetChildren() {
			printRecur(w, c, prefix, 0)
		}
	} else {
		printRecur(w, doc, prefix, 0)
	}
}

// ToString is like Print but returns the result as a string.
func ToString(doc Node) string {
	var buf bytes.Buffer
	Print(&buf, doc)
	return buf.String()
}

func contentToString(d1 []byte, d2 []byte) string {
	if d1 != nil {
		return string(d1)
	}
	return string(d2)
}

func getContent(node Node) string {
	if c := node.AsContainer(); c != nil {
		return contentToString(c.Literal, c.Content)
	}
	leaf := node.AsLeaf()
	return contentToString(leaf.Literal, leaf.Content)
}

func shortenString(s string, maxLen int) string {
	// for cleaner, one-line ouput, replace some white-space chars
	// with their escaped version
	s = strings.Replace(s, "\n", `\n`, -1)
	s = strings.Replace(s, "\r", `\r`, -1)
	s = strings.Replace(s, "\t", `\t`, -1)
	if maxLen < 0 {
		return s
	}
	if utf8.RuneCountInString(s) < maxLen {
		return s
	}
	// add "…" to indicate truncation
	return string(append([]rune(s)[:maxLen-3], '…'))
}

// get a short name of the type of v which excludes package name
// and strips "()" from the end
func getNodeType(node Node) string {
	s := fmt.Sprintf("%T", node)
	s = strings.TrimSuffix(s, "()")
	if idx := strings.Index(s, "."); idx != -1 {
		return s[idx+1:]
	}
	return s
}

func printDefault(w io.Writer, indent string, typeName string, content string) {
	content = strings.TrimSpace(content)
	if content != "" {
		fmt.Fprintf(w, "%s%s '%s'\n", indent, typeName, content)
	} else {
		fmt.Fprintf(w, "%s%s\n", indent, typeName)
	}
}

var listFlagNames = []struct {
	flag ListType
	name string
}{
	{ListTypeOrdered, "ordered"},
	{ListTypeDefinition, "definition"},
	{ListTypeTerm, "term"},
	{ListItemContainsBlock, "has_block"},
	{ListItemBeginningOfList, "start"},
	{ListItemEndOfList, "end"},
}

func listDetails(content string, flags ListType, start int, tight, footnotes bool) string {
	if start > 1 {
		content += fmt.Sprintf("start=%d ", start)
	}
	if tight {
		content += "tight "
	}
	if footnotes {
		content += "footnotes "
	}
	var names []string
	for _, entry := range listFlagNames {
		if flags&entry.flag != 0 {
			names = append(names, entry.name)
		}
	}
	if len(names) > 0 {
		content += "flags=" + strings.Join(names, " ") + " "
	}
	return content
}

func printRecur(w io.Writer, node Node, prefix string, depth int) {
	if node == nil {
		return
	}
	indent := strings.Repeat(prefix, depth)

	content := shortenString(getContent(node), 40)
	typeName := getNodeType(node)
	switch v := node.(type) {
	case *Link:
		content := "url=" + string(v.Destination)
		printDefault(w, indent, typeName, content)
	case *Image:
		content := "url=" + string(v.Destination)
		printDefault(w, indent, typeName, content)
	case *List:
		content = listDetails(content, v.ListFlags, v.Start, v.Tight, v.IsFootnotesList)
		printDefault(w, indent, typeName, content)
	case *ListItem:
		content = listDetails(content, v.ListFlags, 0, v.Tight, v.IsFootnotesList)
		printDefault(w, indent, typeName, content)
	case *CodeBlock:
		printDefault(w, indent, typeName+":"+string(v.Info), content)
	default:
		printDefault(w, indent, typeName, content)
	}
	for _, child := range node.GetChildren() {
		printRecur(w, child, prefix, depth+1)
	}
}
