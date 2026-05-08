package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gibavargas/electron-go/internal/cefbuild"
)

type report struct {
	Target     cefbuild.Manifest  `json:"target"`
	Artifact   *cefbuild.Artifact `json:"artifact,omitempty"`
	AllowMajor bool               `json:"allow_major"`
	Resolved   bool               `json:"resolved"`
	Error      string             `json:"error,omitempty"`
}

func main() {
	manifestPath := flag.String("manifest", cefbuild.DefaultManifestPath, "CEF target manifest")
	indexURL := flag.String("index-url", "", "override CEF index URL")
	indexFile := flag.String("index-file", "", "read CEF index JSON from a local file instead of the network")
	timeout := flag.Duration("timeout", 30*time.Second, "HTTP timeout")
	allowMajor := flag.Bool("allow-major", false, "allow the newest matching Chromium major when exact Chromium version is unavailable")
	jsonOut := flag.Bool("json", false, "emit JSON output")
	flag.Parse()

	result, code := run(*manifestPath, *indexURL, *indexFile, *timeout, *allowMajor)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "cefresolve: encode: %v\n", err)
			os.Exit(2)
		}
		os.Exit(code)
	}
	if result.Resolved {
		fmt.Printf("CEF artifact: %s\n", result.Artifact.URL)
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "cefresolve: %s\n", result.Error)
	os.Exit(code)
}

func run(manifestPath, indexURL, indexFile string, timeout time.Duration, allowMajor bool) (report, int) {
	target, err := cefbuild.LoadManifest(manifestPath)
	if err != nil {
		return report{Error: err.Error()}, 2
	}
	if indexURL != "" {
		target.IndexURL = indexURL
	}

	data, err := readIndex(target.IndexURL, indexFile, timeout)
	if err != nil {
		return report{Target: target, AllowMajor: allowMajor, Error: err.Error()}, 2
	}
	artifact, err := cefbuild.ResolveArtifact(data, target, allowMajor)
	if err != nil {
		return report{Target: target, AllowMajor: allowMajor, Error: err.Error()}, 1
	}
	return report{Target: target, Artifact: &artifact, AllowMajor: allowMajor, Resolved: true}, 0
}

func readIndex(indexURL, indexFile string, timeout time.Duration) ([]byte, error) {
	if indexFile != "" {
		data, err := os.ReadFile(indexFile)
		if err != nil {
			return nil, fmt.Errorf("read CEF index file: %w", err)
		}
		return data, nil
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("timeout must be positive")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build CEF index request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch CEF index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("fetch CEF index: status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, fmt.Errorf("read CEF index: %w", err)
	}
	return data, nil
}
