package compat

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed ledger.json
var ledgerFS embed.FS

type TargetVersions struct {
	Electron string `json:"electron"`
	Chromium string `json:"chromium"`
	Node     string `json:"node"`
	V8       string `json:"v8"`
}

type Ledger struct {
	Target     TargetVersions `json:"target"`
	Generated  string         `json:"generated"`
	Completion string         `json:"completion"`
	Items      []Item         `json:"items"`
}

type Item struct {
	ID       string   `json:"id"`
	Area     string   `json:"area"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
	Notes    string   `json:"notes"`
}

func Target() TargetVersions {
	return MustLoadLedger().Target
}

func MustLoadLedger() Ledger {
	ledger, err := LoadLedger()
	if err != nil {
		panic(err)
	}
	return ledger
}

func LoadLedger() (Ledger, error) {
	data, err := ledgerFS.ReadFile("ledger.json")
	if err != nil {
		return Ledger{}, fmt.Errorf("read compatibility ledger: %w", err)
	}
	var ledger Ledger
	if err := json.Unmarshal(data, &ledger); err != nil {
		return Ledger{}, fmt.Errorf("parse compatibility ledger: %w", err)
	}
	return ledger, nil
}

func (l Ledger) TotalCount() int {
	return len(l.Items)
}

func (l Ledger) CompatibleCount() int {
	total := 0
	for _, item := range l.Items {
		if item.Status == "compatible" {
			total++
		}
	}
	return total
}

func (l Ledger) IsComplete() bool {
	return l.TotalCount() > 0 && l.CompatibleCount() == l.TotalCount()
}
