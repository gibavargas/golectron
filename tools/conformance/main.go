package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type options struct {
	Fixtures      []string
	Electron      string
	ElectronGo    string
	Timeout       time.Duration
	Output        string
	AllowMismatch bool
}

type fixtureFlags []string

func (f *fixtureFlags) String() string {
	return strings.Join(*f, ",")
}

func (f *fixtureFlags) Set(value string) error {
	fixtures := parseFixtures(value)
	*f = append(*f, fixtures...)
	return nil
}

type Report struct {
	OS            string          `json:"os"`
	Arch          string          `json:"arch"`
	StartedAt     string          `json:"started_at"`
	TimeoutMS     int64           `json:"timeout_ms"`
	Electron      string          `json:"electron"`
	ElectronGo    string          `json:"electron_go"`
	AllowMismatch bool            `json:"allow_mismatch"`
	Fixtures      []FixtureReport `json:"fixtures"`
	Mismatch      bool            `json:"mismatch"`
}

type FixtureReport struct {
	Fixture    string        `json:"fixture"`
	Electron   CommandResult `json:"electron"`
	ElectronGo CommandResult `json:"electron_go"`
	Comparison Comparison    `json:"comparison"`
	StartedAt  string        `json:"started_at"`
	DurationMS int64         `json:"duration_ms"`
}

