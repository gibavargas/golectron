package compat

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type npmPackagePlan struct {
	Platform       string `json:"platform"`
	Arch           string `json:"arch"`
	BinaryPath     string `json:"binaryPath"`
	Checksum       string `json:"checksum"`
	ChecksumPath   string `json:"checksumPath"`
	Download       bool   `json:"download"`
	Manual         bool   `json:"manual"`
	RemovedSkipSet bool   `json:"removedSkipSet"`
}

func TestNPMPackageInstallAndLazyBinBehavior(t *testing.T) {
	root := t.TempDir()
	postinstall := runNPMPackageScript(t, root, filepath.Join("npm", "install.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM":     "linux",
		"ELECTRON_INSTALL_ARCH":         "arm64",
		"ELECTRON_SKIP_BINARY_DOWNLOAD": "1",
	})
	if postinstall.Download || postinstall.Manual || !postinstall.RemovedSkipSet {
		t.Fatalf("postinstall = %#v, want no download, manual=false, removed skip observed", postinstall)
	}
	if _, err := os.Stat(postinstall.BinaryPath); !os.IsNotExist(err) {
		t.Fatalf("postinstall binary stat = %v, want missing binary", err)
	}

	manual := runNPMPackageScript(t, root, filepath.Join("npm", "install.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM": "linux",
		"ELECTRON_INSTALL_ARCH":     "arm64",
	}, "--manual")
	if !manual.Download || !manual.Manual || manual.Checksum == "" {
		t.Fatalf("manual install = %#v, want download, manual=true, checksum", manual)
	}
	if _, err := os.Stat(manual.BinaryPath); err != nil {
		t.Fatalf("manual binary stat error = %v", err)
	}
	if _, err := os.Stat(manual.ChecksumPath); err != nil {
		t.Fatalf("manual checksum stat error = %v", err)
	}

	lazyRoot := t.TempDir()
	first := runNPMPackageScript(t, lazyRoot, filepath.Join("npm", "bin", "electron-go.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM":  "darwin",
		"ELECTRON_INSTALL_ARCH":      "x64",
		"ELECTRON_GO_NPM_PRINT_PLAN": "1",
	})
	second := runNPMPackageScript(t, lazyRoot, filepath.Join("npm", "bin", "electron-go.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM":  "darwin",
		"ELECTRON_INSTALL_ARCH":      "x64",
		"ELECTRON_GO_NPM_PRINT_PLAN": "1",
	})
	if !first.Download || first.Manual || second.Download || second.Manual {
		t.Fatalf("bin plans first=%#v second=%#v, want lazy first download only", first, second)
	}
	if first.BinaryPath != second.BinaryPath || first.Checksum == "" || first.Checksum != second.Checksum {
		t.Fatalf("bin artifact mismatch first=%#v second=%#v", first, second)
	}
}

func runNPMPackageScript(t *testing.T, root, script string, env map[string]string, args ...string) npmPackagePlan {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for npm package e2e tests")
	}
	argv := append([]string{script}, args...)
	cmd := exec.Command(node, argv...)
	cmd.Dir = repoRootForNPM(t)
	cmd.Env = append(os.Environ(), "ELECTRON_GO_NPM_ROOT="+root)
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node %v error = %v\n%s", argv, err, string(output))
	}
	var plan npmPackagePlan
	if err := json.Unmarshal(output, &plan); err != nil {
		t.Fatalf("npm package output is not JSON: %v\n%s", err, string(output))
	}
	return plan
}

func repoRootForNPM(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repo root")
		}
		dir = parent
	}
}
