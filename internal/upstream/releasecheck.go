package upstream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultReleasesURL        = "https://releases.electronjs.org/releases.json"
	DefaultReleaseURLTemplate = "https://releases.electronjs.org/release/v%s"
	DefaultTimeout            = 15 * time.Second
)

type Versions struct {
	Electron string `json:"electron"`
	Chromium string `json:"chromium"`
	Node     string `json:"node"`
	V8       string `json:"v8"`
}

type LatestStable struct {
	Versions
	Date       string `json:"date,omitempty"`
	FullDate   string `json:"full_date,omitempty"`
	ReleaseURL string `json:"release_url,omitempty"`
}

type Mismatch struct {
	Component string `json:"component"`
	Target    string `json:"target"`
	Latest    string `json:"latest"`
	Stale     bool   `json:"stale"`
}

type CheckResult struct {
	Target     Versions     `json:"target"`
	Latest     LatestStable `json:"latest"`
	Stale      bool         `json:"stale"`
	Mismatches []Mismatch   `json:"mismatches,omitempty"`
	SourceURL  string       `json:"source_url"`
	CheckedAt  time.Time    `json:"checked_at"`
}

type releaseRecord struct {
	Version  string `json:"version"`
	Date     string `json:"date"`
	FullDate string `json:"fullDate"`
	Node     string `json:"node"`
	V8       string `json:"v8"`
	Chrome   string `json:"chrome"`
}

func FetchLatestStable(ctx context.Context, client *http.Client, releasesURL, releaseURLTemplate string) (LatestStable, error) {
	if client == nil {
		client = http.DefaultClient
	}
	releasesURL = strings.TrimSpace(releasesURL)
	if releasesURL == "" {
		releasesURL = DefaultReleasesURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releasesURL, nil)
	if err != nil {
		return LatestStable{}, fmt.Errorf("build releases request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return LatestStable{}, fmt.Errorf("fetch Electron releases: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return LatestStable{}, fmt.Errorf("fetch Electron releases: status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return LatestStable{}, fmt.Errorf("read Electron releases: %w", err)
	}

	latest, err := ParseLatestStable(data, releaseURLTemplate)
	if err != nil {
		return LatestStable{}, err
	}
	if latest.V8 == "" && latest.ReleaseURL != "" {
		v8, err := fetchReleaseV8(ctx, client, latest.ReleaseURL)
		if err != nil {
			return LatestStable{}, err
		}
		latest.V8 = v8
	}
	if err := validateVersions("latest stable Electron release", latest.Versions); err != nil {
		return LatestStable{}, err
	}
	return latest, nil
}

func ParseLatestStable(data []byte, releaseURLTemplate string) (LatestStable, error) {
	var records []releaseRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return LatestStable{}, fmt.Errorf("parse Electron releases JSON: %w", err)
	}
	var best *releaseRecord
	for i := range records {
		record := records[i]
		if !isStableSemver(record.Version) {
			continue
		}
		if best == nil || compareVersions(record.Version, best.Version) > 0 {
			best = &records[i]
		}
	}
	if best == nil {
		return LatestStable{}, fmt.Errorf("Electron releases feed has no stable releases")
	}
	latest := LatestStable{
		Versions: Versions{
			Electron: best.Version,
			Chromium: best.Chrome,
			Node:     best.Node,
			V8:       best.V8,
		},
		Date:     best.Date,
		FullDate: best.FullDate,
	}
	if strings.TrimSpace(releaseURLTemplate) != "" {
		latest.ReleaseURL = fmt.Sprintf(releaseURLTemplate, best.Version)
	}
	if err := validateCoreVersions("latest stable Electron release", latest.Versions); err != nil {
		return LatestStable{}, err
	}
	return latest, nil
}

func Compare(target Versions, latest LatestStable, checkedAt time.Time, sourceURL string) (CheckResult, error) {
	if err := validateVersions("ledger target", target); err != nil {
		return CheckResult{}, err
	}
	if err := validateVersions("latest stable Electron release", latest.Versions); err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{
		Target:    target,
		Latest:    latest,
		SourceURL: strings.TrimSpace(sourceURL),
		CheckedAt: checkedAt.UTC(),
	}
	result.Mismatches = appendIfDifferent(result.Mismatches, "electron", target.Electron, latest.Electron)
	result.Mismatches = appendIfDifferent(result.Mismatches, "chromium", target.Chromium, latest.Chromium)
	result.Mismatches = appendIfDifferent(result.Mismatches, "node", target.Node, latest.Node)
	result.Mismatches = appendIfDifferent(result.Mismatches, "v8", target.V8, latest.V8)
	for _, mismatch := range result.Mismatches {
		if mismatch.Stale {
			result.Stale = true
			break
		}
	}
	return result, nil
}

func appendIfDifferent(mismatches []Mismatch, component, target, latest string) []Mismatch {
	if target == latest {
		return mismatches
	}
	return append(mismatches, Mismatch{
		Component: component,
		Target:    target,
		Latest:    latest,
		Stale:     compareVersions(target, latest) < 0,
	})
}

func fetchReleaseV8(ctx context.Context, client *http.Client, releaseURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, releaseURL, nil)
	if err != nil {
		return "", fmt.Errorf("build Electron release request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch Electron release detail: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("fetch Electron release detail: status %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read Electron release detail: %w", err)
	}
	v8 := parseReleaseDetailV8(string(data))
	if v8 == "" {
		return "", fmt.Errorf("Electron release detail did not include V8")
	}
	return v8, nil
}

var v8DetailPattern = regexp.MustCompile(`(?is)\bV8\b(?:\s*</?[^>]+>|\s)*([0-9]+(?:\.[0-9A-Za-z-]+)+)`)

func parseReleaseDetailV8(body string) string {
	matches := v8DetailPattern.FindStringSubmatch(body)
	if len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		if strings.EqualFold(strings.TrimSpace(stripTags(line)), "V8") {
			for _, next := range lines[i+1:] {
				candidate := strings.TrimSpace(stripTags(next))
				if candidate == "" {
					continue
				}
				if isVersionLike(candidate) {
					return candidate
				}
				break
			}
		}
	}
	return ""
}

