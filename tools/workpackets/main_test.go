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
	if !packets[1].E2ERequired {
		t.Fatal("partial packet E2ERequired = false, want true")
	}
	if len(packets[1].Acceptance) == 0 || !strings.Contains(strings.Join(packets[1].Acceptance, " "), "compat/fixtures") {
		t.Fatalf("acceptance does not mention compat fixtures: %#v", packets[1].Acceptance)
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

func TestBuildPacketsMarksCompatibleWithoutE2EAsRequired(t *testing.T) {
	ledger := compat.Ledger{
		Items: []compat.Item{
			{ID: "unit-only", Area: "testing", Status: compat.StatusCompatible, Evidence: []string{"internal/unit"}, EvidenceInfo: []compat.Evidence{{Kind: "test", Ref: "internal/unit_test.go"}}, Notes: "unit only"},
			{ID: "conformance", Area: "testing", Status: compat.StatusCompatible, Evidence: []string{"compat/fixture"}, EvidenceInfo: []compat.Evidence{{Kind: "conformance", Ref: "compat/fixture"}}, Notes: "e2e"},
		},
	}
	packets := buildPackets(ledger, nil, nil, nil)
	if len(packets) != 2 {
		t.Fatalf("len(packets) = %d, want 2", len(packets))
	}
	var unitOnly, conformance packet
	for _, packet := range packets {
		if packet.ID == "unit-only" {
			unitOnly = packet
		}
		if packet.ID == "conformance" {
			conformance = packet
		}
	}
	if !unitOnly.E2ERequired {
		t.Fatal("unit-only E2ERequired = false, want true")
	}
	if conformance.E2ERequired {
		t.Fatal("conformance E2ERequired = true, want false")
	}
}
