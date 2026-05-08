package main

import (
	"testing"
)

func TestSafeJoinRejectsTraversal(t *testing.T) {
	if _, err := safeJoin("/tmp/out", "../escape"); err == nil {
		t.Fatal("safeJoin() error = nil, want traversal error")
	}
}

func TestSafeJoinAllowsNestedRelativePath(t *testing.T) {
	got, err := safeJoin("/tmp/out", "cef_binary/Release/libcef.so")
	if err != nil {
		t.Fatalf("safeJoin() error = %v", err)
	}
	if got == "" {
		t.Fatal("safeJoin() returned empty path")
	}
}

func TestValidateLinkTargetRejectsTraversal(t *testing.T) {
	if err := validateLinkTarget("../../escape"); err == nil {
		t.Fatal("validateLinkTarget() error = nil, want traversal error")
	}
}
