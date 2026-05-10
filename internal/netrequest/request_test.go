package netrequest

import (
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestNewClientRequestNormalizesOptions(t *testing.T) {
	body := &UploadBody{ContentType: " application/json ", Bytes: []byte(`{"ok":true}`)}
	req, err := NewClientRequest(RequestOptions{
		Method:           " post ",
		URL:              "https://example.test/path",
		Headers:          http.Header{"X-Test": []string{"one"}},
		SessionPartition: " persist:profile ",
		RedirectPolicy:   RedirectManual,
		UploadBody:       body,
	})
	if err != nil {
		t.Fatalf("NewClientRequest() error = %v", err)
	}
	if req.Method != http.MethodPost || req.URL != "https://example.test/path" {
		t.Fatalf("method/url = %q/%q", req.Method, req.URL)
	}
	if req.Headers.Get("X-Test") != "one" || req.SessionPartition != "persist:profile" || req.RedirectPolicy != RedirectManual {
		t.Fatalf("request = %#v", req)
	}
	if req.UploadBody == nil || req.UploadBody.ContentType != "application/json" || string(req.UploadBody.Bytes) != `{"ok":true}` {
		t.Fatalf("upload = %#v", req.UploadBody)
	}
	body.Bytes[0] = '['
	if string(req.UploadBody.Bytes) != `{"ok":true}` {
		t.Fatal("upload body was not defensively copied")
	}
	req.End()
	if !req.Ended() {
		t.Fatal("Ended() = false after End")
	}
	parsed, err := req.ParsedURL()
	if err != nil {
		t.Fatalf("ParsedURL() error = %v", err)
	}
	if parsed.Path != "/path" {
		t.Fatalf("ParsedURL().Path = %q, want /path", parsed.Path)
	}
}

func TestNewClientRequestDefaults(t *testing.T) {
	req, err := NewClientRequest(RequestOptions{URL: "https://example.test/"})
	if err != nil {
		t.Fatalf("NewClientRequest() error = %v", err)
	}
	if req.Method != http.MethodGet || req.RedirectPolicy != RedirectFollow {
		t.Fatalf("defaults = method %q redirect %q", req.Method, req.RedirectPolicy)
	}
}

func TestNewClientRequestRejectsInvalidOptions(t *testing.T) {
	tests := []struct {
		name string
		opts RequestOptions
		want error
	}{
		{name: "method", opts: RequestOptions{Method: "bad method", URL: "https://example.test/"}, want: ErrInvalidRequest},
		{name: "url", opts: RequestOptions{URL: "missing-host"}, want: ErrInvalidRequest},
		{name: "redirect", opts: RequestOptions{URL: "https://example.test/", RedirectPolicy: "sometimes"}, want: ErrInvalidRequest},
		{name: "header name", opts: RequestOptions{URL: "https://example.test/", Headers: http.Header{"Bad:Name": []string{"ok"}}}, want: ErrInvalidHeader},
		{name: "header value", opts: RequestOptions{URL: "https://example.test/", Headers: http.Header{"X-Test": []string{"bad\r\nvalue"}}}, want: ErrInvalidHeader},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewClientRequest(tt.opts); !errors.Is(err, tt.want) {
				t.Fatalf("NewClientRequest() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHeadersAreDefensivelyCopied(t *testing.T) {
	headers := http.Header{"X-Test": []string{"one"}}
	req, err := NewClientRequest(RequestOptions{URL: "https://example.test/", Headers: headers})
	if err != nil {
		t.Fatalf("NewClientRequest() error = %v", err)
	}
	headers.Set("X-Test", "mutated")
	if got := req.Headers.Values("X-Test"); !reflect.DeepEqual(got, []string{"one"}) {
		t.Fatalf("headers = %#v, want original copy", got)
	}
}

func TestNetLogLifecycle(t *testing.T) {
	var log NetLog
	if log.IsCurrentlyLogging() {
		t.Fatal("IsCurrentlyLogging() = true before start")
	}
	if _, err := log.StopLogging(); !errors.Is(err, ErrNetLogInactive) {
		t.Fatalf("StopLogging(inactive) error = %v, want ErrNetLogInactive", err)
	}
	if err := log.StartLogging(" /tmp/netlog.json "); err != nil {
		t.Fatalf("StartLogging() error = %v", err)
	}
	if !log.IsCurrentlyLogging() {
		t.Fatal("IsCurrentlyLogging() = false after start")
	}
	if err := log.StartLogging("/tmp/other.json"); !errors.Is(err, ErrNetLogActive) {
		t.Fatalf("StartLogging(active) error = %v, want ErrNetLogActive", err)
	}
	path, err := log.StopLogging()
	if err != nil {
		t.Fatalf("StopLogging() error = %v", err)
	}
	if path != "/tmp/netlog.json" {
		t.Fatalf("path = %q, want /tmp/netlog.json", path)
	}
	wantEvents := []string{"start:/tmp/netlog.json", "stop:/tmp/netlog.json"}
	if got := log.Events(); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("Events() = %#v, want %#v", got, wantEvents)
	}
}
