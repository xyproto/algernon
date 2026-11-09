package ast

func removeNodeFromArray(a []Node, node Node) []Node {
	n := len(a)
	for i := 0; i < n; i++ {
		if a[i] == node {
			return append(a[:i], a[i+1:]...)
		}
	}
	return nil
}

// AppendChild appends child to children of parent
// It panics if either node is nil.
func AppendChild(parent Node, child Node) {
	RemoveFromTree(child)
	child.SetParent(parent)
	children := parent.GetChildren()
	prev := GetLastChild(parent)
	linkNodes(prev, child)
	linkNodes(child, nil)
	if container := parent.AsContainer(); container != nil {
		container.Children = append(container.Children, child)
		return
	}
	parent.SetChildren(append(children, child))
}

// RemoveFromTree removes this node from tree
func RemoveFromTree(n Node) {
	p := n.GetParent()
	if p == nil {
		return
	}
	// important: don't clear n.Children if n has no parent
	// we're called from AppendChild and that might happen on a node
	// that accumulated Children but hasn't been inserted into the tree
	n.SetChildren(nil)
	prev := GetPrevNode(n)
	next := GetNextNode(n)
	linkNodes(prev, next)
	linkNodes(nil, n)
	linkNodes(n, nil)
	newChildren := removeNodeFromArray(p.GetChildren(), n)
	if newChildren != nil {
		p.SetChildren(newChildren)
	}
}

// GetLastChild returns last child of node n
// It's implemented as stand-alone function to keep Node interface small
func GetLastChild(n Node) Node {
	a := n.GetChildren()
	if len(a) > 0 {
		return a[len(a)-1]
	}
	return nil
}

// GetFirstChild returns first child of node n
// It's implemented as stand-alone function to keep Node interface small
func GetFirstChild(n Node) Node {
	a := n.GetChildren()
	if len(a) > 0 {
		return a[0]
	}
	return nil
}

// GetNextNode returns next sibling of node n (node after n)
// We can't make it part of Container or Leaf because we loose Node identity
func GetNextNode(n Node) Node {
	if _, next, ok := siblingLinks(n); ok {
		return *next
	}
	parent := n.GetParent()
	if parent == nil {
		return nil
	}
	a := parent.GetChildren()
	for i := 0; i+1 < len(a); i++ {
		if a[i] == n {
			return a[i+1]
		}
	}
	return nil
}

// GetPrevNode returns previous sibling of node n (node before n)
// We can't make it part of Container or Leaf because we loose Node identity
func GetPrevNode(n Node) Node {
	if prev, _, ok := siblingLinks(n); ok {
		return *prev
	}
	parent := n.GetParent()
	if parent == nil {
		return nil
	}
	a := parent.GetChildren()
	len := len(a)
	for i := 1; i < len; i++ {
		if a[i] == n {
			return a[i-1]
		}
	}
	return nil
}

func linkSiblings(children []Node) {
	var prev Node
	for _, child := range children {
		linkNodes(prev, child)
		prev = child
	}
	linkNodes(prev, nil)
}

func siblingLinks(n Node) (prev, next *Node, ok bool) {
	if n == nil {
		return nil, nil, false
	}
	if c := n.AsContainer(); c != nil {
		return &c.Prev, &c.Next, true
	}
	if l := n.AsLeaf(); l != nil {
		return &l.Prev, &l.Next, true
	}
	return nil, nil, false
}

// linkNodes makes b the next sibling of a and a the previous sibling of b.
// Either may be nil, which clears the other's link.
func linkNodes(a, b Node) {
	if _, next, ok := siblingLinks(a); ok {
		*next = b
	}
	if prev, _, ok := siblingLinks(b); ok {
		*prev = a
	}
}
