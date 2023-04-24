/*
 * a data structure to test a string against a set of prefixes
 */

package utils

import (
	"fmt"
)

type Node struct {
	Children   map[rune]*Node
	IsTerminal bool
}

func (node *Node) GetChild(c rune) *Node {
	ret, found := node.Children[c]
	if found {
		return ret
	} else {
		return nil
	}
}

type PrefixMatch struct {
	root Node
}

func (pm *PrefixMatch) Build(prefixes []string) {

	//pm.root.Value = ""
	for _, prefix := range prefixes {
		current := &pm.root
		for i, char := range prefix {
			if current.Children == nil {
				current.Children = make(map[rune]*Node)
			}
			_, exists := current.Children[char]
			if !exists {
				node := new(Node)
				node.IsTerminal = i == len(prefix)-1
				current.Children[char] = node
				//fmt.Printf("adding %c -> %p\n", char, current.GetChild(char))
			}
			current, _ = current.Children[char]
		}
	}
}

func (pm *PrefixMatch) Match(str string) []string {
	result := make([]string, 0)
	pm.match(str, &pm.root, "", &result)
	return result
}

func (pm *PrefixMatch) match(str string, start *Node, path string, found *[]string) {
	if start.IsTerminal {
		*found = append(*found, path)
	}
	if len(str) == 0 {
		return
	}
	for char, child := range start.Children {
		if string(str[0]) != string(char) {
			return
		}
		pm.match(str[1:], child, path+string(char), found)
	}
}

func (pm *PrefixMatch) PPrint(args ...*Node) {
	var start *Node
	if len(args) == 0 {
		start = &pm.root
	} else {
		start = args[0]
	}
	fmt.Printf("%p:\n", start)
	for char, node := range start.Children {
		fmt.Printf("|-- %c: %p\n", char, node)
	}
	fmt.Printf("\n")
	for _, node := range start.Children {
		pm.PPrint(node)
	}
}
