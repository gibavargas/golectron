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
	RunOrder    string          `json:"run_order,omitempty"`
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
	Iteration              int              `json:"iteration"`
	Sequence               int              `json:"sequence,omitempty"`
	Pair                   int              `json:"pair,omitempty"`
	StartedAt              string           `json:"started_at"`
	DurationMS             int64            `json:"duration_ms"`
	ExitCode               int              `json:"exit_code"`
	MaxRSSKB               int64            `json:"max_rss_kb,omitempty"`
	ProcessTreeRSSPeakKB   int64            `json:"process_tree_rss_peak_kb,omitempty"`
	StartupTraceMS         map[string]int64 `json:"startup_trace_ms,omitempty"`
	BenchmarkTraceMS       map[string]int64 `json:"benchmark_trace_ms,omitempty"`
	Error                  string           `json:"error,omitempty"`
	TimedOut               bool             `json:"timed_out,omitempty"`
	TimeToolUsed           string           `json:"time_tool_used,omitempty"`
	ProcessTreeRSSProbe    string           `json:"process_tree_rss_probe,omitempty"`
	ProcessTreeRSSProbeErr string           `json:"process_tree_rss_probe_error,omitempty"`
}

type Summary struct {
	Successes                         int              `json:"successes"`
	Failures                          int              `json:"failures"`
	DurationMinMS                     int64            `json:"duration_min_ms,omitempty"`
	DurationMedianMS                  int64            `json:"duration_median_ms,omitempty"`
	DurationMeanMS                    float64          `json:"duration_mean_ms,omitempty"`
	DurationMaxMS                     int64            `json:"duration_max_ms,omitempty"`
	StartupTraceMedianMS              map[string]int64 `json:"startup_trace_median_ms,omitempty"`
	BenchmarkTraceMedianMS            map[string]int64 `json:"benchmark_trace_median_ms,omitempty"`
	MaxRSSMinKB                       int64            `json:"max_rss_min_kb,omitempty"`
	MaxRSSMedianKB                    int64            `json:"max_rss_median_kb,omitempty"`
	MaxRSSMeanKB                      float64          `json:"max_rss_mean_kb,omitempty"`
	MaxRSSMaxKB                       int64            `json:"max_rss_max_kb,omitempty"`
	MaxRSSSamples                     int              `json:"max_rss_samples,omitempty"`
	MaxRSSAvailable                   bool             `json:"max_rss_available"`
	MaxRSSUnsupportedOS               bool             `json:"max_rss_unsupported_os,omitempty"`
	ProcessTreeRSSPeakMinKB           int64            `json:"process_tree_rss_peak_min_kb,omitempty"`
	ProcessTreeRSSPeakMedianKB        int64            `json:"process_tree_rss_peak_median_kb,omitempty"`
	ProcessTreeRSSPeakMeanKB          float64          `json:"process_tree_rss_peak_mean_kb,omitempty"`
	ProcessTreeRSSPeakMaxKB           int64            `json:"process_tree_rss_peak_max_kb,omitempty"`
	ProcessTreeRSSPeakSamples         int              `json:"process_tree_rss_peak_samples,omitempty"`
	ProcessTreeRSSPeakAvailable       bool             `json:"process_tree_rss_peak_available"`
	ProcessTreeRSSPeakUnsupportedOS   bool             `json:"process_tree_rss_peak_unsupported_os,omitempty"`
	ProcessTreeRSSPeakPollingInterval string           `json:"process_tree_rss_peak_polling_interval,omitempty"`
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
	measureProcessTreeRSS := flag.Bool("measure-process-tree-rss", false, "sample peak process-tree RSS on Linux")
	output := flag.String("output", "", "optional JSON output file; stdout is used when empty")
	requireFaster := flag.String("require-faster", "", "comma-separated lower-is-better comparison metrics that Electron-Go must beat")
	requireRatio := flag.String("require-ratio", "", "comma-separated metric=max-ratio gates using electron_go_over_electron")
	runOrder := flag.String("run-order", "sequential", "sample order: sequential or alternating")
	flag.Parse()

	if *iterations < 1 {
		fmt.Fprintln(os.Stderr, "--iterations must be at least 1")
		os.Exit(2)
	}
	if *runOrder != "sequential" && *runOrder != "alternating" {
		fmt.Fprintln(os.Stderr, "--run-order must be sequential or alternating")
		os.Exit(2)
	}

	report := Report{
		Fixture:    *fixture,
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Iterations: *iterations,
		MeasureRSS: *measureRSS || *measureProcessTreeRSS,
		StartedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	}
	if *runOrder == "alternating" {
		report.RunOrder = *runOrder
	}
	if *runOrder == "alternating" {
		report.Results = runAlternating([]commandSpec{
			{Name: "electron", Command: *electron},
			{Name: "electron-go", Command: *electronGo},
		}, *fixture, *timeout, *iterations, *measureRSS, *measureProcessTreeRSS)
	} else {
		if *electron != "" {
			report.Results = append(report.Results, runCommand("electron", *electron, *fixture, *timeout, *iterations, *measureRSS, *measureProcessTreeRSS))
		}
		if *electronGo != "" {
			report.Results = append(report.Results, runCommand("electron-go", *electronGo, *fixture, *timeout, *iterations, *measureRSS, *measureProcessTreeRSS))
		}
	}
	report.Comparisons = computeComparisons(report.Results)
	requireErr := validateRequiredFaster(report, requiredMetrics(*requireFaster))
	ratios, ratioParseErr := requiredRatios(*requireRatio)
	if ratioErr := validateRequiredRatios(report, ratios, ratioParseErr); requireErr == nil {
		requireErr = ratioErr
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "encode results: %v\n", err)
		os.Exit(1)
	}
	if *output == "" {
		_, _ = os.Stdout.Write(buf.Bytes())
		if requireErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", requireErr)
			os.Exit(1)
		}
		return
	}
	if err := os.WriteFile(*output, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote benchmark report to %s\n", *output)
	if requireErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", requireErr)
		os.Exit(1)
	}
}

