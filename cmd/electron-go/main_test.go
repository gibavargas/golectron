package main

import "testing"

func TestRunHelloUsesFixture(t *testing.T) {
	code := run([]string{"electron-go", "--hello"}, nil)
	if code != 78 {
		t.Fatalf("run(--hello) exit = %d, want bridge-unavailable 78", code)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	code := run([]string{"electron-go", "--not-a-real-flag"}, nil)
	if code != 2 {
		t.Fatalf("run(unknown flag) exit = %d, want 2", code)
	}
}