type CommandResult struct {
	Command    string   `json:"command"`
	ParsedArgs []string `json:"parsed_args"`
	ExitCode   int      `json:"exit_code"`
	Stdout     string   `json:"stdout"`
	Stderr     string   `json:"stderr"`
	DurationMS int64    `json:"duration_ms"`
	TimedOut   bool     `json:"timed_out"`
	Skipped    bool     `json:"skipped,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type Comparison struct {
	Match       bool     `json:"match"`
	Skipped     bool     `json:"skipped,omitempty"`
	Differences []string `json:"differences,omitempty"`
}

func main() {
	code := runCLI(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args)
	if err != nil {
		fmt.Fprintf(stderr, "conformance: %v\n", err)
		return 2
	}

	report, err := runConformance(opts)
	if err != nil {
		fmt.Fprintf(stderr, "conformance: %v\n", err)
		return 1
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(stderr, "conformance: encode: %v\n", err)
		return 1
	}

	if opts.Output == "" {
		_, _ = stdout.Write(buf.Bytes())
	} else if err := os.WriteFile(opts.Output, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(stderr, "conformance: write output: %v\n", err)
		return 1
	}

	if report.Mismatch && !opts.AllowMismatch {
		return 1
	}
	return 0
}

func parseOptions(args []string) (options, error) {
	var fixtures fixtureFlags
	opts := options{
		Electron:   "electron",
		ElectronGo: "go run ./cmd/electron-go",
		Timeout:    30 * time.Second,
	}

	fs := flag.NewFlagSet("conformance", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&fixtures, "fixture", "fixture directory; repeat or pass comma-separated values")
	fs.StringVar(&opts.Electron, "electron", opts.Electron, "official Electron command")
	fs.StringVar(&opts.ElectronGo, "electron-go", opts.ElectronGo, "Electron-Go command")
	fs.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "per-command timeout")
	fs.StringVar(&opts.Output, "output", "", "optional JSON output file; stdout is used when empty")
	fs.BoolVar(&opts.AllowMismatch, "allow-mismatch", false, "return zero even when Electron and Electron-Go differ")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	opts.Fixtures = dedupeFixtures(fixtures)
	if len(opts.Fixtures) == 0 {
		return options{}, errors.New("at least one --fixture is required")
	}
	if strings.TrimSpace(opts.Electron) == "" {
		return options{}, errors.New("--electron must not be empty")
	}
	if strings.TrimSpace(opts.ElectronGo) == "" {
		return options{}, errors.New("--electron-go must not be empty")
	}
	if opts.Timeout <= 0 {
		return options{}, errors.New("--timeout must be greater than zero")
	}
	return opts, nil
}

func runConformance(opts options) (Report, error) {
	report := Report{
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		StartedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		TimeoutMS:     opts.Timeout.Milliseconds(),
		Electron:      opts.Electron,
		ElectronGo:    opts.ElectronGo,
		AllowMismatch: opts.AllowMismatch,
	}

	electronArgs, err := splitCommand(opts.Electron)
	if err != nil {
		return Report{}, fmt.Errorf("parse --electron: %w", err)
	}
	electronGoArgs, err := splitCommand(opts.ElectronGo)
	if err != nil {
		return Report{}, fmt.Errorf("parse --electron-go: %w", err)
	}
	if len(electronArgs) == 0 {
		return Report{}, errors.New("--electron parsed to an empty command")
	}
	if len(electronGoArgs) == 0 {
		return Report{}, errors.New("--electron-go parsed to an empty command")
	}

	for _, fixture := range opts.Fixtures {
		started := time.Now()
		fixtureReport := FixtureReport{
			Fixture:   fixture,
			StartedAt: started.UTC().Format(time.RFC3339Nano),
			Electron:  runFixtureCommand(opts.Electron, electronArgs, fixture, opts.Timeout),
			ElectronGo: runFixtureCommand(
				opts.ElectronGo,
				electronGoArgs,
				fixture,
				opts.Timeout,
			),
		}
		fixtureReport.DurationMS = time.Since(started).Milliseconds()
		fixtureReport.Comparison = compareResults(fixtureReport.Electron, fixtureReport.ElectronGo)
		if !fixtureReport.Comparison.Match {
			report.Mismatch = true
		}
		report.Fixtures = append(report.Fixtures, fixtureReport)
	}
	return report, nil
}

func runFixtureCommand(command string, args []string, fixture string, timeout time.Duration) CommandResult {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result := CommandResult{
		Command:    command,
		ParsedArgs: append([]string{}, args...),
		ExitCode:   -1,
	}
	if shouldSkipSandboxedDarwinElectron(args) {
		result.DurationMS = time.Since(started).Milliseconds()
		result.Skipped = true
		result.Error = "official Electron GUI launch is skipped under the macOS seatbelt sandbox; run outside the sandbox for e2e evidence"
		return result
	}
	cmdArgs := append(append([]string{}, args[1:]...), fixture)
	cmd := exec.CommandContext(ctx, args[0], cmdArgs...)
	cmd.Env = os.Environ()
	prepareCommandForCleanup(cmd)
	cmd.Cancel = func() error {
		terminateCommandGroup(cmd)
		return nil
	}
	cmd.WaitDelay = 2 * time.Second

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result.DurationMS = time.Since(started).Milliseconds()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result
}

func shouldSkipSandboxedDarwinElectron(args []string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	if os.Getenv("CODEX_SANDBOX") == "" {
		return false
	}
	if os.Getenv("ELECTRON_GO_ALLOW_SANDBOXED_DARWIN_CONFORMANCE") == "1" {
		return false
	}
	for _, arg := range args {
		lower := strings.ToLower(arg)
		base := strings.ToLower(filepathBase(arg))
		if lower == "electron" || base == "electron" || strings.Contains(lower, "node_modules/.bin/electron") || strings.Contains(lower, "electron.app/") {
			return true
		}
	}
	return false
}

func filepathBase(path string) string {
	path = strings.TrimRight(path, `/\`)
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}

func compareResults(electron, electronGo CommandResult) Comparison {
	if electron.Skipped || electronGo.Skipped {
		return Comparison{
			Match:   true,
			Skipped: true,
		}
	}

	var differences []string
	if electron.ExitCode != electronGo.ExitCode {
		differences = append(differences, "exit_code")
	}
	if electron.Stdout != electronGo.Stdout {
		differences = append(differences, "stdout")
	}
	if electron.Stderr != electronGo.Stderr {
		differences = append(differences, "stderr")
	}
	if electron.TimedOut != electronGo.TimedOut {
		differences = append(differences, "timed_out")
	}
	if electron.Error != electronGo.Error {
		differences = append(differences, "error")
	}
	return Comparison{
		Match:       len(differences) == 0,
		Differences: differences,
	}
}

func parseFixtures(raw string) []string {
	var fixtures []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			fixtures = append(fixtures, part)
		}
	}
	return fixtures
}

func dedupeFixtures(fixtures []string) []string {
	seen := make(map[string]bool, len(fixtures))
	out := make([]string, 0, len(fixtures))
	for _, fixture := range fixtures {
		if seen[fixture] {
			continue
		}
		seen[fixture] = true
		out = append(out, fixture)
	}
	return out
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