type commandSpec struct {
	Name    string
	Command string
}

type scheduledSample struct {
	Name      string
	Iteration int
	Sequence  int
	Pair      int
}

func runAlternating(specs []commandSpec, fixture string, timeout time.Duration, iterations int, measureRSS, measureProcessTreeRSS bool) []CommandResult {
	results := make([]CommandResult, 0, len(specs))
	argsByName := map[string][]string{}
	for _, spec := range specs {
		if spec.Command == "" {
			continue
		}
		args, result := prepareCommand(spec.Name, spec.Command, iterations)
		results = append(results, result)
		if len(args) > 0 {
			argsByName[spec.Name] = args
		}
	}

	indexByName := map[string]int{}
	for i := range results {
		indexByName[results[i].Name] = i
	}
	for _, item := range alternatingSchedule(specs, argsByName, iterations) {
		sample := runSample(argsByName[item.Name], fixture, timeout, item.Iteration, measureRSS, measureProcessTreeRSS)
		sample.Sequence = item.Sequence
		sample.Pair = item.Pair
		result := &results[indexByName[item.Name]]
		result.Samples = append(result.Samples, sample)
	}
	for i := range results {
		if results[i].Error == "" {
			results[i].Summary = summarize(results[i].Samples, measureRSS, measureProcessTreeRSS)
		}
	}
	return results
}

func prepareCommand(name, command string, iterations int) ([]string, CommandResult) {
	args, err := splitCommand(command)
	result := CommandResult{Name: name, Command: command}
	if err != nil {
		result.Error = err.Error()
		result.Summary.Failures = iterations
		return nil, result
	}
	result.ParsedArgs = args
	if len(args) == 0 {
		result.Error = "empty command"
		result.Summary.Failures = iterations
		return nil, result
	}
	return args, result
}

func alternatingSchedule(specs []commandSpec, argsByName map[string][]string, iterations int) []scheduledSample {
	var enabled []string
	for _, spec := range specs {
		if len(argsByName[spec.Name]) > 0 {
			enabled = append(enabled, spec.Name)
		}
	}
	var schedule []scheduledSample
	sequence := 1
	for iteration := 1; iteration <= iterations; iteration++ {
		names := append([]string{}, enabled...)
		if len(names) > 1 && iteration%2 == 0 {
			for left, right := 0, len(names)-1; left < right; left, right = left+1, right-1 {
				names[left], names[right] = names[right], names[left]
			}
		}
		for _, name := range names {
			schedule = append(schedule, scheduledSample{
				Name:      name,
				Iteration: iteration,
				Sequence:  sequence,
				Pair:      iteration,
			})
			sequence++
		}
	}
	return schedule
}

func runCommand(name, command, fixture string, timeout time.Duration, iterations int, measureRSS, measureProcessTreeRSS bool) CommandResult {
	args, result := prepareCommand(name, command, iterations)
	if result.Error != "" {
		return result
	}

	for i := 1; i <= iterations; i++ {
		result.Samples = append(result.Samples, runSample(args, fixture, timeout, i, measureRSS, measureProcessTreeRSS))
	}
	result.Summary = summarize(result.Samples, measureRSS, measureProcessTreeRSS)
	return result
}

