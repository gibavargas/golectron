package compat

import "testing"

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
