package upstream

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestParseLatestStableIgnoresPrereleasesAndNightlies(t *testing.T) {
	data := []byte(`[
		{"version":"43.0.0-nightly.20260505","date":"2026-05-05","node":"24.15.0","v8":"14.9.205","chrome":"149.0.7820.0"},
		{"version":"43.0.0-alpha.1","date":"2026-05-06","node":"24.15.0","v8":"14.9.207","chrome":"149.0.7827.0"},
		{"version":"41.5.0","date":"2026-05-01","node":"24.15.0","v8":"14.6.231.21","chrome":"146.0.7680.216"},
		{"version":"42.0.0","date":"2026-05-05","fullDate":"2026-05-05T16:00:00.000Z","node":"24.15.0","v8":"14.8.178.14","chrome":"148.0.7778.96"}
	]`)

	latest, err := ParseLatestStable(data, "https://example.test/release/v%s")
	if err != nil {
		t.Fatalf("ParseLatestStable() error = %v", err)
	}
	if latest.Electron != "42.0.0" {
		t.Fatalf("Electron = %q, want 42.0.0", latest.Electron)
	}
	if latest.Chromium != "148.0.7778.96" || latest.Node != "24.15.0" || latest.V8 != "14.8.178.14" {
		t.Fatalf("latest versions = %#v", latest.Versions)
	}
	if latest.ReleaseURL != "https://example.test/release/v42.0.0" {
		t.Fatalf("ReleaseURL = %q", latest.ReleaseURL)
	}
}

func TestFetchLatestStableFallsBackToReleaseDetailForV8(t *testing.T) {
	var releaseDetailRequested bool
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			var body string
			switch req.URL.Path {
			case "/releases.json":
				body = `[
				{"version":"42.0.0","date":"2026-05-05","node":"24.15.0","chrome":"148.0.7778.96"}
			]`
			case "/release/v42.0.0":
				releaseDetailRequested = true
				body = `<main><h2>V8</h2><p>14.8.178.14</p></main>`
			default:
				t.Fatalf("unexpected request path %q", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBufferString(body)),
				Request:    req,
			}, nil
		}),
	}

	latest, err := FetchLatestStable(context.Background(), client, "https://example.test/releases.json", "https://example.test/release/v%s")
	if err != nil {
		t.Fatalf("FetchLatestStable() error = %v", err)
	}
	if !releaseDetailRequested {
		t.Fatal("release detail was not requested")
	}
	if latest.V8 != "14.8.178.14" {
		t.Fatalf("V8 = %q, want 14.8.178.14", latest.V8)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestCompareMarksLedgerTargetStale(t *testing.T) {
	result, err := Compare(
		Versions{Electron: "41.5.0", Chromium: "146.0.7680.216", Node: "24.15.0", V8: "14.6.231.21"},
		LatestStable{Versions: Versions{Electron: "42.0.0", Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"}},
		time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC),
		DefaultReleasesURL,
	)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if !result.Stale {
		t.Fatal("result.Stale = false, want true")
	}
	if len(result.Mismatches) != 3 {
		t.Fatalf("len(Mismatches) = %d, want 3", len(result.Mismatches))
	}
	for _, mismatch := range result.Mismatches {
		if mismatch.Component == "node" {
			t.Fatal("Node should not mismatch")
		}
		if !mismatch.Stale {
			t.Fatalf("mismatch %#v should be stale", mismatch)
		}
	}
}

func TestCompareReportsNonStaleDifferences(t *testing.T) {
	result, err := Compare(
		Versions{Electron: "42.1.0", Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"},
		LatestStable{Versions: Versions{Electron: "42.0.0", Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"}},
		time.Now(),
		DefaultReleasesURL,
	)
	if err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if result.Stale {
		t.Fatal("result.Stale = true, want false")
	}
	if len(result.Mismatches) != 1 || result.Mismatches[0].Stale {
		t.Fatalf("mismatches = %#v", result.Mismatches)
	}
}
