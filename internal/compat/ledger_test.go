package compat

import (
	"strings"
	"testing"
)

func TestLedgerLoadsElectron42Baseline(t *testing.T) {
	ledger, err := LoadLedger()
	if err != nil {
		t.Fatalf("LoadLedger() error = %v", err)
	}
	if ledger.Target.Electron != "42.0.0" {
		t.Fatalf("Electron = %q, want 42.0.0", ledger.Target.Electron)
	}
	if ledger.Target.Chromium == "" || ledger.Target.Node == "" || ledger.Target.V8 == "" {
		t.Fatalf("target versions must all be populated: %#v", ledger.Target)
	}
	if ledger.TotalCount() == 0 {
		t.Fatal("ledger has no items")
	}
	if ledger.IsComplete() {
		t.Fatal("initial ledger must not claim full parity")
	}
}

func TestLedgerItemsHaveImplementationDrivingMetadata(t *testing.T) {
	ledger, err := LoadLedger()
	if err != nil {
		t.Fatalf("LoadLedger() error = %v", err)
	}
	if err := ledger.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	for _, item := range ledger.Items {
		if item.ID == "" || item.Area == "" || item.APIGroup == "" {
			t.Fatalf("item must identify id, area, and api_group: %#v", item)
		}
		if item.Status == "" {
			t.Fatalf("item %q has empty status", item.ID)
		}
		if len(item.Evidence) == 0 {
			t.Fatalf("item %q has no evidence", item.ID)
		}
		for _, evidence := range item.Evidence {
			if strings.TrimSpace(evidence) == "" {
				t.Fatalf("item %q has incomplete evidence: %#v", item.ID, evidence)
			}
		}
		for _, evidence := range item.EvidenceInfo {
			if strings.TrimSpace(evidence.Kind) == "" || strings.TrimSpace(evidence.Ref) == "" {
				t.Fatalf("item %q has incomplete evidence detail: %#v", item.ID, evidence)
			}
		}
	}
}

func TestValidateRejectsInvalidLedgerItems(t *testing.T) {
	valid := Ledger{
		Target: TargetVersions{
			Electron: "42.0.0",
			Chromium: "148.0.7778.96",
			Node:     "24.15.0",
			V8:       "14.8.178.14",
		},
		Items: []Item{{
			ID:       "app-lifecycle",
			Area:     "main-process-api",
			APIGroup: "app",
			Status:   StatusUnstarted,
			Evidence: []string{"electron/docs/api/app.md"},
			EvidenceInfo: []Evidence{{
				Kind: "upstream",
				Ref:  "electron/docs/api/app.md",
			}},
			Notes: "Inventory placeholder for Electron app API parity.",
		}},
	}

	tests := []struct {
		name   string
		mutate func(*Ledger)
		want   string
	}{
		{
			name: "empty id",
			mutate: func(l *Ledger) {
				l.Items[0].ID = ""
			},
			want: "empty id",
		},
		{
			name: "empty area",
			mutate: func(l *Ledger) {
				l.Items[0].Area = ""
			},
			want: "empty area",
		},
		{
			name: "empty api group",
			mutate: func(l *Ledger) {
				l.Items[0].APIGroup = ""
			},
			want: "empty api_group",
		},
		{
			name: "invalid status",
			mutate: func(l *Ledger) {
				l.Items[0].Status = Status("done")
			},
			want: "invalid status",
		},
		{
			name: "missing evidence",
			mutate: func(l *Ledger) {
				l.Items[0].Evidence = nil
			},
			want: "no evidence",
		},
		{
			name: "empty evidence ref",
			mutate: func(l *Ledger) {
				l.Items[0].Evidence[0] = ""
			},
			want: "empty ref",
		},
		{
			name: "compatible needs implementation proof",
			mutate: func(l *Ledger) {
				l.Items[0].Status = StatusCompatible
			},
			want: "without implementation evidence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := valid
			ledger.Items = append([]Item(nil), valid.Items...)
			ledger.Items[0].Evidence = append([]string(nil), valid.Items[0].Evidence...)
			ledger.Items[0].EvidenceInfo = append([]Evidence(nil), valid.Items[0].EvidenceInfo...)
			tt.mutate(&ledger)
			err := ledger.Validate()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
