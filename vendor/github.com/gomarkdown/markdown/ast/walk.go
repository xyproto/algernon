package ast

// WalkStatus allows NodeVisitor to have some control over the tree traversal.
// It is returned from NodeVisitor and different values allow Node.Walk to
// decide which node to go to next.
type WalkStatus int

const (
	// GoToNext is the default traversal of every node.
	GoToNext WalkStatus = iota
	// SkipChildren tells walker to skip all children of current node.
	SkipChildren
	// Terminate tells walker to terminate the traversal.
	Terminate
)

// NodeVisitor is a callback to be called when traversing the syntax tree.
// Called twice for every node: once with entering=true when the branch is
// first visited, then with entering=false after all the children are done.
type NodeVisitor interface {
	Visit(node Node, entering bool) WalkStatus
}

// NodeVisitorFunc casts a function to match NodeVisitor interface
type NodeVisitorFunc func(node Node, entering bool) WalkStatus

// Walk traverses tree recursively
func Walk(n Node, visitor NodeVisitor) WalkStatus {
	isContainer := n.AsContainer() != nil
	status := visitor.Visit(n, true) // entering
	if status == Terminate {
		// even if terminating, close container node
		if isContainer {
			visitor.Visit(n, false)
		}
		return status
	}
	if isContainer && status != SkipChildren {
		children := n.GetChildren()
		for _, n := range children {
			status = Walk(n, visitor)
			if status == Terminate {
				return status
			}
		}
	}
	if isContainer {
		status = visitor.Visit(n, false) // exiting
		if status == Terminate {
			return status
		}
	}
	return GoToNext
}

// Visit calls visitor function
func (f NodeVisitorFunc) Visit(node Node, entering bool) WalkStatus { return f(node, entering) }

// WalkFunc is like Walk but accepts just a callback function
func WalkFunc(n Node, f NodeVisitorFunc) {
	Walk(n, f)
}
