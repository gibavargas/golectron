package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gibavargas/electron-go/internal/compat"
)

type packet struct {
	ID          string   `json:"id"`
	Area        string   `json:"area"`
	Status      string   `json:"status"`
	Evidence    []string `json:"evidence"`
	Notes       string   `json:"notes"`
	E2ERequired bool     `json:"e2e_required"`
	Acceptance  []string `json:"acceptance"`
	Prompt      string   `json:"prompt"`
}

func main() {
	status := flag.String("status", "", "comma-separated status filter")
	area := flag.String("area", "", "comma-separated area filter")
	ids := flag.String("id", "", "comma-separated ledger item ID filter")
	format := flag.String("format", "markdown", "output format: markdown or json")
	limit := flag.Int("limit", 0, "maximum packets to print")
	flag.Parse()

	ledger, err := compat.LoadLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "workpackets: %v\n", err)
		os.Exit(1)
	}

	statuses := filterSet(*status)
	areas := filterSet(*area)
	idSet := filterSet(*ids)
	packets := buildPackets(ledger, statuses, areas, idSet)
	packets = append(packets, buildObjectivePackets(statuses, areas, idSet)...)
	sortPackets(packets)
	if *limit > 0 && len(packets) > *limit {
		packets = packets[:*limit]
	}

	switch *format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(packets); err != nil {
			fmt.Fprintf(os.Stderr, "workpackets: encode: %v\n", err)
			os.Exit(1)
		}
	case "markdown":
		printMarkdown(ledger.Target, packets)
	default:
		fmt.Fprintf(os.Stderr, "workpackets: unsupported format %q\n", *format)
		os.Exit(2)
	}
}

func buildPackets(ledger compat.Ledger, statuses, areas, ids map[string]bool) []packet {
	packets := make([]packet, 0, len(ledger.Items))
	for _, item := range ledger.Items {
		if len(statuses) > 0 && !statuses[item.Status] {
			continue
		}
		if len(areas) > 0 && !areas[item.Area] {
			continue
		}
		if len(ids) > 0 && !ids[item.ID] {
			continue
		}
		packets = append(packets, packet{
			ID:          item.ID,
			Area:        item.Area,
			Status:      item.Status,
			Evidence:    item.Evidence,
			Notes:       item.Notes,
			E2ERequired: e2eRequired(item),
			Acceptance:  acceptanceFor(item),
			Prompt:      promptFor(item),
		})
	}

	sortPackets(packets)

	return packets
}

func buildObjectivePackets(statuses, areas, ids map[string]bool) []packet {
	objectivePackets := []packet{
		{
			ID:          "runtime-ipc-preload-conformance",
			Area:        "objective-runtime-parity",
			Status:      string(compat.StatusPartial),
			Evidence:    []string{"compat/fixtures/hello", "compat/conformance_test.go", "cmd/electron-go --runtime-parity-audit"},
			Notes:       "Unblocks TestHelloFixtureConformance and ELECTRON_GO_ENABLE_IPC_CONFORMANCE. Requires real preload, contextBridge, ipcRenderer.invoke, ipcMain.handle, and renderer-visible result parity against official Electron.",
			E2ERequired: true,
			Acceptance: []string{
				"Electron-Go executes the hello fixture main/preload/renderer flow rather than only direct-loading index.html.",
				"`ELECTRON_GO_ENABLE_IPC_CONFORMANCE=1 go test ./compat -run TestHelloFixtureConformance -count=1` passes against official Electron.",
				"`go run ./cmd/electron-go --runtime-parity-audit` no longer fails this gate.",
			},
			Prompt: "Implement the remaining hello fixture app-runtime parity. Preserve Electron 42.0.0 behavior for app.whenReady, BrowserWindow preload loading, contextIsolation+sandbox, contextBridge.exposeInMainWorld, ipcMain.handle, ipcRenderer.invoke, and renderer-visible output. Prove it against official Electron with TestHelloFixtureConformance before enabling ELECTRON_GO_ENABLE_IPC_CONFORMANCE.",
		},
		{
			ID:          "runtime-main-process-conformance",
			Area:        "objective-runtime-parity",
			Status:      string(compat.StatusPartial),
			Evidence:    []string{"compat/fixtures/benchmark-hello", "compat/conformance_test.go", "docs/benchmarks.md", "cmd/electron-go --runtime-parity-audit"},
			Notes:       "Unblocks TestBenchmarkHelloFixtureConformance and ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE. The current CEF path loads index.html directly and does not execute fixture main.js.",
			E2ERequired: true,
			Acceptance: []string{
				"Electron-Go executes the benchmark fixture main.js lifecycle instead of direct-loading index.html.",
				"The fixture observes app.whenReady, BrowserWindow construction, did-finish-load, close, and app.quit semantics comparable to official Electron.",
				"`ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE=1 go test ./compat -run TestBenchmarkHelloFixtureConformance -count=1` passes against official Electron.",
				"`go run ./cmd/electron-go --runtime-parity-audit` no longer fails this gate.",
			},
			Prompt: "Implement native Chromium/Node/V8 main-process runtime parity for the benchmark fixture. Do not hard-code the fixture outcome; execute the Electron-style main process lifecycle and prove it with TestBenchmarkHelloFixtureConformance before enabling ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE.",
		},
		{
			ID:          "performance-50-percent-startup-target",
			Area:        "objective-performance",
			Status:      string(compat.StatusPartial),
			Evidence:    []string{"docs/benchmarks.md", "cmd/electron-go/goal_evidence.json", "tools/goalevidence", "cmd/electron-go --goal-audit"},
			Notes:       "Latest tracked duration_median_ms ratio is 0.7388 on the scoped runtime path; the objective requires electron_go_over_electron <= 0.5 with full runtime parity still intact.",
			E2ERequired: true,
			Acceptance: []string{
				"Run a published alternating-order benchmark artifact against official Electron and Electron-Go with full runtime parity enabled.",
				"`duration_median_ms` comparison reports `electron_go_over_electron <= 0.5`.",
				"Update cmd/electron-go/goal_evidence.json via tools/goalevidence from that published artifact.",
				"`go run ./cmd/electron-go --goal-audit` passes without disabling runtime parity gates.",
			},
			Prompt: "Optimize Electron-Go startup only after preserving full runtime parity. Use published tools/benchmarks artifacts, update cmd/electron-go/goal_evidence.json via tools/goalevidence, and keep benchmark semantics comparable to official Electron. The target is duration_median_ms electron_go_over_electron <= 0.5.",
		},
	}
	filtered := objectivePackets[:0]
	for _, packet := range objectivePackets {
		if len(statuses) > 0 && !statuses[packet.Status] {
			continue
		}
		if len(areas) > 0 && !areas[packet.Area] {
			continue
		}
		if len(ids) > 0 && !ids[packet.ID] {
			continue
		}
		filtered = append(filtered, packet)
	}
	return filtered
}

