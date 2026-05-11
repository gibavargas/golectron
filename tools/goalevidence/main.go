package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultMetric = "duration_median_ms"
const defaultTargetRatio = 0.5

type benchmarkReport struct {
	Comparisons []benchmarkComparison `json:"comparisons"`
}

type benchmarkComparison struct {
	Metric                 string  `json:"metric"`
	ElectronGoOverElectron float64 `json:"electron_go_over_electron"`
}

type goalEvidence struct {
	Performance goalPerformanceEvidence `json:"performance"`
}

type goalPerformanceEvidence struct {
	TargetElectronGoOverElectron float64 `json:"target_electron_go_over_electron"`
	LatestElectronGoOverElectron float64 `json:"latest_electron_go_over_electron"`
	LatestRunURL                 string  `json:"latest_run_url"`
	LatestCommit                 string  `json:"latest_commit"`
	Metric                       string  `json:"metric"`
}

func main() {
	code := run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("goalevidence", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	benchmarkPath := fs.String("benchmark", "", "benchmark JSON report to convert")
	outputPath := fs.String("output", "", "optional goal evidence JSON output path; stdout is used when empty")
	runURL := fs.String("run-url", "", "published benchmark run URL")
	commit := fs.String("commit", "", "commit SHA measured by the benchmark")
	metric := fs.String("metric", defaultMetric, "comparison metric to extract")
	targetRatio := fs.Float64("target-ratio", defaultTargetRatio, "maximum acceptable electron_go_over_electron ratio")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintf(stderr, "goalevidence: %v\n", err)
		return 2
	}
	evidence, err := buildEvidence(*benchmarkPath, *runURL, *commit, *metric, *targetRatio)
	if err != nil {
		fmt.Fprintf(stderr, "goalevidence: %v\n", err)
		return 1
	}
	payload, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "goalevidence: encode: %v\n", err)
		return 1
	}
	payload = append(payload, '\n')
	if *outputPath == "" {
		_, _ = stdout.Write(payload)
		return 0
	}
	if err := os.WriteFile(*outputPath, payload, 0o644); err != nil {
		fmt.Fprintf(stderr, "goalevidence: write output: %v\n", err)
		return 1
	}
	return 0
}

func buildEvidence(benchmarkPath, runURL, commit, metric string, targetRatio float64) (goalEvidence, error) {
	benchmarkPath = strings.TrimSpace(benchmarkPath)
	runURL = strings.TrimSpace(runURL)
	commit = strings.TrimSpace(commit)
	metric = strings.TrimSpace(metric)
	if benchmarkPath == "" {
		return goalEvidence{}, errors.New("--benchmark is required")
	}
	if runURL == "" {
		return goalEvidence{}, errors.New("--run-url is required")
	}
	if commit == "" {
		return goalEvidence{}, errors.New("--commit is required")
	}
	if metric == "" {
		return goalEvidence{}, errors.New("--metric is required")
	}
	if targetRatio <= 0 {
		return goalEvidence{}, errors.New("--target-ratio must be greater than zero")
	}

	raw, err := os.ReadFile(benchmarkPath)
	if err != nil {
		return goalEvidence{}, err
	}
	var report benchmarkReport
	if err := json.Unmarshal(raw, &report); err != nil {
		return goalEvidence{}, err
	}
	for _, comparison := range report.Comparisons {
		if comparison.Metric == metric {
			if comparison.ElectronGoOverElectron <= 0 {
				return goalEvidence{}, fmt.Errorf("metric %q has invalid electron_go_over_electron ratio", metric)
			}
			return goalEvidence{
				Performance: goalPerformanceEvidence{
					TargetElectronGoOverElectron: targetRatio,
					LatestElectronGoOverElectron: comparison.ElectronGoOverElectron,
					LatestRunURL:                 runURL,
					LatestCommit:                 commit,
					Metric:                       metric,
				},
			}, nil
		}
	}
	return goalEvidence{}, fmt.Errorf("metric %q not found in benchmark comparisons", metric)
}
