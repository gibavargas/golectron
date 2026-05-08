package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type Result struct {
	Name       string `json:"name"`
	Command    string `json:"command"`
	Fixture    string `json:"fixture"`
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	StartedAt  string `json:"started_at"`
	DurationMS int64  `json:"duration_ms"`
	ExitCode   int    `json:"exit_code"`
	Error      string `json:"error,omitempty"`
}

func main() {
	fixture := flag.String("fixture", "./compat/fixtures/hello", "fixture app directory")
	electron := flag.String("electron", "", "official Electron command")
	electronGo := flag.String("electron-go", "go run ./cmd/electron-go", "Electron-Go command")
	timeout := flag.Duration("timeout", 30*time.Second, "per-command timeout")
	flag.Parse()

	var results []Result
	if *electron != "" {
		results = append(results, run("electron", *electron, *fixture, *timeout))
	}
	results = append(results, run("electron-go", *electronGo, *fixture, *timeout))

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(results); err != nil {
		fmt.Fprintf(os.Stderr, "encode results: %v\n", err)
		os.Exit(1)
	}
}

func run(name, command, fixture string, timeout time.Duration) Result {
	started := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := strings.Fields(command)
	result := Result{
		Name:      name,
		Command:   command,
		Fixture:   fixture,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		StartedAt: started.UTC().Format(time.RFC3339Nano),
	}
	if len(args) == 0 {
		result.ExitCode = -1
		result.Error = "empty command"
		return result
	}

	cmdArgs := append(args[1:], fixture)
	cmd := exec.CommandContext(ctx, args[0], cmdArgs...)
	output, err := cmd.CombinedOutput()
	result.DurationMS = time.Since(started).Milliseconds()
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	} else {
		result.ExitCode = -1
	}
	if err != nil {
		result.Error = strings.TrimSpace(string(output))
		if result.Error == "" {
			result.Error = err.Error()
		}
	}
	return result
}