func runSample(args []string, fixture string, timeout time.Duration, iteration int, measureRSS, measureProcessTreeRSS bool) Sample {
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
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		sample.DurationMS = time.Since(started).Milliseconds()
		sample.ExitCode = -1
		sample.Error = err.Error()
		return sample
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	if measureProcessTreeRSS {
		sample.ProcessTreeRSSProbe = "linux-procfs"
		if runtime.GOOS == "linux" {
			var rssErr error
			sample.ProcessTreeRSSPeakKB, err, rssErr = sampleProcessTreeRSSPeakKB(ctx, cmd.Process.Pid, 10*time.Millisecond, done)
			if rssErr != nil {
				sample.ProcessTreeRSSProbeErr = rssErr.Error()
			}
		} else {
			sample.ProcessTreeRSSProbe = ""
			err = <-done
		}
	} else {
		err = <-done
	}
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
		sample.MaxRSSKB = parseMaxRSSKB(output.String(), runtime.GOOS)
	}
	sample.StartupTraceMS = parseStartupTraceMS(output.String())
	sample.BenchmarkTraceMS = parseTraceMS(output.String(), "benchmark-trace:")
	if err != nil {
		sample.Error = strings.TrimSpace(output.String())
		if sample.Error == "" {
			sample.Error = err.Error()
		}
	}
	return sample
}

func parseStartupTraceMS(output string) map[string]int64 {
	return parseTraceMS(output, "electron-go-startup-trace:")
}

func parseTraceMS(output, marker string) map[string]int64 {
	for _, line := range strings.Split(output, "\n") {
		_, payload, ok := strings.Cut(line, marker)
		if !ok {
			continue
		}
		var trace map[string]int64
		if err := json.Unmarshal([]byte(strings.TrimSpace(payload)), &trace); err == nil && len(trace) > 0 {
			return trace
		}
	}
	return nil
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

func summarize(samples []Sample, measureRSS, measureProcessTreeRSS bool) Summary {
	var summary Summary
	var durations []int64
	var startupTraces []map[string]int64
	var benchmarkTraces []map[string]int64
	var rss []int64
	var processTreeRSS []int64
	for _, sample := range samples {
		if sample.ExitCode == 0 && sample.Error == "" && !sample.TimedOut {
			summary.Successes++
			durations = append(durations, sample.DurationMS)
			if len(sample.StartupTraceMS) > 0 {
				startupTraces = append(startupTraces, sample.StartupTraceMS)
			}
			if len(sample.BenchmarkTraceMS) > 0 {
				benchmarkTraces = append(benchmarkTraces, sample.BenchmarkTraceMS)
			}
			if sample.MaxRSSKB > 0 {
				rss = append(rss, sample.MaxRSSKB)
			}
			if sample.ProcessTreeRSSPeakKB > 0 {
				processTreeRSS = append(processTreeRSS, sample.ProcessTreeRSSPeakKB)
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
	summary.StartupTraceMedianMS = medianTraceMS(startupTraces)
	summary.BenchmarkTraceMedianMS = medianTraceMS(benchmarkTraces)
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
	if measureProcessTreeRSS && runtime.GOOS != "linux" {
		summary.ProcessTreeRSSPeakUnsupportedOS = true
	}
	if len(processTreeRSS) > 0 {
		summary.ProcessTreeRSSPeakAvailable = true
		summary.ProcessTreeRSSPeakSamples = len(processTreeRSS)
		summary.ProcessTreeRSSPeakMinKB = minInt64(processTreeRSS)
		summary.ProcessTreeRSSPeakMedianKB = medianInt64(processTreeRSS)
		summary.ProcessTreeRSSPeakMeanKB = meanInt64(processTreeRSS)
		summary.ProcessTreeRSSPeakMaxKB = maxInt64(processTreeRSS)
		summary.ProcessTreeRSSPeakPollingInterval = (10 * time.Millisecond).String()
	}
	return summary
}

func medianTraceMS(traces []map[string]int64) map[string]int64 {
	if len(traces) == 0 {
		return nil
	}
	valuesByKey := map[string][]int64{}
	for _, trace := range traces {
		for key, value := range trace {
			valuesByKey[key] = append(valuesByKey[key], value)
		}
	}
	medians := make(map[string]int64, len(valuesByKey))
	for key, values := range valuesByKey {
		medians[key] = medianInt64(values)
	}
	return medians
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
	add("process_tree_rss_peak_median_kb", float64(electron.Summary.ProcessTreeRSSPeakMedianKB), float64(electronGo.Summary.ProcessTreeRSSPeakMedianKB))
	return comparisons
}

func sampleProcessTreeRSSPeakKB(ctx context.Context, rootPID int, interval time.Duration, done <-chan error) (int64, error, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var peak int64
	var lastErr error
	for {
		rss, err := processTreeRSSKB(rootPID)
		if err == nil && rss > peak {
			peak = rss
		} else if err != nil {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return peak, ctx.Err(), lastErr
		case waitErr := <-done:
			if peak > 0 {
				return peak, waitErr, nil
			}
			return peak, waitErr, lastErr
		case <-ticker.C:
		}
	}
}

func processTreeRSSKB(rootPID int) (int64, error) {
	pids := processTreePIDs(rootPID)
	if len(pids) == 0 {
		return 0, fmt.Errorf("process %d not found", rootPID)
	}
	var total int64
	for _, pid := range pids {
		rss, err := processRSSKB(pid)
		if err == nil {
			total += rss
		}
	}
	return total, nil
}

func processTreePIDs(rootPID int) []int {
	var out []int
	seen := map[int]struct{}{}
	queue := []int{rootPID}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if _, ok := seen[pid]; ok {
			continue
		}
		if !processExists(pid) {
			continue
		}
		seen[pid] = struct{}{}
		out = append(out, pid)
		for _, child := range processChildren(pid) {
			if _, ok := seen[child]; !ok {
				queue = append(queue, child)
			}
		}
	}
	return out
}

func processChildren(pid int) []int {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", pid, pid))
	if err != nil {
		return nil
	}
	fields := strings.Fields(string(data))
	out := make([]int, 0, len(fields))
	for _, field := range fields {
		child, err := strconv.Atoi(field)
		if err == nil {
			out = append(out, child)
		}
	}
	return out
}

func processRSSKB(pid int) (int64, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "VmRSS:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("invalid VmRSS line for pid %d", pid)
		}
		return strconv.ParseInt(fields[1], 10, 64)
	}
	return 0, nil
}

func processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

func requiredMetrics(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}
	var out []string
	for _, metric := range strings.Split(input, ",") {
		metric = strings.TrimSpace(metric)
		if metric != "" {
			out = append(out, metric)
		}
	}
	return out
}

