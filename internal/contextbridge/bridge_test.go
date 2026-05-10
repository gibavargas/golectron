package contextbridge

import (
	"errors"
	"testing"
)

func TestExposeInMainWorldCopiesImmutableAPI(t *testing.T) {
	bridge := New()
	api := Value{Kind: ValueObject, Object: map[string]Value{
		"version": {Kind: ValueString, String: "1.0.0"},
		"send":    {Kind: ValueFunction, Function: "send"},
		"flags":   {Kind: ValueArray, Array: []Value{{Kind: ValueBool, Bool: true}}},
	}}
	if err := bridge.ExposeInMainWorld("app", api); err != nil {
		t.Fatalf("ExposeInMainWorld() error = %v", err)
	}
	api.Object["version"] = Value{Kind: ValueString, String: "mutated"}
	exposed, ok := bridge.Exposure(0, "app")
	if !ok {
		t.Fatal("Exposure() ok = false, want true")
	}
	if exposed.Object["version"].String != "1.0.0" {
		t.Fatalf("exposed version = %q, want 1.0.0", exposed.Object["version"].String)
	}
	exposed.Object["version"] = Value{Kind: ValueString, String: "caller mutation"}
	again, _ := bridge.Exposure(0, "app")
	if again.Object["version"].String != "1.0.0" {
		t.Fatalf("Exposure() returned mutable state: %q", again.Object["version"].String)
	}
	if err := bridge.MutateExposure(0, "app", Value{Kind: ValueNull}); !errors.Is(err, ErrMutationNotAllowed) {
		t.Fatalf("MutateExposure() error = %v, want ErrMutationNotAllowed", err)
	}
}

func TestExposeInIsolatedWorld(t *testing.T) {
	bridge := New()
	if err := bridge.ExposeInIsolatedWorld(999, "api", Value{Kind: ValueString, String: "ok"}); err != nil {
		t.Fatalf("ExposeInIsolatedWorld() error = %v", err)
	}
	if _, ok := bridge.Exposure(0, "api"); ok {
		t.Fatal("main world unexpectedly has isolated exposure")
	}
	if value, ok := bridge.Exposure(999, "api"); !ok || value.String != "ok" {
		t.Fatalf("isolated exposure = %#v ok=%v", value, ok)
	}
}

func TestExposeRejectsInvalidInputs(t *testing.T) {
	bridge := New()
	cases := []struct {
		name    string
		worldID int
		key     string
		value   Value
		want    error
	}{
		{name: "world", worldID: -1, key: "api", value: Value{Kind: ValueNull}, want: ErrInvalidWorld},
		{name: "key", key: "bad.key", value: Value{Kind: ValueNull}, want: ErrInvalidAPIKey},
		{name: "kind", key: "api", value: Value{Kind: "channel"}, want: ErrInvalidValue},
		{name: "function", key: "api", value: Value{Kind: ValueFunction}, want: ErrInvalidValue},
		{name: "object key", key: "api", value: Value{Kind: ValueObject, Object: map[string]Value{"": {Kind: ValueNull}}}, want: ErrInvalidValue},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := bridge.ExposeInIsolatedWorld(tc.worldID, tc.key, tc.value); !errors.Is(err, tc.want) {
				t.Fatalf("ExposeInIsolatedWorld() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestExposeRejectsDuplicateAndPostPreloadExposure(t *testing.T) {
	bridge := New()
	if err := bridge.ExposeInMainWorld("api", Value{Kind: ValueNull}); err != nil {
		t.Fatalf("ExposeInMainWorld() error = %v", err)
	}
	if err := bridge.ExposeInMainWorld("api", Value{Kind: ValueNull}); !errors.Is(err, ErrAlreadyExposed) {
		t.Fatalf("ExposeInMainWorld(duplicate) error = %v, want ErrAlreadyExposed", err)
	}
	bridge.CompletePreload()
	if err := bridge.ExposeInMainWorld("other", Value{Kind: ValueNull}); !errors.Is(err, ErrPreloadOrder) {
		t.Fatalf("ExposeInMainWorld(after preload) error = %v, want ErrPreloadOrder", err)
	}
}
