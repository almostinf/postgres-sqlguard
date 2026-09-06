package parser

import (
	pg_query "github.com/pganalyze/pg_query_go/v6"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const pgQueryNodeName protoreflect.FullName = "pg_query.Node"

// Parse parses complete SQL input with the PostgreSQL grammar and immediately
// converts the backend tree into package-owned immutable values. This function
// is the only place where raw parser errors can enter the library.
func Parse(sql string) (*Result, error) {
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return nil, &Error{category: FailureSyntax}
	}

	statements := make([]*Node, 0, len(tree.GetStmts()))
	for _, rawStatement := range tree.GetStmts() {
		statement := convertPGNode(rawStatement.GetStmt())
		if statement != nil {
			statements = append(statements, statement)
		}
	}

	return &Result{statements: statements}, nil
}

// convertPGNode handles the generated outer Node message separately from all
// concrete AST messages. A nil wrapper represents no syntax node and is omitted.
func convertPGNode(source *pg_query.Node) *Node {
	if source == nil {
		return nil
	}

	return convertMessage(source.ProtoReflect())
}

// convertMessage copies a protobuf message instead of retaining a reference to
// mutable backend state. Fields are read in descriptor order because protobuf's
// Range iteration order is explicitly unspecified.
func convertMessage(source protoreflect.Message) *Node {
	if !source.IsValid() {
		return nil
	}

	descriptor := source.Descriptor()
	if descriptor.FullName() == pgQueryNodeName {
		// pg_query.Node is a generated oneof container, not a semantic AST node.
		// Unwrapping it keeps that backend implementation detail out of our tree.
		oneof := descriptor.Oneofs().Get(0)

		field := source.WhichOneof(oneof)
		if field == nil {
			return nil
		}

		return convertMessage(source.Get(field).Message())
	}

	fields := make([]Field, 0, descriptor.Fields().Len())
	for index := range descriptor.Fields().Len() {
		field := descriptor.Fields().Get(index)
		if !source.Has(field) {
			continue
		}

		fields = append(fields, Field{
			name:  string(field.Name()),
			value: convertValue(field, source.Get(field)),
		})
	}

	return &Node{
		kind:   Kind(descriptor.Name()),
		fields: fields,
	}
}

// convertValue preserves repeated-field order and delegates scalar conversion
// to the same path used by singular protobuf fields.
func convertValue(descriptor protoreflect.FieldDescriptor, source protoreflect.Value) Value {
	if descriptor.IsList() {
		// Copy repeated values eagerly; the protobuf tree is discarded after Parse.
		list := source.List()

		values := make([]Value, 0, list.Len())
		for index := range list.Len() {
			values = append(values, convertSingularValue(descriptor, list.Get(index)))
		}

		return Value{kind: ValueList, listValue: values}
	}

	return convertSingularValue(descriptor, source)
}

// convertSingularValue maps every protobuf scalar family to a bounded internal
// representation and recursively detaches message values from the backend.
func convertSingularValue(descriptor protoreflect.FieldDescriptor, source protoreflect.Value) Value {
	switch descriptor.Kind() {
	case protoreflect.BoolKind:
		return Value{kind: ValueBool, boolValue: source.Bool()}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return Value{kind: ValueInt, intValue: source.Int()}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return Value{kind: ValueUint, uintValue: source.Uint()}
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		return Value{kind: ValueFloat, floatValue: source.Float()}
	case protoreflect.StringKind:
		return Value{kind: ValueString, stringValue: source.String()}
	case protoreflect.BytesKind:
		return Value{kind: ValueBytes, bytesValue: append([]byte(nil), source.Bytes()...)}
	case protoreflect.EnumKind:
		value := descriptor.Enum().Values().ByNumber(source.Enum())
		if value == nil {
			return Value{kind: ValueEnum}
		}

		return Value{kind: ValueEnum, stringValue: string(value.Name())}
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return Value{kind: ValueNode, nodeValue: convertMessage(source.Message())}
	default:
		return Value{}
	}
}
