package globalshortcut

import (
	"fmt"
	"strings"
)

type Accelerator struct {
	Modifiers []string
	Key       string
}

func ParseAccelerator(raw string) (Accelerator, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Accelerator{}, ErrAcceleratorRequired
	}

	parts := strings.Split(raw, "+")
	modifiers := make([]string, 0, len(parts)-1)
	seen := make(map[string]struct{}, len(parts))
	var key string

	for i, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return Accelerator{}, fmt.Errorf("%w: empty token in %q", ErrInvalidAccelerator, raw)
		}
		if modifier, ok := canonicalModifier(token); ok {
			if key != "" {
				return Accelerator{}, fmt.Errorf("%w: modifier %q appears after key in %q", ErrInvalidAccelerator, token, raw)
			}
			if _, exists := seen[modifier]; exists {
				return Accelerator{}, fmt.Errorf("%w: duplicate modifier %q in %q", ErrInvalidAccelerator, modifier, raw)
			}
			seen[modifier] = struct{}{}
			modifiers = append(modifiers, modifier)
			continue
		}
		if i != len(parts)-1 {
			return Accelerator{}, fmt.Errorf("%w: key %q must be the final token in %q", ErrInvalidAccelerator, token, raw)
		}
		key = canonicalKey(token)
	}

	if key == "" {
		return Accelerator{}, fmt.Errorf("%w: missing key in %q", ErrInvalidAccelerator, raw)
	}
	return Accelerator{Modifiers: modifiers, Key: key}, nil
}

func (a Accelerator) String() string {
	if len(a.Modifiers) == 0 {
		return a.Key
	}
	parts := make([]string, 0, len(a.Modifiers)+1)
	parts = append(parts, a.Modifiers...)
	parts = append(parts, a.Key)
	return strings.Join(parts, "+")
}

func canonicalModifier(token string) (string, bool) {
	switch strings.ToLower(strings.ReplaceAll(token, " ", "")) {
	case "commandorcontrol", "cmdorctrl":
		return "CommandOrControl", true
	case "command", "cmd":
		return "Command", true
	case "control", "ctrl":
		return "Control", true
	case "alt", "option":
		return "Alt", true
	case "altgr":
		return "AltGr", true
	case "shift":
		return "Shift", true
	case "super", "meta":
		return "Super", true
	default:
		return "", false
	}
}

func canonicalKey(token string) string {
	if len([]rune(token)) == 1 {
		return strings.ToUpper(token)
	}
	return token
}
