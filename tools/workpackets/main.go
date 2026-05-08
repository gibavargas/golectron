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
	ID       string   `json:"id"`
	Area     string   `json:"area"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
	Notes    string   `json:"notes"`
	Prompt   string   `json:"prompt"`
}

func main() {
	status := flag.String("status", "", "comma-separated status filter")
	area := flag.String("area", "", "comma-separated area filter")
	format := flag.String("format", "markdown", "output format: markdown or json")
	limit := flag.Int("limit", 0, "maximum packets to print")
	flag.Parse()

	ledger, err := compat.LoadLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "workpackets: %v\n", err)
		os.Exit(1)
	}

	packets := buildPackets(ledger, filterSet(*status), filterSet(*area))
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

func buildPackets(ledger compat.Ledger, statuses, areas map[string]bool) []packet {
	packets := make([]packet, 0, len(ledger.Items))
	for _, item := range ledger.Items {
		if len(statuses) > 0 && !statuses[item.Status] {
			continue
		}
		if len(areas) > 0 && !areas[item.Area] {
			continue
		}
		packets = append(packets, packet{
			ID:       item.ID,
			Area:     item.Area,
			Status:   item.Status,
			Evidence: item.Evidence,
			Notes:    item.Notes,
			Prompt:   promptFor(item),
		})
	}

	sort.SliceStable(packets, func(i, j int) bool {
		if packets[i].Status != packets[j].Status {
			return rankStatus(packets[i].Status) < rankStatus(packets[j].Status)
		}
		if packets[i].Area != packets[j].Area {
			return packets[i].Area < packets[j].Area
		}
		return packets[i].ID < packets[j].ID
	})

	return packets
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
	return fmt.Sprintf("Implement Electron-Go parity for ledger item %q in area %q. Preserve Electron 42.0.0 behavior, add conformance tests against official Electron fixtures, update evidence only after tests prove compatibility, and keep the ledger honest.", item.ID, item.Area)
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
