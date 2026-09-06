package parser

// Kind identifies a parser-neutral PostgreSQL AST node type. It deliberately
// avoids backend-generated enums so callers do not depend on pg_query types.
type Kind string

// ValueKind identifies the data stored in a Value.
type ValueKind uint8

// Parser-neutral value kinds.
const (
	ValueInvalid ValueKind = iota
	ValueBool
	ValueInt
	ValueUint
	ValueFloat
	ValueString
	ValueBytes
	ValueEnum
	ValueNode
	ValueList
)

// Field is an immutable named value on a Node. Its state stays private so a
// parsed tree cannot be changed through values returned to another package.
type Field struct {
	name  string
	value Value
}

// Name returns the parser-neutral field name.
func (f Field) Name() string {
	return f.name
}

// Value returns the field value.
func (f Field) Value() Value {
	return f.value
}

// Node is an immutable parser-neutral PostgreSQL AST node. Accessors return
// snapshots for slice-backed data; child nodes are safe to share because all
// their state is private and has no mutating API.
type Node struct {
	kind   Kind
	fields []Field
}

// Kind returns the PostgreSQL AST node kind.
func (n *Node) Kind() Kind {
	if n == nil {
		return ""
	}

	return n.kind
}

// Field returns a named field value.
func (n *Node) Field(name string) (Value, bool) {
	if n == nil {
		return Value{}, false
	}

	for _, field := range n.fields {
		if field.name == name {
			return field.value, true
		}
	}

	return Value{}, false
}

// Fields returns a snapshot of the node fields in schema declaration order.
// Declaration order makes structural traversal reproducible across calls.
func (n *Node) Fields() []Field {
	if n == nil {
		return nil
	}

	return append([]Field(nil), n.fields...)
}

// Value is an immutable parser-neutral scalar, node, or list. The tagged union
// keeps protobuf reflection values from crossing the parser boundary.
type Value struct {
	kind        ValueKind
	boolValue   bool
	intValue    int64
	uintValue   uint64
	floatValue  float64
	stringValue string
	bytesValue  []byte
	nodeValue   *Node
	listValue   []Value
}

// Kind returns the stored value kind.
func (v Value) Kind() ValueKind {
	return v.kind
}

// Bool returns the stored boolean.
func (v Value) Bool() (bool, bool) {
	return v.boolValue, v.kind == ValueBool
}

// Int returns the stored signed integer.
func (v Value) Int() (int64, bool) {
	return v.intValue, v.kind == ValueInt
}

// Uint returns the stored unsigned integer.
func (v Value) Uint() (uint64, bool) {
	return v.uintValue, v.kind == ValueUint
}

// Float returns the stored floating-point number.
func (v Value) Float() (float64, bool) {
	return v.floatValue, v.kind == ValueFloat
}

// String returns the stored string.
func (v Value) String() (string, bool) {
	return v.stringValue, v.kind == ValueString
}

// Bytes returns a snapshot of the stored bytes so callers cannot mutate the
// parsed tree through a shared backing array.
func (v Value) Bytes() ([]byte, bool) {
	if v.kind != ValueBytes {
		return nil, false
	}

	return append([]byte(nil), v.bytesValue...), true
}

// Enum returns the stored symbolic enum value.
func (v Value) Enum() (string, bool) {
	return v.stringValue, v.kind == ValueEnum
}

// Node returns the stored node.
func (v Value) Node() (*Node, bool) {
	return v.nodeValue, v.kind == ValueNode
}

// List returns a snapshot of the stored list so callers can reorder or replace
// returned values without changing the parsed tree.
func (v Value) List() ([]Value, bool) {
	if v.kind != ValueList {
		return nil, false
	}

	return append([]Value(nil), v.listValue...), true
}