func stripTags(in string) string {
	var out strings.Builder
	inTag := false
	for _, r := range in {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}

func validateVersions(label string, versions Versions) error {
	if err := validateCoreVersions(label, versions); err != nil {
		return err
	}
	if strings.TrimSpace(versions.V8) == "" {
		return fmt.Errorf("%s V8 version is empty", label)
	}
	return nil
}

func validateCoreVersions(label string, versions Versions) error {
	if strings.TrimSpace(versions.Electron) == "" {
		return fmt.Errorf("%s Electron version is empty", label)
	}
	if strings.TrimSpace(versions.Chromium) == "" {
		return fmt.Errorf("%s Chromium version is empty", label)
	}
	if strings.TrimSpace(versions.Node) == "" {
		return fmt.Errorf("%s Node.js version is empty", label)
	}
	return nil
}

func isStableSemver(version string) bool {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	if strings.Contains(version, "-") {
		return false
	}
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}

func isVersionLike(version string) bool {
	version = strings.TrimSpace(version)
	if version == "" {
		return false
	}
	for _, part := range strings.Split(version, ".") {
		if part == "" {
			return false
		}
		if _, err := strconv.Atoi(strings.Trim(part, "vV")); err != nil {
			return false
		}
	}
	return strings.Contains(version, ".")
}

func compareVersions(a, b string) int {
	as := versionParts(a)
	bs := versionParts(b)
	n := len(as)
	if len(bs) > n {
		n = len(bs)
	}
	for i := 0; i < n; i++ {
		var av, bv int
		if i < len(as) {
			av = as[i]
		}
		if i < len(bs) {
			bv = bs[i]
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

func versionParts(version string) []int {
	version = strings.TrimPrefix(strings.TrimSpace(version), "v")
	version = strings.Split(version, "-")[0]
	raw := strings.Split(version, ".")
	parts := make([]int, 0, len(raw))
	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			parts = append(parts, 0)
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			parts = append(parts, 0)
			continue
		}
		parts = append(parts, n)
	}
	return parts
}
