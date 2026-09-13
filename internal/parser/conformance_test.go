package parser

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const updateConformanceGoldensEnv = "UPDATE_CONFORMANCE_GOLDENS"

type conformanceCase struct {
	path      string
	wantParse bool
}

type canonicalResult struct {
	Statements []*canonicalNode `json:"statements"`
}

type canonicalNode struct {
	Kind   string           `json:"kind"`
	Fields []canonicalField `json:"fields"`
}

type canonicalField struct {
	Name  string         `json:"name"`
	Value canonicalValue `json:"value"`
}

type canonicalValue struct {
	Kind  string `json:"kind"`
	Value any    `json:"value"`
}

func TestParserConformanceCorpus(t *testing.T) {
	t.Parallel()

	tests := loadConformanceCases(t)
	names := sortedConformanceCaseNames(tests)
	updateGoldens := os.Getenv(updateConformanceGoldensEnv) == "1"

	if !updateGoldens {
		requireGoldenSet(t, names, tests)
	}

	for _, name := range names {
		testCase := tests[name]

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(testCase.path)
			require.NoError(t, err)

			result, parseErr := Parse(string(input))
			if !testCase.wantParse {
				require.Nil(t, result)
				requireSyntaxFailure(t, parseErr)

				return
			}

			require.NoError(t, parseErr)

			snapshot, err := canonicalSnapshot(result)
			require.NoError(t, err)

			goldenPath := conformanceGoldenPath(name)
			if updateGoldens {
				require.NoError(t, os.MkdirAll(filepath.Dir(goldenPath), 0o750))
				require.NoError(t, os.WriteFile(goldenPath, snapshot, 0o600))

				return
			}

			golden, err := os.ReadFile(goldenPath)
			require.NoError(t, err)
			require.Equal(t, string(golden), string(snapshot))
		})
	}
}

func TestCanonicalSnapshotPreservesKindsValuesAndOrder(t *testing.T) {
	t.Parallel()

	result := &Result{statements: []*Node{
		{
			kind: "FixtureStmt",
			fields: []Field{
				{name: "invalid", value: Value{}},
				{name: "bool", value: Value{kind: ValueBool, boolValue: false}},
				{name: "int", value: Value{kind: ValueInt, intValue: -7}},
				{name: "uint", value: Value{kind: ValueUint, uintValue: 9}},
				{name: "float", value: Value{kind: ValueFloat, floatValue: 3.25}},
				{name: "string", value: Value{kind: ValueString, stringValue: "value"}},
				{name: "bytes", value: Value{kind: ValueBytes, bytesValue: []byte{0, 1, 255}}},
				{name: "enum", value: Value{kind: ValueEnum, stringValue: "OPTION"}},
				{name: "node", value: Value{kind: ValueNode, nodeValue: &Node{kind: "Child", fields: []Field{}}}},
				{name: "list", value: Value{kind: ValueList, listValue: []Value{
					{kind: ValueString, stringValue: "first"},
					{kind: ValueString, stringValue: "second"},
				}}},
			},
		},
	}}

	snapshot, err := canonicalSnapshot(result)

	require.NoError(t, err)
	require.JSONEq(t, canonicalSnapshotFixture, string(snapshot))

	repeated, err := canonicalSnapshot(result)
	require.NoError(t, err)
	require.Equal(t, snapshot, repeated)
}

func loadConformanceCases(t *testing.T) map[string]conformanceCase {
	t.Helper()

	tests := make(map[string]conformanceCase)

	for _, classification := range []struct {
		directory string
		wantParse bool
	}{
		{directory: "valid", wantParse: true},
		{directory: "invalid", wantParse: false},
	} {
		root := filepath.Join("testdata", "conformance", classification.directory)
		paths := conformanceFiles(t, root, ".sql")
		require.NotEmpty(t, paths, "%s corpus must not be empty", classification.directory)

		for _, path := range paths {
			relative, err := filepath.Rel(root, path)
			require.NoError(t, err)

			name := classification.directory + "/" + strings.TrimSuffix(filepath.ToSlash(relative), filepath.Ext(relative))
			_, duplicate := tests[name]
			require.False(t, duplicate, "duplicate conformance case %s", name)

			tests[name] = conformanceCase{path: path, wantParse: classification.wantParse}
		}
	}

	return tests
}

func sortedConformanceCaseNames(tests map[string]conformanceCase) []string {
	names := make([]string, 0, len(tests))
	for name := range tests {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

func requireGoldenSet(t *testing.T, names []string, tests map[string]conformanceCase) {
	t.Helper()

	want := make([]string, 0, len(names))
	for _, name := range names {
		if tests[name].wantParse {
			want = append(want, strings.TrimPrefix(name, "valid/")+".json")
		}
	}

	root := filepath.Join("testdata", "conformance", "golden")
	paths := conformanceFiles(t, root, ".json")

	got := make([]string, 0, len(paths))
	for _, path := range paths {
		relative, err := filepath.Rel(root, path)
		require.NoError(t, err)

		got = append(got, filepath.ToSlash(relative))
	}

	slices.Sort(want)
	slices.Sort(got)
	require.Equal(t, want, got, "golden files must correspond exactly to valid corpus cases")
}

func conformanceGoldenPath(name string) string {
	relative := filepath.FromSlash(strings.TrimPrefix(name, "valid/") + ".json")

	return filepath.Join("testdata", "conformance", "golden", relative)
}

func conformanceFiles(t *testing.T, root string, extension string) []string {
	t.Helper()

	paths := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if !entry.IsDir() && filepath.Ext(path) == extension {
			paths = append(paths, path)
		}

		return nil
	})
	require.NoError(t, err)

	return paths
}

