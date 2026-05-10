package webcontents

import (
	"errors"
	"reflect"
	"testing"
)

func TestLoadURLEmitsDocumentNavigationOrder(t *testing.T) {
	wc := New(99)
	if wc.ID() != 99 {
		t.Fatalf("ID() = %d, want 99", wc.ID())
	}
	if err := wc.LoadURL("https://example.test/"); err != nil {
		t.Fatalf("LoadURL() error = %v", err)
	}
	want := []Event{
		EventDidStartNavigation,
		EventWillFrameNavigate,
		EventWillNavigate,
		EventDidStartLoading,
		EventDidFrameNavigate,
		EventDidNavigate,
		EventDidStopLoading,
		EventDidFinishLoad,
	}
	if got := wc.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if wc.URL() != "https://example.test/" || wc.IsLoading() {
		t.Fatalf("URL/loading = %q/%v", wc.URL(), wc.IsLoading())
	}
}

func TestLoadURLCanBeCancelledByWillNavigate(t *testing.T) {
	wc := New(1)
	wc.On(EventWillNavigate, func(ctx *EventContext) {
		if ctx.Details.URL != "https://example.test/" || !ctx.Details.IsMainFrame {
			t.Fatalf("will-navigate details = %#v", ctx.Details)
		}
		ctx.PreventDefault()
	})
	if err := wc.LoadURL("https://example.test/"); !errors.Is(err, ErrNavigationCancelled) {
		t.Fatalf("LoadURL() error = %v, want ErrNavigationCancelled", err)
	}
	if wc.URL() != "" || len(wc.History()) != 0 {
		t.Fatalf("cancelled navigation mutated state: url=%q history=%#v", wc.URL(), wc.History())
	}
	want := []Event{EventDidStartNavigation, EventWillFrameNavigate, EventWillNavigate}
	if got := wc.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
}

func TestNavigateInPageEmitsSameDocumentEvents(t *testing.T) {
	wc := New(1)
	if err := wc.LoadURL("https://example.test/page"); err != nil {
		t.Fatalf("LoadURL() error = %v", err)
	}
	wc.events = nil
	if err := wc.NavigateInPage("https://example.test/page#section"); err != nil {
		t.Fatalf("NavigateInPage() error = %v", err)
	}
	want := []Event{EventDidStartNavigation, EventDidNavigateInPage}
	if got := wc.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if wc.URL() != "https://example.test/page#section" {
		t.Fatalf("URL() = %q", wc.URL())
	}
}

func TestHistoryBackForwardAndTruncation(t *testing.T) {
	wc := New(1)
	for _, target := range []string{"https://example.test/1", "https://example.test/2", "https://example.test/3"} {
		if err := wc.LoadURL(target); err != nil {
			t.Fatalf("LoadURL(%s) error = %v", target, err)
		}
	}
	if !wc.CanGoBack() || wc.CanGoForward() {
		t.Fatalf("back/forward = %v/%v", wc.CanGoBack(), wc.CanGoForward())
	}
	if err := wc.GoBack(); err != nil {
		t.Fatalf("GoBack() error = %v", err)
	}
	if wc.URL() != "https://example.test/2" || !wc.CanGoForward() {
		t.Fatalf("after GoBack url=%q forward=%v", wc.URL(), wc.CanGoForward())
	}
	if err := wc.GoForward(); err != nil {
		t.Fatalf("GoForward() error = %v", err)
	}
	if wc.URL() != "https://example.test/3" {
		t.Fatalf("after GoForward url=%q", wc.URL())
	}
	if err := wc.GoBack(); err != nil {
		t.Fatalf("GoBack() error = %v", err)
	}
	if err := wc.LoadURL("https://example.test/4"); err != nil {
		t.Fatalf("LoadURL(4) error = %v", err)
	}
	if wc.CanGoForward() {
		t.Fatal("CanGoForward() = true after new navigation truncated forward history")
	}
	wantHistory := []string{"https://example.test/1", "https://example.test/2", "https://example.test/4"}
	if got := wc.History(); !reflect.DeepEqual(got, wantHistory) {
		t.Fatalf("History() = %#v, want %#v", got, wantHistory)
	}
}

