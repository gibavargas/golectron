package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeJoinRejectsTraversal(t *testing.T) {
	if _, err := safeJoin("/tmp/out", "../escape"); err == nil {
		t.Fatal("safeJoin() error = nil, want traversal error")
	}
}

func TestSafeJoinAllowsNestedRelativePath(t *testing.T) {
	got, err := safeJoin("/tmp/out", "cef_binary/Release/libcef.so")
	if err != nil {
		t.Fatalf("safeJoin() error = %v", err)
	}
	if got == "" {
		t.Fatal("safeJoin() returned empty path")
	}
}

func TestValidateLinkTargetRejectsTraversal(t *testing.T) {
	if err := validateLinkTarget("../../escape"); err == nil {
		t.Fatal("validateLinkTarget() error = nil, want traversal error")
	}
}

func TestKnownChecksumIgnoresUnresolved(t *testing.T) {
	if knownChecksum("UNRESOLVED") {
		t.Fatal("knownChecksum(UNRESOLVED) = true, want false")
	}
	if !knownChecksum("abc123") {
		t.Fatal("knownChecksum(abc123) = false, want true")
	}
}

func TestStageCEFLayout(t *testing.T) {
	root := t.TempDir()
	cefDir := filepath.Join(root, "cef_binary_test_linux64_minimal")
	releaseDir := filepath.Join(cefDir, "Release")
	resourcesDir := filepath.Join(cefDir, "Resources")
	localesDir := filepath.Join(resourcesDir, "locales")
	for _, dir := range []string{releaseDir, localesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	files := map[string]string{
		filepath.Join(releaseDir, "libcef.so"):                      "cef",
		filepath.Join(resourcesDir, "icudtl.dat"):                   "icu",
		filepath.Join(resourcesDir, "v8_context_snapshot.bin"):      "v8",
		filepath.Join(resourcesDir, "resources.pak"):                "resources",
		filepath.Join(resourcesDir, "chrome_100_percent.pak"):       "chrome",
		filepath.Join(localesDir, "en-US.pak"):                      "locale",
		filepath.Join(releaseDir, "vk_swiftshader_icd.json"):        "{}",
		filepath.Join(releaseDir, "libvk_swiftshader.so"):           "vk",
		filepath.Join(releaseDir, "chrome-sandbox"):                 "sandbox",
		filepath.Join(resourcesDir, "chrome_200_percent.pak"):       "chrome2",
		filepath.Join(resourcesDir, "v8_context_snapshot_blob.bin"): "v8blob",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}

	binDir := filepath.Join(root, "bin")
	if err := stageCEFLayout(cefDir, binDir); err != nil {
		t.Fatalf("stageCEFLayout() error = %v", err)
	}
	for _, path := range []string{
		filepath.Join(binDir, "libcef.so"),
		filepath.Join(binDir, "icudtl.dat"),
		filepath.Join(binDir, "v8_context_snapshot.bin"),
		filepath.Join(binDir, "resources.pak"),
		filepath.Join(binDir, "chrome_100_percent.pak"),
		filepath.Join(binDir, "locales", "en-US.pak"),
		filepath.Join(binDir, "libvk_swiftshader.so"),
		filepath.Join(binDir, "vk_swiftshader_icd.json"),
	} {
		if !regularFile(path) {
			t.Fatalf("%s was not staged", path)
		}
	}
}

func TestRunReusesExtractedCEFLayout(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "CEF_VERSION")
	archive := "cef_binary_147.0.10+gd58e84d+chromium-147.0.7727.118_linux64_minimal.tar.bz2"
	if err := os.WriteFile(manifest, []byte("INDEX_URL=https://invalid.test/index.json\nDOWNLOAD_BASE_URL=https://invalid.test/\nPLATFORM=linux64\nDISTRIBUTION=minimal\nCHROMIUM_MAJOR=147\nCHROMIUM_VERSION=147.0.7727.118\nCEF_VERSION=147.0.10+gd58e84d+chromium-147.0.7727.118\nARCHIVE="+archive+"\nSHA1=UNRESOLVED\nSHA256=UNRESOLVED\nSIZE=1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(manifest) error = %v", err)
	}
	index := filepath.Join(root, "index.json")
	if err := os.WriteFile(index, []byte(`{"linux64":{"versions":[{"cef_version":"147.0.10+gd58e84d+chromium-147.0.7727.118","chromium_version":"147.0.7727.118","channel":"stable","files":[{"type":"minimal","name":"`+archive+`","sha1":"UNRESOLVED","size":1}]}]}}`), 0o644); err != nil {
		t.Fatalf("WriteFile(index) error = %v", err)
	}
	outputDir := filepath.Join(root, "cef")
	extractedDir := filepath.Join(outputDir, extractedDirName(archive))
	createMinimalCEFLayout(t, extractedDir)

	binDir := filepath.Join(root, "bin")
	report, err := run(manifest, "", index, outputDir, binDir, 0, false, false, true)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if !report.Extracted || report.SHA1 != "" || report.SHA256 != "" {
		t.Fatalf("report = %#v, want reused extracted layout without download checksums", report)
	}
	if !regularFile(filepath.Join(binDir, "libcef.so")) {
		t.Fatal("cached CEF layout was not staged")
	}
}

func createMinimalCEFLayout(t *testing.T, cefDir string) {
	t.Helper()
	releaseDir := filepath.Join(cefDir, "Release")
	resourcesDir := filepath.Join(cefDir, "Resources")
	localesDir := filepath.Join(resourcesDir, "locales")
	for _, dir := range []string{releaseDir, localesDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	files := map[string]string{
		filepath.Join(releaseDir, "libcef.so"):                 "cef",
		filepath.Join(resourcesDir, "icudtl.dat"):              "icu",
		filepath.Join(resourcesDir, "v8_context_snapshot.bin"): "v8",
		filepath.Join(resourcesDir, "resources.pak"):           "resources",
		filepath.Join(resourcesDir, "chrome_100_percent.pak"):  "chrome",
		filepath.Join(localesDir, "en-US.pak"):                 "locale",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
}
