package sqlguard_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	diagnosticLimit    = 2 * 1024
	modulePath         = "github.com/almostinf/postgres-sqlguard"
	internalParserPath = modulePath + "/internal/parser"
	pgQueryPath        = "github.com/pganalyze/pg_query_go/v6"
	wasmQueryPath      = "github.com/wasilibs/go-pgquery"
)

type listedPackage struct {
	ImportPath      string
	CompiledGoFiles []string
	CgoFiles        []string
	CFiles          []string
	CXXFiles        []string
	MFiles          []string
	HFiles          []string
	FFiles          []string
	SFiles          []string
	SwigFiles       []string
	SwigCXXFiles    []string
	SysoFiles       []string
	Imports         []string
	TestImports     []string
	XTestImports    []string
}

func TestParserEntryPointImportsStayInsideInternalParser(t *testing.T) {
	tests := map[string]struct {
		cgoEnabled     string
		expectedImport map[string]bool
	}{
		"cgo_enabled": {
			cgoEnabled: "1",
			expectedImport: map[string]bool{
				pgQueryPath: true,
			},
		},
		"cgo_disabled": {
			cgoEnabled: "0",
			expectedImport: map[string]bool{
				pgQueryPath:   true,
				wasmQueryPath: true,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			packages := listPackages(t, test.cgoEnabled, false)
			importers := map[string][]string{
				pgQueryPath:   nil,
				wasmQueryPath: nil,
			}

			for _, pkg := range packages {
				if pkg.ImportPath != modulePath && !strings.HasPrefix(pkg.ImportPath, modulePath+"/") {
					continue
				}

				for _, imported := range appendImports(pkg.Imports, pkg.TestImports, pkg.XTestImports) {
					if _, guarded := importers[imported]; guarded {
						importers[imported] = append(importers[imported], pkg.ImportPath)
					}
				}
			}

			for imported, gotImporters := range importers {
				sort.Strings(gotImporters)

				if test.expectedImport[imported] {
					require.NotEmpty(t, gotImporters, "%s import check must not be vacuous", imported)
				}

				for _, importer := range gotImporters {
					require.Equal(t, internalParserPath, importer,
						"%s must only be imported by the internal parser package", imported)
				}
			}
		})
	}
}

func TestNoCGOPackageGraphSelectsOnlyPortableParserSources(t *testing.T) {
	packages := listPackages(t, "0", true)

	parserPackage, found := packages[internalParserPath]
	require.True(t, found, "%s is absent from the selected package graph", internalParserPath)
	require.Contains(t, parserPackage.CompiledGoFiles, "backend_nocgo.go")
	require.NotContains(t, parserPackage.CompiledGoFiles, "backend_cgo.go")

	for importPath, pkg := range packages {
		if importPath != internalParserPath &&
			!strings.HasPrefix(importPath, pgQueryPath) &&
			!strings.HasPrefix(importPath, wasmQueryPath) {
			continue
		}

		require.Empty(t, pkg.CgoFiles, "%s selected Cgo source files", importPath)
		require.Empty(t, pkg.CFiles, "%s selected C source files", importPath)
		require.Empty(t, pkg.CXXFiles, "%s selected C++ source files", importPath)
		require.Empty(t, pkg.MFiles, "%s selected Objective-C source files", importPath)
		require.Empty(t, pkg.HFiles, "%s selected C header files", importPath)
		require.Empty(t, pkg.FFiles, "%s selected Fortran source files", importPath)
		require.Empty(t, pkg.SFiles, "%s selected assembly source files", importPath)
		require.Empty(t, pkg.SwigFiles, "%s selected SWIG source files", importPath)
		require.Empty(t, pkg.SwigCXXFiles, "%s selected SWIG C++ source files", importPath)
		require.Empty(t, pkg.SysoFiles, "%s selected system object files", importPath)
	}
}

func listPackages(t *testing.T, cgoEnabled string, includeDependencies bool) map[string]listedPackage {
	t.Helper()

	arguments := []string{"list", "-mod=readonly", "-json"}
	if includeDependencies {
		arguments = append(arguments, "-compiled", "-deps")
	}

	arguments = append(arguments, "./...")

	command := exec.CommandContext(t.Context(), "go", arguments...)
	command.Env = goListEnvironment(cgoEnabled)

	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	require.NoError(t, err, "go list failed; stderr: %q; stdout tail: %q",
		boundedTail(stderr.Bytes()), boundedTail(stdout.Bytes()))

	packages := make(map[string]listedPackage)
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))

	for {
		var pkg listedPackage

		decodeErr := decoder.Decode(&pkg)
		if errors.Is(decodeErr, io.EOF) {
			break
		}

		require.NoError(t, decodeErr,
			"decode go list JSON near byte %d; stderr: %q; stdout context: %q",
			decoder.InputOffset(), boundedTail(stderr.Bytes()),
			boundedWindow(stdout.Bytes(), decoder.InputOffset()))

		packages[pkg.ImportPath] = pkg
	}

	return packages
}

func goListEnvironment(cgoEnabled string) []string {
	environment := make([]string, 0, len(os.Environ())+3)
	for _, variable := range os.Environ() {
		name, _, _ := strings.Cut(variable, "=")
		if name == "CGO_ENABLED" || name == "CC" || name == "GOFLAGS" {
			continue
		}

		environment = append(environment, variable)
	}

	return append(environment,
		"CGO_ENABLED="+cgoEnabled,
		"CC=/definitely/missing/postgres-sqlguard-cc",
		"GOFLAGS=",
	)
}

func boundedTail(output []byte) string {
	if len(output) <= diagnosticLimit {
		return string(output)
	}

	return "...<truncated>..." + string(output[len(output)-diagnosticLimit:])
}

func boundedWindow(output []byte, offset int64) string {
	center := min(max(int(offset), 0), len(output))
	start := max(center-diagnosticLimit/2, 0)
	end := min(start+diagnosticLimit, len(output))
	start = max(end-diagnosticLimit, 0)

	context := string(output[start:end])
	if start > 0 {
		context = "...<truncated>..." + context
	}

	if end < len(output) {
		context += "...<truncated>..."
	}

	return context
}

func appendImports(importSets ...[]string) []string {
	var imports []string
	for _, importSet := range importSets {
		imports = append(imports, importSet...)
	}

	return imports
}
