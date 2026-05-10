package session

import (
	"errors"
	"testing"
)

func TestSelectWebAuthnAccountDispatchesSelection(t *testing.T) {
	s := NewRegistry().DefaultSession()
	var got WebAuthnAccountSelection
	s.SetWebAuthnAccountSelectionHandler(func(selection WebAuthnAccountSelection) (string, error) {
		got = selection
		return "credential-2", nil
	})
	selected, err := s.SelectWebAuthnAccount(WebAuthnAccountSelection{
		RequestID: " request ",
		Origin:    " https://example.test ",
		Credentials: []WebAuthnCredential{
			{ID: "credential-1", RelyingParty: " example.test ", UserName: "one"},
			{ID: " credential-2 ", DisplayName: " Two "},
		},
	})
	if err != nil {
		t.Fatalf("SelectWebAuthnAccount() error = %v", err)
	}
	if selected != "credential-2" {
		t.Fatalf("selected = %q, want credential-2", selected)
	}
	if got.RequestID != "request" || got.Origin != "https://example.test" {
		t.Fatalf("normalized selection = %#v", got)
	}
	if got.Credentials[0].RelyingParty != "example.test" || got.Credentials[1].DisplayName != "Two" {
		t.Fatalf("normalized credentials = %#v", got.Credentials)
	}
}

func TestSelectWebAuthnAccountRejectsInvalidSelection(t *testing.T) {
	s := NewRegistry().DefaultSession()
	validCredential := WebAuthnCredential{ID: "credential"}
	cases := []WebAuthnAccountSelection{
		{Origin: "https://example.test", Credentials: []WebAuthnCredential{validCredential}},
		{RequestID: "request", Credentials: []WebAuthnCredential{validCredential}},
		{RequestID: "request", Origin: "https://example.test"},
		{RequestID: "request", Origin: "https://example.test", Credentials: []WebAuthnCredential{{}}},
		{RequestID: "request", Origin: "https://example.test", Credentials: []WebAuthnCredential{validCredential, validCredential}},
	}
	for _, tc := range cases {
		if _, err := s.SelectWebAuthnAccount(tc); !errors.Is(err, ErrInvalidWebAuthnSelection) {
			t.Fatalf("SelectWebAuthnAccount(%#v) error = %v, want ErrInvalidWebAuthnSelection", tc, err)
		}
	}
}

func TestSelectWebAuthnAccountRequiresHandlerAndKnownCredential(t *testing.T) {
	s := NewRegistry().DefaultSession()
	selection := WebAuthnAccountSelection{
		RequestID:   "request",
		Origin:      "https://example.test",
		Credentials: []WebAuthnCredential{{ID: "credential"}},
	}
	if _, err := s.SelectWebAuthnAccount(selection); !errors.Is(err, ErrInvalidWebAuthnSelection) {
		t.Fatalf("SelectWebAuthnAccount(no handler) error = %v, want ErrInvalidWebAuthnSelection", err)
	}
	s.SetWebAuthnAccountSelectionHandler(func(WebAuthnAccountSelection) (string, error) {
		return "missing", nil
	})
	if _, err := s.SelectWebAuthnAccount(selection); !errors.Is(err, ErrInvalidWebAuthnSelection) {
		t.Fatalf("SelectWebAuthnAccount(missing credential) error = %v, want ErrInvalidWebAuthnSelection", err)
	}
}
