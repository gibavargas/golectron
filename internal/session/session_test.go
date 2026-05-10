package session

import (
	"errors"
	"testing"
)

func TestRegistryDefaultAndPartitions(t *testing.T) {
	registry := NewRegistry()
	defaultSession := registry.DefaultSession()
	emptyPartition, err := registry.FromPartition("")
	if err != nil {
		t.Fatalf("FromPartition(empty) error = %v", err)
	}
	if defaultSession != emptyPartition {
		t.Fatal("empty partition did not return default session")
	}
	persistent, err := registry.FromPartition("persist:profile")
	if err != nil {
		t.Fatalf("FromPartition(persist) error = %v", err)
	}
	if !persistent.IsPersistent() || persistent.Partition() != "persist:profile" {
		t.Fatalf("persistent session = partition %q persistent %v", persistent.Partition(), persistent.IsPersistent())
	}
	again, err := registry.FromPartition("persist:profile")
	if err != nil {
		t.Fatalf("FromPartition(existing) error = %v", err)
	}
	if persistent != again {
		t.Fatal("same partition returned a different Session")
	}
	inMemory, err := registry.FromPartition("temporary")
	if err != nil {
		t.Fatalf("FromPartition(memory) error = %v", err)
	}
	if inMemory.IsPersistent() {
		t.Fatal("non-persist partition is persistent")
	}
	if _, err := registry.FromPartition("bad\npartition"); !errors.Is(err, ErrInvalidPartition) {
		t.Fatalf("FromPartition(invalid) error = %v, want ErrInvalidPartition", err)
	}
}

func TestPermissionHandler(t *testing.T) {
	s := NewRegistry().DefaultSession()
	if s.CheckPermission(PermissionRequest{Permission: "media"}) {
		t.Fatal("CheckPermission() = true without handler")
	}
	s.SetPermissionRequestHandler(func(req PermissionRequest) bool {
		return req.Permission == "media" && req.RequestingURL == "https://example.test/"
	})
	if err := s.RequestPermission(PermissionRequest{Permission: "media", RequestingURL: "https://example.test/"}); err != nil {
		t.Fatalf("RequestPermission(media) error = %v", err)
	}
	if err := s.RequestPermission(PermissionRequest{Permission: "notifications"}); !errors.Is(err, ErrPermissionDenied) {
		t.Fatalf("RequestPermission(denied) error = %v, want ErrPermissionDenied", err)
	}
}

func TestCookieStoreNormalizesAndRecordsChanges(t *testing.T) {
	s := NewRegistry().DefaultSession()
	if err := s.SetCookie(Cookie{URL: "https://example.test/path", Name: "sid", Value: "1", Secure: true}); err != nil {
		t.Fatalf("SetCookie() error = %v", err)
	}
	cookies := s.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("Cookies() len = %d, want 1", len(cookies))
	}
	if cookies[0].Domain != "example.test" || cookies[0].Path != "/" {
		t.Fatalf("normalized cookie = %#v", cookies[0])
	}
	if err := s.SetCookie(Cookie{URL: "https://example.test/other", Name: "sid", Value: "2"}); err != nil {
		t.Fatalf("SetCookie(overwrite) error = %v", err)
	}
	changes := s.CookieChanges()
	if len(changes) != 3 || changes[0].Cause != CookieChangeExplicit || changes[1].Cause != CookieChangeOverwrite || !changes[1].Removed || changes[2].Cause != CookieChangeExplicit {
		t.Fatalf("CookieChanges() = %#v", changes)
	}
	if err := s.DeleteCookie("https://example.test/", "sid"); err != nil {
		t.Fatalf("DeleteCookie() error = %v", err)
	}
	if len(s.Cookies()) != 0 {
		t.Fatalf("Cookies() after delete = %#v, want empty", s.Cookies())
	}
	changes = s.CookieChanges()
	if len(changes) != 4 || !changes[3].Removed || changes[3].Cause != CookieChangeExplicit {
		t.Fatalf("delete change = %#v", changes)
	}
}

func TestCookieValidation(t *testing.T) {
	s := NewRegistry().DefaultSession()
	cases := []Cookie{
		{URL: "missing-host", Name: "sid"},
		{URL: "https://example.test/", Name: ""},
		{URL: "https://example.test/", Name: "bad;name"},
		{URL: "http://example.test/", Name: "sid", Secure: true},
	}
	for _, tc := range cases {
		if err := s.SetCookie(tc); !errors.Is(err, ErrInvalidCookie) {
			t.Fatalf("SetCookie(%#v) error = %v, want ErrInvalidCookie", tc, err)
		}
	}
	if err := s.DeleteCookie("missing-host", "sid"); !errors.Is(err, ErrInvalidCookie) {
		t.Fatalf("DeleteCookie(invalid) error = %v, want ErrInvalidCookie", err)
	}
}

func TestClearCacheAndStorageData(t *testing.T) {
	s := NewRegistry().DefaultSession()
	s.ClearCache()
	if !s.CacheCleared() {
		t.Fatal("CacheCleared() = false, want true")
	}
	options := ClearStorageOptions{
		Storages: []string{"cookies", "localstorage"},
		Origins:  []string{"https://example.test"},
	}
	if err := s.SetCookie(Cookie{URL: "https://example.test/", Name: "clearme", Value: "1"}); err != nil {
		t.Fatalf("SetCookie(clearme) error = %v", err)
	}
	if err := s.SetCookie(Cookie{URL: "https://other.test/", Name: "keepme", Value: "1"}); err != nil {
		t.Fatalf("SetCookie(keepme) error = %v", err)
	}
	if err := s.ClearStorageData(options); err != nil {
		t.Fatalf("ClearStorageData() error = %v", err)
	}
	cookies := s.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "keepme" {
		t.Fatalf("Cookies() after origin clear = %#v, want only keepme", cookies)
	}
	options.Storages[0] = "mutated"
	clears := s.StorageClears()
	if len(clears) != 1 || clears[0].Storages[0] != "cookies" {
		t.Fatalf("StorageClears() = %#v", clears)
	}
	if err := s.ClearStorageData(ClearStorageOptions{Quotas: []string{"temporary"}}); err != nil {
		t.Fatalf("ClearStorageData(quotas) error = %v, want nil ignored option", err)
	}
	clears = s.StorageClears()
	if len(clears) != 2 || len(clears[1].Quotas) != 0 {
		t.Fatalf("StorageClears() after quotas = %#v, want ignored quotas", clears)
	}
	if err := s.ClearStorageData(ClearStorageOptions{Origins: []string{"not-an-origin"}}); !errors.Is(err, ErrUnsupportedClearOption) {
		t.Fatalf("ClearStorageData(invalid origin) error = %v, want ErrUnsupportedClearOption", err)
	}
}
