package cefbuild

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultManifestPath = "native/CEF_VERSION"
	DefaultIndexURL     = "https://cef-builds.spotifycdn.com/index.json"
	DefaultDownloadBase = "https://cef-builds.spotifycdn.com/"
)

type Manifest struct {
	IndexURL        string `json:"index_url"`
	DownloadBaseURL string `json:"download_base_url"`
	Platform        string `json:"platform"`
	Distribution    string `json:"distribution"`
	ChromiumMajor   int    `json:"chromium_major"`
	ChromiumVersion string `json:"chromium_version"`
	CEFVersion      string `json:"cef_version"`
	Archive         string `json:"archive"`
	SHA1            string `json:"sha1"`
	Size            int64  `json:"size"`
}

type Artifact struct {
	Platform        string `json:"platform"`
	Distribution    string `json:"distribution"`
	Channel         string `json:"channel"`
	ChromiumVersion string `json:"chromium_version"`
	CEFVersion      string `json:"cef_version"`
	Archive         string `json:"archive"`
	SHA1            string `json:"sha1"`
	Size            int64  `json:"size"`
	URL             string `json:"url"`
	ExactChromium   bool   `json:"exact_chromium"`
}

type indexFile map[string]platformBuilds

type platformBuilds struct {
	Versions []buildVersion `json:"versions"`
}

type buildVersion struct {
	CEFVersion      string      `json:"cef_version"`
	Channel         string      `json:"channel"`
	ChromiumVersion string      `json:"chromium_version"`
	Files           []buildFile `json:"files"`
}

type buildFile struct {
	LastModified string `json:"last_modified"`
	Name         string `json:"name"`
	SHA1         string `json:"sha1"`
	Size         int64  `json:"size"`
	Type         string `json:"type"`
}

func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, fmt.Errorf("read CEF manifest: %w", err)
	}
	return ParseManifest(data)
}

func ParseManifest(data []byte) (Manifest, error) {
	m := Manifest{
		IndexURL:        DefaultIndexURL,
		DownloadBaseURL: DefaultDownloadBase,
	}

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Manifest{}, fmt.Errorf("invalid CEF manifest line %q", line)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		switch key {
		case "INDEX_URL":
			m.IndexURL = value
		case "DOWNLOAD_BASE_URL":
			m.DownloadBaseURL = value
		case "PLATFORM":
			m.Platform = value
		case "DISTRIBUTION":
			m.Distribution = value
		case "CHROMIUM_MAJOR":
			n, err := strconv.Atoi(value)
			if err != nil {
				return Manifest{}, fmt.Errorf("parse CHROMIUM_MAJOR: %w", err)
			}
			m.ChromiumMajor = n
		case "CHROMIUM_VERSION":
			m.ChromiumVersion = value
		case "CEF_VERSION":
			m.CEFVersion = value
		case "ARCHIVE":
			m.Archive = value
		case "SHA1":
			m.SHA1 = value
		case "SIZE":
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return Manifest{}, fmt.Errorf("parse SIZE: %w", err)
			}
			m.Size = n
		default:
			return Manifest{}, fmt.Errorf("unknown CEF manifest key %q", key)
		}
	}
	if err := scanner.Err(); err != nil {
		return Manifest{}, fmt.Errorf("scan CEF manifest: %w", err)
	}
	if err := m.ValidateTarget(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m Manifest) ValidateTarget() error {
	if strings.TrimSpace(m.IndexURL) == "" {
		return fmt.Errorf("CEF manifest INDEX_URL is required")
	}
	if strings.TrimSpace(m.DownloadBaseURL) == "" {
		return fmt.Errorf("CEF manifest DOWNLOAD_BASE_URL is required")
	}
	if strings.TrimSpace(m.Platform) == "" {
		return fmt.Errorf("CEF manifest PLATFORM is required")
	}
	if strings.TrimSpace(m.Distribution) == "" {
		return fmt.Errorf("CEF manifest DISTRIBUTION is required")
	}
	if m.ChromiumMajor <= 0 {
		return fmt.Errorf("CEF manifest CHROMIUM_MAJOR must be positive")
	}
	if strings.TrimSpace(m.ChromiumVersion) == "" {
		return fmt.Errorf("CEF manifest CHROMIUM_VERSION is required")
	}
	if majorVersion(m.ChromiumVersion) != m.ChromiumMajor {
		return fmt.Errorf("CEF manifest CHROMIUM_MAJOR %d does not match CHROMIUM_VERSION %s", m.ChromiumMajor, m.ChromiumVersion)
	}
	return nil
}

func ResolveArtifact(indexData []byte, target Manifest, allowMajor bool) (Artifact, error) {
	if err := target.ValidateTarget(); err != nil {
		return Artifact{}, err
	}

	var idx indexFile
	if err := json.Unmarshal(indexData, &idx); err != nil {
		return Artifact{}, fmt.Errorf("parse CEF index JSON: %w", err)
	}

	platform, ok := idx[target.Platform]
	if !ok {
		return Artifact{}, fmt.Errorf("CEF index has no platform %q", target.Platform)
	}

	var majorCandidate *Artifact
	for _, build := range platform.Versions {
		exact := build.ChromiumVersion == target.ChromiumVersion
		major := majorVersion(build.ChromiumVersion) == target.ChromiumMajor
		if !exact && (!allowMajor || !major) {
			continue
		}
		for _, file := range build.Files {
			if file.Type != target.Distribution {
				continue
			}
			artifact := Artifact{
				Platform:        target.Platform,
				Distribution:    target.Distribution,
				Channel:         build.Channel,
				ChromiumVersion: build.ChromiumVersion,
				CEFVersion:      build.CEFVersion,
				Archive:         file.Name,
				SHA1:            file.SHA1,
				Size:            file.Size,
				URL:             artifactURL(target.DownloadBaseURL, file.Name),
				ExactChromium:   exact,
			}
			if exact {
				return artifact, nil
			}
			if majorCandidate == nil {
				majorCandidate = &artifact
			}
		}
	}
	if majorCandidate != nil {
		return *majorCandidate, nil
	}

	mode := "exact"
	if allowMajor {
		mode = "major"
	}
	return Artifact{}, fmt.Errorf("no %s CEF artifact for platform=%s distribution=%s chromium=%s", mode, target.Platform, target.Distribution, target.ChromiumVersion)
}

func artifactURL(base, name string) string {
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return base + strings.ReplaceAll(url.PathEscape(name), "+", "%2B")
}

func majorVersion(version string) int {
	head, _, _ := strings.Cut(strings.TrimSpace(version), ".")
	n, err := strconv.Atoi(head)
	if err != nil {
		return 0
	}
	return n
}
