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
	setPrevNode(child, prev)
	setNextNode(child, nil)
	setNextNode(prev, child)
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
	setNextNode(prev, next)
	setPrevNode(next, prev)
	setPrevNode(n, nil)
	setNextNode(n, nil)
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
	for i, child := range children {
		var prev, next Node
		if i > 0 {
			prev = children[i-1]
		}
		if i+1 < len(children) {
			next = children[i+1]
		}
		setPrevNode(child, prev)
		setNextNode(child, next)
	}
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

func setPrevNode(n Node, prev Node) {
	if link, _, ok := siblingLinks(n); ok {
		*link = prev
	}
}

func setNextNode(n Node, next Node) {
	if _, link, ok := siblingLinks(n); ok {
		*link = next
	}
}
