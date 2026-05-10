package autoupdater

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

var (
	ErrInvalidFeed     = errors.New("invalid update feed")
	ErrInvalidProvider = errors.New("invalid updater provider")
	ErrNotDownloaded   = errors.New("update has not been downloaded")
)

type Provider string

const (
	ProviderSquirrel Provider = "squirrel"
	ProviderNSIS     Provider = "nsis"
	ProviderMSIX     Provider = "msix"
)

type Feed struct {
	URL      string
	Headers  map[string]string
	Provider Provider
}

type UpdateInfo struct {
	Version      string
	ReleaseName  string
	ReleaseNotes string
	Downloaded   bool
}

type Event string

const (
	EventCheckingForUpdate   Event = "checking-for-update"
	EventUpdateAvailable     Event = "update-available"
	EventUpdateNotAvailable  Event = "update-not-available"
	EventUpdateDownloaded    Event = "update-downloaded"
	EventBeforeQuitForUpdate Event = "before-quit-for-update"
	EventError               Event = "error"
)

type EventRecord struct {
	Event Event
	Info  UpdateInfo
	Error string
}

type Updater struct {
	feed       Feed
	current    string
	available  *UpdateInfo
	downloaded *UpdateInfo
	events     []EventRecord
}

func New(currentVersion string) *Updater {
	return &Updater{current: strings.TrimSpace(currentVersion)}
}

func (u *Updater) SetFeedURL(feed Feed) error {
	normalized, err := normalizeFeed(feed)
	if err != nil {
		return err
	}
	u.feed = normalized
	return nil
}

func (u *Updater) FeedURL() Feed {
	feed := u.feed
	feed.Headers = cloneHeaders(feed.Headers)
	return feed
}

func (u *Updater) CheckForUpdates(remote *UpdateInfo) (bool, error) {
	if strings.TrimSpace(u.feed.URL) == "" {
		err := fmt.Errorf("%w: feed URL is required", ErrInvalidFeed)
		u.events = append(u.events, EventRecord{Event: EventError, Error: err.Error()})
		return false, err
	}
	u.events = append(u.events, EventRecord{Event: EventCheckingForUpdate})
	if remote == nil || strings.TrimSpace(remote.Version) == "" || strings.TrimSpace(remote.Version) == u.current {
		u.events = append(u.events, EventRecord{Event: EventUpdateNotAvailable})
		return false, nil
	}
	info := *remote
	info.Version = strings.TrimSpace(info.Version)
	info.ReleaseName = strings.TrimSpace(info.ReleaseName)
	u.available = &info
	u.events = append(u.events, EventRecord{Event: EventUpdateAvailable, Info: info})
	return true, nil
}

func (u *Updater) DownloadUpdate() error {
	if u.available == nil {
		u.events = append(u.events, EventRecord{Event: EventError, Error: ErrNotDownloaded.Error()})
		return ErrNotDownloaded
	}
	info := *u.available
	info.Downloaded = true
	u.downloaded = &info
	u.events = append(u.events, EventRecord{Event: EventUpdateDownloaded, Info: info})
	return nil
}

func (u *Updater) QuitAndInstall() error {
	if u.downloaded == nil {
		u.events = append(u.events, EventRecord{Event: EventError, Error: ErrNotDownloaded.Error()})
		return ErrNotDownloaded
	}
	u.events = append(u.events, EventRecord{Event: EventBeforeQuitForUpdate, Info: *u.downloaded})
	return nil
}

func (u *Updater) Events() []EventRecord {
	return append([]EventRecord(nil), u.events...)
}

func SupportsProvider(provider Provider, targetOS string) bool {
	switch provider {
	case ProviderSquirrel:
		return targetOS == "darwin" || targetOS == "win32" || targetOS == "windows"
	case ProviderNSIS:
		return targetOS == "win32" || targetOS == "windows"
	case ProviderMSIX:
		return targetOS == "win32" || targetOS == "windows"
	default:
		return false
	}
}

func normalizeFeed(feed Feed) (Feed, error) {
	feed.URL = strings.TrimSpace(feed.URL)
	parsed, err := url.Parse(feed.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Feed{}, fmt.Errorf("%w: URL", ErrInvalidFeed)
	}
	if feed.Provider == "" {
		feed.Provider = ProviderSquirrel
	}
	if feed.Provider != ProviderSquirrel && feed.Provider != ProviderNSIS && feed.Provider != ProviderMSIX {
		return Feed{}, fmt.Errorf("%w: %s", ErrInvalidProvider, feed.Provider)
	}
	headers := cloneHeaders(feed.Headers)
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n:") || strings.ContainsAny(value, "\r\n") {
			return Feed{}, fmt.Errorf("%w: header", ErrInvalidFeed)
		}
	}
	feed.Headers = headers
	return feed, nil
}

func cloneHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return nil
	}
	out := make(map[string]string, len(headers))
	for key, value := range headers {
		out[key] = value
	}
	return out
}
