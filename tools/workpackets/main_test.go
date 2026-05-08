package main

import (
	"strings"
	"testing"

	"github.com/gibavargas/electron-go/internal/compat"
)

func TestBuildPacketsFiltersAndOrdersByStatus(t *testing.T) {
	ledger := compat.Ledger{
		Items: []compat.Item{
			{ID: "partial-ipc", Area: "ipc", Status: compat.StatusPartial, Evidence: []string{"internal/ipc"}, Notes: "partial"},
			{ID: "unstarted-app", Area: "main-process-api", Status: compat.StatusUnstarted, Evidence: []string{"electron/docs/api/app.md"}, Notes: "unstarted"},
			{ID: "stub-runtime", Area: "runtime", Status: compat.StatusStubbed, Evidence: []string{"internal/runtime"}, Notes: "stubbed"},
		},
	}

	packets := buildPackets(ledger, filterSet("unstarted,partial"), nil, nil)
	if len(packets) != 2 {
		t.Fatalf("len(packets) = %d, want 2", len(packets))
	}
	if packets[0].ID != "unstarted-app" {
		t.Fatalf("first packet = %q, want unstarted-app", packets[0].ID)
	}
	if packets[1].ID != "partial-ipc" {
		t.Fatalf("second packet = %q, want partial-ipc", packets[1].ID)
	}
	if packets[0].Prompt == "" {
		t.Fatal("prompt is empty")
	}
}

func TestBuildPacketsAreaFilter(t *testing.T) {
	ledger := compat.Ledger{
		Items: []compat.Item{
			{ID: "ipc", Area: "ipc", Status: compat.StatusPartial, Evidence: []string{"internal/ipc"}, Notes: "partial"},
			{ID: "native", Area: "native", Status: compat.StatusStubbed, Evidence: []string{"internal/native"}, Notes: "stubbed"},
		},
	}

	packets := buildPackets(ledger, nil, filterSet("native"), nil)
	if len(packets) != 1 || packets[0].ID != "native" {
		t.Fatalf("packets = %#v, want only native", packets)
	}
}

func TestBuildPacketsIDFilterAndCEFBootstrapPrompt(t *testing.T) {
	ledger := compat.Ledger{
		Items: []compat.Item{
			{ID: "cef_bootstrap", Area: "chromium-integration", Status: compat.StatusUnstarted, Evidence: []string{"docs/architecture.md"}, Notes: "first boss"},
			{ID: "ipc", Area: "ipc", Status: compat.StatusPartial, Evidence: []string{"internal/ipc"}, Notes: "partial"},
		},
	}

	packets := buildPackets(ledger, nil, nil, filterSet("cef_bootstrap"))
	if len(packets) != 1 {
		t.Fatalf("len(packets) = %d, want 1", len(packets))
	}
	if packets[0].ID != "cef_bootstrap" {
		t.Fatalf("packet ID = %q, want cef_bootstrap", packets[0].ID)
	}
	if !strings.Contains(packets[0].Prompt, "runtime.LockOSThread") {
		t.Fatalf("CEF bootstrap prompt does not mention thread ownership: %q", packets[0].Prompt)
	}
}
