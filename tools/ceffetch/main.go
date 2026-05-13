package main

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/sha1"
	"crypto/sha256"
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
	SHA256    string            `json:"sha256"`
	Extracted bool              `json:"extracted"`
	Current   string            `json:"current,omitempty"`
	StageBin  string            `json:"stage_bin,omitempty"`
}

type downloadSums struct {
	SHA1   string
	SHA256 string
}

func main() {
	manifestPath := flag.String("manifest", cefbuild.DefaultManifestPath, "CEF target manifest")
	indexURL := flag.String("index-url", "", "override CEF index URL")
	indexFile := flag.String("index-file", "", "read CEF index JSON from a local file instead of the network")
	outputDir := flag.String("output", "native/cef", "directory where CEF should be extracted")
	stageBin := flag.String("stage-bin", "", "optional bin/runtime directory to populate from the extracted CEF distribution")
	timeout := flag.Duration("timeout", 30*time.Minute, "HTTP timeout")
	allowMajor := flag.Bool("allow-major", false, "allow a matching Chromium major when exact Chromium version is unavailable")
	keepArchive := flag.Bool("keep-archive", false, "keep the downloaded archive after extraction")
	linkCurrent := flag.Bool("link-current", true, "create or update output/current to point at the extracted CEF distribution")
	jsonOut := flag.Bool("json", false, "emit JSON output")
	flag.Parse()

	report, err := run(*manifestPath, *indexURL, *indexFile, *outputDir, *stageBin, *timeout, *allowMajor, *keepArchive, *linkCurrent)
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

func run(manifestPath, indexURL, indexFile, outputDir, stageBin string, timeout time.Duration, allowMajor, keepArchive, linkCurrent bool) (*fetchReport, error) {
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
	extractedDir := filepath.Join(outputDir, extractedDirName(artifact.Archive))
	if regularFile(filepath.Join(extractedDir, "Release", "libcef.so")) {
		current := ""
		if linkCurrent {
			current = filepath.Join(outputDir, "current")
			if err := replaceSymlink(extractedDirName(artifact.Archive), current); err != nil {
				return nil, err
			}
		}
		if stageBin != "" {
			if err := stageCEFLayout(extractedDir, stageBin); err != nil {
				return nil, err
			}
		}
		return &fetchReport{
			Artifact:  artifact,
			Output:    outputDir,
			Archive:   archivePath,
			Extracted: true,
			Current:   current,
			StageBin:  stageBin,
		}, nil
	}
	sums, err := downloadFile(artifact.URL, archivePath, timeout)
	if err != nil {
		return nil, err
	}
	if knownChecksum(artifact.SHA1) && sums.SHA1 != artifact.SHA1 {
		return nil, fmt.Errorf("CEF archive sha1 mismatch: got %s want %s", sums.SHA1, artifact.SHA1)
	}
	if knownChecksum(artifact.SHA256) && sums.SHA256 != artifact.SHA256 {
		return nil, fmt.Errorf("CEF archive sha256 mismatch: got %s want %s", sums.SHA256, artifact.SHA256)
	}
	if knownChecksum(target.SHA256) && sums.SHA256 != target.SHA256 {
		return nil, fmt.Errorf("CEF archive sha256 mismatch: got %s want %s", sums.SHA256, target.SHA256)
	}
	if err := extractTarBzip2(archivePath, outputDir); err != nil {
		return nil, err
	}
	current := ""
	if linkCurrent {
		current = filepath.Join(outputDir, "current")
		if err := replaceSymlink(extractedDirName(artifact.Archive), current); err != nil {
			return nil, err
		}
	}
	if stageBin != "" {
		if err := stageCEFLayout(extractedDir, stageBin); err != nil {
			return nil, err
		}
	}
	if !keepArchive {
		_ = os.Remove(archivePath)
	}

	return &fetchReport{
		Artifact:  artifact,
		Output:    outputDir,
		Archive:   archivePath,
		SHA1:      sums.SHA1,
		SHA256:    sums.SHA256,
		Extracted: true,
		Current:   current,
		StageBin:  stageBin,
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

func downloadFile(rawURL, path string, timeout time.Duration) (downloadSums, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return downloadSums{}, fmt.Errorf("build CEF download request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return downloadSums{}, fmt.Errorf("download CEF archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return downloadSums{}, fmt.Errorf("download CEF archive: status %s", resp.Status)
	}
	out, err := os.Create(path)
	if err != nil {
		return downloadSums{}, fmt.Errorf("create CEF archive: %w", err)
	}
	defer out.Close()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(out, sha1Hash, sha256Hash), resp.Body); err != nil {
		return downloadSums{}, fmt.Errorf("write CEF archive: %w", err)
	}
	return downloadSums{
		SHA1:   hex.EncodeToString(sha1Hash.Sum(nil)),
		SHA256: hex.EncodeToString(sha256Hash.Sum(nil)),
	}, nil
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

func knownChecksum(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && value != "UNRESOLVED"
}

func extractedDirName(archive string) string {
	name := strings.TrimSuffix(archive, ".tar.bz2")
	return strings.TrimSuffix(name, ".tbz2")
}

func replaceSymlink(target, link string) error {
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return fmt.Errorf("create symlink parent: %w", err)
	}
	_ = os.Remove(link)
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("create current symlink: %w", err)
	}
	return nil
}

func stageCEFLayout(cefDir, binDir string) error {
	releaseDir := filepath.Join(cefDir, "Release")
	resourcesDir := filepath.Join(cefDir, "Resources")
	if !directoryExists(releaseDir) {
		return fmt.Errorf("CEF Release directory missing: %s", releaseDir)
	}
	if !directoryExists(resourcesDir) {
		return fmt.Errorf("CEF Resources directory missing: %s", resourcesDir)
	}
	if err := os.MkdirAll(filepath.Join(binDir, "locales"), 0o755); err != nil {
		return fmt.Errorf("create bin locales: %w", err)
	}
	for _, pattern := range []string{
		filepath.Join(releaseDir, "lib*.so"),
		filepath.Join(releaseDir, "*.bin"),
		filepath.Join(releaseDir, "*.json"),
		filepath.Join(resourcesDir, "*.pak"),
	} {
		if err := copyGlob(pattern, binDir); err != nil {
			return err
		}
	}
	for _, name := range []string{"chrome-sandbox"} {
		src := filepath.Join(releaseDir, name)
		if regularFile(src) {
			if err := copyFile(src, filepath.Join(binDir, name)); err != nil {
				return err
			}
		}
	}
	for _, name := range []string{"icudtl.dat", "v8_context_snapshot.bin", "v8_context_snapshot_blob.bin", "snapshot_blob.bin"} {
		src := filepath.Join(resourcesDir, name)
		if regularFile(src) {
			if err := copyFile(src, filepath.Join(binDir, name)); err != nil {
				return err
			}
		}
	}
	if err := copyGlob(filepath.Join(resourcesDir, "locales", "*.pak"), filepath.Join(binDir, "locales")); err != nil {
		return err
	}
	return nil
}

func copyGlob(pattern, dstDir string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob %s: %w", pattern, err)
	}
	for _, src := range matches {
		if regularFile(src) {
			if err := copyFile(src, filepath.Join(dstDir, filepath.Base(src))); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent for %s: %w", dst, err)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy %s to %s: %w", src, dst, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", dst, err)
	}
	return nil
}

func regularFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
