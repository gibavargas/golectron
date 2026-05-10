package apppaths

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrProtocolRequired    = errors.New("protocol is required")
	ErrInvalidProtocol     = errors.New("invalid protocol")
	ErrProtocolUnavailable = errors.New("protocol unavailable")
)

var protocolNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

type ProtocolRegistration struct {
	Protocol string
	Path     string
	Args     []string
}

func (s *Store) SetAsDefaultProtocolClient(protocol string, path string, args []string) (bool, error) {
	registration, err := s.protocolRegistration(protocol, path, args)
	if err != nil {
		return false, err
	}
	if s.env.GOOS == "darwin" && !s.protocolDeclared(protocol) {
		return false, fmt.Errorf("%w: %s", ErrProtocolUnavailable, protocol)
	}
	s.protocols[registration.Protocol] = registration
	return true, nil
}

func (s *Store) IsDefaultProtocolClient(protocol string, path string, args []string) (bool, error) {
	registration, err := s.protocolRegistration(protocol, path, args)
	if err != nil {
		return false, err
	}
	current, ok := s.protocols[registration.Protocol]
	if !ok {
		return false, nil
	}
	return current.Path == registration.Path && equalStringSlices(current.Args, registration.Args), nil
}

func (s *Store) RemoveAsDefaultProtocolClient(protocol string, path string, args []string) (bool, error) {
	isDefault, err := s.IsDefaultProtocolClient(protocol, path, args)
	if err != nil {
		return false, err
	}
	if !isDefault {
		return false, nil
	}
	delete(s.protocols, canonicalProtocol(protocol))
	return true, nil
}

func (s *Store) protocolRegistration(protocol string, path string, args []string) (ProtocolRegistration, error) {
	protocol = canonicalProtocol(protocol)
	if protocol == "" {
		return ProtocolRegistration{}, ErrProtocolRequired
	}
	if strings.Contains(protocol, "://") || !protocolNamePattern.MatchString(protocol) {
		return ProtocolRegistration{}, fmt.Errorf("%w: %s", ErrInvalidProtocol, protocol)
	}
	if strings.TrimSpace(path) == "" {
		path = s.env.Executable
	}
	if !s.isAbsPath(path) {
		return ProtocolRegistration{}, fmt.Errorf("%w: %s", ErrPathNotAbsolute, path)
	}
	return ProtocolRegistration{
		Protocol: protocol,
		Path:     path,
		Args:     append([]string(nil), args...),
	}, nil
}

func (s *Store) protocolDeclared(protocol string) bool {
	if s.env.DeclaredProtocols == nil {
		return false
	}
	_, ok := s.env.DeclaredProtocols[canonicalProtocol(protocol)]
	return ok
}

func canonicalProtocol(protocol string) string {
	return strings.ToLower(strings.TrimSpace(protocol))
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
