package nativetheme

import (
	"context"
	"runtime"
	"testing"
)

func TestSnapshotReportsDifferentiateWithoutColorSupport(t *testing.T) {
	report, err := Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if report.Platform == "" {
		t.Fatal("Platform is empty")
	}
	if runtime.GOOS == "darwin" && !report.SupportsDifferentiateWithoutColor {
		t.Fatal("darwin shouldDifferentiateWithoutColor support is disabled")
	}
	if runtime.GOOS != "darwin" && report.SupportsDifferentiateWithoutColor {
		t.Fatalf("SupportsDifferentiateWithoutColor = true on %s, want false", runtime.GOOS)
	}
	if !report.SupportsDifferentiateWithoutColor && report.ShouldDifferentiateWithoutColor {
		t.Fatal("unsupported platform reported ShouldDifferentiateWithoutColor=true")
	}
}

func TestSnapshotHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Snapshot(ctx); err == nil {
		t.Fatal("Snapshot(cancelled context) error = nil, want error")
	}
}
