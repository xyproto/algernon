package utils

import "unicode/utf8"

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
	for _, prefix := range prefixes {
		current := &pm.root
		for _, char := range prefix {
			if current.Children == nil {
				current.Children = make(map[rune]*Node)
			}
			child, exists := current.Children[char]
			if !exists {
				child = new(Node)
				current.Children[char] = child
			}
			current = child
		}
		current.IsTerminal = true
	}
}

// Match returns the prefixes from the trie that match the start of str.
func (pm *PrefixMatch) Match(str string) []string {
	result := make([]string, 0)
	current := &pm.root
	for i := 0; ; {
		if current.IsTerminal {
			result = append(result, str[:i])
		}
		if i >= len(str) {
			break
		}
		r, size := utf8.DecodeRuneInString(str[i:])
		child, ok := current.Children[r]
		if !ok {
			break
		}
		current = child
		i += size
	}
	return result
}
