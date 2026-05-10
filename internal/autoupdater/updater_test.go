package autoupdater

import (
	"errors"
	"reflect"
	"testing"
)

func TestSetFeedURLNormalizesAndCopies(t *testing.T) {
	updater := New("1.0.0")
	headers := map[string]string{"Authorization": "token"}
	if err := updater.SetFeedURL(Feed{URL: "https://updates.example.test/feed", Headers: headers, Provider: ProviderMSIX}); err != nil {
		t.Fatalf("SetFeedURL() error = %v", err)
	}
	headers["Authorization"] = "mutated"
	feed := updater.FeedURL()
	if feed.URL != "https://updates.example.test/feed" || feed.Provider != ProviderMSIX || feed.Headers["Authorization"] != "token" {
		t.Fatalf("FeedURL() = %#v", feed)
	}
}

func TestSetFeedURLRejectsInvalidInput(t *testing.T) {
	updater := New("1.0.0")
	cases := []Feed{
		{URL: "missing-host"},
		{URL: "https://updates.example.test", Provider: "custom"},
		{URL: "https://updates.example.test", Headers: map[string]string{"Bad\nHeader": "x"}},
	}
	for _, tc := range cases {
		if err := updater.SetFeedURL(tc); !errors.Is(err, ErrInvalidFeed) && !errors.Is(err, ErrInvalidProvider) {
			t.Fatalf("SetFeedURL(%#v) error = %v, want validation error", tc, err)
		}
	}
}

func TestCheckDownloadAndQuitEventOrder(t *testing.T) {
	updater := New("1.0.0")
	if err := updater.SetFeedURL(Feed{URL: "https://updates.example.test/feed"}); err != nil {
		t.Fatalf("SetFeedURL() error = %v", err)
	}
	available, err := updater.CheckForUpdates(&UpdateInfo{Version: " 2.0.0 ", ReleaseName: " Release "})
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}
	if !available {
		t.Fatal("available = false, want true")
	}
	if err := updater.DownloadUpdate(); err != nil {
		t.Fatalf("DownloadUpdate() error = %v", err)
	}
	if err := updater.QuitAndInstall(); err != nil {
		t.Fatalf("QuitAndInstall() error = %v", err)
	}
	want := []EventRecord{
		{Event: EventCheckingForUpdate},
		{Event: EventUpdateAvailable, Info: UpdateInfo{Version: "2.0.0", ReleaseName: "Release"}},
		{Event: EventUpdateDownloaded, Info: UpdateInfo{Version: "2.0.0", ReleaseName: "Release", Downloaded: true}},
		{Event: EventBeforeQuitForUpdate, Info: UpdateInfo{Version: "2.0.0", ReleaseName: "Release", Downloaded: true}},
	}
	if got := updater.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
}

func TestNoUpdateAndMissingDownload(t *testing.T) {
	updater := New("1.0.0")
	if err := updater.SetFeedURL(Feed{URL: "https://updates.example.test/feed"}); err != nil {
		t.Fatalf("SetFeedURL() error = %v", err)
	}
	available, err := updater.CheckForUpdates(&UpdateInfo{Version: "1.0.0"})
	if err != nil {
		t.Fatalf("CheckForUpdates() error = %v", err)
	}
	if available {
		t.Fatal("available = true, want false")
	}
	if err := updater.DownloadUpdate(); !errors.Is(err, ErrNotDownloaded) {
		t.Fatalf("DownloadUpdate() error = %v, want ErrNotDownloaded", err)
	}
}

func TestSupportsProvider(t *testing.T) {
	if !SupportsProvider(ProviderMSIX, "windows") {
		t.Fatal("MSIX should be supported on Windows")
	}
	if SupportsProvider(ProviderMSIX, "darwin") {
		t.Fatal("MSIX should not be supported on macOS")
	}
	if !SupportsProvider(ProviderSquirrel, "darwin") {
		t.Fatal("Squirrel should be supported on macOS")
	}
}
