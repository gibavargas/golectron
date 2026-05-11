package compat

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed ledger.json
var ledgerFS embed.FS

type TargetVersions struct {
	Electron  string `json:"electron"`
	Chromium  string `json:"chromium"`
	Node      string `json:"node"`
	V8        string `json:"v8"`
	V8Process string `json:"v8_process,omitempty"`
	Modules   string `json:"modules,omitempty"`
}

var staticTarget = TargetVersions{
	Electron:  "42.0.0",
	Chromium:  "148.0.7778.96",
	Node:      "24.15.0",
	V8:        "14.8.178.14",
	V8Process: "14.8.178.14-electron.0",
	Modules:   "146",
}

type Ledger struct {
	Target     TargetVersions `json:"target"`
	Generated  string         `json:"generated"`
	Completion string         `json:"completion"`
	Items      []Item         `json:"items"`
}

type Status = string

const (
	StatusUnstarted  Status = "unstarted"
	StatusStubbed    Status = "stubbed"
	StatusPartial    Status = "partial"
	StatusCompatible Status = "compatible"
)

type Item struct {
	ID           string     `json:"id"`
	Area         string     `json:"area"`
	APIGroup     string     `json:"api_group"`
	Status       Status     `json:"status"`
	Evidence     []string   `json:"evidence"`
	EvidenceInfo []Evidence `json:"evidence_details,omitempty"`
	ElectronRefs []string   `json:"electron_refs,omitempty"`
	Notes        string     `json:"notes"`
}

type E2EAudit struct {
	Target     TargetVersions `json:"target"`
	Pass       bool           `json:"pass"`
	Missing    []string       `json:"missing_e2e_evidence"`
	Incomplete []string       `json:"incomplete_items"`
}

type Evidence struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
	Note string `json:"note,omitempty"`
}

func (item *Item) UnmarshalJSON(data []byte) error {
	type rawItem struct {
		ID           string          `json:"id"`
		Area         string          `json:"area"`
		APIGroup     string          `json:"api_group"`
		Status       Status          `json:"status"`
		Evidence     json.RawMessage `json:"evidence"`
		EvidenceInfo []Evidence      `json:"evidence_details"`
		ElectronRefs []string        `json:"electron_refs,omitempty"`
		Notes        string          `json:"notes"`
	}
	var raw rawItem
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	item.ID = raw.ID
	item.Area = raw.Area
	item.APIGroup = raw.APIGroup
	item.Status = raw.Status
	item.ElectronRefs = raw.ElectronRefs
	item.Notes = raw.Notes
	item.EvidenceInfo = raw.EvidenceInfo

	var refs []string
	if err := json.Unmarshal(raw.Evidence, &refs); err == nil {
		item.Evidence = refs
		return nil
	}

	var details []Evidence
	if err := json.Unmarshal(raw.Evidence, &details); err != nil {
		return fmt.Errorf("parse item %q evidence: %w", raw.ID, err)
	}
	item.EvidenceInfo = append(item.EvidenceInfo, details...)
	item.Evidence = make([]string, 0, len(details))
	for _, detail := range details {
		item.Evidence = append(item.Evidence, detail.Ref)
	}
	return nil
}

func Target() TargetVersions {
	return staticTarget
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
	if err := ledger.Validate(); err != nil {
		return Ledger{}, err
	}
	return ledger, nil
}