type requiredRatio struct {
	Metric   string
	MaxRatio float64
}

func requiredRatios(input string) ([]requiredRatio, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil
	}
	var out []requiredRatio
	for _, raw := range strings.Split(input, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		metric, value, ok := strings.Cut(raw, "=")
		if !ok {
			return nil, fmt.Errorf("invalid required ratio %q; want metric=max-ratio", raw)
		}
		metric = strings.TrimSpace(metric)
		if metric == "" {
			return nil, fmt.Errorf("invalid required ratio %q; metric is required", raw)
		}
		maxRatio, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		if err != nil || maxRatio <= 0 {
			return nil, fmt.Errorf("invalid required ratio %q; max-ratio must be positive", raw)
		}
		out = append(out, requiredRatio{Metric: metric, MaxRatio: maxRatio})
	}
	return out, nil
}

func validateRequiredFaster(report Report, metrics []string) error {
	if len(metrics) == 0 {
		return nil
	}
	for _, result := range report.Results {
		if result.Summary.Failures > 0 || result.Summary.Successes != report.Iterations {
			return fmt.Errorf("%s benchmark had %d successes and %d failures; refusing speed claim", result.Name, result.Summary.Successes, result.Summary.Failures)
		}
	}
	for _, metric := range metrics {
		var found *Comparison
		for i := range report.Comparisons {
			if report.Comparisons[i].Metric == metric {
				found = &report.Comparisons[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("required benchmark metric %q was not produced", metric)
		}
		if !found.ElectronGoFasterOrLighter {
			return fmt.Errorf("Electron-Go did not beat Electron for %s: electron=%.2f electron-go=%.2f ratio=%.4f", metric, found.Electron, found.ElectronGo, found.ElectronGoOverElectron)
		}
	}
	return nil
}

func validateRequiredRatios(report Report, ratios []requiredRatio, parseErr error) error {
	if parseErr != nil {
		return parseErr
	}
	if len(ratios) == 0 {
		return nil
	}
	for _, result := range report.Results {
		if result.Summary.Failures > 0 || result.Summary.Successes != report.Iterations {
			return fmt.Errorf("%s benchmark had %d successes and %d failures; refusing ratio claim", result.Name, result.Summary.Successes, result.Summary.Failures)
		}
	}
	for _, ratio := range ratios {
		var found *Comparison
		for i := range report.Comparisons {
			if report.Comparisons[i].Metric == ratio.Metric {
				found = &report.Comparisons[i]
				break
			}
		}
		if found == nil {
			return fmt.Errorf("required benchmark ratio metric %q was not produced", ratio.Metric)
		}
		if found.ElectronGoOverElectron > ratio.MaxRatio {
			return fmt.Errorf("Electron-Go exceeded ratio for %s: electron=%.2f electron-go=%.2f ratio=%.4f max=%.4f", ratio.Metric, found.Electron, found.ElectronGo, found.ElectronGoOverElectron, ratio.MaxRatio)
		}
	}
	return nil
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
