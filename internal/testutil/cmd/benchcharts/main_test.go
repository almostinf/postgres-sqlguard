package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDatasetValidate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		mutate  func(*dataset)
		wantErr string
	}{
		"accepts_complete_dataset": {
			mutate: func(*dataset) {},
		},
		"rejects_missing_raw_result": {
			mutate: func(input *dataset) {
				input.Methodology.RawResults.CGO = ""
			},
			wantErr: "raw result",
		},
		"rejects_wrong_run_count": {
			mutate: func(input *dataset) {
				input.Methodology.RunCount = 4
			},
			wantErr: "at least 5",
		},
		"rejects_duplicate_case_order": {
			mutate: func(input *dataset) {
				input.Cases[1].Order = input.Cases[0].Order
			},
			wantErr: "case order",
		},
		"rejects_wrong_units": {
			mutate: func(input *dataset) {
				input.Units.Latency = "milliseconds"
			},
			wantErr: "latency unit",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			input := validDataset()
			test.mutate(&input)

			err := input.validate()
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestRenderChartsIsStableAccessibleAndEscaped(t *testing.T) {
	t.Parallel()

	input := validDataset()
	input.Cases[0].Label = `Simple <select> & "quote"`
	input.Cases[0].Order = 2
	input.Cases[1].Order = 1

	first, err := renderCharts(input)
	require.NoError(t, err)
	second, err := renderCharts(input)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Len(t, first, 4)

	latency := first["latency.svg"]
	require.Contains(t, string(latency), `role="img"`)
	require.Contains(t, string(latency), "<title")
	require.Contains(t, string(latency), "<desc")
	require.Contains(t, string(latency), "ns/op")
	require.Contains(t, string(latency), `Simple &lt;select&gt; &amp; &#34;quote&#34;`)
	require.Less(t, bytes.Index(latency, []byte("Medium")), bytes.Index(latency, []byte("Simple")))

	require.Contains(t, string(first["allocations.svg"]), "B/op")
	require.Contains(t, string(first["cold-rss.svg"]), "MiB")
	require.Contains(t, string(first["binary-size.svg"]), "MiB")
}

func validDataset() dataset {
	return dataset{
		SchemaVersion:   1,
		MeasurementDate: "2026-09-14",
		Environment: environment{
			Host:         "Apple M4 Pro",
			OS:           "darwin",
			Architecture: "arm64",
			LogicalCPUs:  14,
			GoVersion:    "go1.26.0",
			CGOCompiler:  "Apple Clang 17.0.0",
		},
		Methodology: methodology{
			RunCount:  5,
			Benchtime: "1s",
			Commands: backendStrings{
				CGO:   "CGO_ENABLED=1 CC=cc go test . -run '^$' -bench '^BenchmarkEngineValidateByComplexity$' -benchmem -benchtime=1s -count=5",
				NoCGO: "CGO_ENABLED=0 CC=/definitely/missing/postgres-sqlguard-cc go test . -run '^$' -bench '^BenchmarkEngineValidateByComplexity$' -benchmem -benchtime=1s -count=5",
			},
			RawResults:   backendStrings{CGO: "raw/cgo.txt", NoCGO: "raw/no-cgo.txt"},
			RSSCommands:  backendStrings{CGO: "/usr/bin/time -l ./sqlguard-cgo", NoCGO: "/usr/bin/time -l ./sqlguard-no-cgo"},
			SizeCommands: backendStrings{CGO: "stat -f '%z' ./sqlguard-cgo", NoCGO: "stat -f '%z' ./sqlguard-no-cgo"},
		},
		Units: units{Latency: "ns/op", AllocatedBytes: "B/op", ColdRSS: "bytes", BinarySize: "bytes"},
		Cases: []complexityCase{
			{ID: "simple", Label: "Simple", Order: 1, CGO: benchmarkMedian{Latency: 10, AllocatedBytes: 20}, NoCGO: benchmarkMedian{Latency: 30, AllocatedBytes: 40}},
			{ID: "medium", Label: "Medium", Order: 2, CGO: benchmarkMedian{Latency: 50, AllocatedBytes: 60}, NoCGO: benchmarkMedian{Latency: 70, AllocatedBytes: 80}},
		},
		Footprint: footprint{
			Workload:   "size probe",
			RunCount:   5,
			RawResults: backendStrings{CGO: "raw/cgo-footprint.txt", NoCGO: "raw/no-cgo-footprint.txt"},
			CGO:        footprintMedian{ColdRSS: 1024, BinarySize: 2048},
			NoCGO:      footprintMedian{ColdRSS: 4096, BinarySize: 8192},
		},
	}
}
