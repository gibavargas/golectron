package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Report struct {
	Fixture     string          `json:"fixture"`
	OS          string          `json:"os"`
	Arch        string          `json:"arch"`
	Iterations  int             `json:"iterations"`
	MeasureRSS  bool            `json:"measure_rss"`
	StartedAt   string          `json:"started_at"`
	Results     []CommandResult `json:"results"`
	Comparisons []Comparison    `json:"comparisons,omitempty"`
}

type CommandResult struct {
	Name       string   `json:"name"`
	Command    string   `json:"command"`
	ParsedArgs []string `json:"parsed_args,omitempty"`
	Samples    []Sample `json:"samples"`
	Summary    Summary  `json:"summary"`
	Error      string   `json:"error,omitempty"`
}

type Sample struct {
	Iteration    int    `json:"iteration"`
	StartedAt    string `json:"started_at"`
	DurationMS   int64  `json:"duration_ms"`
	ExitCode     int    `json:"exit_code"`
	MaxRSSKB     int64  `json:"max_rss_kb,omitempty"`
	Error        string `json:"error,omitempty"`
	TimedOut     bool   `json:"timed_out,omitempty"`
	TimeToolUsed string `json:"time_tool_used,omitempty"`
}

type Summary struct {
	Successes           int     `json:"successes"`
	Failures            int     `json:"failures"`
	DurationMinMS       int64   `json:"duration_min_ms,omitempty"`
	DurationMedianMS    int64   `json:"duration_median_ms,omitempty"`
	DurationMeanMS      float64 `json:"duration_mean_ms,omitempty"`
	DurationMaxMS       int64   `json:"duration_max_ms,omitempty"`
	MaxRSSMinKB         int64   `json:"max_rss_min_kb,omitempty"`
	MaxRSSMedianKB      int64   `json:"max_rss_median_kb,omitempty"`
	MaxRSSMeanKB        float64 `json:"max_rss_mean_kb,omitempty"`
	MaxRSSMaxKB         int64   `json:"max_rss_max_kb,omitempty"`
	MaxRSSSamples       int     `json:"max_rss_samples,omitempty"`
	MaxRSSAvailable     bool    `json:"max_rss_available"`
	MaxRSSUnsupportedOS bool    `json:"max_rss_unsupported_os,omitempty"`
}

type Comparison struct {
	Metric                    string  `json:"metric"`
	Electron                  float64 `json:"electron"`
	ElectronGo                float64 `json:"electron_go"`
	ElectronGoOverElectron    float64 `json:"electron_go_over_electron"`
	ElectronOverElectronGo    float64 `json:"electron_over_electron_go"`
	LowerIsBetter             bool    `json:"lower_is_better"`
	ElectronGoFasterOrLighter bool    `json:"electron_go_faster_or_lighter"`
}