func sortPackets(packets []packet) {
	sort.SliceStable(packets, func(i, j int) bool {
		if packets[i].Status != packets[j].Status {
			return rankStatus(packets[i].Status) < rankStatus(packets[j].Status)
		}
		if packets[i].Area != packets[j].Area {
			return packets[i].Area < packets[j].Area
		}
		return packets[i].ID < packets[j].ID
	})
}

func filterSet(raw string) map[string]bool {
	out := make(map[string]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out[part] = true
		}
	}
	return out
}

func promptFor(item compat.Item) string {
	if item.ID == "cef_bootstrap" {
		return "Implement Electron-Go CEF bootstrap parity. Preserve process-original OS main-thread ownership with runtime.LockOSThread at executable entry, route CEF subprocesses before normal app initialization, initialize CEF through the native C ABI, route callbacks through internal/native.Dispatcher, create one visible BrowserWindow, load the hello fixture file URL, run the CEF message loop, exit cleanly on window close with no zombie renderer processes, and keep JavaScript execution and IPC out of scope until this packet is proven."
	}
	return fmt.Sprintf("Implement Electron-Go parity for ledger item %q in area %q. Preserve Electron 42.0.0 behavior, add or extend a fixture under compat/fixtures, prove it with tools/conformance or go test ./compat against official Electron, update evidence only after e2e tests prove compatibility, and keep the ledger honest.", item.ID, item.Area)
}

func e2eRequired(item compat.Item) bool {
	return item.Status != compat.StatusCompatible || !hasE2EEvidence(item.EvidenceInfo)
}

func acceptanceFor(item compat.Item) []string {
	return []string{
		"Implement the runtime behavior behind the Electron API, not only internal state.",
		"Add or extend an Electron fixture under compat/fixtures for this ledger item.",
		"Run the fixture against official Electron and Electron-Go with tools/conformance or go test ./compat.",
		"Move the ledger item to compatible only after conformance passes and evidence includes implementation plus e2e/conformance refs.",
	}
}

func hasE2EEvidence(evidence []compat.Evidence) bool {
	for _, entry := range evidence {
		if entry.Kind == "conformance" {
			return true
		}
		if strings.HasPrefix(entry.Ref, "compat/") || strings.HasPrefix(entry.Ref, ".github/workflows/") {
			return true
		}
	}
	return false
}

func printMarkdown(target compat.TargetVersions, packets []packet) {
	fmt.Printf("# Electron-Go Work Packets\n\n")
	fmt.Printf("Target: Electron %s, Chromium %s, Node.js %s, V8 %s\n\n", target.Electron, target.Chromium, target.Node, target.V8)
	if len(packets) == 0 {
		fmt.Println("No packets matched the filters.")
		return
	}
	for _, packet := range packets {
		fmt.Printf("## %s\n\n", packet.ID)
		fmt.Printf("- Area: `%s`\n", packet.Area)
		fmt.Printf("- Status: `%s`\n", packet.Status)
		if len(packet.Evidence) > 0 {
			fmt.Printf("- Existing evidence: `%s`\n", strings.Join(packet.Evidence, "`, `"))
		}
		fmt.Printf("- E2E required: `%t`\n", packet.E2ERequired)
		if len(packet.Acceptance) > 0 {
			fmt.Printf("- Acceptance:\n")
			for _, item := range packet.Acceptance {
				fmt.Printf("  - %s\n", item)
			}
		}
		fmt.Printf("- Notes: %s\n", packet.Notes)
		fmt.Printf("- Agent prompt: %s\n\n", packet.Prompt)
	}
}

func rankStatus(status string) int {
	switch status {
	case "unstarted":
		return 0
	case "stubbed":
		return 1
	case "partial":
		return 2
	case "compatible":
		return 3
	default:
		return 4
	}
}
