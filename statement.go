package sqlguard

import "github.com/almostinf/postgres-sqlguard/internal/parser"

// Kind identifies a stable structural PostgreSQL AST node kind.
type Kind string

// Statement is an immutable view of one statement root. Its values are valid
// for the duration of a Rule call and may be copied, but rules must not retain
// them after Evaluate returns.
type Statement struct {
	root Node
}

// newStatement creates a public facade without exposing the internal node.
func newStatement(root *parser.Node) Statement {
	return Statement{root: Node{node: root}}
}

// Kind returns the statement root kind.
func (s Statement) Kind() Kind {
	return s.root.Kind()
}

// Root returns the statement root as a generic read-only node.
func (s Statement) Root() Node {
	return s.root
}

// Walk visits the statement root and its descendants in structural pre-order.
// Returning false from visit stops traversal. A nil visitor performs no work.
func (s Statement) Walk(visit func(Node) bool) {
	s.root.Walk(visit)
}

// Node is an immutable view of one structural PostgreSQL AST node.
type Node struct {
	node *parser.Node
}

// Kind returns the structural node kind.
func (n Node) Kind() Kind {
	return Kind(n.node.Kind())
}

// Child returns a named field when it contains a node.
func (n Node) Child(name string) (Node, bool) {
	value, ok := n.value(name)
	if !ok {
		return Node{}, false
	}

	child, ok := value.Node()
	if !ok {
		return Node{}, false
	}

	return Node{node: child}, true
}

// Children returns a snapshot of node values stored in a named list field.
func (n Node) Children(name string) []Node {
	value, ok := n.value(name)
	if !ok {
		return nil
	}

	values, ok := value.List()
	if !ok {
		return nil
	}

	children := make([]Node, 0, len(values))
	for _, item := range values {
		child, isNode := item.Node()
		if isNode {
			children = append(children, Node{node: child})
		}
	}

	return children
}

// Bool returns a named boolean field.
func (n Node) Bool(name string) (bool, bool) {
	value, ok := n.value(name)
	if !ok {
		return false, false
	}

	return value.Bool()
}

// Int returns a named signed-integer field.
func (n Node) Int(name string) (int64, bool) {
	value, ok := n.value(name)
	if !ok {
		return 0, false
	}

	return value.Int()
}

// Uint returns a named unsigned-integer field.
func (n Node) Uint(name string) (uint64, bool) {
	value, ok := n.value(name)
	if !ok {
		return 0, false
	}

	return value.Uint()
}

// Float returns a named floating-point field.
func (n Node) Float(name string) (float64, bool) {
	value, ok := n.value(name)
	if !ok {
		return 0, false
	}

	return value.Float()
}

// String returns a named string field.
func (n Node) String(name string) (string, bool) {
	value, ok := n.value(name)
	if !ok {
		return "", false
	}

	return value.String()
}

// Bytes returns a snapshot of a named bytes field.
func (n Node) Bytes(name string) ([]byte, bool) {
	value, ok := n.value(name)
	if !ok {
		return nil, false
	}

	return value.Bytes()
}

// Enum returns the symbolic value of a named enum field.
func (n Node) Enum(name string) (string, bool) {
	value, ok := n.value(name)
	if !ok {
		return "", false
	}

	return value.Enum()
}

// Walk visits this node and its descendants in structural pre-order. Returning
// false from visit stops traversal. A nil visitor performs no work.
func (n Node) Walk(visit func(Node) bool) {
	if n.node == nil || visit == nil {
		return
	}

	walkNode(n.node, visit)
}

func (n Node) value(name string) (parser.Value, bool) {
	if n.node == nil {
		return parser.Value{}, false
	}

	return n.node.Field(name)
}

func walkNode(node *parser.Node, visit func(Node) bool) bool {
	if node == nil || !visit(Node{node: node}) {
		return false
	}

	for _, field := range node.Fields() {
		value := field.Value()
		if child, ok := value.Node(); ok {
			if !walkNode(child, visit) {
				return false
			}

			continue
		}

		values, ok := value.List()
		if !ok {
			continue
		}

		for _, item := range values {
			child, isNode := item.Node()
			if isNode && !walkNode(child, visit) {
				return false
			}
		}
	}

	return true
}
