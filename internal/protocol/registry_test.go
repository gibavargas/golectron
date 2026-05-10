package protocol

import (
	"errors"
	"net/http"
	"testing"
)

func TestRegisterSchemesAsPrivileged(t *testing.T) {
	registry := NewRegistry()
	err := registry.RegisterSchemesAsPrivileged([]SchemePrivilege{
		{Scheme: "app", Privileges: Privileges{Standard: true, Secure: true, SupportFetchAPI: true, AllowExtensions: true}},
		{Scheme: "stream+v1:", Privileges: Privileges{Stream: true, CorsEnabled: true}},
	})
	if err != nil {
		t.Fatalf("RegisterSchemesAsPrivileged() error = %v", err)
	}
	privileges, ok := registry.PrivilegesFor("app:")
	if !ok {
		t.Fatal("PrivilegesFor(app) ok = false")
	}
	if !privileges.Standard || !privileges.Secure || !privileges.SupportFetchAPI || !privileges.AllowExtensions {
		t.Fatalf("app privileges = %#v", privileges)
	}
	stream, ok := registry.PrivilegesFor("stream+v1")
	if !ok || !stream.Stream || !stream.CorsEnabled {
		t.Fatalf("stream privileges = %#v ok=%v", stream, ok)
	}
}

func TestRegisterSchemesRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		in   []SchemePrivilege
		want error
	}{
		{name: "uppercase", in: []SchemePrivilege{{Scheme: "App"}}, want: ErrInvalidScheme},
		{name: "starts with digit", in: []SchemePrivilege{{Scheme: "1app"}}, want: ErrInvalidScheme},
		{name: "allowExtensions non-standard", in: []SchemePrivilege{{Scheme: "app", Privileges: Privileges{AllowExtensions: true}}}, want: ErrInvalidPrivileges},
		{name: "duplicate", in: []SchemePrivilege{{Scheme: "app"}, {Scheme: "app:"}}, want: ErrAlreadyRegistered},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := NewRegistry().RegisterSchemesAsPrivileged(tt.in); !errors.Is(err, tt.want) {
				t.Fatalf("RegisterSchemesAsPrivileged() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestPrivilegeRegistrationLocksAfterReady(t *testing.T) {
	registry := NewRegistry()
	registry.LockPrivileges()
	if err := registry.RegisterSchemesAsPrivileged([]SchemePrivilege{{Scheme: "app"}}); !errors.Is(err, ErrRegistryLocked) {
		t.Fatalf("RegisterSchemesAsPrivileged(locked) error = %v, want ErrRegistryLocked", err)
	}
}

func TestProtocolHandlers(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterHandler("app", Handler{Kind: HandlerString}); err != nil {
		t.Fatalf("RegisterHandler() error = %v", err)
	}
	if !registry.IsProtocolHandled("app:") {
		t.Fatal("IsProtocolHandled(app) = false")
	}
	if err := registry.RegisterHandler("app", Handler{Kind: HandlerBuffer}); !errors.Is(err, ErrAlreadyRegistered) {
		t.Fatalf("RegisterHandler(duplicate) error = %v, want ErrAlreadyRegistered", err)
	}
	if err := registry.UnregisterProtocol("app"); err != nil {
		t.Fatalf("UnregisterProtocol() error = %v", err)
	}
	if registry.IsProtocolHandled("app") {
		t.Fatal("IsProtocolHandled(app) = true after unregister")
	}
	if err := registry.UnregisterProtocol("app"); !errors.Is(err, ErrNotRegistered) {
		t.Fatalf("UnregisterProtocol(missing) error = %v, want ErrNotRegistered", err)
	}
}

func TestValidateResponse(t *testing.T) {
	if err := ValidateResponse(Response{StatusCode: http.StatusCreated, Headers: http.Header{"X-App": []string{"ok"}}}); err != nil {
		t.Fatalf("ValidateResponse(valid) error = %v", err)
	}
	cases := []Response{
		{StatusCode: 99},
		{StatusCode: 600},
		{StatusCode: http.StatusOK, Headers: http.Header{"Bad:Name": []string{"ok"}}},
		{StatusCode: http.StatusOK, Headers: http.Header{"X-App": []string{"bad\r\nvalue"}}},
	}
	for _, tc := range cases {
		if err := ValidateResponse(tc); !errors.Is(err, ErrInvalidResponse) {
			t.Fatalf("ValidateResponse(%#v) error = %v, want ErrInvalidResponse", tc, err)
		}
	}
}
