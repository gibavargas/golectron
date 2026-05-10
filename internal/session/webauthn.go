package session

import (
	"errors"
	"fmt"
	"strings"
)

var ErrInvalidWebAuthnSelection = errors.New("invalid WebAuthn account selection")

type WebAuthnCredential struct {
	ID           string
	RelyingParty string
	UserName     string
	DisplayName  string
}

type WebAuthnAccountSelection struct {
	RequestID   string
	Origin      string
	Credentials []WebAuthnCredential
}

type WebAuthnAccountSelectionHandler func(WebAuthnAccountSelection) (string, error)

func (s *Session) SetWebAuthnAccountSelectionHandler(handler WebAuthnAccountSelectionHandler) {
	s.webauthnHandler = handler
}

func (s *Session) SelectWebAuthnAccount(selection WebAuthnAccountSelection) (string, error) {
	normalized, err := normalizeWebAuthnSelection(selection)
	if err != nil {
		return "", err
	}
	if s.webauthnHandler == nil {
		return "", fmt.Errorf("%w: no handler", ErrInvalidWebAuthnSelection)
	}
	selectedID, err := s.webauthnHandler(normalized)
	if err != nil {
		return "", err
	}
	selectedID = strings.TrimSpace(selectedID)
	for _, credential := range normalized.Credentials {
		if credential.ID == selectedID {
			return selectedID, nil
		}
	}
	return "", fmt.Errorf("%w: unknown credential id", ErrInvalidWebAuthnSelection)
}

func normalizeWebAuthnSelection(selection WebAuthnAccountSelection) (WebAuthnAccountSelection, error) {
	selection.RequestID = strings.TrimSpace(selection.RequestID)
	selection.Origin = strings.TrimSpace(selection.Origin)
	if selection.RequestID == "" {
		return WebAuthnAccountSelection{}, fmt.Errorf("%w: request id is required", ErrInvalidWebAuthnSelection)
	}
	if selection.Origin == "" {
		return WebAuthnAccountSelection{}, fmt.Errorf("%w: origin is required", ErrInvalidWebAuthnSelection)
	}
	if len(selection.Credentials) == 0 {
		return WebAuthnAccountSelection{}, fmt.Errorf("%w: credentials are required", ErrInvalidWebAuthnSelection)
	}
	seen := make(map[string]bool, len(selection.Credentials))
	for i, credential := range selection.Credentials {
		credential.ID = strings.TrimSpace(credential.ID)
		credential.RelyingParty = strings.TrimSpace(credential.RelyingParty)
		credential.UserName = strings.TrimSpace(credential.UserName)
		credential.DisplayName = strings.TrimSpace(credential.DisplayName)
		if credential.ID == "" {
			return WebAuthnAccountSelection{}, fmt.Errorf("%w: credential id is required", ErrInvalidWebAuthnSelection)
		}
		if seen[credential.ID] {
			return WebAuthnAccountSelection{}, fmt.Errorf("%w: duplicate credential id", ErrInvalidWebAuthnSelection)
		}
		seen[credential.ID] = true
		selection.Credentials[i] = credential
	}
	return selection, nil
}
