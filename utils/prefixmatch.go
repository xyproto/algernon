// Package utils provides a data structure for testing a string against a set
// of prefixes, along with other small helpers.
package utils

// Node is a trie node used by PrefixMatch.
type Node struct {
	Children   map[rune]*Node
	IsTerminal bool
}

// PrefixMatch tests whether a string starts with any of a set of prefixes.
type PrefixMatch struct {
	root Node
}

// Build constructs the prefix trie from the given list of prefixes.
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
			current = current.Children[char]
		}
	}
}

// Match returns the prefixes from the trie that match the start of str.
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
		if string(str[0]) == string(char) {
			pm.match(str[1:], child, path+string(char), found)
		}
	}
}