func requireSyntaxFailure(t *testing.T, err error) {
	t.Helper()

	var parseError *Error
	require.ErrorAs(t, err, &parseError)
	require.Equal(t, FailureSyntax, parseError.Category())
}

func canonicalSnapshot(result *Result) ([]byte, error) {
	snapshot := canonicalResult{Statements: make([]*canonicalNode, 0)}
	if result != nil {
		for _, statement := range result.Statements() {
			node, err := canonicalizeNode(statement)
			if err != nil {
				return nil, err
			}

			snapshot.Statements = append(snapshot.Statements, node)
		}
	}

	var output bytes.Buffer

	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(snapshot); err != nil {
		return nil, fmt.Errorf("encode canonical parser snapshot: %w", err)
	}

	return output.Bytes(), nil
}

func canonicalizeNode(node *Node) (*canonicalNode, error) {
	if node == nil {
		return nil, nil
	}

	snapshot := &canonicalNode{
		Kind:   string(node.Kind()),
		Fields: make([]canonicalField, 0, len(node.Fields())),
	}

	for _, field := range node.Fields() {
		value, err := canonicalizeValue(field.Value())
		if err != nil {
			return nil, fmt.Errorf("canonicalize field %q on %q: %w", field.Name(), node.Kind(), err)
		}

		snapshot.Fields = append(snapshot.Fields, canonicalField{Name: field.Name(), Value: value})
	}

	return snapshot, nil
}

func canonicalizeValue(value Value) (canonicalValue, error) {
	switch value.Kind() {
	case ValueInvalid:
		return canonicalValue{Kind: "invalid", Value: nil}, nil
	case ValueBool:
		stored, _ := value.Bool()
		return canonicalValue{Kind: "bool", Value: stored}, nil
	case ValueInt:
		stored, _ := value.Int()
		return canonicalValue{Kind: "int", Value: strconv.FormatInt(stored, 10)}, nil
	case ValueUint:
		stored, _ := value.Uint()
		return canonicalValue{Kind: "uint", Value: strconv.FormatUint(stored, 10)}, nil
	case ValueFloat:
		stored, _ := value.Float()
		return canonicalValue{Kind: "float", Value: strconv.FormatFloat(stored, 'g', -1, 64)}, nil
	case ValueString:
		stored, _ := value.String()
		return canonicalValue{Kind: "string", Value: stored}, nil
	case ValueBytes:
		stored, _ := value.Bytes()
		return canonicalValue{Kind: "bytes", Value: base64.StdEncoding.EncodeToString(stored)}, nil
	case ValueEnum:
		stored, _ := value.Enum()
		return canonicalValue{Kind: "enum", Value: stored}, nil
	case ValueNode:
		stored, _ := value.Node()

		node, err := canonicalizeNode(stored)
		if err != nil {
			return canonicalValue{}, err
		}

		return canonicalValue{Kind: "node", Value: node}, nil
	case ValueList:
		stored, _ := value.List()

		items := make([]canonicalValue, 0, len(stored))
		for _, item := range stored {
			canonical, err := canonicalizeValue(item)
			if err != nil {
				return canonicalValue{}, err
			}

			items = append(items, canonical)
		}

		return canonicalValue{Kind: "list", Value: items}, nil
	default:
		return canonicalValue{}, fmt.Errorf("unsupported parser value kind %d", value.Kind())
	}
}

const canonicalSnapshotFixture = `{
  "statements": [
    {
      "kind": "FixtureStmt",
      "fields": [
        {"name":"invalid","value":{"kind":"invalid","value":null}},
        {"name":"bool","value":{"kind":"bool","value":false}},
        {"name":"int","value":{"kind":"int","value":"-7"}},
        {"name":"uint","value":{"kind":"uint","value":"9"}},
        {"name":"float","value":{"kind":"float","value":"3.25"}},
        {"name":"string","value":{"kind":"string","value":"value"}},
        {"name":"bytes","value":{"kind":"bytes","value":"AAH/"}},
        {"name":"enum","value":{"kind":"enum","value":"OPTION"}},
        {"name":"node","value":{"kind":"node","value":{"kind":"Child","fields":[]}}},
        {"name":"list","value":{"kind":"list","value":[
          {"kind":"string","value":"first"},
          {"kind":"string","value":"second"}
        ]}}
      ]
    }
  ]
}`
