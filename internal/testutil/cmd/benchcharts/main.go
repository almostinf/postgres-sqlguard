// Command benchcharts generates deterministic SVG charts from release benchmark evidence.
package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	chartWidth  = 960
	chartHeight = 480
	plotTop     = 92
	plotBottom  = 390
	plotLeft    = 92
	plotRight   = 928
	mebibyte    = 1024 * 1024
)

type dataset struct {
	SchemaVersion   int              `json:"schema_version"`
	MeasurementDate string           `json:"measurement_date"`
	Environment     environment      `json:"environment"`
	Methodology     methodology      `json:"methodology"`
	Units           units            `json:"units"`
	Cases           []complexityCase `json:"cases"`
	Footprint       footprint        `json:"footprint"`
}

type environment struct {
	Host         string `json:"host"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	LogicalCPUs  int    `json:"logical_cpus"`
	GoVersion    string `json:"go_version"`
	CGOCompiler  string `json:"cgo_compiler"`
}

type methodology struct {
	RunCount     int            `json:"run_count"`
	Benchtime    string         `json:"benchtime"`
	Commands     backendStrings `json:"commands"`
	RawResults   backendStrings `json:"raw_results"`
	RSSCommands  backendStrings `json:"rss_commands"`
	SizeCommands backendStrings `json:"size_commands"`
}

type backendStrings struct {
	CGO   string `json:"cgo"`
	NoCGO string `json:"no_cgo"`
}

type units struct {
	Latency        string `json:"latency"`
	AllocatedBytes string `json:"allocated_bytes"`
	ColdRSS        string `json:"cold_rss"`
	BinarySize     string `json:"binary_size"`
}

type complexityCase struct {
	ID    string          `json:"id"`
	Label string          `json:"label"`
	Order int             `json:"order"`
	CGO   benchmarkMedian `json:"cgo"`
	NoCGO benchmarkMedian `json:"no_cgo"`
}

type benchmarkMedian struct {
	Latency        float64 `json:"latency_ns_per_op_median"`
	AllocatedBytes float64 `json:"allocated_bytes_per_op_median"`
}

type footprint struct {
	Workload   string          `json:"workload"`
	RunCount   int             `json:"run_count"`
	RawResults backendStrings  `json:"raw_results"`
	CGO        footprintMedian `json:"cgo"`
	NoCGO      footprintMedian `json:"no_cgo"`
}

type footprintMedian struct {
	ColdRSS    float64 `json:"cold_rss_bytes_median"`
	BinarySize float64 `json:"stripped_binary_size_bytes"`
}

type chartCase struct {
	label string
	cgo   float64
	noCGO float64
}

func main() {
	dataPath := flag.String("data", "", "path to benchmark dataset JSON")
	outDir := flag.String("out", "", "directory for generated SVG files")

	flag.Parse()

	if *dataPath == "" || *outDir == "" {
		fatal(errors.New("both -data and -out are required"))
	}

	input, err := loadDataset(*dataPath)
	if err != nil {
		fatal(err)
	}

	charts, err := renderCharts(input)
	if err != nil {
		fatal(err)
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(fmt.Errorf("create output directory: %w", err))
	}

	names := make([]string, 0, len(charts))
	for name := range charts {
		names = append(names, name)
	}

	slices.Sort(names)

	for _, name := range names {
		if err := os.WriteFile(filepath.Join(*outDir, name), charts[name], 0o644); err != nil {
			fatal(fmt.Errorf("write %s: %w", name, err))
		}
	}
}

func fatal(err error) {
	_, _ = fmt.Fprintln(os.Stderr, err)

	os.Exit(1)
}

func loadDataset(path string) (dataset, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return dataset{}, fmt.Errorf("read dataset: %w", err)
	}

	var input dataset

	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		return dataset{}, fmt.Errorf("decode dataset: %w", err)
	}

	if err := input.validate(); err != nil {
		return dataset{}, err
	}

	return input, nil
}

func (input dataset) validate() error {
	if input.SchemaVersion != 1 {
		return fmt.Errorf("schema version must be 1")
	}

	if input.MeasurementDate == "" || input.Environment.Host == "" || input.Environment.OS == "" ||
		input.Environment.Architecture == "" || input.Environment.LogicalCPUs < 1 ||
		input.Environment.GoVersion == "" || input.Environment.CGOCompiler == "" {
		return fmt.Errorf("measurement date and environment metadata are required")
	}

	if input.Methodology.RunCount < 5 || input.Footprint.RunCount < 5 {
		return fmt.Errorf("benchmark and footprint run counts must be at least 5")
	}

	if input.Methodology.Benchtime == "" || !input.Methodology.Commands.complete() ||
		!input.Methodology.RSSCommands.complete() || !input.Methodology.SizeCommands.complete() {
		return fmt.Errorf("benchmark methodology commands are required for both backends")
	}

	if !input.Methodology.RawResults.complete() || !input.Footprint.RawResults.complete() {
		return fmt.Errorf("raw result locations are required for both backends")
	}

	if input.Units.Latency != "ns/op" {
		return fmt.Errorf("latency unit must be ns/op")
	}

	if input.Units.AllocatedBytes != "B/op" {
		return fmt.Errorf("allocated-bytes unit must be B/op")
	}

	if input.Units.ColdRSS != "bytes" || input.Units.BinarySize != "bytes" {
		return fmt.Errorf("footprint units must be bytes")
	}

	if len(input.Cases) == 0 {
		return fmt.Errorf("at least one SQL-complexity case is required")
	}

	orders := make(map[int]struct{}, len(input.Cases))
	ids := make(map[string]struct{}, len(input.Cases))

	for _, benchmarkCase := range input.Cases {
		if benchmarkCase.ID == "" || benchmarkCase.Label == "" || benchmarkCase.Order < 1 {
			return fmt.Errorf("case id, label, and positive order are required")
		}

		if _, exists := orders[benchmarkCase.Order]; exists {
			return fmt.Errorf("case order values must be unique")
		}

		if _, exists := ids[benchmarkCase.ID]; exists {
			return fmt.Errorf("case ids must be unique")
		}

		orders[benchmarkCase.Order] = struct{}{}
		ids[benchmarkCase.ID] = struct{}{}

		if !positiveFinite(benchmarkCase.CGO.Latency) || !positiveFinite(benchmarkCase.NoCGO.Latency) ||
			!positiveFinite(benchmarkCase.CGO.AllocatedBytes) || !positiveFinite(benchmarkCase.NoCGO.AllocatedBytes) {
			return fmt.Errorf("case %q measurements must be positive finite values", benchmarkCase.ID)
		}
	}

	if input.Footprint.Workload == "" || !positiveFinite(input.Footprint.CGO.ColdRSS) ||
		!positiveFinite(input.Footprint.NoCGO.ColdRSS) || !positiveFinite(input.Footprint.CGO.BinarySize) ||
		!positiveFinite(input.Footprint.NoCGO.BinarySize) {
		return fmt.Errorf("footprint workload and positive finite measurements are required")
	}

	return nil
}

func (values backendStrings) complete() bool {
	return values.CGO != "" && values.NoCGO != ""
}

func positiveFinite(value float64) bool {
	return value > 0 && !math.IsInf(value, 0) && !math.IsNaN(value)
}

func renderCharts(input dataset) (map[string][]byte, error) {
	if err := input.validate(); err != nil {
		return nil, err
	}

	cases := slices.Clone(input.Cases)
	slices.SortFunc(cases, func(left, right complexityCase) int {
		return left.Order - right.Order
	})

	latency := make([]chartCase, 0, len(cases))
	allocations := make([]chartCase, 0, len(cases))

	for _, benchmarkCase := range cases {
		latency = append(latency, chartCase{benchmarkCase.Label, benchmarkCase.CGO.Latency, benchmarkCase.NoCGO.Latency})
		allocations = append(allocations, chartCase{benchmarkCase.Label, benchmarkCase.CGO.AllocatedBytes, benchmarkCase.NoCGO.AllocatedBytes})
	}

	hostLine := input.MeasurementDate + " · " + input.Environment.Host + " · " + input.Environment.GoVersion

	return map[string][]byte{
		"latency.svg":     renderGroupedBars("Direct validation latency", "Median execution time; lower is better", input.Units.Latency, hostLine, latency, 1),
		"allocations.svg": renderGroupedBars("Allocated memory", "Median heap bytes per validation; lower is better", input.Units.AllocatedBytes, hostLine, allocations, 1),
		"cold-rss.svg":    renderGroupedBars("Cold-process maximum RSS", "Median of fixed size-probe executions; lower is better", "MiB", hostLine, []chartCase{{input.Footprint.Workload, input.Footprint.CGO.ColdRSS / mebibyte, input.Footprint.NoCGO.ColdRSS / mebibyte}}, 1),
		"binary-size.svg": renderGroupedBars("Stripped linked binary size", "Equivalent size-probe builds; lower is better", "MiB", hostLine, []chartCase{{input.Footprint.Workload, input.Footprint.CGO.BinarySize / mebibyte, input.Footprint.NoCGO.BinarySize / mebibyte}}, 2),
	}, nil
}

func renderGroupedBars(title, description, unit, metadata string, cases []chartCase, precision int) []byte {
	maximum := 0.0
	for _, benchmarkCase := range cases {
		maximum = max(maximum, benchmarkCase.cgo, benchmarkCase.noCGO)
	}

	maximum *= 1.1

	var output strings.Builder
	output.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="960" height="480" viewBox="0 0 960 480" role="img" aria-labelledby="title desc">`)
	output.WriteString(`<title id="title">` + escape(title) + `</title>`)
	output.WriteString(`<desc id="desc">` + escape(description+". CGO is blue and no-CGO is amber. Values are in "+unit+".") + `</desc>`)
	output.WriteString(`<rect width="960" height="480" rx="16" fill="#0d1117"/>`)
	output.WriteString(`<text x="40" y="42" fill="#f0f6fc" font-family="system-ui,sans-serif" font-size="24" font-weight="700">` + escape(title) + `</text>`)
	output.WriteString(`<text x="40" y="68" fill="#8b949e" font-family="system-ui,sans-serif" font-size="13">` + escape(metadata+" · "+unit) + `</text>`)
	output.WriteString(`<line x1="92" y1="390" x2="928" y2="390" stroke="#30363d"/>`)
	output.WriteString(`<rect x="730" y="28" width="12" height="12" rx="2" fill="#58a6ff"/><text x="748" y="39" fill="#c9d1d9" font-family="system-ui,sans-serif" font-size="13">CGO</text>`)
	output.WriteString(`<rect x="805" y="28" width="12" height="12" rx="2" fill="#d29922"/><text x="823" y="39" fill="#c9d1d9" font-family="system-ui,sans-serif" font-size="13">no-CGO</text>`)

	groupWidth := float64(plotRight-plotLeft) / float64(len(cases))
	barWidth := math.Min(70, groupWidth*0.28)

	for index, benchmarkCase := range cases {
		center := float64(plotLeft) + groupWidth*(float64(index)+0.5)
		writeBar(&output, center-barWidth-3, barWidth, benchmarkCase.cgo, maximum, "#58a6ff", unit, precision)
		writeBar(&output, center+3, barWidth, benchmarkCase.noCGO, maximum, "#d29922", unit, precision)
		_, _ = fmt.Fprintf(&output, `<text x="%.1f" y="420" text-anchor="middle" fill="#c9d1d9" font-family="system-ui,sans-serif" font-size="13">%s</text>`, center, escape(benchmarkCase.label))
	}

	output.WriteString(`</svg>`)

	return []byte(output.String())
}

func writeBar(output *strings.Builder, x, width, value, maximum float64, color, unit string, precision int) {
	height := value / maximum * float64(plotBottom-plotTop)
	y := float64(plotBottom) - height

	_, _ = fmt.Fprintf(output, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="%s"/>`, x, y, width, height, color)
	_, _ = fmt.Fprintf(output, `<text x="%.1f" y="%.1f" text-anchor="middle" fill="#f0f6fc" font-family="system-ui,sans-serif" font-size="12">%s</text>`, x+width/2, y-8, formatValue(value, unit, precision))
}

func formatValue(value float64, unit string, precision int) string {
	if unit == "ns/op" || unit == "B/op" {
		return fmt.Sprintf("%.0f", value)
	}

	return fmt.Sprintf("%.*f", precision, value)
}

func escape(value string) string {
	var output strings.Builder

	_ = xml.EscapeText(&output, []byte(value))

	return output.String()
}
