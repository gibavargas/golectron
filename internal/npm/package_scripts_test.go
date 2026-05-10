package npm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type scriptPlan struct {
	Platform       string `json:"platform"`
	Arch           string `json:"arch"`
	BinaryPath     string `json:"binaryPath"`
	Checksum       string `json:"checksum"`
	ChecksumPath   string `json:"checksumPath"`
	Download       bool   `json:"download"`
	Manual         bool   `json:"manual"`
	RemovedSkipSet bool   `json:"removedSkipSet"`
}

func TestNPMInstallScriptDoesNotDownloadOnPostinstall(t *testing.T) {
	root := t.TempDir()
	plan := runNPMScript(t, root, filepath.Join("npm", "install.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM":     "linux",
		"ELECTRON_INSTALL_ARCH":         "arm64",
		"ELECTRON_SKIP_BINARY_DOWNLOAD": "1",
	})
	if plan.Platform != "linux" || plan.Arch != "arm64" || plan.Download || plan.Manual || !plan.RemovedSkipSet {
		t.Fatalf("postinstall plan = %#v", plan)
	}
	if _, err := os.Stat(plan.BinaryPath); !os.IsNotExist(err) {
		t.Fatalf("postinstall binary stat error = %v, want missing binary", err)
	}
}

func TestNPMManualInstallDownloadsBinary(t *testing.T) {
	root := t.TempDir()
	plan := runNPMScript(t, root, filepath.Join("npm", "install.js"), map[string]string{
		"ELECTRON_INSTALL_PLATFORM": "darwin",
		"ELECTRON_INSTALL_ARCH":     "arm64",
	}, "--manual")
	if plan.Platform != "darwin" || plan.Arch != "arm64" || !plan.Download || !plan.Manual {
		t.Fatalf("manual install plan = %#v", plan)
	}
	if _, err := os.Stat(plan.BinaryPath); err != nil {
		t.Fatalf("manual install binary stat error = %v", err)
	}
	if plan.Checksum == "" {
		t.Fatalf("manual install checksum is empty: %#v", plan)
	}
	if _, err := os.Stat(plan.ChecksumPath); err != nil {
		t.Fatalf("manual install checksum stat error = %v", err)
	}
}

func TestNPMBinScriptDownloadsLazilyOnce(t *testing.T) {
	root := t.TempDir()
	env := map[string]string{
		"ELECTRON_INSTALL_PLATFORM":  "linux",
		"ELECTRON_INSTALL_ARCH":      "x64",
		"ELECTRON_GO_NPM_PRINT_PLAN": "1",
	}
	first := runNPMScript(t, root, filepath.Join("npm", "bin", "electron-go.js"), env)
	if !first.Download || first.Manual {
		t.Fatalf("first bin plan = %#v, want lazy download", first)
	}
	second := runNPMScript(t, root, filepath.Join("npm", "bin", "electron-go.js"), env)
	if second.Download || second.BinaryPath != first.BinaryPath {
		t.Fatalf("second bin plan = %#v, want no download and same binary %q", second, first.BinaryPath)
	}
	if second.Checksum == "" || second.Checksum != first.Checksum {
		t.Fatalf("second checksum = %q, want %q", second.Checksum, first.Checksum)
	}
}

func runNPMScript(t *testing.T, root, script string, env map[string]string, args ...string) scriptPlan {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is required for npm package script e2e tests")
	}
	argv := append([]string{script}, args...)
	cmd := exec.Command(node, argv...)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "ELECTRON_GO_NPM_ROOT="+root)
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node %v error = %v\n%s", argv, err, string(output))
	}
	var plan scriptPlan
	if err := json.Unmarshal(output, &plan); err != nil {
		t.Fatalf("script output is not JSON: %v\n%s", err, string(output))
	}
	return plan
}

func repoRoot(t *testing.T) string {
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