func TestFailLoadAndValidation(t *testing.T) {
	wc := New(1)
	if err := wc.LoadURL("missing-scheme"); !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("LoadURL(invalid) error = %v, want ErrInvalidURL", err)
	}
	if err := wc.FailLoad("https://example.test/fail", -3, "aborted"); err != nil {
		t.Fatalf("FailLoad() error = %v", err)
	}
	want := []Event{EventDidStartLoading, EventDidStopLoading, EventDidFailLoad}
	if got := wc.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if wc.IsLoading() {
		t.Fatal("IsLoading() = true after FailLoad")
	}
	if err := wc.GoBack(); !errors.Is(err, ErrHistoryUnavailable) {
		t.Fatalf("GoBack(empty) error = %v, want ErrHistoryUnavailable", err)
	}
}

func TestDevToolsTargetIDIsStableAndLookupable(t *testing.T) {
	wc := New(42)
	first := wc.GetOrCreateDevToolsTargetID()
	second := wc.GetOrCreateDevToolsTargetID()
	if first == "" {
		t.Fatal("GetOrCreateDevToolsTargetID() returned empty string")
	}
	if first != second {
		t.Fatalf("target ids differ: %q != %q", first, second)
	}
	got, ok := FromDevToolsTargetID(first)
	if !ok {
		t.Fatal("FromDevToolsTargetID() ok = false, want true")
	}
	if got != wc {
		t.Fatal("FromDevToolsTargetID() returned a different WebContents")
	}
	if _, ok := FromDevToolsTargetID("missing"); ok {
		t.Fatal("FromDevToolsTargetID(missing) ok = true, want false")
	}
}

func TestPrintNormalizesOptionsAndRecordsJob(t *testing.T) {
	wc := New(7)
	size := &PageSize{WidthMicrons: 210000, HeightMicrons: 297000}
	job, err := wc.Print(PrintOptions{
		Silent:          true,
		PrintBackground: true,
		DeviceName:      "  Office Printer  ",
		Copies:          2,
		PageSize:        size,
	})
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if job.WebContentsID != 7 || !job.Silent || !job.PrintBackground {
		t.Fatalf("job flags/id = %#v", job)
	}
	if job.DeviceName != "Office Printer" || job.Copies != 2 {
		t.Fatalf("job device/copies = %q/%d", job.DeviceName, job.Copies)
	}
	if job.PageSize == nil || *job.PageSize != *size {
		t.Fatalf("job PageSize = %#v, want %#v", job.PageSize, size)
	}
	size.WidthMicrons = 1
	stored, ok := wc.LastPrintJob()
	if !ok {
		t.Fatal("LastPrintJob() ok = false, want true")
	}
	if stored.PageSize.WidthMicrons != 210000 {
		t.Fatalf("LastPrintJob() retained caller-owned page size: %#v", stored.PageSize)
	}
}

func TestPrintSupportsPrinterDefaultPageSize(t *testing.T) {
	wc := New(1)
	job, err := wc.Print(PrintOptions{UsePrinterDefaultPageSize: true})
	if err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if !job.UsePrinterDefaultPageSize {
		t.Fatal("UsePrinterDefaultPageSize = false, want true")
	}
	if job.PageSize != nil {
		t.Fatalf("PageSize = %#v, want nil when printer default is requested", job.PageSize)
	}
	if job.Copies != 1 {
		t.Fatalf("Copies = %d, want default 1", job.Copies)
	}
}

func TestPrintRejectsInvalidOptions(t *testing.T) {
	wc := New(1)
	cases := []PrintOptions{
		{Copies: -1},
		{PageSize: &PageSize{WidthMicrons: 0, HeightMicrons: 1}},
		{PageSize: &PageSize{WidthMicrons: 1, HeightMicrons: -1}},
		{UsePrinterDefaultPageSize: true, PageSize: &PageSize{WidthMicrons: 1, HeightMicrons: 1}},
	}
	for _, tc := range cases {
		if _, err := wc.Print(tc); !errors.Is(err, ErrInvalidPrintOptions) {
			t.Fatalf("Print(%#v) error = %v, want ErrInvalidPrintOptions", tc, err)
		}
	}
}
