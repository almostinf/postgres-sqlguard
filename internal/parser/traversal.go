package parser

// StatementSequence returns top-level statements and their statement-bearing
// CTEs in deterministic root-first, depth-first declaration order. A new slice
// is built for every call, so consumers cannot mutate Result state.
func StatementSequence(result *Result) []*Node {
	if result == nil {
		return nil
	}

	sequence := make([]*Node, 0, len(result.statements))
	for _, statement := range result.statements {
		sequence = appendStatement(sequence, statement)
	}

	return sequence
}

// appendStatement adds one statement and then recursively adds the statements
// stored by its CTE declarations before moving to the next sibling statement.
func appendStatement(sequence []*Node, statement *Node) []*Node {
	if statement == nil {
		return sequence
	}

	sequence = append(sequence, statement)
	// PostgreSQL attaches a statement's CTE declarations through with_clause.
	// Visiting it after the root implements the documented pre-order.
	withClause := nodeField(statement, "with_clause")
	if withClause == nil {
		return sequence
	}

	ctes, ok := withClause.Field("ctes")
	if !ok {
		return sequence
	}

	cteValues, ok := ctes.List()
	if !ok {
		return sequence
	}

	for _, cteValue := range cteValues {
		cte, ok := cteValue.Node()
		if !ok {
			continue
		}

		// Recursing per declaration before continuing the list produces a stable
		// depth-first order and covers CTEs nested inside another CTE statement.
		sequence = appendStatement(sequence, nodeField(cte, "ctequery"))
	}

	return sequence
}

// nodeField reads a named child only when that field contains a node value.
// Missing fields and scalar fields are intentionally treated alike by traversal.
func nodeField(node *Node, name string) *Node {
	value, ok := node.Field(name)
	if !ok {
		return nil
	}

	child, ok := value.Node()
	if !ok {
		return nil
	}

	return child
}
