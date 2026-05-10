package globalshortcut

import (
	"errors"
	"testing"
)

func TestRegistryRegisterInvokeAndUnregister(t *testing.T) {
	registry := NewRegistry()
	calls := 0
	registered, err := registry.Register("CommandOrControl+X", func() {
		calls++
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if !registered {
		t.Fatal("Register() = false, want true")
	}
	if !registry.IsRegistered("CommandOrControl+X") {
		t.Fatal("IsRegistered() = false, want true")
	}
	if !registry.Trigger("CommandOrControl+X") {
		t.Fatal("Trigger() = false, want true")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	registry.Unregister("CommandOrControl+X")
	if registry.IsRegistered("CommandOrControl+X") {
		t.Fatal("IsRegistered() = true after Unregister")
	}
	if registry.Trigger("CommandOrControl+X") {
		t.Fatal("Trigger() = true after Unregister")
	}
}

func TestRegistryRejectsDuplicateWithoutReplacingCallback(t *testing.T) {
	registry := NewRegistry()
	calls := 0
	if registered, err := registry.Register("Alt+Space", func() {
		calls += 1
	}); err != nil || !registered {
		t.Fatalf("first Register() = %v, %v, want true nil", registered, err)
	}
	if registered, err := registry.Register("Option+Space", func() {
		calls += 100
	}); err != nil || registered {
		t.Fatalf("second Register() = %v, %v, want false nil", registered, err)
	}
	registry.Trigger("Option+Space")
	if calls != 1 {
		t.Fatalf("calls = %d, want original callback only", calls)
	}
}

func TestRegistryUsesCanonicalAccelerators(t *testing.T) {
	registry := NewRegistry()
	var triggered string
	registry.SetTriggerHook(func(accelerator string) {
		triggered = accelerator
	})
	if registered, err := registry.Register("cmdorctrl+shift+p", func() {}); err != nil || !registered {
		t.Fatalf("Register() = %v, %v, want true nil", registered, err)
	}
	if !registry.IsRegistered("CommandOrControl+Shift+P") {
		t.Fatal("IsRegistered(canonical alias) = false, want true")
	}
	if !registry.Trigger("CommandOrControl+Shift+P") {
		t.Fatal("Trigger(canonical alias) = false, want true")
	}
	if triggered != "CommandOrControl+Shift+P" {
		t.Fatalf("triggered = %q, want canonical accelerator", triggered)
	}
	registry.Unregister("CommandOrControl+Shift+P")
	if registry.IsRegistered("cmdorctrl+shift+p") {
		t.Fatal("IsRegistered(alias) = true after canonical unregister")
	}
}

func TestRegistrySuspensionSuppressesCallbacksWithoutLosingRegistrations(t *testing.T) {
	registry := NewRegistry()
	calls := 0
	if registered, err := registry.Register("Control+Shift+P", func() {
		calls++
	}); err != nil || !registered {
		t.Fatalf("Register() = %v, %v, want true nil", registered, err)
	}

	registry.SetSuspended(true)
	if !registry.IsSuspended() {
		t.Fatal("IsSuspended() = false, want true")
	}
	if !registry.Trigger("Control+Shift+P") {
		t.Fatal("Trigger() = false while suspended, want true because registration remains")
	}
	if calls != 0 {
		t.Fatalf("calls = %d while suspended, want 0", calls)
	}
	if !registry.IsRegistered("Control+Shift+P") {
		t.Fatal("IsRegistered() = false while suspended, want true")
	}

	registry.SetSuspended(false)
	if registry.IsSuspended() {
		t.Fatal("IsSuspended() = true after resume")
	}
	registry.Trigger("Control+Shift+P")
	if calls != 1 {
		t.Fatalf("calls = %d after resume, want 1", calls)
	}
}

func TestRegistryUnregisterAllKeepsSuspensionState(t *testing.T) {
	registry := NewRegistry()
	registry.SetSuspended(true)
	registry.Register("A", func() {})
	registry.Register("B", func() {})

	registry.UnregisterAll()
	if registry.IsRegistered("A") || registry.IsRegistered("B") {
		t.Fatal("registered shortcuts remain after UnregisterAll")
	}
	if !registry.IsSuspended() {
		t.Fatal("UnregisterAll cleared suspension state")
	}
}

func TestRegistryValidation(t *testing.T) {
	registry := NewRegistry()
	if _, err := registry.Register(" ", func() {}); !errors.Is(err, ErrAcceleratorRequired) {
		t.Fatalf("Register(empty accelerator) error = %v, want ErrAcceleratorRequired", err)
	}
	if _, err := registry.Register("Shift+Shift+X", func() {}); !errors.Is(err, ErrInvalidAccelerator) {
		t.Fatalf("Register(invalid accelerator) error = %v, want ErrInvalidAccelerator", err)
	}
	if _, err := registry.Register("A", nil); !errors.Is(err, ErrCallbackRequired) {
		t.Fatalf("Register(nil callback) error = %v, want ErrCallbackRequired", err)
	}
	results := registry.RegisterAll([]string{"A", " ", "ctrl+b"}, func() {})
	if !results["A"] || results[""] || !results["Control+B"] {
		t.Fatalf("RegisterAll() = %#v", results)
	}
}
