package cefbuild

import (
	"strings"
	"testing"
)

func TestParseManifest(t *testing.T) {
	manifest, err := ParseManifest([]byte(`
INDEX_URL=https://cef-builds.spotifycdn.com/index.json
DOWNLOAD_BASE_URL=https://cef-builds.spotifycdn.com/
PLATFORM=linux64
DISTRIBUTION=minimal
CHROMIUM_MAJOR=148
CHROMIUM_VERSION=148.0.7778.96
CEF_VERSION=UNRESOLVED
ARCHIVE=UNRESOLVED
SHA1=UNRESOLVED
SIZE=0
`))
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}
	if manifest.ChromiumMajor != 148 {
		t.Fatalf("ChromiumMajor = %d, want 148", manifest.ChromiumMajor)
	}
	if manifest.Platform != "linux64" {
		t.Fatalf("Platform = %q, want linux64", manifest.Platform)
	}
}

func TestParseManifestRejectsMismatchedMajor(t *testing.T) {
	_, err := ParseManifest([]byte(`
PLATFORM=linux64
DISTRIBUTION=minimal
CHROMIUM_MAJOR=130
CHROMIUM_VERSION=148.0.7778.96
`))
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("ParseManifest() error = %v, want mismatch", err)
	}
}

func TestResolveArtifactExactChromium(t *testing.T) {
	index := []byte(`{
  "linux64": {
    "versions": [
      {
        "cef_version": "148.0.1+gabc1234+chromium-148.0.7778.96",
        "channel": "stable",
        "chromium_version": "148.0.7778.96",
        "files": [
          {"name": "cef_binary_148.0.1+gabc1234+chromium-148.0.7778.96_linux64_minimal.tar.bz2", "sha1": "abc", "size": 42, "type": "minimal"}
        ]
      }
    ]
  }
}`)
	target := Manifest{
		IndexURL:        DefaultIndexURL,
		DownloadBaseURL: DefaultDownloadBase,
		Platform:        "linux64",
		Distribution:    "minimal",
		ChromiumMajor:   148,
		ChromiumVersion: "148.0.7778.96",
	}

	artifact, err := ResolveArtifact(index, target, false)
	if err != nil {
		t.Fatalf("ResolveArtifact() error = %v", err)
	}
	if !artifact.ExactChromium {
		t.Fatal("ExactChromium = false, want true")
	}
	if !strings.Contains(artifact.URL, "%2B") {
		t.Fatalf("URL = %q, want escaped plus signs", artifact.URL)
	}
}

func TestResolveArtifactRejectsWrongMajorWithoutFallback(t *testing.T) {
	index := []byte(`{
  "linux64": {
    "versions": [
      {
        "cef_version": "130.0.1+gabc1234+chromium-130.0.1.1",
        "channel": "stable",
        "chromium_version": "130.0.1.1",
        "files": [
          {"name": "cef_binary_130.0.1+gabc1234+chromium-130.0.1.1_linux64_minimal.tar.bz2", "sha1": "abc", "size": 42, "type": "minimal"}
        ]
      }
    ]
  }
}`)
	target := Manifest{
		IndexURL:        DefaultIndexURL,
		DownloadBaseURL: DefaultDownloadBase,
		Platform:        "linux64",
		Distribution:    "minimal",
		ChromiumMajor:   148,
		ChromiumVersion: "148.0.7778.96",
	}

	_, err := ResolveArtifact(index, target, false)
	if err == nil {
		t.Fatal("ResolveArtifact() error = nil, want error")
	}
}