func main() {
	fixture := flag.String("fixture", "./compat/fixtures/hello", "fixture app directory appended to each command")
	electron := flag.String("electron", "", "official Electron command")
	electronGo := flag.String("electron-go", "go run ./cmd/electron-go", "Electron-Go command")
	timeout := flag.Duration("timeout", 30*time.Second, "per-command timeout")
	iterations := flag.Int("iterations", 1, "number of times to run each command")
	measureRSS := flag.Bool("measure-rss", false, "collect max RSS with /usr/bin/time on macOS/Linux")
	output := flag.String("output", "", "optional JSON output file; stdout is used when empty")
	flag.Parse()

	if *iterations < 1 {
		fmt.Fprintln(os.Stderr, "--iterations must be at least 1")
		os.Exit(2)
	}

	report := Report{
		Fixture:    *fixture,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Iterations: *iterations,
		MeasureRSS: *measureRSS,
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	if *electron != "" {
		report.Results = append(report.Results, runCommand("electron", *electron, *fixture, *timeout, *iterations, *measureRSS))
	}
	if *electronGo != "" {
		report.Results = append(report.Results, runCommand("electron-go", *electronGo, *fixture, *timeout, *iterations, *measureRSS))
	}
	report.Comparisons = computeComparisons(report.Results)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode results: %v\n", err)
		os.Exit(1)
	}
	if *output == "" {
		_, _ = os.Stdout.Write(buf.Bytes())
		return
	}
	if err := os.WriteFile(*output, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote benchmark report to %s\n", *output)
}

func runCommand(name, command, fixture string, timeout time.Duration, iterations int, measureRSS bool) CommandResult {
	args, err := splitCommand(command)
	result := CommandResult{Name: name, Command: command}
	if err != nil {
		result.Error = err.Error()
		result.Summary.Failures = iterations
		return result
	}
	result.ParsedArgs = args
	if len(args) == 0 {
		result.Error = "empty command"
		result.Summary.Failures = iterations
		return result
	}

	for i := 1; i <= iterations; i++ {
		result.Samples = append(result.Samples, runSample(args, fixture, timeout, i, measureRSS))
	}
	result.Summary = summarize(result.Samples, measureRSS)
	return result
}

func runSample(args []string, fixture string, timeout time.Duration, iteration int, measureRSS bool) Sample {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	sample := Sample{
		Iteration: iteration,
		StartedAt: started.UTC().Format(time.RFC3339Nano),
	}
	cmdArgs := append(append([]string{}, args[1:]...), fixture)
	name := args[0]
	if measureRSS {
		timeArgs, tool := timeCommandArgs(args[0], cmdArgs)
		if len(timeArgs) > 0 {
			name = timeArgs[0]
			cmdArgs = timeArgs[1:]
			sample.TimeToolUsed = tool
		}
	}

	cmd := exec.CommandContext(ctx, name, cmdArgs...)
	output, err := cmd.CombinedOutput()
	sample.DurationMS = time.Since(started).Milliseconds()
	if ctx.Err() == context.DeadlineExceeded {
		sample.TimedOut = true
	}
	if cmd.ProcessState != nil {
		sample.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		sample.ExitCode = -1
	}
	if sample.TimeToolUsed != "" {
		sample.MaxRSSKB = parseMaxRSSKB(string(output), runtime.GOOS)
	}
	if err != nil {
		sample.Error = strings.TrimSpace(string(output))
		if sample.Error == "" {
			sample.Error = err.Error()
		}
	}
	return sample
}

func timeCommandArgs(name string, args []string) ([]string, string) {
	switch runtime.GOOS {
	case "darwin":
		all := append([]string{"/usr/bin/time", "-l", name}, args...)
		return all, "/usr/bin/time -l"
	case "linux":
		all := append([]string{"/usr/bin/time", "-v", name}, args...)
		return all, "/usr/bin/time -v"
	default:
		return nil, ""
	}
}

func summarize(samples []Sample, measureRSS bool) Summary {
	var summary Summary
	var durations []int64
	var rss []int64
	for _, sample := range samples {
		if sample.ExitCode == 0 && sample.Error == "" && !sample.TimedOut {
			summary.Successes++
			durations = append(durations, sample.DurationMS)
			if sample.MaxRSSKB > 0 {
				rss = append(rss, sample.MaxRSSKB)
			}
			continue
		}
		summary.Failures++
	}
	if len(durations) > 0 {
		summary.DurationMinMS = minInt64(durations)
		summary.DurationMedianMS = medianInt64(durations)
		summary.DurationMeanMS = meanInt64(durations)
		summary.DurationMaxMS = maxInt64(durations)
	}
	if measureRSS && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		summary.MaxRSSUnsupportedOS = true
	}
	if len(rss) > 0 {
		summary.MaxRSSAvailable = true
		summary.MaxRSSSamples = len(rss)
		summary.MaxRSSMinKB = minInt64(rss)
		summary.MaxRSSMedianKB = medianInt64(rss)
		summary.MaxRSSMeanKB = meanInt64(rss)
		summary.MaxRSSMaxKB = maxInt64(rss)
	}
	return summary
}

func computeComparisons(results []CommandResult) []Comparison {
	var electron, electronGo *CommandResult
	for i := range results {
		switch results[i].Name {
		case "electron":
			electron = &results[i]
		case "electron-go":
			electronGo = &results[i]
		}
	}
	if electron == nil || electronGo == nil {
		return nil
	}

	var comparisons []Comparison
	add := func(metric string, e, eg float64) {
		if e <= 0 || eg <= 0 {
			return
		}
		comparisons = append(comparisons, Comparison{
			Metric:                    metric,
			Electron:                  e,
			ElectronGo:                eg,
			ElectronGoOverElectron:    eg / e,
			ElectronOverElectronGo:    e / eg,
			LowerIsBetter:             true,
			ElectronGoFasterOrLighter: eg < e,
		})
	}
	add("duration_median_ms", float64(electron.Summary.DurationMedianMS), float64(electronGo.Summary.DurationMedianMS))
	add("max_rss_median_kb", float64(electron.Summary.MaxRSSMedianKB), float64(electronGo.Summary.MaxRSSMedianKB))
	return comparisons
}

func parseMaxRSSKB(output, goos string) int64 {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch goos {
		case "darwin":
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "maximum" && fields[2] == "resident" {
				bytes, err := strconv.ParseInt(fields[0], 10, 64)
				if err == nil && bytes > 0 {
					return bytes / 1024
				}
			}
		case "linux":
			const prefix = "Maximum resident set size (kbytes):"
			if strings.HasPrefix(line, prefix) {
				kb := strings.TrimSpace(strings.TrimPrefix(line, prefix))
				value, err := strconv.ParseInt(kb, 10, 64)
				if err == nil {
					return value
				}
			}
		}
	}
	return 0
}

func splitCommand(input string) ([]string, error) {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	inToken := false

	for _, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			inToken = true
			continue
		}
		if r == '\\' {
			escaped = true
			inToken = true
			continue
		}
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			inToken = true
		case r == '\'' || r == '"':
			quote = r
			inToken = true
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			if inToken {
				args = append(args, current.String())
				current.Reset()
				inToken = false
			}
		default:
			current.WriteRune(r)
			inToken = true
		}
	}
	if escaped {
		return nil, errors.New("command ends with unfinished escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("command has unterminated %q quote", quote)
	}
	if inToken {
		args = append(args, current.String())
	}
	return args, nil
}

func medianInt64(values []int64) int64 {
	sorted := append([]int64{}, values...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func meanInt64(values []int64) float64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return float64(total) / float64(len(values))
}

func minInt64(values []int64) int64 {
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

func maxInt64(values []int64) int64 {
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}