func (l Ledger) Validate() error {
	if strings.TrimSpace(l.Target.Electron) == "" {
		return fmt.Errorf("compatibility ledger target electron version is empty")
	}
	if strings.TrimSpace(l.Target.Chromium) == "" {
		return fmt.Errorf("compatibility ledger target chromium version is empty")
	}
	if strings.TrimSpace(l.Target.Node) == "" {
		return fmt.Errorf("compatibility ledger target node version is empty")
	}
	if strings.TrimSpace(l.Target.V8) == "" {
		return fmt.Errorf("compatibility ledger target v8 version is empty")
	}
	if len(l.Items) == 0 {
		return fmt.Errorf("compatibility ledger has no items")
	}
	if l.Completion != "complete" && l.Completion != "incomplete" {
		return fmt.Errorf("compatibility ledger completion has invalid value %q", l.Completion)
	}

	seen := make(map[string]struct{}, len(l.Items))
	for i, item := range l.Items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("compatibility ledger item %d has empty id", i)
		}
		if _, ok := seen[item.ID]; ok {
			return fmt.Errorf("compatibility ledger item %q is duplicated", item.ID)
		}
		seen[item.ID] = struct{}{}
		if strings.TrimSpace(item.Area) == "" {
			return fmt.Errorf("compatibility ledger item %q has empty area", item.ID)
		}
		if strings.TrimSpace(item.APIGroup) == "" {
			return fmt.Errorf("compatibility ledger item %q has empty api_group", item.ID)
		}
		if !validStatus(item.Status) {
			return fmt.Errorf("compatibility ledger item %q has invalid status %q", item.ID, item.Status)
		}
		if len(item.Evidence) == 0 {
			return fmt.Errorf("compatibility ledger item %q has no evidence", item.ID)
		}
		if item.Status == StatusCompatible && !hasImplementationEvidence(item.EvidenceInfo) {
			return fmt.Errorf("compatibility ledger item %q is compatible without implementation evidence", item.ID)
		}
		if item.Status == StatusCompatible && !hasVerificationEvidence(item.EvidenceInfo) {
			return fmt.Errorf("compatibility ledger item %q is compatible without test or conformance evidence", item.ID)
		}
		for j, evidence := range item.Evidence {
			if strings.TrimSpace(evidence) == "" {
				return fmt.Errorf("compatibility ledger item %q evidence %d has empty ref", item.ID, j)
			}
		}
		for j, evidence := range item.EvidenceInfo {
			if strings.TrimSpace(evidence.Kind) == "" {
				return fmt.Errorf("compatibility ledger item %q evidence_details %d has empty kind", item.ID, j)
			}
			if strings.TrimSpace(evidence.Ref) == "" {
				return fmt.Errorf("compatibility ledger item %q evidence_details %d has empty ref", item.ID, j)
			}
		}
		if strings.TrimSpace(item.Notes) == "" {
			return fmt.Errorf("compatibility ledger item %q has empty notes", item.ID)
		}
	}
	if l.IsComplete() && l.Completion != "complete" {
		return fmt.Errorf("compatibility ledger completion is %q but all items are compatible", l.Completion)
	}
	if !l.IsComplete() && l.Completion != "incomplete" {
		return fmt.Errorf("compatibility ledger completion is %q but only %d/%d items are compatible", l.Completion, l.CompatibleCount(), l.TotalCount())
	}
	return nil
}

func (l Ledger) TotalCount() int {
	return len(l.Items)
}

func (l Ledger) CompatibleCount() int {
	total := 0
	for _, item := range l.Items {
		if item.Status == StatusCompatible {
			total++
		}
	}
	return total
}

func (l Ledger) IsComplete() bool {
	return l.TotalCount() > 0 && l.CompatibleCount() == l.TotalCount()
}

func (l Ledger) AuditE2EEvidence() E2EAudit {
	audit := E2EAudit{
		Target:     l.Target,
		Missing:    []string{},
		Incomplete: []string{},
	}
	for _, item := range l.Items {
		if item.Status != StatusCompatible {
			audit.Incomplete = append(audit.Incomplete, item.ID)
			continue
		}
		if !hasE2EEvidence(item.EvidenceInfo) {
			audit.Missing = append(audit.Missing, item.ID)
		}
	}
	audit.Pass = len(audit.Missing) == 0 && len(audit.Incomplete) == 0
	return audit
}

func validStatus(status Status) bool {
	switch status {
	case StatusUnstarted, StatusStubbed, StatusPartial, StatusCompatible:
		return true
	default:
		return false
	}
}

func hasImplementationEvidence(evidence []Evidence) bool {
	for _, entry := range evidence {
		switch entry.Kind {
		case "implementation":
			return true
		}
	}
	return false
}

func hasVerificationEvidence(evidence []Evidence) bool {
	for _, entry := range evidence {
		switch entry.Kind {
		case "test", "conformance":
			return true
		}
	}
	return false
}

func hasE2EEvidence(evidence []Evidence) bool {
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
