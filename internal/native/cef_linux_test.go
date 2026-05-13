//go:build linux && (amd64 || arm64) && cgo && electron_go_cef

package native

import (
	"strings"
	"testing"
)

func TestAppendCEFBrowserProcessSwitchesFastPath(t *testing.T) {
	got := appendCEFBrowserProcessSwitches([]string{"electron-go", "compat/fixtures/benchmark-hello"})

	if len(got) != 2+len(cefBrowserProcessSwitches)+1 {
		t.Fatalf("switch count = %d, want %d: %#v", len(got), 2+len(cefBrowserProcessSwitches)+1, got)
	}
	for _, want := range cefBrowserProcessSwitches {
		if !hasSwitch(got, want) {
			t.Fatalf("missing switch %q in %#v", want, got)
		}
	}
	if !hasSwitch(got, "--browser-subprocess-path") {
		t.Fatalf("missing browser subprocess path in %#v", got)
	}
}

func TestAppendCEFBrowserProcessSwitchesPreservesOverrides(t *testing.T) {
	args := []string{
		"electron-go",
		"--disable-background-networking=0",
		"--browser-subprocess-path=/tmp/custom-helper",
	}
	got := appendCEFBrowserProcessSwitches(args)

	if countSwitch(got, "--disable-background-networking") != 1 {
		t.Fatalf("disable-background-networking count = %d in %#v", countSwitch(got, "--disable-background-networking"), got)
	}
	if countSwitch(got, "--browser-subprocess-path") != 1 {
		t.Fatalf("browser-subprocess-path count = %d in %#v", countSwitch(got, "--browser-subprocess-path"), got)
	}
	for _, want := range cefBrowserProcessSwitches {
		if !hasSwitch(got, want) {
			t.Fatalf("missing default switch %q in %#v", want, got)
		}
	}
}

func countSwitch(args []string, name string) int {
	count := 0
	for _, arg := range args {
		if arg == name || strings.HasPrefix(arg, name+"=") {
			count++
		}
	}
	return count
}
