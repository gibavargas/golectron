package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gibavargas/electron-go/internal/compat"
	"github.com/gibavargas/electron-go/internal/upstream"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	flags := flag.NewFlagSet("releasecheck", flag.ContinueOnError)
	flags.SetOutput(stderr)
	releasesURL := flags.String("url", upstream.DefaultReleasesURL, "Electron releases JSON URL")
	releaseURLTemplate := flags.String("release-url-template", upstream.DefaultReleaseURLTemplate, "Electron release detail URL template; must contain one %s for the version")
	timeout := flags.Duration("timeout", upstream.DefaultTimeout, "HTTP timeout")
	jsonOut := flags.Bool("json", false, "print JSON output")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *timeout <= 0 {
		fmt.Fprintln(stderr, "releasecheck: --timeout must be greater than zero")
		return 2
	}

	ledger, err := compat.LoadLedger()
	if err != nil {
		fmt.Fprintf(stderr, "releasecheck: %v\n", err)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	client := &http.Client{Timeout: *timeout}
	latest, err := upstream.FetchLatestStable(ctx, client, *releasesURL, *releaseURLTemplate)
	if err != nil {
		fmt.Fprintf(stderr, "releasecheck: %v\n", err)
		return 2
	}
	result, err := upstream.Compare(toUpstreamVersions(ledger.Target), latest, time.Now(), *releasesURL)
	if err != nil {
		fmt.Fprintf(stderr, "releasecheck: %v\n", err)
		return 2
	}

	if *jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(stderr, "releasecheck: encode JSON: %v\n", err)
			return 2
		}
	} else {
		printHuman(stdout, result)
	}
	if result.Stale {
		return 1
	}
	return 0
}

func toUpstreamVersions(target compat.TargetVersions) upstream.Versions {
	return upstream.Versions{
		Electron: target.Electron,
		Chromium: target.Chromium,
		Node:     target.Node,
		V8:       target.V8,
	}
}

func printHuman(out *os.File, result upstream.CheckResult) {
	fmt.Fprintf(out, "Electron release gate\n")
	fmt.Fprintf(out, "source: %s\n", result.SourceURL)
	fmt.Fprintf(out, "target: Electron %s, Chromium %s, Node.js %s, V8 %s\n", result.Target.Electron, result.Target.Chromium, result.Target.Node, result.Target.V8)
	fmt.Fprintf(out, "latest: Electron %s, Chromium %s, Node.js %s, V8 %s\n", result.Latest.Electron, result.Latest.Chromium, result.Latest.Node, result.Latest.V8)
	if result.Latest.ReleaseURL != "" {
		fmt.Fprintf(out, "release: %s\n", result.Latest.ReleaseURL)
	}
	if len(result.Mismatches) == 0 {
		fmt.Fprintln(out, "status: ledger target matches latest stable")
		return
	}
	fmt.Fprintln(out, "mismatches:")
	for _, mismatch := range result.Mismatches {
		status := "different"
		if mismatch.Stale {
			status = "stale"
		}
		fmt.Fprintf(out, "- %s: target %s, latest %s (%s)\n", mismatch.Component, mismatch.Target, mismatch.Latest, status)
	}
	if result.Stale {
		fmt.Fprintln(out, "status: ledger target is stale")
		return
	}
	fmt.Fprintln(out, "status: ledger target is not stale")
}
