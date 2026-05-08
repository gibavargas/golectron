package main

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gibavargas/electron-go/internal/cefbuild"
)

type fetchReport struct {
	Artifact  cefbuild.Artifact `json:"artifact"`
	Output    string            `json:"output"`
	Archive   string            `json:"archive"`
	SHA1      string            `json:"sha1"`
	Extracted bool              `json:"extracted"`
}

func main() {
	manifestPath := flag.String("manifest", cefbuild.DefaultManifestPath, "CEF target manifest")
	indexURL := flag.String("index-url", "", "override CEF index URL")
	indexFile := flag.String("index-file", "", "read CEF index JSON from a local file instead of the network")
	outputDir := flag.String("output", "native/cef", "directory where CEF should be extracted")
	timeout := flag.Duration("timeout", 30*time.Minute, "HTTP timeout")
	allowMajor := flag.Bool("allow-major", false, "allow a matching Chromium major when exact Chromium version is unavailable")
	keepArchive := flag.Bool("keep-archive", false, "keep the downloaded archive after extraction")
	jsonOut := flag.Bool("json", false, "emit JSON output")
	flag.Parse()

	report, err := run(*manifestPath, *indexURL, *indexFile, *outputDir, *timeout, *allowMajor, *keepArchive)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if report != nil {
			_ = enc.Encode(report)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ceffetch: %v\n", err)
		os.Exit(1)
	}
	if !*jsonOut {
		fmt.Printf("CEF extracted to %s\n", report.Output)
	}
}

func run(manifestPath, indexURL, indexFile, outputDir string, timeout time.Duration, allowMajor, keepArchive bool) (*fetchReport, error) {
	target, err := cefbuild.LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}
	if indexURL != "" {
		target.IndexURL = indexURL
	}
	indexData, err := readIndex(target.IndexURL, indexFile, timeout)
	if err != nil {
		return nil, err
	}
	artifact, err := cefbuild.ResolveArtifact(indexData, target, allowMajor)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	archivePath := filepath.Join(outputDir, artifact.Archive)
	sum, err := downloadFile(artifact.URL, archivePath, timeout)
	if err != nil {
		return nil, err
	}
	if artifact.SHA1 != "" && sum != artifact.SHA1 {
		return nil, fmt.Errorf("CEF archive sha1 mismatch: got %s want %s", sum, artifact.SHA1)
	}
	if err := extractTarBzip2(archivePath, outputDir); err != nil {
		return nil, err
	}
	if !keepArchive {
		_ = os.Remove(archivePath)
	}

	return &fetchReport{
		Artifact:  artifact,
		Output:    outputDir,
		Archive:   archivePath,
		SHA1:      sum,
		Extracted: true,
	}, nil
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

func downloadFile(rawURL, path string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build CEF download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download CEF archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("download CEF archive: status %s", resp.Status)
	}
	out, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create CEF archive: %w", err)
	}
	defer out.Close()
	hash := sha1.New()
	if _, err := io.Copy(out, io.TeeReader(resp.Body, hash)); err != nil {
		return "", fmt.Errorf("write CEF archive: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func extractTarBzip2(path, outputDir string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open CEF archive: %w", err)
	}
	defer file.Close()
	reader := tar.NewReader(bzip2.NewReader(file))
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read CEF archive: %w", err)
		}
		target, err := safeJoin(outputDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create dir %s: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent for %s: %w", target, err)
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("create file %s: %w", target, err)
			}
			if _, err := io.Copy(out, reader); err != nil {
				_ = out.Close()
				return fmt.Errorf("extract file %s: %w", target, err)
			}
			if err := out.Close(); err != nil {
				return fmt.Errorf("close file %s: %w", target, err)
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create parent for %s: %w", target, err)
			}
			if err := validateLinkTarget(header.Linkname); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return fmt.Errorf("create symlink %s: %w", target, err)
			}
		}
	}
}

func safeJoin(root, name string) (string, error) {
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("archive entry has absolute path: %s", name)
	}
	clean := filepath.Clean(name)
	if clean == "." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", fmt.Errorf("archive entry escapes output dir: %s", name)
	}
	return filepath.Join(root, clean), nil
}

func validateLinkTarget(name string) error {
	if filepath.IsAbs(name) {
		return fmt.Errorf("archive symlink has absolute target: %s", name)
	}
	clean := filepath.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("archive symlink escapes output dir: %s", name)
	}
	return nil
}
